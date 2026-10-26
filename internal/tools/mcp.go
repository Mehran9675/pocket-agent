package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MCPServer configures a remote Model Context Protocol server reached
// over Streamable HTTP. (Phones can't spawn stdio servers.)
type MCPServer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
	// Headers carries authentication, e.g. Authorization: Bearer ....
	Headers map[string]string `json:"headers,omitempty"`
}

// MCPClient talks to one MCP server.
type MCPClient struct {
	cfg     MCPServer
	http    *http.Client
	mu      sync.Mutex
	session string
	ready   bool
	nextID  atomic.Int64
	tools   []*Tool
}

// NewMCPClient returns a client for cfg; it connects lazily.
func NewMCPClient(cfg MCPServer) *MCPClient {
	return &MCPClient{cfg: cfg, http: &http.Client{Timeout: 90 * time.Second}}
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

// call sends a JSON-RPC request and waits for its response, which may
// come as plain JSON or inside an SSE stream.
func (c *MCPClient) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	msg := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	resp, err := c.post(ctx, msg)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var rr *rpcResponse
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		rr, err = readRPCFromSSE(resp.Body, id)
	} else {
		rr = &rpcResponse{}
		err = json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(rr)
	}
	if err != nil {
		return nil, err
	}
	if rr.Error != nil {
		return nil, fmt.Errorf("%s: %s", c.cfg.Name, rr.Error.Message)
	}
	return rr.Result, nil
}

func (c *MCPClient) notify(ctx context.Context, method string) {
	resp, err := c.post(ctx, map[string]any{"jsonrpc": "2.0", "method": method})
	if err == nil {
		resp.Body.Close()
	}
}

func (c *MCPClient) post(ctx context.Context, msg any) (*http.Response, error) {
	body, _ := json.Marshal(msg)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}
	c.mu.Lock()
	if c.session != "" {
		req.Header.Set("Mcp-Session-Id", c.session)
	}
	c.mu.Unlock()
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if s := resp.Header.Get("Mcp-Session-Id"); s != "" {
		c.mu.Lock()
		c.session = s
		c.mu.Unlock()
	}
	if resp.StatusCode == http.StatusAccepted {
		return resp, nil
	}
	if resp.StatusCode/100 != 2 {
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("%s: unauthorized - add an Authorization header in the server settings", c.cfg.Name)
		}
		return nil, fmt.Errorf("%s: HTTP %d %s", c.cfg.Name, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return resp, nil
}

func readRPCFromSSE(r io.Reader, id int64) (*rpcResponse, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 32<<20)
	var data strings.Builder
	want := fmt.Sprint(id)
	check := func() (*rpcResponse, bool) {
		if data.Len() == 0 {
			return nil, false
		}
		var rr rpcResponse
		raw := data.String()
		data.Reset()
		if json.Unmarshal([]byte(raw), &rr) == nil && strings.Trim(string(rr.ID), `"`) == want {
			return &rr, true
		}
		return nil, false
	}
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "data:") {
			data.WriteString(strings.TrimPrefix(line[5:], " "))
			continue
		}
		if line == "" {
			if rr, ok := check(); ok {
				return rr, nil
			}
		}
	}
	if rr, ok := check(); ok {
		return rr, nil
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("the server closed the stream without answering")
}

func (c *MCPClient) connect(ctx context.Context) error {
	c.mu.Lock()
	ready := c.ready
	c.mu.Unlock()
	if ready {
		return nil
	}
	_, err := c.call(ctx, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "pocket-agent", "version": "1.0.0"},
	})
	if err != nil {
		return err
	}
	c.notify(ctx, "notifications/initialized")
	c.mu.Lock()
	c.ready = true
	c.mu.Unlock()
	return nil
}

var toolNameRE = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// Tools connects if needed and lists the server's tools, adapted to
// Pocket Agent tools named mcp_<server>_<tool>.
func (c *MCPClient) Tools(ctx context.Context) ([]*Tool, error) {
	if err := c.connect(ctx); err != nil {
		return nil, err
	}
	var tools []*Tool
	cursor := ""
	for page := 0; page < 10; page++ {
		var params any
		if cursor != "" {
			params = map[string]any{"cursor": cursor}
		}
		raw, err := c.call(ctx, "tools/list", params)
		if err != nil {
			return nil, err
		}
		var res struct {
			Tools []struct {
				Name        string         `json:"name"`
				Title       string         `json:"title"`
				Description string         `json:"description"`
				InputSchema map[string]any `json:"inputSchema"`
				Annotations struct {
					ReadOnlyHint    bool `json:"readOnlyHint"`
					DestructiveHint bool `json:"destructiveHint"`
				} `json:"annotations"`
			} `json:"tools"`
			NextCursor string `json:"nextCursor"`
		}
		if err := json.Unmarshal(raw, &res); err != nil {
			return nil, err
		}
		prefix := "mcp_" + strings.ToLower(toolNameRE.ReplaceAllString(c.cfg.Name, "_")) + "_"
		for _, t := range res.Tools {
			remote := t.Name
			title := t.Title
			if title == "" {
				title = t.Name
			}
			name := prefix + toolNameRE.ReplaceAllString(t.Name, "_")
			if len(name) > 64 {
				name = name[:64]
			}
			schema := t.InputSchema
			if schema == nil {
				schema = object(map[string]any{})
			}
			tools = append(tools, &Tool{
				Name:        name,
				Title:       title,
				Description: fmt.Sprintf("[%s] %s", c.cfg.Name, t.Description),
				Icon:        "plug",
				Group:       "MCP · " + c.cfg.Name,
				Schema:      schema,
				Approval:    !t.Annotations.ReadOnlyHint,
				Run: func(ctx context.Context, _ *Env, args json.RawMessage) Result {
					return c.CallTool(ctx, remote, args)
				},
			})
		}
		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}
	c.mu.Lock()
	c.tools = tools
	c.mu.Unlock()
	return tools, nil
}

// CallTool runs a tool on the server.
func (c *MCPClient) CallTool(ctx context.Context, name string, args json.RawMessage) Result {
	if err := c.connect(ctx); err != nil {
		return Errorf("%v", err)
	}
	raw, err := c.call(ctx, "tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		// Sessions expire; reconnect once.
		c.mu.Lock()
		c.ready, c.session = false, ""
		c.mu.Unlock()
		if err2 := c.connect(ctx); err2 != nil {
			return Errorf("%v", err)
		}
		if raw, err = c.call(ctx, "tools/call", map[string]any{"name": name, "arguments": args}); err != nil {
			return Errorf("%v", err)
		}
	}
	var res struct {
		Content []struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			MimeType string `json:"mimeType"`
			Resource struct {
				URI  string `json:"uri"`
				Text string `json:"text"`
			} `json:"resource"`
		} `json:"content"`
		StructuredContent json.RawMessage `json:"structuredContent"`
		IsError           bool            `json:"isError"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return Errorf("bad response from %s: %v", c.cfg.Name, err)
	}
	var b strings.Builder
	for _, part := range res.Content {
		switch part.Type {
		case "text":
			b.WriteString(part.Text)
		case "resource":
			b.WriteString(part.Resource.Text)
		case "image", "audio":
			fmt.Fprintf(&b, "[%s content: %s]", part.Type, part.MimeType)
		}
		b.WriteByte('\n')
	}
	if b.Len() == 0 && len(res.StructuredContent) > 0 {
		b.Write(res.StructuredContent)
	}
	return Result{Text: truncate(strings.TrimSpace(b.String()), 30000), IsError: res.IsError}
}

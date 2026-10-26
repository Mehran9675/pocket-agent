package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// httpClient is shared by every provider. Streams can run for minutes,
// so there's no overall timeout; contexts cancel them instead.
var httpClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: 3 * time.Minute,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConnsPerHost:   4,
	},
}

// postJSON sends body to url and returns the response, turning non-2xx
// answers into *APIError.
func postJSON(ctx context.Context, provider, url string, headers map[string]string, body any) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		defer resp.Body.Close()
		return nil, apiError(provider, resp)
	}
	return resp, nil
}

// getJSON fetches url and decodes the JSON answer into out.
func getJSON(ctx context.Context, provider, url string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return apiError(provider, resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// apiError reads the error message out of the usual JSON shapes.
func apiError(provider string, resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	msg := extractErrorMessage(raw)
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		if t := strings.TrimSpace(string(raw)); t != "" && len(t) < 300 {
			msg += ": " + t
		}
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		msg = "authentication failed - check the API key (" + msg + ")"
	case http.StatusTooManyRequests:
		msg = "rate limited - " + msg
	}
	return &APIError{Provider: provider, Status: resp.StatusCode, Message: msg}
}

func extractErrorMessage(raw []byte) string {
	var v struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
		Detail  string          `json:"detail"`
	}
	if json.Unmarshal(raw, &v) != nil {
		// Some servers answer with an array (Gemini streams).
		var arr []json.RawMessage
		if json.Unmarshal(raw, &arr) == nil && len(arr) > 0 {
			return extractErrorMessage(arr[0])
		}
		return ""
	}
	if len(v.Error) > 0 {
		var s string
		if json.Unmarshal(v.Error, &s) == nil {
			return s
		}
		var obj struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(v.Error, &obj) == nil && obj.Message != "" {
			return obj.Message
		}
	}
	if v.Message != "" {
		return v.Message
	}
	return v.Detail
}

// readSSE calls fn with the event name and data of every server-sent
// event in r until the stream ends, fn returns an error, or the data
// is "[DONE]".
func readSSE(r io.Reader, fn func(event, data string) error) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	var event string
	var data strings.Builder
	dispatch := func() error {
		if data.Len() == 0 {
			event = ""
			return nil
		}
		d := data.String()
		data.Reset()
		ev := event
		event = ""
		if d == "[DONE]" {
			return io.EOF
		}
		return fn(ev, d)
	}
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if err := dispatch(); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		case strings.HasPrefix(line, ":"):
			// comment / keep-alive
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(line[len("event:"):])
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(line[len("data:"):], " "))
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if err := dispatch(); err != nil && err != io.EOF {
		return err
	}
	return nil
}

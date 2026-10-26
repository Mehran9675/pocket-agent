package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/mehran9675/pocket-agent/internal/docs"
)

// SearchEngine names a web search backend.
type SearchEngine string

const (
	SearchDuckDuckGo SearchEngine = "duckduckgo"
	SearchBrave      SearchEngine = "brave"
	SearchTavily     SearchEngine = "tavily"
	SearchSearXNG    SearchEngine = "searxng"
)

// SearchConfig selects and configures web search.
type SearchConfig struct {
	Engine SearchEngine `json:"engine"`
	APIKey string       `json:"apiKey,omitempty"`
	// BaseURL is the SearXNG instance.
	BaseURL string `json:"baseURL,omitempty"`
}

var webClient = &http.Client{Timeout: 25 * time.Second}

const userAgent = "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Mobile Safari/537.36 PocketAgent/1.0"

// searchOverride and fetchOverride replace the network in the devmock
// build, which drives the UI offline.
var (
	searchOverride func(query string, n int) ([]Source, error)
	fetchOverride  func(url string) (*Page, error)
)

// Search runs a web search with cfg.
func Search(ctx context.Context, cfg SearchConfig, query string, n int) ([]Source, error) {
	if n <= 0 {
		n = 6
	}
	if searchOverride != nil {
		return searchOverride(query, n)
	}
	switch cfg.Engine {
	case SearchBrave:
		return searchBrave(ctx, cfg, query, n)
	case SearchTavily:
		return searchTavily(ctx, cfg, query, n)
	case SearchSearXNG:
		return searchSearXNG(ctx, cfg, query, n)
	default:
		return searchDuckDuckGo(ctx, query, n)
	}
}

func doJSON(req *http.Request, out any) error {
	req.Header.Set("Accept", "application/json")
	resp, err := webClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("search failed: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func searchBrave(ctx context.Context, cfg SearchConfig, q string, n int) ([]Source, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("Brave Search needs an API key (Settings → Web search)")
	}
	u := "https://api.search.brave.com/res/v1/web/search?count=" + fmt.Sprint(n) + "&q=" + url.QueryEscape(q)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("X-Subscription-Token", cfg.APIKey)
	var out struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := doJSON(req, &out); err != nil {
		return nil, err
	}
	var res []Source
	for _, r := range out.Web.Results {
		res = append(res, Source{Title: stripTags(r.Title), URL: r.URL, Snippet: stripTags(r.Description)})
	}
	return res, nil
}

func searchTavily(ctx context.Context, cfg SearchConfig, q string, n int) ([]Source, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("Tavily needs an API key (Settings → Web search)")
	}
	body, _ := json.Marshal(map[string]any{"query": q, "max_results": n, "search_depth": "basic"})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tavily.com/search", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	var out struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := doJSON(req, &out); err != nil {
		return nil, err
	}
	var res []Source
	for _, r := range out.Results {
		res = append(res, Source{Title: r.Title, URL: r.URL, Snippet: r.Content})
	}
	return res, nil
}

func searchSearXNG(ctx context.Context, cfg SearchConfig, q string, n int) ([]Source, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("set your SearXNG instance URL in Settings → Web search")
	}
	u := strings.TrimRight(cfg.BaseURL, "/") + "/search?format=json&q=" + url.QueryEscape(q)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	var out struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := doJSON(req, &out); err != nil {
		return nil, err
	}
	var res []Source
	for i, r := range out.Results {
		if i >= n {
			break
		}
		res = append(res, Source{Title: r.Title, URL: r.URL, Snippet: r.Content})
	}
	return res, nil
}

// searchDuckDuckGo reads DuckDuckGo's HTML results page; it needs no
// key, which makes it the default.
func searchDuckDuckGo(ctx context.Context, q string, n int) ([]Source, error) {
	form := url.Values{"q": {q}, "kl": {"wt-wt"}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://html.duckduckgo.com/html/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	resp, err := webClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("search failed: HTTP %d", resp.StatusCode)
	}
	doc, err := html.Parse(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var res []Source
	var cur *Source
	var walk func(*html.Node)
	walk = func(nd *html.Node) {
		if len(res) >= n {
			return
		}
		if nd.Type == html.ElementNode {
			class := attrOf(nd, "class")
			switch {
			case nd.DataAtom == atom.A && strings.Contains(class, "result__a"):
				href := attrOf(nd, "href")
				if u, err := url.Parse(href); err == nil {
					if real := u.Query().Get("uddg"); real != "" {
						href = real
					} else if strings.HasPrefix(href, "//") {
						href = "https:" + href
					}
				}
				if strings.Contains(href, "duckduckgo.com/y.js") {
					// sponsored
					cur = nil
					return
				}
				res = append(res, Source{Title: nodeText(nd), URL: href})
				cur = &res[len(res)-1]
				return
			case strings.Contains(class, "result__snippet") && cur != nil:
				cur.Snippet = nodeText(nd)
				return
			}
		}
		for c := nd.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(res) == 0 {
		return nil, errors.New("no results (DuckDuckGo may be rate limiting - try again, or pick another engine in Settings)")
	}
	return res, nil
}

func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

func stripTags(s string) string {
	if !strings.Contains(s, "<") {
		return html.UnescapeString(s)
	}
	n, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	return nodeText(n)
}

// Page is a fetched web page.
type Page struct {
	URL   string
	Title string
	Text  string
}

// Fetch downloads a URL and extracts its readable text.
func Fetch(ctx context.Context, rawURL string) (*Page, error) {
	if fetchOverride != nil {
		return fetchOverride(rawURL)
	}
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("not a web URL: %s", rawURL)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/pdf,text/plain,application/json;q=0.9,*/*;q=0.5")
	resp, err := webClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	ct := resp.Header.Get("Content-Type")
	page := &Page{URL: resp.Request.URL.String()}
	switch {
	case strings.Contains(ct, "pdf"):
		page.Text, _, err = docs.PDFText(data)
		if err != nil {
			return nil, err
		}
		page.Title = u.Path
	case strings.Contains(ct, "html") || ct == "":
		page.Title, page.Text = docs.HTMLText(data)
	case strings.HasPrefix(ct, "text/") || strings.Contains(ct, "json") || strings.Contains(ct, "xml"):
		page.Text = string(data)
	default:
		return nil, fmt.Errorf("unsupported content type %s", ct)
	}
	if page.Title == "" {
		page.Title = u.Host
	}
	return page, nil
}

func webSearchTool() *Tool {
	return &Tool{
		Name:  "web_search",
		Title: "Web search",
		Icon:  "globe",
		Group: "Web",
		Description: "Search the web. Returns titles, URLs and snippets. Use it for anything recent, " +
			"niche or factual you aren't sure about, then fetch_url the most promising results for detail. " +
			"Cite sources as markdown links.",
		Schema: object(map[string]any{
			"query":       str("The search query, phrased like a search engine query."),
			"max_results": integer("How many results to return (1-10, default 6)."),
		}, "query"),
		Run: func(ctx context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Query      string `json:"query"`
				MaxResults int    `json:"max_results"`
			}
			if err := json.Unmarshal(raw, &a); err != nil || strings.TrimSpace(a.Query) == "" {
				return Errorf("query is required")
			}
			n := a.MaxResults
			if n <= 0 || n > 10 {
				n = 6
			}
			res, err := Search(ctx, env.Search, a.Query, n)
			if err != nil {
				return Errorf("search failed: %v", err)
			}
			var b strings.Builder
			for i, r := range res {
				fmt.Fprintf(&b, "[%d] %s\n%s\n%s\n\n", i+1, r.Title, r.URL, r.Snippet)
			}
			return Result{Text: b.String(), Sources: res}
		},
	}
}

func fetchTool() *Tool {
	return &Tool{
		Name:        "fetch_url",
		Title:       "Read web page",
		Icon:        "link",
		Group:       "Web",
		Description: "Download a web page (HTML, PDF, text or JSON) and return its readable text. Long pages are paged: pass offset to continue.",
		Schema: object(map[string]any{
			"url":    str("The http(s) URL to read."),
			"offset": integer("Character offset to start from, for continuing a long page (default 0)."),
		}, "url"),
		Run: func(ctx context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				URL    string `json:"url"`
				Offset int    `json:"offset"`
			}
			if err := json.Unmarshal(raw, &a); err != nil || a.URL == "" {
				return Errorf("url is required")
			}
			p, err := Fetch(ctx, a.URL)
			if err != nil {
				return Errorf("couldn't read %s: %v", a.URL, err)
			}
			text := []rune(p.Text)
			if a.Offset > 0 && a.Offset < len(text) {
				text = text[a.Offset:]
			}
			const page = 12000
			body := string(text)
			more := ""
			if len(text) > page {
				body = string(text[:page])
				more = fmt.Sprintf("\n\n[... %d more characters; call fetch_url again with offset=%d]", len(text)-page, a.Offset+page)
			}
			return Result{
				Text:    fmt.Sprintf("Title: %s\nURL: %s\n\n%s%s", p.Title, p.URL, body, more),
				Sources: []Source{{Title: p.Title, URL: p.URL}},
			}
		},
	}
}

//go:build devmock

package tools

import (
	"fmt"
	"strings"
)

// The devmock build answers searches and fetches from canned data so
// the research and agent views can be exercised offline.
func init() {
	sites := []Source{
		{Title: "Solid-state batteries: where the industry stands", URL: "https://www.example-energy.org/solid-state-2026"},
		{Title: "Toyota outlines sulfide electrolyte production plan", URL: "https://news.example-auto.com/toyota-solid-state"},
		{Title: "QuantumScape QSE-5 B-sample shipments", URL: "https://ir.example-qs.com/qse5-update"},
		{Title: "Review: interface stability in solid electrolytes", URL: "https://journals.example-science.org/sse-review"},
		{Title: "Cost curves for next-generation cells", URL: "https://www.example-bnef.com/battery-costs"},
		{Title: "Samsung SDI pilot line enters validation", URL: "https://www.example-korea-tech.com/sdi-pilot"},
	}
	searchOverride = func(q string, n int) ([]Source, error) {
		out := make([]Source, 0, n)
		for i := 0; i < n && i < len(sites); i++ {
			s := sites[(i+len(q))%len(sites)]
			s.Snippet = "Coverage of " + q + " with figures, timelines and analysis."
			out = append(out, s)
		}
		return out, nil
	}
	fetchOverride = func(url string) (*Page, error) {
		title := url
		for _, s := range sites {
			if s.URL == url {
				title = s.Title
			}
		}
		body := strings.Repeat(fmt.Sprintf("%s. Energy density of 400-500 Wh/kg is targeted for 2027-2028 pilot vehicles; costs remain 2-3x conventional lithium-ion at pilot scale. ", title), 6)
		return &Page{URL: url, Title: title, Text: body}, nil
	}
}

//go:build devmock

package main

import (
	"log"
	"net/http"
	_ "net/http/pprof" // profiles for UI performance work
	"os"
)

// POCKET_AGENT_PPROF=localhost:6060 serves Go profiles in dev builds.
func init() {
	if addr := os.Getenv("POCKET_AGENT_PPROF"); addr != "" {
		go func() { log.Print(http.ListenAndServe(addr, nil)) }()
	}
}

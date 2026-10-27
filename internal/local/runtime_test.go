package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPickBuild(t *testing.T) {
	files := []ManifestFile{
		{OS: "android", Arch: "arm64", Variant: "armv8", URL: "base"},
		{OS: "android", Arch: "arm64", Variant: "armv8.2", URL: "fast"},
		{OS: "linux", Arch: "amd64", URL: "linux"},
	}
	if f := pickBuild(files, "android", "arm64", []string{"armv8.2", "armv8"}); f == nil || f.URL != "fast" {
		t.Fatalf("modern phone got %+v", f)
	}
	if f := pickBuild(files, "android", "arm64", []string{"armv8"}); f == nil || f.URL != "base" {
		t.Fatalf("old phone got %+v", f)
	}
	if f := pickBuild(files, "linux", "amd64", []string{""}); f == nil || f.URL != "linux" {
		t.Fatalf("linux got %+v", f)
	}
	if f := pickBuild(files, "darwin", "arm64", []string{""}); f != nil {
		t.Fatalf("darwin got %+v", f)
	}
}

// engineServer publishes lib as version v of the engine.
func engineServer(t *testing.T, lib []byte, version *string, corrupt bool) *httptest.Server {
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/engine-manifest.json", func(w http.ResponseWriter, r *http.Request) {
		sum := sha256.Sum256(lib)
		if corrupt {
			sum[0] ^= 1
		}
		json.NewEncoder(w).Encode(Manifest{Version: *version, ABI: engineABI, Files: []ManifestFile{{
			OS: runtime.GOOS, Arch: runtime.GOARCH, Variant: cpuVariants()[0],
			URL: srv.URL + "/lib", SHA256: hex.EncodeToString(sum[:]), Size: int64(len(lib)),
		}}})
	})
	mux.HandleFunc("/lib", func(w http.ResponseWriter, r *http.Request) { w.Write(lib) })
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestRuntimeInstallAndUpdate(t *testing.T) {
	if !downloadable() {
		t.Skip("no downloadable engine on this platform")
	}
	dir := t.TempDir()
	version := "v1"
	srv := engineServer(t, []byte("engine one"), &version, false)
	rt := NewRuntime(dir, srv.URL+"/engine-manifest.json")
	if rt.Ready() || rt.Problem() == "" {
		t.Fatal("ready before install")
	}
	if err := rt.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(rt.LibraryPath()); string(got) != "engine one" || !rt.Ready() {
		t.Fatalf("installed %q", got)
	}

	// A new release is installed at launch; the old version is removed.
	version = "v2"
	rt = NewRuntime(dir, srv.URL+"/engine-manifest.json")
	if rt.Status().Installed != "v1" {
		t.Fatal("install not persisted")
	}
	rt.CheckAndUpdate(context.Background())
	if s := rt.Status(); s.Installed != "v2" || s.UpdateAvailable() {
		t.Fatalf("status after update %+v", s)
	}
	if _, err := os.Stat(filepath.Join(dir, "v1")); !os.IsNotExist(err) {
		t.Fatal("old version kept")
	}

	if err := rt.Remove(); err != nil || rt.Ready() {
		t.Fatal("remove failed")
	}
}

func TestRuntimeRejectsCorruptDownload(t *testing.T) {
	if !downloadable() {
		t.Skip("no downloadable engine on this platform")
	}
	version := "v1"
	srv := engineServer(t, []byte("engine"), &version, true)
	rt := NewRuntime(t.TempDir(), srv.URL+"/engine-manifest.json")
	if err := rt.Install(context.Background()); err == nil || rt.Ready() {
		t.Fatal("corrupt engine installed")
	}
}

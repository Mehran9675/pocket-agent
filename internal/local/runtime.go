package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"time"
)

// DefaultManifestURL lists the engine builds of the latest release.
const DefaultManifestURL = "https://github.com/Mehran9675/pocket-agent/releases/latest/download/engine-manifest.json"

// Manifest describes the published engine builds.
type Manifest struct {
	// Version identifies the build (llama.cpp commit and shim revision);
	// any change is an update.
	Version string `json:"version"`
	// ABI is the PA_ABI_VERSION the builds implement.
	ABI   int            `json:"abi"`
	Files []ManifestFile `json:"files"`
}

// ManifestFile is one platform's engine library.
type ManifestFile struct {
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Variant string `json:"variant,omitempty"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
}

// RuntimeStatus is what the UI shows about the engine.
type RuntimeStatus struct {
	// Supported is false where no engine can run: iOS builds without
	// the engine linked in, and platforms with no published build.
	Supported bool
	// Builtin is set when the engine is part of the app (iOS).
	Builtin bool
	// Installed is the installed version, "" if none.
	Installed string
	// Latest is the newest published version, "" until checked; Size
	// is its download size.
	Latest string
	Size   int64
	// Loaded is the version running in this process, "" until a model
	// is loaded. When it differs from Installed, the update applies on
	// the next launch.
	Loaded string

	Checking    bool
	Downloading bool
	Done        int64
	Err         string
}

// UpdateAvailable reports whether a newer engine than the installed
// one is published.
func (s RuntimeStatus) UpdateAvailable() bool {
	return s.Installed != "" && s.Latest != "" && s.Latest != s.Installed
}

// RestartPending reports whether an installed update waits for the
// next launch.
func (s RuntimeStatus) RestartPending() bool {
	return s.Loaded != "" && s.Installed != "" && s.Loaded != s.Installed && !s.Builtin
}

// installedEngine is <dir>/current.json.
type installedEngine struct {
	Version string `json:"version"`
	File    string `json:"file"`
	SHA256  string `json:"sha256"`
}

// Runtime manages the on-device engine library: on Android and
// desktops it is downloaded from the app's releases, verified,
// installed under dir and updated when a new build is published; on
// iOS it is part of the app.
type Runtime struct {
	dir         string
	manifestURL string
	client      *http.Client

	// OnChange runs (on any goroutine) whenever the status changes.
	OnChange func()

	mu      sync.Mutex
	local   string
	current installedEngine
	latest  *ManifestFile
	st      RuntimeStatus
	cancel  context.CancelFunc
}

// NewRuntime opens the engine directory dir. manifestURL "" uses
// DefaultManifestURL.
func NewRuntime(dir, manifestURL string) *Runtime {
	if manifestURL == "" {
		manifestURL = DefaultManifestURL
	}
	r := &Runtime{
		dir: dir, manifestURL: manifestURL,
		client: &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, ResponseHeaderTimeout: time.Minute}},
	}
	r.st.Builtin = builtinEngine
	r.st.Supported = builtinEngine || downloadable()
	if b, err := os.ReadFile(filepath.Join(dir, "current.json")); err == nil && json.Unmarshal(b, &r.current) == nil {
		if _, err := os.Stat(filepath.Join(dir, r.current.Version, r.current.File)); err == nil {
			r.st.Installed = r.current.Version
		} else {
			r.current = installedEngine{}
		}
	}
	return r
}

// UseLocal runs the engine library at path - one built with
// scripts/build-engine.sh - instead of downloading one.
func (r *Runtime) UseLocal(path string) {
	r.mu.Lock()
	r.local = path
	r.mu.Unlock()
	r.update(func(s *RuntimeStatus) { s.Supported, s.Installed = true, "local" })
}

// downloadable reports whether this platform runs a downloaded engine.
// iOS doesn't allow it.
func downloadable() bool {
	switch runtime.GOOS {
	case "android":
		return runtime.GOARCH == "arm64"
	case "linux", "darwin":
		return true
	}
	return false
}

// libraryName is the engine file name on this platform.
func libraryName() string {
	if runtime.GOOS == "darwin" {
		return "libpocketllama.dylib"
	}
	return "libpocketllama.so"
}

// Status returns a snapshot of the engine's state.
func (r *Runtime) Status() RuntimeStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.st
}

// Ready reports whether a model can be loaded.
func (r *Runtime) Ready() bool {
	s := r.Status()
	return s.Builtin || s.Installed != ""
}

// Problem explains why Ready is false.
func (r *Runtime) Problem() string {
	s := r.Status()
	switch {
	case s.Builtin || s.Installed != "":
		return ""
	case !s.Supported:
		return "This build can't run models on the device. Use a cloud model or Ollama / LM Studio on your network."
	case s.Downloading:
		return "The on-device engine is downloading."
	default:
		return "The on-device engine isn't installed yet. Download it to run models on this device."
	}
}

// LibraryPath is the installed engine library, "" if none.
func (r *Runtime) LibraryPath() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.local != "" {
		return r.local
	}
	if r.current.Version == "" {
		return ""
	}
	return filepath.Join(r.dir, r.current.Version, r.current.File)
}

func (r *Runtime) setLoaded(version string) {
	r.update(func(s *RuntimeStatus) { s.Loaded = version })
}

func (r *Runtime) update(fn func(s *RuntimeStatus)) {
	r.mu.Lock()
	fn(&r.st)
	r.mu.Unlock()
	if r.OnChange != nil {
		r.OnChange()
	}
}

// Check fetches the manifest and records the newest build for this
// device.
func (r *Runtime) Check(ctx context.Context) error {
	if builtinEngine || !downloadable() {
		return nil
	}
	r.update(func(s *RuntimeStatus) { s.Checking, s.Err = true, "" })
	f, version, err := r.fetch(ctx)
	r.mu.Lock()
	r.latest = f
	r.mu.Unlock()
	r.update(func(s *RuntimeStatus) {
		s.Checking = false
		if err != nil {
			s.Err = err.Error()
			return
		}
		s.Latest, s.Size = version, f.Size
	})
	return err
}

func (r *Runtime) fetch(ctx context.Context) (*ManifestFile, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.manifestURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, "", errors.New("couldn't check for the engine - check your internet connection")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("couldn't check for the engine: HTTP %d", resp.StatusCode)
	}
	var m Manifest
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m); err != nil {
		return nil, "", fmt.Errorf("the engine list is malformed: %w", err)
	}
	if m.ABI != engineABI {
		return nil, "", errors.New("the latest engine needs a newer version of Pocket Agent - update the app")
	}
	f := pickBuild(m.Files, runtime.GOOS, runtime.GOARCH, cpuVariants())
	if f == nil {
		return nil, "", fmt.Errorf("no engine is published for %s/%s yet", runtime.GOOS, runtime.GOARCH)
	}
	return f, m.Version, nil
}

// pickBuild chooses the build for a platform, preferring variants in
// order.
func pickBuild(files []ManifestFile, goos, goarch string, variants []string) *ManifestFile {
	var best *ManifestFile
	rank := len(variants)
	for i := range files {
		f := &files[i]
		if f.OS != goos || f.Arch != goarch {
			continue
		}
		if k := slices.Index(variants, f.Variant); k >= 0 && k < rank {
			best, rank = f, k
		}
	}
	return best
}

// Install downloads the latest build (checking first if needed),
// verifies its checksum and makes it the installed engine.
func (r *Runtime) Install(ctx context.Context) error {
	r.mu.Lock()
	if r.st.Downloading {
		r.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.mu.Unlock()
	defer cancel()

	if r.Status().Latest == "" {
		if err := r.Check(ctx); err != nil {
			return err
		}
	}
	r.mu.Lock()
	f, version := r.latest, r.st.Latest
	r.mu.Unlock()
	if f == nil || version == r.Status().Installed {
		return nil
	}
	r.update(func(s *RuntimeStatus) { s.Downloading, s.Done, s.Err = true, 0, "" })
	err := r.download(ctx, f, version)
	r.update(func(s *RuntimeStatus) {
		s.Downloading = false
		switch {
		case err == nil:
			s.Installed = version
		case errors.Is(err, context.Canceled):
		default:
			s.Err = err.Error()
		}
	})
	return err
}

// Cancel stops a download.
func (r *Runtime) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
	}
}

// CheckAndUpdate runs at launch: it checks for a new build and, when an
// engine is installed, installs the update. A first install waits for
// the user.
func (r *Runtime) CheckAndUpdate(ctx context.Context) {
	r.mu.Lock()
	local := r.local
	r.mu.Unlock()
	if local != "" {
		return
	}
	if err := r.Check(ctx); err != nil {
		return
	}
	if r.Status().UpdateAvailable() {
		_ = r.Install(ctx)
	}
}

func (r *Runtime) download(ctx context.Context, f *ManifestFile, version string) error {
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.URL, nil)
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("couldn't download the engine - check your internet connection and try again")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("engine download failed: HTTP %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp(r.dir, "download-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	h := sha256.New()
	var done int64
	last := time.Now()
	buf := make([]byte, 128<<10)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := tmp.Write(buf[:n]); werr != nil {
				tmp.Close()
				return fmt.Errorf("couldn't save the engine (is the device full?): %w", werr)
			}
			h.Write(buf[:n])
			done += int64(n)
		}
		if now := time.Now(); now.Sub(last) > 300*time.Millisecond {
			last = now
			r.update(func(s *RuntimeStatus) { s.Done = done })
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			tmp.Close()
			return rerr
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if f.Size > 0 && done != f.Size {
		return fmt.Errorf("the engine download was incomplete (%d of %d bytes)", done, f.Size)
	}
	if sum := hex.EncodeToString(h.Sum(nil)); sum != f.SHA256 {
		return errors.New("the downloaded engine failed its integrity check - try again")
	}

	vdir := filepath.Join(r.dir, version)
	if err := os.MkdirAll(vdir, 0o755); err != nil {
		return err
	}
	lib := filepath.Join(vdir, libraryName())
	_ = os.Chmod(lib, 0o644)
	if err := os.Rename(tmp.Name(), lib); err != nil {
		return err
	}
	// Loaded code must not be writable (Android 14 enforces it).
	if err := os.Chmod(lib, 0o555); err != nil {
		return err
	}
	cur := installedEngine{Version: version, File: libraryName(), SHA256: f.SHA256}
	b, _ := json.MarshalIndent(cur, "", "  ")
	if err := os.WriteFile(filepath.Join(r.dir, "current.json.tmp"), b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(r.dir, "current.json.tmp"), filepath.Join(r.dir, "current.json")); err != nil {
		return err
	}
	r.mu.Lock()
	r.current = cur
	loaded := r.st.Loaded
	r.mu.Unlock()
	r.prune(version, loaded)
	return nil
}

// prune removes engine versions other than keep and the one running.
func (r *Runtime) prune(keep, running string) {
	entries, _ := os.ReadDir(r.dir)
	for _, e := range entries {
		if e.IsDir() && e.Name() != keep && e.Name() != running {
			_ = os.RemoveAll(filepath.Join(r.dir, e.Name()))
		}
	}
}

// Remove uninstalls the engine; a running one keeps working until the
// app restarts.
func (r *Runtime) Remove() error {
	r.mu.Lock()
	r.current = installedEngine{}
	r.mu.Unlock()
	if err := os.Remove(filepath.Join(r.dir, "current.json")); err != nil && !os.IsNotExist(err) {
		return err
	}
	r.update(func(s *RuntimeStatus) { s.Installed = "" })
	r.prune("", r.Status().Loaded)
	return nil
}

package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Installed describes a model on disk.
type Installed struct {
	File      string    `json:"file"`
	Name      string    `json:"name"`
	Repo      string    `json:"repo,omitempty"`
	Size      int64     `json:"size"`
	Context   int       `json:"context,omitempty"`
	Reasoning bool      `json:"reasoning,omitempty"`
	Added     time.Time `json:"added"`
}

// Download is an in-flight download.
type Download struct {
	File  string
	Name  string
	Done  int64
	Total int64
	// Speed in bytes per second, smoothed.
	Speed float64
	Err   error

	cancel context.CancelFunc
}

// Progress returns 0..1.
func (d Download) Progress() float32 {
	if d.Total <= 0 {
		return 0
	}
	return float32(float64(d.Done) / float64(d.Total))
}

// Store manages the models directory and downloads into it.
type Store struct {
	dir string
	// OnChange runs (on any goroutine) whenever downloads progress or
	// the installed set changes.
	OnChange func()

	mu        sync.Mutex
	manifest  map[string]Installed
	downloads map[string]*Download
}

// NewStore opens the models directory dir.
func NewStore(dir string) *Store {
	_ = os.MkdirAll(dir, 0o755)
	s := &Store{dir: dir, manifest: map[string]Installed{}, downloads: map[string]*Download{}}
	if b, err := os.ReadFile(filepath.Join(dir, "manifest.json")); err == nil {
		_ = json.Unmarshal(b, &s.manifest)
	}
	return s
}

// Dir is the models directory.
func (s *Store) Dir() string { return s.dir }

func (s *Store) changed() {
	if s.OnChange != nil {
		s.OnChange()
	}
}

func (s *Store) saveLocked() {
	b, _ := json.MarshalIndent(s.manifest, "", "  ")
	_ = os.WriteFile(filepath.Join(s.dir, "manifest.json"), b, 0o644)
}

// Installed lists the GGUF files present, newest first.
func (s *Store) Installed() []Installed {
	entries, _ := os.ReadDir(s.dir)
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Installed
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".gguf") {
			continue
		}
		m, ok := s.manifest[e.Name()]
		if !ok {
			info, _ := e.Info()
			m = Installed{File: e.Name(), Name: prettyName(e.Name()), Size: info.Size(), Added: info.ModTime()}
			if c, ok := CatalogByFile(e.Name()); ok {
				m.Name, m.Context, m.Reasoning, m.Repo = c.Name, c.Context, c.Reasoning, c.Repo
			}
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Added.After(out[j].Added) })
	return out
}

// IsInstalled reports whether file is on disk.
func (s *Store) IsInstalled(file string) bool {
	_, err := os.Stat(filepath.Join(s.dir, file))
	return err == nil
}

// prettyName turns "Qwen3-1.7B-Q4_K_M.gguf" into "Qwen3 1.7B Q4_K_M".
func prettyName(file string) string {
	n := strings.TrimSuffix(file, filepath.Ext(file))
	n = strings.ReplaceAll(n, "-", " ")
	return n
}

// Delete removes an installed model.
func (s *Store) Delete(file string) error {
	if strings.ContainsAny(file, `/\`) {
		return errors.New("bad file name")
	}
	err := os.Remove(filepath.Join(s.dir, file))
	s.mu.Lock()
	delete(s.manifest, file)
	s.saveLocked()
	s.mu.Unlock()
	s.changed()
	return err
}

// Downloads snapshots the in-flight downloads.
func (s *Store) Downloads() map[string]Download {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]Download, len(s.downloads))
	for k, d := range s.downloads {
		out[k] = *d
	}
	return out
}

// Cancel stops a download, keeping the partial file for resuming.
func (s *Store) Cancel(file string) {
	s.mu.Lock()
	d := s.downloads[file]
	s.mu.Unlock()
	if d != nil && d.cancel != nil {
		d.cancel()
	}
}

// Import copies a GGUF the user picked into the store.
func (s *Store) Import(name string, r io.Reader) (Installed, error) {
	if !strings.HasSuffix(strings.ToLower(name), ".gguf") {
		name += ".gguf"
	}
	name = filepath.Base(name)
	tmp := filepath.Join(s.dir, name+".part")
	f, err := os.Create(tmp)
	if err != nil {
		return Installed{}, err
	}
	head := make([]byte, 4)
	n, _ := io.ReadFull(r, head)
	if n < 4 || string(head) != "GGUF" {
		f.Close()
		os.Remove(tmp)
		return Installed{}, errors.New("that file isn't a GGUF model")
	}
	size, err := io.Copy(f, io.MultiReader(strings.NewReader(string(head)), r))
	f.Close()
	if err != nil {
		os.Remove(tmp)
		return Installed{}, err
	}
	if err := os.Rename(tmp, filepath.Join(s.dir, name)); err != nil {
		return Installed{}, err
	}
	m := Installed{File: name, Name: prettyName(name), Size: size, Added: time.Now()}
	s.mu.Lock()
	s.manifest[name] = m
	s.saveLocked()
	s.mu.Unlock()
	s.changed()
	return m, nil
}

var dlClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, ResponseHeaderTimeout: time.Minute}}

// Start downloads a model in the background, resuming a partial file.
func (s *Store) Start(meta Installed, rawURL string) {
	s.mu.Lock()
	if _, busy := s.downloads[meta.File]; busy {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	d := &Download{File: meta.File, Name: meta.Name, Total: meta.Size, cancel: cancel}
	s.downloads[meta.File] = d
	s.mu.Unlock()
	s.changed()
	go func() {
		err := s.download(ctx, d, rawURL)
		s.mu.Lock()
		if err == nil {
			meta.Size = d.Total
			meta.Added = time.Now()
			s.manifest[meta.File] = meta
			s.saveLocked()
			delete(s.downloads, meta.File)
		} else if errors.Is(err, context.Canceled) {
			delete(s.downloads, meta.File)
		} else {
			d.Err = err
		}
		s.mu.Unlock()
		s.changed()
	}()
}

// Dismiss clears a failed download.
func (s *Store) Dismiss(file string) {
	s.mu.Lock()
	delete(s.downloads, file)
	s.mu.Unlock()
	s.changed()
}

func (s *Store) download(ctx context.Context, d *Download, rawURL string) error {
	part := filepath.Join(s.dir, d.File+".part")
	var have int64
	if info, err := os.Stat(part); err == nil {
		have = info.Size()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	if have > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", have))
	}
	resp, err := dlClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("couldn't connect to Hugging Face - check your internet connection and try again")
	}
	defer resp.Body.Close()
	flags := os.O_CREATE | os.O_WRONLY
	switch resp.StatusCode {
	case http.StatusPartialContent:
		flags |= os.O_APPEND
	case http.StatusOK:
		have = 0
		flags |= os.O_TRUNC
	case http.StatusRequestedRangeNotSatisfiable:
		// Already complete.
		return os.Rename(part, filepath.Join(s.dir, d.File))
	case http.StatusUnauthorized, http.StatusForbidden:
		return errors.New("this model is gated on Hugging Face - accept its license there, or pick another")
	default:
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > 0 {
		s.mu.Lock()
		d.Total = have + resp.ContentLength
		s.mu.Unlock()
	}
	f, err := os.OpenFile(part, flags, 0o644)
	if err != nil {
		return err
	}
	buf := make([]byte, 256<<10)
	done := have
	last, lastDone := time.Now(), done
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return fmt.Errorf("couldn't save the model (is the device full?): %w", werr)
			}
			done += int64(n)
		}
		if now := time.Now(); now.Sub(last) > 400*time.Millisecond || rerr != nil {
			s.mu.Lock()
			d.Done = done
			inst := float64(done-lastDone) / now.Sub(last).Seconds()
			if d.Speed == 0 {
				d.Speed = inst
			} else {
				d.Speed = d.Speed*0.7 + inst*0.3
			}
			s.mu.Unlock()
			last, lastDone = now, done
			s.changed()
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return rerr
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(part, filepath.Join(s.dir, d.File))
}

// HFModel is a Hugging Face search hit.
type HFModel struct {
	ID        string `json:"id"`
	Downloads int    `json:"downloads"`
	Likes     int    `json:"likes"`
}

// HFFile is a GGUF file in a repo.
type HFFile struct {
	Path string
	Size int64
}

// SearchHF finds GGUF repos on Hugging Face.
func SearchHF(ctx context.Context, query string) ([]HFModel, error) {
	u := "https://huggingface.co/api/models?filter=gguf&sort=downloads&direction=-1&limit=25&search=" + url.QueryEscape(query)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := dlClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Hugging Face search failed: HTTP %d", resp.StatusCode)
	}
	var out []HFModel
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

// RepoFiles lists the GGUF files of a repo, smallest first.
func RepoFiles(ctx context.Context, repo string) ([]HFFile, error) {
	u := "https://huggingface.co/api/models/" + repo + "/tree/main?recursive=true"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := dlClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("couldn't list %s: HTTP %d", repo, resp.StatusCode)
	}
	var tree []struct {
		Type string `json:"type"`
		Path string `json:"path"`
		Size int64  `json:"size"`
		LFS  *struct {
			Size int64 `json:"size"`
		} `json:"lfs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return nil, err
	}
	var files []HFFile
	for _, t := range tree {
		low := strings.ToLower(t.Path)
		if t.Type != "file" || !strings.HasSuffix(low, ".gguf") || strings.Contains(low, "mmproj") {
			continue
		}
		// Split models (-00001-of-00003) aren't supported.
		if strings.Contains(low, "-of-0") {
			continue
		}
		size := t.Size
		if t.LFS != nil {
			size = t.LFS.Size
		}
		files = append(files, HFFile{Path: t.Path, Size: size})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Size < files[j].Size })
	return files, nil
}

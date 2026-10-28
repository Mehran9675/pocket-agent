// Package skills stores installed skills: folders holding a SKILL.md -
// a name and description in YAML frontmatter, then instructions - and
// any files those instructions refer to. Models see each skill's name
// and description and load the rest on demand.
package skills

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Skill is an installed skill.
type Skill struct {
	// Name identifies the skill and names its folder.
	Name        string
	Description string
	// Body is SKILL.md after the frontmatter.
	Body string
	// Files are the skill's other files, slash-separated and relative.
	Files []string
}

// Library is the folder skills are installed in. It caches the list,
// which changes only through its methods.
type Library struct {
	dir   string
	mu    sync.Mutex
	cache []*Skill
}

// Open opens (creating) the library at dir.
func Open(dir string) *Library {
	_ = os.MkdirAll(dir, 0o755)
	return &Library{dir: dir}
}

const (
	maxFile  = 1 << 20 // per file
	maxTotal = 8 << 20 // per skill
)

var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// Slug turns a display name into a skill name.
func Slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	dash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if len(out) > 64 {
		out = strings.TrimRight(out[:64], "-")
	}
	return out
}

// List returns the installed skills sorted by name. Folders without a
// readable SKILL.md are skipped.
func (l *Library) List() []*Skill {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cache != nil {
		return l.cache
	}
	ents, _ := os.ReadDir(l.dir)
	var out []*Skill
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if s, err := l.read(e.Name()); err == nil {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if out == nil {
		out = []*Skill{}
	}
	l.cache = out
	return out
}

func (l *Library) changed() {
	l.mu.Lock()
	l.cache = nil
	l.mu.Unlock()
}

// Get returns the skill named name.
func (l *Library) Get(name string) (*Skill, error) {
	for _, s := range l.List() {
		if s.Name == name {
			return s, nil
		}
	}
	return nil, fmt.Errorf("no skill named %q", name)
}

func (l *Library) read(name string) (*Skill, error) {
	if !nameRE.MatchString(name) {
		return nil, fmt.Errorf("no skill named %q", name)
	}
	dir := filepath.Join(l.dir, name)
	b, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return nil, fmt.Errorf("no skill named %q", name)
	}
	meta, body := parse(string(b))
	s := &Skill{Name: name, Description: meta["description"], Body: body}
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if rel = filepath.ToSlash(rel); rel != "SKILL.md" {
			s.Files = append(s.Files, rel)
		}
		return nil
	})
	return s, nil
}

// Source returns the skill's SKILL.md as written.
func (l *Library) Source(name string) (string, error) {
	if !nameRE.MatchString(name) {
		return "", fmt.Errorf("no skill named %q", name)
	}
	b, err := os.ReadFile(filepath.Join(l.dir, name, "SKILL.md"))
	return string(b), err
}

// ReadFile returns one of a skill's files.
func (l *Library) ReadFile(name, rel string) (string, error) {
	if !nameRE.MatchString(name) {
		return "", fmt.Errorf("no skill named %q", name)
	}
	clean := path.Clean("/" + strings.ReplaceAll(rel, "\\", "/"))[1:]
	if clean == "" {
		return "", errors.New("give a file path")
	}
	b, err := os.ReadFile(filepath.Join(l.dir, name, filepath.FromSlash(clean)))
	if err != nil {
		return "", fmt.Errorf("skill %s has no file %s", name, clean)
	}
	return string(b), nil
}

// Delete removes a skill.
func (l *Library) Delete(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf("no skill named %q", name)
	}
	defer l.changed()
	return os.RemoveAll(filepath.Join(l.dir, name))
}

// Save writes source as the SKILL.md of the skill it names, creating
// the skill or replacing its SKILL.md; prev, when set and different,
// is the skill's old name, whose folder is renamed.
func (l *Library) Save(prev, source string) (*Skill, error) {
	name, err := checkSource(source)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(l.dir, name)
	if prev != "" && prev != name {
		if _, err := os.Stat(dir); err == nil {
			return nil, fmt.Errorf("a skill named %s already exists", name)
		}
		if nameRE.MatchString(prev) {
			if err := os.Rename(filepath.Join(l.dir, prev), dir); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(source), 0o644); err != nil {
		return nil, err
	}
	l.changed()
	return l.Get(name)
}

// Create writes a new skill from its parts.
func (l *Library) Create(name, description, body string) (*Skill, error) {
	name = Slug(name)
	if name == "" {
		return nil, errors.New("give the skill a name")
	}
	if _, err := os.Stat(filepath.Join(l.dir, name)); err == nil {
		return nil, fmt.Errorf("a skill named %s already exists", name)
	}
	return l.Save("", Compose(name, description, body))
}

// Compose builds a SKILL.md.
func Compose(name, description, body string) string {
	description = strings.Join(strings.Fields(description), " ")
	return fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n", name, yamlString(description), strings.TrimSpace(body))
}

// Install adds a skill from a file's bytes: a zip holding a SKILL.md
// (at its root or in one folder), or a SKILL.md itself. An installed
// skill of the same name is replaced.
func (l *Library) Install(filename string, data []byte) (*Skill, error) {
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		return l.installZip(data)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "---") {
		return nil, fmt.Errorf("%s is not a skill: expected a .zip or a SKILL.md with name and description", filename)
	}
	name, err := checkSource(string(data))
	if err != nil {
		return nil, err
	}
	_ = os.RemoveAll(filepath.Join(l.dir, name))
	return l.Save("", string(data))
}

func (l *Library) installZip(data []byte) (*Skill, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("can't read the zip: %w", err)
	}
	// The shallowest SKILL.md marks the skill's root.
	root := ""
	found := false
	for _, f := range zr.File {
		p := path.Clean(f.Name)
		if path.Base(p) != "SKILL.md" || strings.HasPrefix(p, "__MACOSX/") {
			continue
		}
		if dir := path.Dir(p); !found || depth(dir) < depth(root) {
			root, found = dir, true
		}
	}
	if !found {
		return nil, errors.New("the zip has no SKILL.md")
	}
	files := map[string][]byte{}
	total := 0
	for _, f := range zr.File {
		p := path.Clean(f.Name)
		if f.FileInfo().IsDir() || strings.HasPrefix(p, "__MACOSX/") || strings.HasPrefix(p, "../") || path.IsAbs(p) {
			continue
		}
		rel := p
		if root != "." {
			if !strings.HasPrefix(p, root+"/") {
				continue
			}
			rel = strings.TrimPrefix(p, root+"/")
		}
		if f.UncompressedSize64 > maxFile {
			return nil, fmt.Errorf("%s is larger than 1 MB", rel)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(rc, maxFile+1))
		rc.Close()
		if err != nil {
			return nil, err
		}
		if total += len(b); total > maxTotal {
			return nil, errors.New("the skill is larger than 8 MB")
		}
		files[rel] = b
	}
	return l.installFiles(files)
}

// installFiles writes a skill from its files, replacing any installed
// skill of the same name.
func (l *Library) installFiles(files map[string][]byte) (*Skill, error) {
	name, err := checkSource(string(files["SKILL.md"]))
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(l.dir, name)
	tmp := dir + ".installing"
	_ = os.RemoveAll(tmp)
	for rel, b := range files {
		p := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			return nil, err
		}
	}
	_ = os.RemoveAll(dir)
	if err := os.Rename(tmp, dir); err != nil {
		return nil, err
	}
	l.changed()
	return l.Get(name)
}

func depth(dir string) int {
	if dir == "." {
		return 0
	}
	return strings.Count(dir, "/") + 1
}

// InstallURL downloads and installs a skill: a link to a zip or to a
// SKILL.md. GitHub page links are turned into their raw files.
func (l *Library) InstallURL(ctx context.Context, raw string) (*Skill, error) {
	raw = strings.TrimSpace(raw)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	// A GitHub folder comes with all its files when the API allows.
	if owner, repo, ref, dir, ok := githubTree(raw); ok {
		files := map[string][]byte{}
		if err := fetchTree(ctx, owner, repo, ref, dir, "", files, new(int)); err == nil && files["SKILL.md"] != nil {
			return l.installFiles(files)
		}
	}
	u := rawURL(raw)
	data, err := download(ctx, u, maxTotal)
	if err != nil {
		return nil, err
	}
	return l.Install(path.Base(u), data)
}

func download(ctx context.Context, u string, limit int) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("%s is larger than %d MB", path.Base(u), limit>>20)
	}
	return data, nil
}

// githubTree splits a github.com folder link.
func githubTree(u string) (owner, repo, ref, dir string, ok bool) {
	const gh = "https://github.com/"
	if !strings.HasPrefix(u, gh) {
		return
	}
	parts := strings.SplitN(strings.TrimSuffix(strings.TrimPrefix(u, gh), "/"), "/", 5)
	if len(parts) < 4 || parts[2] != "tree" || strings.HasSuffix(u, ".md") {
		return
	}
	if len(parts) == 5 {
		dir = parts[4]
	}
	return parts[0], parts[1], parts[3], dir, true
}

// fetchTree downloads a repository folder through GitHub's contents
// API into files, keyed by path relative to the folder.
func fetchTree(ctx context.Context, owner, repo, ref, dir, rel string, files map[string][]byte, total *int) error {
	api := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, path.Join(dir, rel), ref)
	b, err := download(ctx, api, maxFile)
	if err != nil {
		return err
	}
	var ents []struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Size        int    `json:"size"`
		DownloadURL string `json:"download_url"`
	}
	if err := json.Unmarshal(b, &ents); err != nil {
		return err
	}
	for _, e := range ents {
		p := path.Join(rel, e.Name)
		switch e.Type {
		case "dir":
			if err := fetchTree(ctx, owner, repo, ref, dir, p, files, total); err != nil {
				return err
			}
		case "file":
			if *total += e.Size; *total > maxTotal {
				return errors.New("the skill is larger than 8 MB")
			}
			data, err := download(ctx, e.DownloadURL, maxFile)
			if err != nil {
				return err
			}
			files[p] = data
		}
	}
	return nil
}

// rawURL maps github.com links to files and folders onto downloadable
// ones: a blob to its raw file, a tree to the SKILL.md inside it (the
// fallback when the folder can't be listed).
func rawURL(u string) string {
	const gh = "https://github.com/"
	if !strings.HasPrefix(u, gh) {
		return u
	}
	parts := strings.SplitN(strings.TrimPrefix(u, gh), "/", 5)
	if len(parts) < 5 || (parts[2] != "blob" && parts[2] != "tree") {
		return u
	}
	p := parts[4]
	if parts[2] == "tree" && !strings.HasSuffix(p, "SKILL.md") {
		p = strings.TrimSuffix(p, "/") + "/SKILL.md"
	}
	return "https://raw.githubusercontent.com/" + parts[0] + "/" + parts[1] + "/" + parts[3] + "/" + p
}

// checkSource validates a SKILL.md and returns its skill name.
func checkSource(src string) (string, error) {
	meta, body := parse(src)
	name := Slug(meta["name"])
	if name == "" || !nameRE.MatchString(name) {
		return "", errors.New("SKILL.md needs a name in its frontmatter")
	}
	if strings.TrimSpace(meta["description"]) == "" {
		return "", errors.New("SKILL.md needs a description saying when to use the skill")
	}
	if strings.TrimSpace(body) == "" {
		return "", errors.New("SKILL.md has no instructions after its frontmatter")
	}
	return name, nil
}

// parse splits SKILL.md into its frontmatter's top-level string fields
// and its body. Only the simple YAML skills use is understood: key:
// value lines, quoted values, and folded or literal blocks.
func parse(src string) (map[string]string, string) {
	meta := map[string]string{}
	src = strings.TrimPrefix(strings.ReplaceAll(src, "\r\n", "\n"), "\ufeff")
	if !strings.HasPrefix(src, "---\n") {
		return meta, src
	}
	end := strings.Index(src[4:], "\n---")
	if end < 0 {
		return meta, src
	}
	head := src[4 : 4+end]
	body := strings.TrimLeft(src[4+end+4:], "-")
	body = strings.TrimPrefix(body, "\n")
	lines := strings.Split(head, "\n")
	for i := 0; i < len(lines); i++ {
		ln := lines[i]
		if ln == "" || ln[0] == ' ' || ln[0] == '\t' || ln[0] == '#' {
			continue
		}
		k, v, ok := strings.Cut(ln, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if v == ">" || v == "|" || v == ">-" || v == "|-" {
			var block []string
			for i+1 < len(lines) && (lines[i+1] == "" || lines[i+1][0] == ' ' || lines[i+1][0] == '\t') {
				i++
				block = append(block, strings.TrimSpace(lines[i]))
			}
			sep := " "
			if v[0] == '|' {
				sep = "\n"
			}
			v = strings.TrimSpace(strings.Join(block, sep))
		} else if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
			if ln[len(ln)-1] == '"' {
				v = strings.ReplaceAll(strings.ReplaceAll(v, `\"`, `"`), `\\`, `\`)
			} else {
				v = strings.ReplaceAll(v, "''", "'")
			}
		}
		meta[strings.TrimSpace(k)] = v
	}
	return meta, body
}

// yamlString quotes s when plain YAML would misread it.
func yamlString(s string) string {
	if s == "" || strings.ContainsAny(s, ":#'\"{}[]&*!|>%@`") || strings.HasPrefix(s, "-") {
		return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
	}
	return s
}

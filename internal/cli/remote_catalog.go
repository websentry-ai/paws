package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"testing/fstest"

	"github.com/websentry-ai/paws/internal/catalog"
	"github.com/websentry-ai/paws/internal/pack"
)

// The catalog lives on the paws backend. `paws add` asks it for the slice one name needs, search asks it for matches,
// and every skill install downloads the files by content hash, which pack checks before writing anything.

// maxSkillFiles bounds a skill download: the backend keeps skill directories up to 32 MB, base64 in JSON.
const maxSkillFiles = 48 << 20

// catalogAPI is the backend `add` and `search` ask: PAWS_PORTAL_API, else the one this machine logged in to.
func catalogAPI() (string, error) {
	api := portalAPI()
	if c, err := loadCredentials(); err == nil && c != nil && c.API != "" && api == defaultPortalAPI {
		api = c.API
	}
	if _, err := checkPortalAPI(api); err != nil {
		return "", err
	}
	return api, nil
}

// fetchCatalog GETs a public catalog endpoint into out; limit bounds the body.
func fetchCatalog(ctx context.Context, rawURL string, limit int64, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := portalClient.Do(req)
	if err != nil {
		return fmt.Errorf("the paws catalog is unreachable: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return err
	}
	if int64(len(body)) > limit {
		return fmt.Errorf("the paws catalog sent more than %d bytes", limit)
	}
	if resp.StatusCode == http.StatusNotFound {
		return &catalogNotFound{err: portalError(resp.StatusCode, body)}
	}
	if resp.StatusCode != http.StatusOK {
		return portalError(resp.StatusCode, body)
	}
	return json.Unmarshal(body, out)
}

// catalogNotFound is a 404 from the catalog. It prints as the portal error it wraps.
type catalogNotFound struct{ err error }

func (e *catalogNotFound) Error() string { return e.err.Error() }

// unknownName is what `add` reports when the catalog has no pack, rule file or skill of that name.
type unknownName struct{ name string }

func (e *unknownName) Error() string {
	return fmt.Sprintf("No pack, rule file or skill named %s.", printable(e.name))
}

// remoteFile is a file as the backend sends it: base64, and whether it is a runnable script.
type remoteFile struct {
	Path string `json:"path"`
	Data string `json:"data"`
	Exec bool   `json:"exec"`
}

// addFiles puts files into m under dir, refusing a path that leaves it. A backslash, colon or NUL is refused too:
// Windows reads \ as a separator and C: as a drive, so a\..\..\x would leave the skill folder there.
func addFiles(m fstest.MapFS, dir string, files []remoteFile) error {
	for _, f := range files {
		clean := path.Clean(f.Path)
		if f.Path == "" || clean != f.Path || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") ||
			strings.ContainsAny(f.Path, "\\:\x00") {
			return fmt.Errorf("the paws catalog sent a bad file path %q", printable(f.Path))
		}
		data, err := base64.StdEncoding.DecodeString(f.Data)
		if err != nil {
			return fmt.Errorf("%s: %w", printable(f.Path), err)
		}
		mode := fs.FileMode(0o644)
		if f.Exec {
			mode = 0o755
		}
		m[path.Join(dir, clean)] = &fstest.MapFile{Data: data, Mode: mode}
	}
	return nil
}

// skillFilesFrom returns pack.SkillFiles for one backend: GET /skill-files/<hash>.
func skillFilesFrom(api string) func(ctx context.Context, hash string) (fs.FS, error) {
	return func(ctx context.Context, hash string) (fs.FS, error) {
		var got struct {
			Files []remoteFile `json:"files"`
		}
		if err := fetchCatalog(ctx, api+"/paws/v1/skill-files/"+url.PathEscape(hash), maxSkillFiles, &got); err != nil {
			return nil, err
		}
		m := fstest.MapFS{}
		if err := addFiles(m, ".", got.Files); err != nil {
			return nil, err
		}
		return m, nil
	}
}

// catalogText is a rule file or digest: its path under catalog/ and its text.
type catalogText struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

// catalogSlice is GET /catalog/source: what `paws add <name>` installs.
type catalogSlice struct {
	Packs []struct {
		Name string `json:"name"`
		Text string `json:"text"`
	} `json:"packs"`
	Bindings []catalogText `json:"bindings"`
	Digests  []catalogText `json:"digests"`
	Bundled  []struct {
		Dir         string `json:"dir"`
		ContentHash string `json:"content_hash"`
	} `json:"bundled"`
}

// remoteCatalog builds, from the backend, a catalog holding just what these names need: each pack or rule file
// named, every rule file of the skills they involve with their digests, and those skills' bundled files. It is validated like a
// catalog on disk.
func remoteCatalog(api string, names []string) (*catalog.Catalog, error) {
	m := fstest.MapFS{}
	for _, name := range names {
		var s catalogSlice
		if err := fetchCatalog(portalCtx, api+"/paws/v1/catalog/source?name="+url.QueryEscape(name), maxPortalResponse, &s); err != nil {
			var missing *catalogNotFound
			if errors.As(err, &missing) {
				return nil, &unknownName{name: name}
			}
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		for _, p := range s.Packs {
			if !catalog.ValidSkillName(p.Name) {
				return nil, fmt.Errorf("the paws catalog sent a bad pack name %q", printable(p.Name))
			}
			m["packs/"+p.Name+".yaml"] = &fstest.MapFile{Data: []byte(p.Text), Mode: 0o644}
		}
		for _, b := range append(s.Bindings, s.Digests...) {
			if err := addFiles(m, "catalog", []remoteFile{{Path: b.Path, Data: base64.StdEncoding.EncodeToString([]byte(b.Text))}}); err != nil {
				return nil, err
			}
		}
		for _, b := range s.Bundled {
			if dir := path.Clean(b.Dir); dir != b.Dir || path.IsAbs(dir) || strings.HasPrefix(dir, "..") || dir == "." {
				return nil, fmt.Errorf("the paws catalog sent a bad skill directory %q", printable(b.Dir))
			}
			// A skill Unbound wrote comes by its hash, like any other skill, and must hash to it.
			files, err := skillFilesFor(api)(portalCtx, b.ContentHash)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", printable(b.Dir), err)
			}
			if got, err := pack.HashSkill(files); err != nil || got != b.ContentHash {
				return nil, fmt.Errorf("the files of %s don't hash to the catalog's %q; refusing to install", printable(b.Dir), printable(b.ContentHash))
			}
			if err := copyInto(m, path.Join("catalog", b.Dir), files); err != nil {
				return nil, err
			}
		}
	}
	c, probs := catalog.ValidateFS(m, "paws catalog")
	if len(probs) > 0 {
		return nil, fmt.Errorf("the paws catalog sent a catalog with problems: %v", probs[0])
	}
	return c, nil
}

// copyInto puts every file of fsys into m under dir, keeping its mode.
func copyInto(m fstest.MapFS, dir string, fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		m[path.Join(dir, p)] = &fstest.MapFile{Data: data, Mode: info.Mode()}
		return nil
	})
}

// catalogRow is one search result from GET /catalog/skills.
type catalogRow struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Add         string `json:"add"`
}

func searchCatalog(api, query string) ([]catalogRow, int, error) {
	var page struct {
		Skills []catalogRow `json:"skills"`
		Total  int          `json:"total"`
	}
	err := fetchCatalog(portalCtx, api+"/paws/v1/catalog/skills?limit=20&q="+url.QueryEscape(query), maxPortalResponse, &page)
	return page.Skills, page.Total, err
}

// catalogNames lists every name `add` accepts that the catalog listing shows: each skill's name and each target of
// its install command (a pack such as "writing", or a rule file such as "testing/tdd"). It pages through the listing.
func catalogNames(api string) ([]string, error) {
	seen := map[string]bool{}
	var names []string
	offset := 0
	for page := 0; page < 20; page++ {
		var rows struct {
			Skills     []catalogRow `json:"skills"`
			NextOffset *int         `json:"next_offset"`
		}
		if err := fetchCatalog(portalCtx, fmt.Sprintf("%s/paws/v1/catalog/skills?limit=50&offset=%d", api, offset), maxPortalResponse, &rows); err != nil {
			return names, err
		}
		for _, r := range rows.Skills {
			cands := append([]string{r.Name}, strings.Fields(strings.TrimPrefix(r.Add, cliName+" add"))...)
			for _, n := range cands {
				if n != "" && !seen[n] {
					seen[n] = true
					names = append(names, n)
				}
			}
		}
		if rows.NextOffset == nil {
			break
		}
		offset = *rows.NextOffset
	}
	return names, nil
}

// closestNames gives up to three of names within a small edit distance of name, nearest first.
func closestNames(name string, names []string) []string {
	name = strings.ToLower(name)
	limit := max(2, len(name)/3)
	type hit struct {
		name string
		dist int
	}
	var hits []hit
	for _, n := range names {
		if d := editDistance(name, strings.ToLower(n)); d <= limit {
			hits = append(hits, hit{n, d})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].dist != hits[j].dist {
			return hits[i].dist < hits[j].dist
		}
		return hits[i].name < hits[j].name
	})
	var out []string
	for i := 0; i < len(hits) && i < 3; i++ {
		out = append(out, hits[i].name)
	}
	return out
}

// editDistance is the Levenshtein distance between a and b, counted in bytes.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// skillFilesFor gives the skill-file source for a backend; tests replace it with files of their own.
var skillFilesFor = skillFilesFrom

// useSkillFiles makes skill installs download from this backend.
func useSkillFiles(api string) { pack.SkillFiles = skillFilesFor(api) }

package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
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
	if resp.StatusCode != http.StatusOK {
		return portalError(resp.StatusCode, body)
	}
	return json.Unmarshal(body, out)
}

// remoteFile is a file as the backend sends it: base64, and whether it is a runnable script.
type remoteFile struct {
	Path string `json:"path"`
	Data string `json:"data"`
	Exec bool   `json:"exec"`
}

// addFiles puts files into m under dir, refusing a path that leaves it.
func addFiles(m fstest.MapFS, dir string, files []remoteFile) error {
	for _, f := range files {
		clean := path.Clean(f.Path)
		if f.Path == "" || clean != f.Path || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
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
		Dir         string       `json:"dir"`
		ContentHash string       `json:"content_hash"`
		Files       []remoteFile `json:"files"`
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
			// A skill Unbound wrote installs from these files: they must hash to its pin, like any other skill.
			files := fstest.MapFS{}
			if err := addFiles(files, ".", b.Files); err != nil {
				return nil, err
			}
			if got, err := pack.HashSkill(files); err != nil || got != b.ContentHash {
				return nil, fmt.Errorf("the files of %s don't hash to the catalog's %q; refusing to install", printable(b.Dir), printable(b.ContentHash))
			}
			if err := addFiles(m, path.Join("catalog", b.Dir), b.Files); err != nil {
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

// skillFilesFor gives the skill-file source for a backend; tests replace it with files of their own.
var skillFilesFor = skillFilesFrom

// useSkillFiles makes skill installs download from this backend.
func useSkillFiles(api string) { pack.SkillFiles = skillFilesFor(api) }

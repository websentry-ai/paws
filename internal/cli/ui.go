package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/websentry-ai/paws"
	"github.com/websentry-ai/paws/internal/pack"
)

// UI serves the local picker: `paws ui [--port N] [--no-open]`. Read-only by design: it shows
// the catalog, what is installed and what fired, with `paws add` commands to copy. It binds
// to 127.0.0.1 and answers only requests addressed to that host, so another site can't read
// it through DNS rebinding.
func UI(args []string) int {
	port, open := 0, true
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--no-open":
			open = false
		case "--port":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "paws: --port needs a value")
				return 2
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "paws: bad port %q\n", args[i+1])
				return 2
			}
			port = n
			i++
		default:
			fmt.Fprintf(os.Stderr, "paws: unknown flag %q\nusage: paws ui [--port N] [--no-open]\n", args[i])
			return 2
		}
	}
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	addr := ln.Addr().String()
	url := "http://" + addr + "/"
	fmt.Printf("paws ui on %s (Ctrl-C to stop)\n", url)
	if open {
		openBrowser(url)
	}
	if err := http.Serve(ln, uiHandler(addr)); err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	return 0
}

// uiHandler serves the embedded page and its JSON endpoints for a server listening on addr.
func uiHandler(addr string) http.Handler {
	_, port, _ := net.SplitHostPort(addr)
	allowed := map[string]bool{"127.0.0.1:" + port: true, "localhost:" + port: true}
	page, _ := fs.Sub(paws.Web, "web/dist")
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(page)))
	mux.HandleFunc("/api/catalog", func(w http.ResponseWriter, r *http.Request) {
		c, err := loadCatalog("")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, buildIndex(c))
	})
	mux.HandleFunc("/api/installed", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, installedView())
	})
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		st, log := localStats()
		writeJSON(w, map[string]any{"stats": st, "log": log})
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowed[r.Host] {
			http.Error(w, "paws ui answers only on its own address", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "read-only", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}

type installedBinding struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Fires int    `json:"fires"`
}

type installedPack struct {
	Name     string             `json:"name"`
	Scope    string             `json:"scope"`
	At       string             `json:"at"`
	Picks    map[string]string  `json:"picks,omitempty"`
	Skills   []string           `json:"skills,omitempty"`
	Bindings []installedBinding `json:"bindings"`
}

// installedView lists user and project installs with each binding's local fire count.
func installedView() []installedPack {
	st, _ := localStats()
	fires := map[string]int{}
	for _, r := range st.Rules {
		fires[r.Rule] = r.Fires
	}
	out := []installedPack{}
	for _, project := range []bool{false, true} {
		t := target(project)
		state, err := pack.LoadState(t.StateFile)
		if err != nil {
			continue
		}
		scope := "user"
		if project {
			scope = "project"
		}
		for _, n := range sortedKeys(state.Packs) {
			inst := state.Packs[n]
			ip := installedPack{Name: n, Scope: scope, At: inst.At, Picks: inst.Picks, Skills: inst.Skills, Bindings: []installedBinding{}}
			for _, rel := range inst.Bindings {
				name := ruleName(filepath.Join(pack.InstallDir(t, n), filepath.FromSlash(rel)))
				ip.Bindings = append(ip.Bindings, installedBinding{Path: rel, Name: name, Fires: fires[name]})
			}
			out = append(out, ip)
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil && !strings.Contains(err.Error(), "not found") {
		fmt.Fprintf(os.Stderr, "paws: open the page yourself: %v\n", err)
	}
}

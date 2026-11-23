//go:build chrome

package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/web"
)

// importedSite is a site whose sheet imports every way an @import can:
// each element's width says which rule won in it.
var importedSite = map[string]string{ //nolint:gochecknoglobals // the site both browsers read
	"/css/site.css": `@layer base;
@import url(parts/theme.css) layer(theme) (min-width: 700px);
@import "parts/grid.css" supports(display: grid);
@import url(parts/never.css) supports(display: bogus);
@import url(cycle.css);
@import url(plain.css);
@import url(untyped.css);
@import url(gone.css);
@import url(parts/anon.css) layer;
@import url(one.css);
@import url(two.css);
@import url(one.css);
@layer base { #layers, #anon { width: 10px; } }
div { height: 10px; }
#order { width: 30px; }
@import url(late.css);`,
	"/css/parts/theme.css": `@import "deep.css"; #layers { width: 50px; } #media { width: 55px; }`,
	"/css/parts/deep.css":  `#deep { width: 60px; }`,
	"/css/parts/grid.css":  `#supports { width: 70px; }`,
	"/css/parts/never.css": `#never { width: 80px; }`,
	"/css/cycle.css":       `@import url(site.css); #cycle { width: 90px; }`,
	"/css/parts/anon.css":  `#anon { width: 110px; }`,
	"/css/one.css":         `#twice { width: 120px; } #order { width: 130px; }`,
	"/css/two.css":         `#twice { width: 125px; }`,
	"/css/late.css":        `#late { width: 140px; }`,
}

// TestImportsCascadeAsChromes lays out one page in ui and in Chrome,
// both following its @import rules from the same server, at a width
// where the media of an import holds and at one where it doesn't.
func TestImportsCascadeAsChromes(t *testing.T) {
	mux := http.NewServeMux()
	site := map[string]bool{"/css/plain.css": true, "/css/untyped.css": true}
	for path, text := range importedSite {
		site[path] = true
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/css")
			_, _ = w.Write([]byte(text))
		})
	}
	mux.HandleFunc("/css/plain.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(`#plain { width: 100px; }`))
	})
	mux.HandleFunc("/css/untyped.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header()["Content-Type"] = nil
		_, _ = w.Write([]byte(`#untyped { width: 105px; }`))
	})
	var (
		mu     sync.Mutex
		served []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if site[r.URL.Path] {
			mu.Lock()
			served = append(served, r.URL.Path)
			mu.Unlock()
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	ids := []string{"layers", "media", "deep", "supports", "never", "cycle", "plain", "untyped", "anon", "twice",
		"order", "late"}
	var body strings.Builder
	for _, id := range ids {
		body.WriteString(`<div id="` + id + `"></div>`)
	}
	sheet := `@import url(css/site.css); body { margin: 0; }`
	mux.HandleFunc("/index.html", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><style>` + sheet + `</style>` + chrome.EvalScript(`return {
	boxes: [...document.body.children].map(e => { const r = e.getBoundingClientRect(); return [r.width, r.y]; }),
	fetched: performance.getEntriesByType("resource").map(e => [new URL(e.name).pathname, e.responseStatus]),
};`) + `</head><body>` + body.String() + `</body></html>`))
	})
	for _, size := range []geom.Vec2{geom.V2(800, 600), geom.V2(600, 600)} {
		// A loaded runner's Chrome can let a fetch from the local server
		// fail, and lays the page out without that sheet: Chrome's page
		// counts only once every sheet the server sent came to it whole.
		var want chromePage
		for try := 0; ; try++ {
			mu.Lock()
			served = nil
			mu.Unlock()
			out, err := chrome.EvalPage(srv.URL+"/index.html", size)
			require.NoError(t, err)
			want = chromePage{}
			require.NoError(t, json.Unmarshal(out, &want))
			mu.Lock()
			missing := want.missing(site, served)
			mu.Unlock()
			if len(missing) == 0 {
				break
			}
			require.Less(t, try, 4, "%v: Chrome never had every sheet, the last time not %v", size, missing)
		}

		d := ui.NewDocument(ui.BrowserSheets(), ui.FetchSheets(web.Sheets(srv.URL+"/")),
			ui.CSS(sheet, ui.From(srv.URL+"/index.html")))
		var els []*ui.Node
		for _, id := range ids {
			els = append(els, ui.Div(ui.ID(id)))
		}
		d.Body().Append(els...)
		require.Eventually(t, func() bool {
			d.Layout(size)
			return d.SheetsLoading() == 0
		}, 5*time.Second, time.Millisecond)
		d.Layout(size)
		require.Len(t, want.Boxes, len(els))
		for i, n := range els {
			assert.InDelta(t, want.Boxes[i][0], n.Box().W, chrome.Tolerance, "%v #%s", size, ids[i])
			assert.InDelta(t, want.Boxes[i][1], n.Box().Y, chrome.Tolerance, "%v #%s", size, ids[i])
		}
	}
}

// chromePage is what Chrome laid out: each element's width and top, and
// the path and HTTP status of every resource it fetched, 0 where the
// fetch failed.
type chromePage struct {
	Boxes   [][2]float32 `json:"boxes"`
	Fetched [][2]any     `json:"fetched"`
}

// missing returns the sheets of the site, which every path of served
// is, that Chrome has no answer for, and those it fetched and lost.
func (p chromePage) missing(site map[string]bool, served []string) []string {
	answered := map[string]bool{}
	var lost []string
	for _, f := range p.Fetched {
		path, _ := f[0].(string)
		if status, _ := f[1].(float64); status != 0 {
			answered[path] = true
		} else if site[path] {
			lost = append(lost, path)
		}
	}
	if !slices.Contains(served, "/css/site.css") {
		lost = append(lost, "/css/site.css")
	}
	for _, path := range served {
		if !answered[path] && !slices.Contains(lost, path) {
			lost = append(lost, path)
		}
	}
	return lost
}

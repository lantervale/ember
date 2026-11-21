//go:build chrome

package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
	for path, text := range importedSite {
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
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	ids := []string{"layers", "media", "deep", "supports", "never", "cycle", "plain", "untyped", "anon", "twice",
		"order", "late"}
	var body strings.Builder
	for _, id := range ids {
		body.WriteString(`<div id="` + id + `"></div>`)
	}
	sheet := `@import url(css/site.css); body { margin: 0; }`
	for _, size := range []geom.Vec2{geom.V2(800, 600), geom.V2(600, 600)} {
		want, err := chrome.PageBoxes(`<base href="`+srv.URL+`/index.html"><style>`+sheet+`</style>`, body.String(),
			size)
		require.NoError(t, err)

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
		require.Len(t, want, len(els)+2)
		for i, n := range els {
			assert.InDelta(t, want[i+2].W, n.Box().W, chrome.Tolerance, "%v #%s", size, ids[i])
			assert.InDelta(t, want[i+2].Y, n.Box().Y, chrome.Tolerance, "%v #%s", size, ids[i])
		}
	}
}

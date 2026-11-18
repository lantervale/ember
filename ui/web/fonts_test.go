package web_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/web"
)

// foundry serves a font in each format, a file too big, and notes the
// paths it was asked for with the Accept each came with.
func foundry(t *testing.T) (*httptest.Server, *sync.Map) {
	t.Helper()
	asked := &sync.Map{}
	mux := http.NewServeMux()
	for _, name := range []string{"go-regular-latin.ttf", "go-regular-latin.woff", "go-regular-latin.woff2"} {
		data, err := os.ReadFile("../internal/woff/testdata/" + name)
		require.NoError(t, err)
		mux.HandleFunc("/fonts/"+name, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) })
	}
	mux.HandleFunc("/fonts/huge.woff2", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, web.MaxFontBytes+1))
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked.Store(r.URL.Path, r.Header.Get("Accept"))
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, asked
}

func TestFontsResolvesAReferenceFromItsBaseAndAsksForAFont(t *testing.T) {
	srv, asked := foundry(t)
	fetch := web.Fonts(srv.URL + "/pages/crypt.html")
	for _, ref := range []string{
		"../fonts/go-regular-latin.woff2", "/fonts/go-regular-latin.woff", srv.URL + "/fonts/go-regular-latin.ttf",
	} {
		data, err := fetch(ref, "")
		require.NoError(t, err, ref)
		assert.NotEmpty(t, data, ref)
	}
	accept, ok := asked.Load("/fonts/go-regular-latin.woff2")
	require.True(t, ok)
	assert.Contains(t, accept, "font/woff2")
}

func TestFontsSaysWhyThereIsNoFont(t *testing.T) {
	srv, _ := foundry(t)
	fetch := web.Fonts(srv.URL + "/")
	_, err := fetch("fonts/gone.woff2", "")
	assert.ErrorContains(t, err, "404")
	_, err = fetch("fonts/huge.woff2", "")
	assert.ErrorContains(t, err, "over")
	_, err = fetch("file:///System/Library/Fonts/Helvetica.ttc", "")
	assert.ErrorContains(t, err, "no http or https address")
	_, err = web.Fonts("crypt/pages")("door.woff2", "")
	assert.ErrorContains(t, err, "no http or https address")
}

func TestADocumentShowsTheFontsOfTheWeb(t *testing.T) {
	srv, asked := foundry(t)
	d := ui.NewDocument(ui.FetchFonts(web.Fonts(srv.URL+"/")), ui.CSS(`
@font-face { font-family: Web;
  src: url(go-regular-latin.woff2) format("woff2"), url(go-regular-latin.woff) format("woff"); }
@font-face { font-family: Plain; src: url(go-regular-latin.ttf); }
.web { font-family: Web, monospace; } .plain { font-family: Plain, monospace; } .mono { font-family: monospace; }`,
		ui.From(srv.URL+"/fonts/site.css")))
	spans := map[string]*ui.Node{}
	for _, class := range []string{"web", "plain", "mono"} {
		spans[class] = ui.Span(ui.Class(class), ui.Text("the fonts of the web"))
		d.Body().Append(ui.Div(spans[class]))
	}
	size := geom.V2(400, 300)
	d.Layout(size)
	require.Eventually(t, func() bool {
		d.Layout(size)
		return d.FontsLoading() == 0 && spans["web"].Box().W == spans["plain"].Box().W
	}, 5*time.Second, 5*time.Millisecond, "the WOFF2 font never measured as the font it packs")
	assert.NotEqual(t, spans["mono"].Box().W, spans["web"].Box().W)
	assert.Empty(t, d.Warnings())
	_, woff := asked.Load("/fonts/go-regular-latin.woff")
	assert.False(t, woff, "the second of a src list is not asked for when the first is read")
}

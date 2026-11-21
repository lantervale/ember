package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/web"
)

// stylist serves sheets by path as text/css, one as text/plain, one as
// no type, one too big, and notes the Accept and Referer each path was asked with.
func stylist(t *testing.T, sheets map[string]string) (*httptest.Server, *sync.Map) {
	t.Helper()
	asked := &sync.Map{}
	mux := http.NewServeMux()
	for path, text := range sheets {
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			_, _ = w.Write([]byte(text))
		})
	}
	mux.HandleFunc("/css/plain.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(".box { width: 1px; }"))
	})
	mux.HandleFunc("/css/untyped.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header()["Content-Type"] = nil
		_, _ = w.Write([]byte(".box { width: 1px; }"))
	})
	mux.HandleFunc("/css/huge.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte(strings.Repeat(" ", web.MaxSheetBytes+1)))
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked.Store(r.URL.Path, [2]string{r.Header.Get("Accept"), r.Header.Get("Referer")})
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, asked
}

func TestSheetsResolvesAReferenceFromItsBaseAndAsksForCSS(t *testing.T) {
	srv, asked := stylist(t, map[string]string{"/css/site.css": ".box { width: 1px; }"})
	fetch := web.Sheets(srv.URL + "/pages/crypt.html")
	for _, ref := range []string{"../css/site.css", "/css/site.css", srv.URL + "/css/site.css"} {
		data, err := fetch(ref, "")
		require.NoError(t, err, ref)
		assert.Equal(t, ".box { width: 1px; }", string(data), ref)
	}
	_, err := fetch("/css/site.css", srv.URL+"/css/main.css")
	require.NoError(t, err)
	got, ok := asked.Load("/css/site.css")
	require.True(t, ok)
	assert.Equal(t, [2]string{"text/css,*/*;q=0.1", srv.URL + "/css/main.css"}, got,
		"it asks for CSS, from the sheet that imports it")
}

func TestSheetsSaysWhyThereIsNoSheet(t *testing.T) {
	srv, _ := stylist(t, nil)
	fetch := web.Sheets(srv.URL + "/")
	_, err := fetch("css/gone.css", "")
	assert.ErrorContains(t, err, "404")
	_, err = fetch("css/plain.css", "")
	assert.ErrorContains(t, err, `the sheet is text/plain, not text/css`)
	_, err = fetch("css/untyped.css", "")
	assert.NoError(t, err, "a sheet that says no type is read, as Chrome reads it")
	_, err = fetch("css/huge.css", "")
	assert.ErrorContains(t, err, "over")
	_, err = web.Sheets("crypt/pages")("site.css", "")
	assert.ErrorContains(t, err, "no http or https address")
}

func TestADocumentImportsTheSheetsOfTheWeb(t *testing.T) {
	srv, _ := stylist(t, map[string]string{
		"/css/site.css":      `@import url(parts/box.css) (min-width: 300px); @import "plain.css"; .box { height: 10px; }`,
		"/css/parts/box.css": `.box { width: 120px; }`,
	})
	box := ui.Div(ui.Class("box"))
	d := ui.NewDocument(ui.FetchSheets(web.Sheets(srv.URL+"/")), ui.CSS(`@import url(css/site.css);`,
		ui.From(srv.URL+"/index.html")))
	d.Body().Append(box)
	size := geom.V2(400, 300)
	require.Eventually(t, func() bool {
		d.Layout(size)
		return d.SheetsLoading() == 0 && box.Box().W == 120
	}, 5*time.Second, 5*time.Millisecond, "the imported sheets never landed")
	assert.InDelta(t, 10, box.Box().H, 0.01)
	require.Len(t, d.Warnings(), 1)
	assert.ErrorContains(t, d.Warnings()[0], "plain.css: web: the sheet is text/plain, not text/css")
}

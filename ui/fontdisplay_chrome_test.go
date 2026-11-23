//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// lateFace is an @font-face whose file comes after delay, and whether
// Chrome, at the end, draws its text in it.
type lateFace struct {
	display string
	delay   time.Duration
}

//nolint:gochecknoglobals // the test's faces
var lateFaces = []lateFace{
	{"block", time.Second}, {"block", 4500 * time.Millisecond}, {"auto", 4500 * time.Millisecond},
	{"swap", 4500 * time.Millisecond}, {"fallback", time.Second}, {"fallback", 4500 * time.Millisecond},
	{"optional", time.Second},
}

func (f lateFace) id() string { return fmt.Sprintf("%s-%d", f.display, f.delay.Milliseconds()) }

// lateSheet is the faces' sheet: each its own file, the font asking
// after delay ms, its text falling back to Go Regular, which comes at
// once.
func lateSheet() string {
	var b strings.Builder
	b.WriteString("@font-face { font-family: Plain; src: url(font?delay=0&n=plain); font-display: swap; }\n" +
		"span { display: inline-block; font: 20px Plain; }\n")
	for _, f := range lateFaces {
		fmt.Fprintf(&b, "@font-face { font-family: F%s; src: url(font?delay=%d&n=%s); font-display: %s; }\n"+
			"#%s { font-family: F%s, Plain; }\n", f.id(), f.delay.Milliseconds(), f.id(), f.display, f.id(), f.id())
	}
	return b.String()
}

const lateText = "AVATAR, as its face's display has it"

func TestFontDisplayKeepsOrDropsALateFontAsChromeDoes(t *testing.T) {
	sheet := lateSheet()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/font" {
			ms, _ := strconv.Atoi(r.URL.Query().Get("delay"))
			time.Sleep(time.Duration(ms) * time.Millisecond)
			w.Header().Set("Content-Type", "font/ttf")
			if r.URL.Query().Get("n") == "plain" {
				_, _ = w.Write(goregular.TTF)
				return
			}
			_, _ = w.Write(gobold.TTF)
			return
		}
		var body strings.Builder
		body.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><style>body { margin: 8px; }\n" + sheet +
			"</style>" + chrome.EvalScript(`return Object.fromEntries([...document.querySelectorAll("span")].map(
	s => [s.id, s.getBoundingClientRect().width]))`) + "</head><body>")
		for _, f := range lateFaces {
			fmt.Fprintf(&body, "<div><span id=%q>%s</span></div>", f.id(), lateText)
		}
		body.WriteString("</body></html>")
		_, _ = w.Write([]byte(body.String()))
	}))
	defer srv.Close()
	require.True(t, strings.HasPrefix(srv.URL, "http://127.0.0.1:"))
	out, err := chrome.EvalPage(srv.URL+"/", geom.V2(600, 400))
	require.NoError(t, err)
	var want map[string]float32
	require.NoError(t, json.Unmarshal(out, &want))

	got := lateWidths(t, sheet)
	for _, f := range lateFaces {
		assert.InDelta(t, want[f.id()], got[f.id()], 0.02, "%s", f.id())
	}
	assert.Greater(t, want["block-4500"], want["fallback-4500"], "the bold face, against Go Regular")
}

// lateWidths are the widths of the spans of the faces' text in ember, its
// fonts given the same delays on the World's real time.
func lateWidths(t *testing.T, sheet string) map[string]float32 {
	t.Helper()
	var mu sync.Mutex
	gates := map[int]chan struct{}{}
	gate := func(ms int) chan struct{} {
		mu.Lock()
		defer mu.Unlock()
		if gates[ms] == nil {
			gates[ms] = make(chan struct{})
		}
		return gates[ms]
	}
	d := ui.NewDocument(ui.FetchFonts(func(address, _ string) ([]byte, error) {
		u, err := url.Parse(address)
		if err != nil {
			return nil, err
		}
		ms, _ := strconv.Atoi(u.Query().Get("delay"))
		if ms > 0 {
			<-gate(ms)
		}
		if u.Query().Get("n") == "plain" {
			return goregular.TTF, nil
		}
		return gobold.TTF, nil
	}), ui.CSS(sheet))
	for _, f := range lateFaces {
		d.Body().Append(ui.Div(ui.Span(ui.ID(f.id()), ui.Text(lateText))))
	}
	win := platform.NewWindow(platform.Size(600, 400))
	win.Add(d)
	w := ember.New().AddEntities(win)
	for _, at := range []int{1000, 4500} {
		for float64(w.Time().RealElapsed) < float64(at)/1000 {
			w.RunFrames(1)
		}
		close(gate(at))
		left := 0
		for _, f := range lateFaces {
			if f.delay.Milliseconds() > int64(at) {
				left++
			}
		}
		require.Eventually(t, func() bool {
			w.RunFrames(1)
			return d.FontsLoading() == left
		}, 5*time.Second, time.Millisecond)
	}
	w.RunFrames(2)
	got := map[string]float32{}
	for _, f := range lateFaces {
		got[f.id()] = d.ByID(f.id()).Box().W
	}
	return got
}

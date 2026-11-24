//go:build chrome

package chrome

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lantervale/ember/geom"
)

func TestMajorOfAUserAgent(t *testing.T) {
	for agent, want := range map[string]int{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) " +
			"HeadlessChrome/158.0.8127.37 Safari/537.36": 158,
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36": 156,
	} {
		if got, err := majorOf(agent); err != nil || got != want {
			t.Errorf("%q: %d, %v; want %d", agent, got, err, want)
		}
	}
	if _, err := majorOf("Firefox/140.0"); err == nil {
		t.Error("a user agent without Chrome's version has none")
	}
}

func TestVersionIsTheBrowsers(t *testing.T) {
	v, err := Version()
	if err != nil || v < 100 {
		t.Fatalf("Chrome %d, %v", v, err)
	}
}

// forget empties the known version until t ends, as at the process's start.
func forget(t *testing.T) {
	t.Helper()
	known.Lock()
	set, major, err := known.set, known.major, known.err
	known.set, known.major, known.err = false, 0, nil
	known.Unlock()
	t.Cleanup(func() {
		known.Lock()
		known.set, known.major, known.err = set, major, err
		known.Unlock()
	})
}

func TestASessionTellsTheVersionItHoldsTheLockFor(t *testing.T) {
	forget(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<!DOCTYPE html><p>x</p>"))
	}))
	defer srv.Close()
	Open(t, srv.URL, geom.V2(100, 100), 1)
	asked := make(chan error, 1)
	go func() {
		v, err := Version()
		if err == nil && v < 100 {
			err = fmt.Errorf("Chrome %d", v)
		}
		asked <- err
	}()
	select {
	case err := <-asked:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the version waited for the lock its own session holds")
	}
}

//go:build chrome

package chrome

import "testing"

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

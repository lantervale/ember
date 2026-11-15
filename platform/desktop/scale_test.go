package desktop //nolint:testpackage // white-box: framebuffer is the runner's own rule

import "testing"

func TestAHiddenWindowKeepsTheSizeItAskedWhateverTheScreen(t *testing.T) {
	for _, c := range []struct {
		name           string
		ww, wh, fw, fh int
		askW, askH     int
		w, h, pw, ph   int
	}{
		{"nothing asked: the system's", 640, 360, 1280, 720, 0, 0, 640, 360, 1280, 720},
		{"the system gave what was asked", 640, 360, 640, 360, 640, 360, 640, 360, 640, 360},
		{"fitted to a 1024×768 screen", 1028, 749, 1028, 749, 2400, 1400, 2400, 1400, 2400, 1400},
		{"fitted on a Retina screen: its pixels a point", 1440, 875, 2880, 1750, 2400, 1400, 2400, 1400, 4800, 2800},
		{"fitted at 1.5 pixels a point", 1000, 700, 1500, 1050, 1201, 801, 1201, 801, 1802, 1202},
		{"a window with no size yet", 0, 0, 0, 0, 2400, 1400, 2400, 1400, 2400, 1400},
	} {
		w, h, pw, ph := asked(c.ww, c.wh, c.fw, c.fh, c.askW, c.askH)
		if w != c.w || h != c.h || pw != c.pw || ph != c.ph {
			t.Errorf("%s: %d×%d points, %d×%d pixels, want %d×%d and %d×%d", c.name, w, h, pw, ph, c.w, c.h, c.pw, c.ph)
		}
	}
}

func TestAHiddenRunDrawsAtTheScaleItAsksForWhateverTheScreen(t *testing.T) {
	for _, c := range []struct {
		name           string
		ww, wh, fw, fh int
		pixels         float32
		w, h           int
	}{
		{"no scale asked: the system's", 640, 360, 640, 360, 0, 640, 360},
		{"a 1× screen asked for 2: twice the points", 640, 360, 640, 360, 2, 1280, 720},
		{"a Retina screen asked for 2: as it is", 640, 360, 1280, 720, 2, 1280, 720},
		{"rounded to whole pixels", 333, 201, 333, 201, 1.5, 500, 302},
	} {
		if w, h := framebuffer(c.ww, c.wh, c.fw, c.fh, c.pixels); w != c.w || h != c.h {
			t.Errorf("%s: %d×%d, want %d×%d", c.name, w, h, c.w, c.h)
		}
	}
}

//go:build chrome

package web_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/animated"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// chromeFrames is what Chrome's ImageDecoder reads of each picture of
// the page, by its title: how many times it repeats after its first
// play (null for ever), and each frame's duration in microseconds and
// its pixels, not premultiplied. The decoder works off the main thread,
// which headless Chrome's virtual clock does not wait for: with the page
// idle it runs the clock out and dumps the page before the frames come
// back (Chrome for Testing 156), so held keeps the main thread busy with
// messages, which stop the clock, until each step settles.
const chromeFrames = `
const held = p => new Promise((resolve, reject) => {
	let settled = false;
	p.then(v => { settled = true; resolve(v); }, e => { settled = true; reject(e); });
	const ch = new MessageChannel();
	ch.port1.onmessage = () => { if (!settled) ch.port2.postMessage(0); };
	ch.port2.postMessage(0);
});
const out = {};
try {
for (const img of document.querySelectorAll("img")) {
	const type = img.src.slice(5, img.src.indexOf(";"));
	const data = await held((async () => (await fetch(img.src)).arrayBuffer())());
	const dec = new ImageDecoder({data, type});
	await held(dec.tracks.ready);
	await held(dec.completed);
	const track = dec.tracks.selectedTrack;
	const frames = [];
	for (let i = 0; i < track.frameCount; i++) {
		const f = (await held(dec.decode({frameIndex: i}))).image;
		const c = new OffscreenCanvas(f.displayWidth, f.displayHeight);
		const g = c.getContext("2d");
		g.drawImage(f, 0, 0);
		frames.push({duration: f.duration, pixels: [...g.getImageData(0, 0, c.width, c.height).data]});
		f.close();
	}
	out[img.title] = {repetitions: track.repetitionCount, frames};
}
} catch (e) { return {error: String(e)}; }
return out;`

type chromeAnimation struct {
	Repetitions *int `json:"repetitions"`
	Frames      []struct {
		Duration int64   `json:"duration"`
		Pixels   []uint8 `json:"pixels"`
	} `json:"frames"`
}

func TestAnimationsDecodeToChromesFramesTimesAndLoops(t *testing.T) {
	pictures := map[string][]byte{}
	for name, data := range animatedGIFs(t) {
		pictures[name+".gif"] = data
	}
	for name, data := range animatedWebPs() {
		pictures[name+".webp"] = data
	}
	files := fstest.MapFS{}
	d := ui.NewDocument(ui.Images(files))
	for name, data := range pictures {
		files[name] = &fstest.MapFile{Data: data}
		d.Body().Append(ui.Img(ui.Attr("src", name), ui.Attr("title", name)))
	}
	out, err := chrome.Eval(d, geom.V2(400, 300), chromeFrames)
	require.NoError(t, err)
	var failed struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(out, &failed))
	require.Empty(t, failed.Error, "Chrome's decoder")
	var want map[string]chromeAnimation
	require.NoError(t, json.Unmarshal(out, &want))
	require.Len(t, want, len(pictures))
	for name, data := range pictures {
		w := want[name]
		got, decodeErr := animated.Decode(data)
		require.NoError(t, decodeErr, name)
		require.NotNil(t, got, name)
		plays := 0
		if w.Repetitions != nil {
			plays = *w.Repetitions + 1
		}
		if len(w.Frames) > 1 {
			assert.Equal(t, plays, got.Plays, "%s plays", name)
		}
		require.Len(t, got.Frames, len(w.Frames), name)
		for i, f := range w.Frames {
			assert.Equal(t, time.Duration(f.Duration)*time.Microsecond, got.Delays[i], "%s frame %d", name, i)
			assertPixels(t, f.Pixels, got.Frames[i].Pix, fmt.Sprintf("%s frame %d", name, i))
		}
	}
}

// assertPixels fails where a pixel's channel is off by more than Chrome's
// canvas rounds a colour it premultiplied, a clear pixel's colour aside.
func assertPixels(t *testing.T, want, got []uint8, name string) {
	t.Helper()
	require.Len(t, got, len(want), name)
	for i := 0; i < len(want); i += 4 {
		if want[i+3] == 0 && got[i+3] == 0 {
			continue
		}
		for c := range 4 {
			if d := int(want[i+c]) - int(got[i+c]); d < -3 || d > 3 {
				t.Errorf("%s: pixel %d is %v, Chrome's %v", name, i/4, got[i:i+4], want[i:i+4])
				break
			}
		}
	}
}

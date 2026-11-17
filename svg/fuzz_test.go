package svg //nolint:testpackage // white-box: reads the parsed model's unexported parts

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzPicture reads and draws whatever bytes it is given: no panic, and
// a picture whose alpha is premultiplied.
func FuzzPicture(f *testing.F) {
	files, err := filepath.Glob("testdata/*.svg")
	if err != nil || len(files) == 0 {
		f.Fatalf("no seed in testdata: %v", err)
	}
	for _, name := range files {
		data, readErr := os.ReadFile(name)
		if readErr != nil {
			f.Fatal(readErr)
		}
		f.Add(data)
	}
	f.Add([]byte(`<!DOCTYPE svg [<!ENTITY a "fill:red">]><svg xmlns="http://www.w3.org/2000/svg" width="1e9" ` +
		`viewBox="0 0 1e-9 5">` +
		`<style>g > .a, #b { stroke: hsl(1 2% 3%) }</style><g id="g" opacity=".5" transform="rotate(1e9) ` +
		`skewX(90)">` +
		`<use href="#g"/><path class="a" style="&a;" d="M0 0a1e9 1e-9 1e9 1 0 5 5T1 1z" ` +
		`stroke-dasharray="1e-9 0 1" stroke-linecap="round"/></g></svg>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		p, parseErr := NewPicture(data)
		if parseErr != nil {
			return
		}
		p.Size()
		p.Unsupported()
		for _, size := range [][2]int{{48, 48}, {33, 7}} {
			img, drawErr := p.Rasterise(size[0], size[1])
			if drawErr != nil {
				t.Fatal(drawErr)
			}
			for i := 0; i < len(img.Pix); i += 4 {
				if a := img.Pix[i+3]; img.Pix[i] > a || img.Pix[i+1] > a || img.Pix[i+2] > a {
					t.Fatalf("pixel %d: %v is brighter than its alpha", i/4, img.Pix[i:i+4])
				}
			}
		}
	})
}

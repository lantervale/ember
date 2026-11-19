package svg //nolint:testpackage // white-box: reads the parsed model's unexported parts

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// mosaic is a picture of many paths: 400 leaves of curves, a third of
// them filled with a gradient, a third stroked, a third in half-clear
// groups.
func mosaic() []byte {
	var b strings.Builder
	b.WriteString(`<svg ` + xmlns + ` viewBox="0 0 400 400">` +
		`<linearGradient id="l"><stop stop-color="#fc0"/><stop offset="1" stop-color="#c06"/></linearGradient>` +
		`<radialGradient id="r"><stop stop-color="#fff"/><stop offset="1" stop-color="#06c"/></radialGradient>`)
	for i := range 400 {
		x, y := i%20*20, i/20*20
		leaf := fmt.Sprintf(`<path transform="translate(%d %d) rotate(%d 10 10)" d="M10 1C16 4 19 9 10 19C1 9 `+
			`4 4 10 1zM10 6a3 3 0 1 0 .1 0z" `, x, y, i*7)
		switch i % 6 {
		case 0:
			b.WriteString(leaf + `fill="url(#l)"/>`)
		case 1:
			b.WriteString(leaf + `fill="url(#r)" fill-rule="evenodd"/>`)
		case 2, 3:
			b.WriteString(leaf + `fill="none" stroke="#333" stroke-width="1.5" stroke-linejoin="round"/>`)
		case 4:
			b.WriteString(`<g opacity=".5">` + leaf + `fill="#396" stroke="#000"/></g>`)
		case 5:
			b.WriteString(leaf + `fill="#936" fill-opacity=".7"/>`)
		}
	}
	b.WriteString(`</svg>`)
	return []byte(b.String())
}

func benchmarkPicture(b *testing.B, data []byte) {
	b.Run("parse", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := NewPicture(data); err != nil {
				b.Fatal(err)
			}
		}
	})
	p, err := NewPicture(data)
	if err != nil {
		b.Fatal(err)
	}
	for _, size := range []int{64, 256, 1024} {
		b.Run(fmt.Sprintf("rasterise-%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, drawErr := p.Rasterise(size, size); drawErr != nil {
					b.Fatal(drawErr)
				}
			}
		})
	}
}

func BenchmarkIcon(b *testing.B) {
	data, err := os.ReadFile("testdata/feather-settings.svg")
	if err != nil {
		b.Fatal(err)
	}
	benchmarkPicture(b, data)
}

func BenchmarkManyPaths(b *testing.B) { benchmarkPicture(b, mosaic()) }

// BenchmarkFile measures the file EMBER_SVG_BENCH names.
func BenchmarkFile(b *testing.B) {
	name := os.Getenv("EMBER_SVG_BENCH")
	if name == "" {
		b.Skip("EMBER_SVG_BENCH names no file")
	}
	data, err := os.ReadFile(name)
	if err != nil {
		b.Fatal(err)
	}
	benchmarkPicture(b, data)
}

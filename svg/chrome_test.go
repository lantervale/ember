//go:build chrome

package svg_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lantervale/ember/svg"
)

// The Chrome suite draws every picture of testdata as headless Chrome
// draws it in an img of the same size, and holds the difference under
// each one's threshold in testdata/chrome.txt. Run with -tags chrome;
// EMBER_CHROME names the browser when it isn't in /Applications.
//
//	EMBER_SVG_SETS=dir:dir   further folders of pictures, each with its own chrome.txt
//	EMBER_SVG_RECORD=1       rewrite each chrome.txt from what was measured
//	EMBER_SVG_STRIPS=dir     write Chrome | ember | difference pictures there
//	EMBER_SVG_SHOTS=768x768-ff,…  other sizes and page greys to look at, measured against no threshold

// shot is one way the pictures are compared: a size and a page colour.
type shot struct {
	w, h int
	bg   uint8
}

func (s shot) String() string { return fmt.Sprintf("%dx%d-%02x", s.w, s.h, s.bg) }

//nolint:gochecknoglobals // EMBER_SVG_SHOTS replaces them
var shots = []shot{{256, 256, 0xff}, {256, 256, 0x80}, {48, 48, 0xff}, {200, 120, 0xff}}

const (
	gap     = 8
	columns = 8
)

func browser() string {
	if p := os.Getenv("EMBER_CHROME"); p != "" {
		return p
	}
	return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
}

func cell(s shot, i int) image.Rectangle {
	x, y := (i%columns)*(s.w+gap), (i/columns)*(s.h+gap)
	return image.Rect(x, y, x+s.w, y+s.h)
}

// chromeShot is Chrome's picture of files, each in an img of the shot's
// size, laid out in the cells of one page.
func chromeShot(t *testing.T, files []string, s shot) image.Image {
	t.Helper()
	dir := t.TempDir()
	var page bytes.Buffer
	fmt.Fprintf(&page, `<!DOCTYPE html><html><body style="margin:0;background:#%02x%02x%02x">`, s.bg, s.bg, s.bg)
	for i, f := range files {
		abs, absErr := filepath.Abs(f)
		if absErr != nil {
			t.Fatal(absErr)
		}
		c := cell(s, i)
		fmt.Fprintf(&page, `<img src="file://%s" width="%d" height="%d" `+
			`style="position:absolute;left:%dpx;top:%dpx">`,
			abs, s.w, s.h, c.Min.X, c.Min.Y)
	}
	page.WriteString("</body></html>")
	html, out := filepath.Join(dir, "page.html"), filepath.Join(dir, "page.png")
	if err := os.WriteFile(html, page.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	last := cell(s, len(files)-1)
	width, height := max(columns*(s.w+gap), 500), max(last.Max.Y+gap, 500)
	unlock := lockChrome(t)
	defer unlock()
	var stderr bytes.Buffer
	for try := 0; ; try++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		cmd := exec.CommandContext(ctx, browser(),
			"--headless=new", "--disable-gpu", "--run-all-compositor-stages-before-draw",
			"--force-device-scale-factor=1", "--no-first-run", "--no-default-browser-check", "--disable-extensions",
			"--allow-file-access-from-files",
			"--user-data-dir="+filepath.Join(dir, fmt.Sprintf("profile-%d", try)),
			fmt.Sprintf("--window-size=%d,%d", width, height),
			"--virtual-time-budget=10000", "--screenshot="+out, "file://"+html)
		stderr.Reset()
		cmd.Stderr = &stderr
		runErr := cmd.Run()
		cancel()
		if f, openErr := os.Open(out); openErr == nil {
			img, decodeErr := png.Decode(f)
			f.Close()
			if decodeErr == nil {
				return img
			}
		}
		if try == 1 {
			t.Fatalf("chrome: %v: %s", runErr, stderr.String())
		}
	}
}

// thresholds reads dir's chrome.txt: lines of file, shot and the most
// its mean difference may be.
func thresholds(t *testing.T, dir string) map[string]float64 {
	t.Helper()
	out := map[string]float64{}
	f, err := os.Open(filepath.Join(dir, "chrome.txt"))
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 3 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		v, parseErr := strconv.ParseFloat(fields[2], 64)
		if parseErr != nil {
			t.Fatalf("%s/chrome.txt: %q: %v", dir, sc.Text(), parseErr)
		}
		out[fields[0]+" "+fields[1]] = v
	}
	return out
}

// difference is the mean absolute difference per colour channel, 0–255,
// between Chrome's cell and ember's picture over the same page colour,
// and the two side by side with where they differ.
func difference(chrome image.Image, at image.Rectangle, got *image.RGBA, bg uint8) (float64, *image.RGBA) {
	w, h := at.Dx(), at.Dy()
	ref := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(ref, ref.Bounds(), chrome, at.Min, draw.Src)
	mine := image.NewRGBA(ref.Bounds())
	draw.Draw(mine, mine.Bounds(), image.NewUniform(color.Gray{bg}), image.Point{}, draw.Src)
	draw.Draw(mine, mine.Bounds(), got, image.Point{}, draw.Over)
	strip := image.NewRGBA(image.Rect(0, 0, 3*w+2*gap, h))
	draw.Draw(strip, strip.Bounds(), image.NewUniform(color.Gray{0x40}), image.Point{}, draw.Src)
	draw.Draw(strip, ref.Bounds(), ref, image.Point{}, draw.Src)
	draw.Draw(strip, ref.Bounds().Add(image.Pt(w+gap, 0)), mine, image.Point{}, draw.Src)
	total := 0
	for i := 0; i < len(ref.Pix); i += 4 {
		worst := 0
		for c := range 3 {
			d := int(ref.Pix[i+c]) - int(mine.Pix[i+c])
			if d < 0 {
				d = -d
			}
			total += d
			worst = max(worst, d)
		}
		heat := uint8(min(255, worst*4))
		x, y := i/4%w, i/4/w
		strip.SetRGBA(2*(w+gap)+x, y, color.RGBA{heat, heat, heat, 255})
	}
	return float64(total) / float64(3*w*h), strip
}

func TestPicturesAreDrawnAsChromeDrawsThem(t *testing.T) {
	if _, err := os.Stat(browser()); err != nil {
		t.Skipf("no Chrome at %s", browser())
	}
	if list := os.Getenv("EMBER_SVG_SHOTS"); list != "" {
		shots = nil
		for _, one := range strings.Split(list, ",") {
			var s shot
			if _, err := fmt.Sscanf(one, "%dx%d-%x", &s.w, &s.h, &s.bg); err != nil {
				t.Fatalf("EMBER_SVG_SHOTS: %q: %v", one, err)
			}
			shots = append(shots, s)
		}
	}
	dirs := []string{"testdata"}
	if sets := os.Getenv("EMBER_SVG_SETS"); sets != "" {
		dirs = append(dirs, filepath.SplitList(sets)...)
	}
	for _, dir := range dirs {
		t.Run(filepath.Base(dir), func(t *testing.T) { compareSet(t, dir) })
	}
}

func compareSet(t *testing.T, dir string) {
	files, err := filepath.Glob(filepath.Join(dir, "*.svg"))
	if err != nil || len(files) == 0 {
		t.Skipf("no pictures in %s", dir)
	}
	sort.Strings(files)
	pictures := make([]*svg.Picture, len(files))
	for i, f := range files {
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if pictures[i], readErr = svg.NewPicture(data); readErr != nil {
			t.Errorf("%s: %v", f, readErr)
		}
	}
	allowed := thresholds(t, dir)
	record := os.Getenv("EMBER_SVG_RECORD") != ""
	look := os.Getenv("EMBER_SVG_SHOTS") != ""
	strips := os.Getenv("EMBER_SVG_STRIPS")
	measured := map[string]float64{}
	var table strings.Builder
	fmt.Fprintf(&table, "\n| %s |", filepath.Base(dir))
	for _, s := range shots {
		fmt.Fprintf(&table, " %s |", s)
	}
	table.WriteString(" not drawn |\n")
	rows := make([]string, len(files))
	sums := make([]float64, len(shots))
	for si, s := range shots {
		chrome := chromeShot(t, files, s)
		for i, f := range files {
			name := filepath.Base(f)
			if pictures[i] == nil {
				continue
			}
			got, drawErr := pictures[i].Rasterise(s.w, s.h)
			if drawErr != nil {
				t.Errorf("%s: %v", name, drawErr)
				continue
			}
			diff, strip := difference(chrome, cell(s, i), got, s.bg)
			key := name + " " + s.String()
			measured[key] = diff
			sums[si] += diff
			rows[i] += fmt.Sprintf(" %.2f |", diff)
			if limit, ok := allowed[key]; !record && !look && (!ok || diff > limit) {
				t.Errorf("%s: mean difference %.3f from Chrome, over its threshold %.3f", key, diff, limit)
			}
			if strips != "" {
				file := strings.TrimSuffix(name, ".svg") + "-" + s.String() + ".png"
				writeStrip(t, filepath.Join(strips, filepath.Base(dir)), file, strip)
			}
		}
	}
	for i, f := range files {
		missing := ""
		if pictures[i] != nil {
			missing = strings.Join(pictures[i].Unsupported(), ", ")
		}
		fmt.Fprintf(&table, "| %s |%s %s |\n", filepath.Base(f), rows[i], missing)
	}
	table.WriteString("| mean |")
	for _, sum := range sums {
		fmt.Fprintf(&table, " %.2f |", sum/float64(len(files)))
	}
	t.Log(table.String())
	if record && !look {
		writeThresholds(t, dir, measured)
	}
}

func writeStrip(t *testing.T, dir, name string, strip image.Image) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = png.Encode(f, strip); err != nil {
		t.Fatal(err)
	}
}

// writeThresholds records each measure with a little room over it.
func writeThresholds(t *testing.T, dir string, measured map[string]float64) {
	t.Helper()
	keys := make([]string, 0, len(measured))
	for k := range measured {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out bytes.Buffer
	out.WriteString("# file, shot (size and page grey), and the most the mean difference per channel from\n" +
		"# Chrome's picture may be, 0-255. Recorded with EMBER_SVG_RECORD=1 go test -tags chrome ./svg.\n")
	for _, k := range keys {
		fmt.Fprintf(&out, "%s %.2f\n", k, math.Ceil((measured[k]*1.15+0.03)*100)/100)
	}
	if err := os.WriteFile(filepath.Join(dir, "chrome.txt"), out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

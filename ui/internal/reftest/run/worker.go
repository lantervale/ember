package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/reftest"
	"github.com/lantervale/ember/world"
)

// Frames a page is given at least, and at most, before it is captured:
// it is captured once nothing is loading for settled frames in a row.
const (
	firstFrames = 6
	lastFrames  = 600
	settled     = 5
)

// drawer draws one page after another in its window.
type drawer struct {
	root     string
	win      *platform.Window
	jobs     <-chan [2]string
	out      io.Writer
	doc      *ui.Document
	fetches  *reftest.Fetches
	job      [2]string
	frames   int
	quiet    int
	captured bool
}

// work draws the pages in reads, one "path\tpicture" a line, in one
// hidden 800×600 window, each captured to its picture once settled, and
// writes a line for each to out: its kind (reftest.Drawn…), path and
// detail, by tabs.
func work(root string, in io.Reader, out io.Writer) {
	jobs := make(chan [2]string)
	go func() {
		s := bufio.NewScanner(in)
		for s.Scan() {
			if p, shot, ok := strings.Cut(s.Text(), "\t"); ok {
				jobs <- [2]string{p, shot}
			}
		}
		close(jobs)
	}()
	d := &drawer{root: root, jobs: jobs, out: out, win: platform.NewWindow(
		platform.Size(reftest.Width, reftest.Height), platform.FixedSize(), platform.Background(color.Hex("#ffffff")))}
	ember.New().AddEntities(d.win).AddSystems(world.Func(world.Update, "reftest", d.step)).MustRun()
	os.Exit(0)
}

func (d *drawer) reply(kind, detail string) {
	fmt.Fprintf(d.out, "%s\t%s\t%s\n", kind, d.job[0], strings.ReplaceAll(detail, "\n", " "))
}

func (d *drawer) step(f world.Frame) {
	if d.doc == nil {
		d.next(f)
		return
	}
	d.frames++
	switch {
	case d.captured:
		d.reply(reftest.Drawn, "")
		f.World.RemoveEntity(d.doc)
		d.doc = nil
	case d.frames >= lastFrames || d.frames >= firstFrames && d.quiet >= settled:
		d.win.Capture(d.job[1])
		d.captured = true
	case d.doc.FontsLoading() == 0 && d.doc.SheetsLoading() == 0 && d.doc.ImagesLoading() == 0 && !d.fetches.Busy():
		d.quiet++
	default:
		d.quiet = 0
	}
}

// next shows the next page that can be drawn, replying for those that
// can't, and quits after the last.
func (d *drawer) next(f world.Frame) {
	for d.doc == nil {
		var ok bool
		if d.job, ok = <-d.jobs; !ok {
			f.Quit()
			return
		}
		page, err := reftest.ReadPage(d.root, d.job[0])
		switch {
		case err != nil:
			d.reply(reftest.Failed, err.Error())
		case page.Skip != "":
			d.reply(reftest.Skipped, page.Skip)
		default:
			d.fetches = &reftest.Fetches{}
			d.doc = page.Document(d.root, d.fetches)
			d.frames, d.quiet, d.captured = 0, 0, false
			d.win.Add(d.doc)
		}
	}
}

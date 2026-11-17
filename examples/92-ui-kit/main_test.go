package main

import (
	"testing"
	"time"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/align"
)

func TestElementsLineUp(t *testing.T) {
	for _, name := range names() {
		t.Run(name, func(t *testing.T) {
			g, err := page(elements()[name])
			if err != nil {
				t.Fatal(err)
			}
			g.doc.Layout(geom.V2(480, 320))
			for _, f := range align.Check(g.doc) {
				t.Error(f)
			}
		})
	}
}

func TestALatePictureWaitsForItsFrame(t *testing.T) {
	l := newLate()
	got := make(chan []byte)
	go func() {
		data, _ := l.fetch("wall.png")
		got <- data
	}()
	for range wallFrame - 1 {
		l.Update(world.Frame{})
	}
	select {
	case <-got:
		t.Fatal("the wall came before its frame")
	case <-time.After(20 * time.Millisecond):
	}
	l.Update(world.Frame{})
	if data := <-got; len(data) == 0 {
		t.Fatal("the wall came empty")
	}
	if _, err := l.fetch("moon.png"); err == nil {
		t.Fatal("a picture the source lacks is no error")
	}
}

func TestFetchedPicturesLandOnThePage(t *testing.T) {
	g, err := page(fetchedImages)
	if err != nil {
		t.Fatal(err)
	}
	win := platform.NewWindow(platform.Size(480, 320))
	win.Add(g.doc)
	w := ember.New().AddEntities(win).AddSystems(g.systems...)
	portrait := g.doc.ByID("portrait")
	w.RunFrames(portraitFrame - 1)
	if portrait.Box().H == 72 {
		t.Fatal("the img has its picture's height before the picture came")
	}
	w.RunFrames(wallFrame)
	for range 400 {
		w.RunFrames(1)
		if portrait.Box().H == 72 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if portrait.Box().H != 72 {
		t.Fatalf("the img is %v high once its 24×24 picture landed, want 72", portrait.Box().H)
	}
	if warns := g.doc.Warnings(); len(warns) > 0 {
		t.Fatalf("warnings: %v", warns)
	}
}

func TestTheQuestsTextGoesRoundItsPortraitAndTheRewardBelowIt(t *testing.T) {
	g, err := page(floated)
	if err != nil {
		t.Fatal(err)
	}
	g.doc.Layout(geom.V2(480, 320))
	portrait, first, reward := g.doc.ByID("portrait").Box(), g.doc.ByID("first").Box(), g.doc.ByID("reward").Box()
	if first.Y >= portrait.Y+portrait.H || first.X != portrait.X {
		t.Fatalf("the first paragraph %v starts behind the portrait %v, its lines beside it", first, portrait)
	}
	if first.Y+first.H <= portrait.Y+portrait.H {
		t.Fatalf("the first paragraph %v goes on under the portrait %v", first, portrait)
	}
	tip, second := g.doc.ByID("tip").Box(), g.doc.ByID("second").Box()
	if tip.X+tip.W != first.X+first.W || second.Y+second.H >= tip.Y+tip.H {
		t.Fatalf("the tip %v floats right, taller than the paragraph %v beside it", tip, second)
	}
	if reward.Y < tip.Y+tip.H {
		t.Fatalf("the reward %v clears the tip %v", reward, tip)
	}
	if warns := g.doc.Warnings(); len(warns) > 0 {
		t.Fatalf("warnings: %v", warns)
	}
}

func TestTheJournalFillsTwoColumnsAndKeepsItsLastEntryWhole(t *testing.T) {
	g, err := page(columned)
	if err != nil {
		t.Fatal(err)
	}
	g.doc.Layout(geom.V2(480, 320))
	journal, monday := g.doc.ByID("journal").Box(), g.doc.ByID("monday").Box()
	tuesday, wednesday := g.doc.ByID("tuesday").Box(), g.doc.ByID("wednesday").Box()
	const column = (380 - 2*13 - 24) / 2
	if monday.W != column || monday.X != journal.X+13 {
		t.Fatalf("Monday %v is one column of %v wide, in the first", monday, column)
	}
	if tuesday.W != 2*column+24 {
		t.Fatalf("Tuesday %v is cut over both columns", tuesday)
	}
	if wednesday.W != column || wednesday.X != monday.X+column+24 || wednesday.Y+wednesday.H > journal.Y+journal.H {
		t.Fatalf("Wednesday %v is whole in the second column of %v", wednesday, journal)
	}
	if warns := g.doc.Warnings(); len(warns) > 0 {
		t.Fatalf("warnings: %v", warns)
	}
}

func TestTheContentsAreNumberedAndDrawnByPseudoElements(t *testing.T) {
	g, err := page(generated)
	if err != nil {
		t.Fatal(err)
	}
	g.doc.Layout(geom.V2(480, 320))
	if warns := g.doc.Warnings(); len(warns) > 0 {
		t.Fatalf("warnings: %v", warns)
	}
	if faults := align.Check(g.doc); len(faults) > 0 {
		t.Fatalf("faults: %v", faults)
	}
	contents := g.doc.ByID("contents")
	if n := len(contents.Children()); n != 5 {
		t.Fatalf("the contents have %d children: a generated box is none", n)
	}
	if text := g.doc.ByID("road").Text(); text != "The road" {
		t.Fatalf("the title reads %q: generated text is not the element's", text)
	}
	road, hills := g.doc.ByID("road").Box(), g.doc.ByID("hills").Box()
	mill, miller := g.doc.ByID("mill").Box(), g.doc.ByID("miller").Box()
	if road.H != 18 || hills.H != 18 || mill.H != 18 || miller.H != 18 {
		t.Fatalf("each title and part is one line: %v %v %v %v", road, hills, mill, miller)
	}
}

func TestTheLetterIsSelectedWhereItsSheetSaysSo(t *testing.T) {
	g, err := page(selection)
	if err != nil {
		t.Fatal(err)
	}
	g.doc.Layout(geom.V2(480, 320))
	if warns := g.doc.Warnings(); len(warns) > 0 {
		t.Fatalf("warnings: %v", warns)
	}
	if faults := align.Check(g.doc); len(faults) > 0 {
		t.Fatalf("faults: %v", faults)
	}
	const opened = "crypt under the mill is open again.\n\nSay ember and ash at the third arch"
	if got := g.doc.Selection().Text; got != opened {
		t.Fatalf("the letter opens with %q selected, want %q", got, opened)
	}
	if !g.doc.SelectAll() {
		t.Fatal("the letter takes no selection")
	}
	const all = "Aldric, the crypt under the mill is open again.\n\n" +
		"Say ember and ash at the third arch and the keeper lets you through."
	if got := g.doc.Selection().Text; got != all {
		t.Fatalf("select all took %q, want %q: the seal and the kit's own text take no selection", got, all)
	}
}

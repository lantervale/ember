package wm //nolint:testpackage // tests the unexported answer of a close rule

import (
	"testing"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

func TestARuleAnswersACloseByTheWindowsRank(t *testing.T) {
	for _, c := range []struct {
		name          string
		rule          platform.CloseRule
		primary, last bool
		closes, quits bool
	}{
		{"by default the primary window's close ends the program", platform.CloseQuitsIfPrimary, true, false, false, true},
		{"by default another window's close closes it", platform.CloseQuitsIfPrimary, false, false, true, false},
		{"CloseQuits ends the program from any window", platform.CloseQuits, false, false, false, true},
		{"CloseQuitsIfLast closes a window among others", platform.CloseQuitsIfLast, true, false, true, false},
		{"CloseQuitsIfLast closes the last and ends the program", platform.CloseQuitsIfLast, true, true, true, true},
		{"CloseOnly closes the last and runs on", platform.CloseOnly, true, true, true, false},
		{"CloseAsks does nothing", platform.CloseAsks, true, true, false, false},
	} {
		if closes, quits := answer(c.rule, c.primary, c.last); closes != c.closes || quits != c.quits {
			t.Errorf("%s: closes %v, quits %v", c.name, closes, quits)
		}
	}
}

// closes counts the close events of a World, read after a frame.
type closes struct {
	asked  world.Reader[platform.WindowCloseRequested]
	closed world.Reader[platform.WindowClosed]
}

func (c *closes) read(w *world.World) (int, int) {
	return c.asked.Count(w), c.closed.Count(w)
}

func TestByDefaultThePrimaryWindowsCloseEndsTheProgramAndAnothersClosesIt(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	main, tool := platform.NewWindow(), platform.NewWindow()
	w.AddEntities(main, tool)
	var c closes
	tool.RequestClose()
	w.RunFrames(1)
	if w.Has(tool) || !w.Has(main) || w.ShouldQuit() {
		t.Fatal("a secondary window asked to close leaves the World, which runs on")
	}
	if asked, closed := c.read(w); asked != 1 || closed != 1 {
		t.Fatalf("%d requests and %d closes, want one of each", asked, closed)
	}
	main.RequestClose()
	w.Frame(w.Time().FixedDelta)
	if !w.ShouldQuit() || !w.Has(main) {
		t.Fatal("the primary window asked to close ends the program, the window still in the World")
	}
	if asked, closed := c.read(w); asked != 1 || closed != 0 {
		t.Fatalf("%d requests and %d closes, want the request alone", asked, closed)
	}
}

func TestAProgramQuitsWithItsLastWindow(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	a, b, c := lastQuits(), lastQuits(), lastQuits()
	w.AddEntities(a, b, c)
	a.RequestClose()
	w.RunFrames(1)
	if w.Has(a) || w.ShouldQuit() || Primary(w) != b {
		t.Fatal("the first window closes, the program runs on and the next oldest is the primary")
	}
	b.RequestClose()
	c.RequestClose()
	w.RunFrames(1)
	if !w.ShouldQuit() || Count(w) != 0 {
		t.Fatal("two windows closed in one frame: the second is the last, and ends the program")
	}
}

func lastQuits() *platform.Window {
	return platform.NewWindow(platform.Closing(platform.CloseQuitsIfLast))
}

func TestAProgramRunsOnWithoutAWindowAndOpensOneLater(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	only := platform.NewWindow(platform.Closing(platform.CloseOnly))
	w.AddEntities(only)
	only.RequestClose()
	w.RunFrames(3)
	if w.ShouldQuit() || Count(w) != 0 || Primary(w) != nil || Target(w) != nil {
		t.Fatal("the last window closed, the World runs on with none")
	}
	if in := Input(w); in.Mouse().X != 0 || len(in.Gamepads()) != 0 {
		t.Fatal("without a window the input is idle")
	}
	again := platform.NewWindow()
	w.AddEntities(again)
	w.RunFrames(1)
	if Primary(w) != again || Input(w) != again.Input() {
		t.Fatal("a window opened later is the primary")
	}
}

func TestAWindowThatAsksLeavesTheCloseToTheProgram(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	doc := platform.NewWindow(platform.Closing(platform.CloseAsks))
	w.AddEntities(doc)
	unsaved := true
	var asked world.Reader[platform.WindowCloseRequested]
	w.On(world.Update, "confirm", func(f world.Frame) {
		for r := range asked.Read(f.World) {
			if !unsaved {
				f.World.RemoveEntity(r.Window)
				f.Quit()
			}
		}
	})
	doc.RequestClose()
	w.RunFrames(2)
	if !w.Has(doc) || w.ShouldQuit() {
		t.Fatal("unsaved work: the program refuses the close, and nothing happens")
	}
	unsaved = false
	doc.RequestClose()
	w.RunFrames(2)
	if w.Has(doc) || !w.ShouldQuit() {
		t.Fatal("once saved, the program closes the window and quits itself")
	}
}

func TestARuleChangesWhileTheWindowIsOpen(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	main, tool := platform.NewWindow(), platform.NewWindow()
	w.AddEntities(main, tool)
	if main.Closing() != platform.CloseQuitsIfPrimary {
		t.Fatal("a window given no rule has the default")
	}
	main.SetClosing(platform.CloseOnly)
	tool.SetClosing(platform.CloseQuits)
	main.RequestClose()
	w.RunFrames(1)
	if w.Has(main) || w.ShouldQuit() {
		t.Fatal("CloseOnly on the primary closes it alone")
	}
	tool.RequestClose()
	w.RunFrames(1)
	if !w.ShouldQuit() {
		t.Fatal("CloseQuits ends the program from the window that has it")
	}
}

func TestTheOldestWindowLeftIsThePrimary(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	a, b, c := platform.NewWindow(), platform.NewWindow(), platform.NewWindow()
	w.AddEntities(a, b, c)
	var gone []*platform.Window
	OnGone(w, func(win *platform.Window) { gone = append(gone, win) })
	w.RemoveEntity(a)
	if Primary(w) != b {
		t.Fatal("with the first window gone the second is the primary, not the last added")
	}
	w.RemoveEntity(b)
	if Primary(w) != c || len(gone) != 2 || gone[0] != a || gone[1] != b {
		t.Fatalf("primary and windows told gone: %v", gone)
	}
}

func TestThePadsPassToTheNextPrimaryWithoutConnectingAgain(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	main, tool := platform.NewWindow(), platform.NewWindow()
	w.AddEntities(main, tool)
	main.Input().ConnectGamepad(0, "pad")
	w.RunFrames(1)
	var joined world.Reader[input.GamepadConnected]
	if joined.Count(w) != 1 {
		t.Fatal("the pad connects once, on the primary window's input")
	}
	w.RemoveEntity(main)
	w.RunFrames(2)
	if len(tool.Input().Gamepads()) != 1 || tool.Input().Gamepad(0).Name() != "pad" {
		t.Fatal("the next primary's input holds the pad")
	}
	if n := joined.Count(w); n != 0 {
		t.Fatalf("the pad connected %d more times as the primary closed", n)
	}
}

func TestAWindowsOwnersAndCoversLeaveWithIt(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	main, tool := platform.NewWindow(), platform.NewWindow()
	modal := world.NewBasic()
	tool.AppendChild(modal)
	w.AddEntities(main, tool)
	Own(w, tool, modal)
	Cover(w, tool, modal)
	SetCurrent(w, tool)
	w.RemoveEntity(tool)
	r := of(w)
	if len(r.owners) != 0 || len(r.covers) != 0 || r.current != nil {
		t.Fatal("a closed window keeps no owner, cover or presentation")
	}
}

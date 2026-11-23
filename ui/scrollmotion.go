package ui

import (
	"math"
	"runtime"

	"github.com/lantervale/ember/ease"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/paint"
)

// InstantScrolls has the wheel, the keys and a click on a scrollbar's
// track move a scroll at once, where by default they glide as Chrome's
// smooth scrolling on the desktop does: a game that reads a list the
// frame after it scrolls, or wants its menus snappy. The program's own
// scrolls still glide under scroll-behavior: smooth, as the page asks.
// As WheelScroll, it is the reader's setting, so the Document in the
// window rules those it shows in its iframes (decision 0169).
func InstantScrolls() DocumentOption { return func(d *Document) { d.instant = true } }

// InstantScrolls reports whether the user's scrolls move at once: the
// Document's own setting.
func (d *Document) InstantScrolls() bool { return d.instant }

// SetInstantScrolls changes it from the next scroll.
func (d *Document) SetInstantScrolls(on bool) { d.instant = on }

// instantScrolls is the setting of the Document in the window.
func (d *Document) instantScrolls() bool {
	root := d
	for root.framed != nil {
		root = root.framed.host
	}
	return root.instant
}

// motion is what moves a glide, which sets its curve and its length, as
// Chromium's ScrollOffsetAnimationCurve: the program (scroll-behavior:
// smooth, a snap) on its programmatic curve over √distance frames, the
// keys and a track's click on ease-in-out over 9 frames, and a mouse's
// notches on ease-in-out over 12 frames down to 6 the farther they go.
type motion uint8

const (
	byProgram motion = iota
	byKeys
	byWheel
)

// lineScroll is how far an arrow scrolls, Chrome's kPixelsPerLineStep.
const lineScroll = 40

// pointsPerPreciseLine is how many points a trackpad's line is: GLFW
// counts macOS's precise deltas in tenths.
const pointsPerPreciseLine = 10

//nolint:gochecknoglobals // Chrome's curves for a programmatic and a user's smooth scroll on the desktop
var (
	programCurve = css.CubicBezier(0.4, 0, 0, 1)
	userCurve    = css.CubicBezier(0.42, 0, 0.58, 1)
)

// glide is a smooth scroll along curve: from and to, begun at start on
// the Document's clock (-1 until the next frame), lasting dur seconds,
// by what moved it.
type glide struct {
	from, to   geom.Vec2
	start, dur float32
	by         motion
	curve      ease.Func
}

func newGlide(from, to geom.Vec2, by motion) *glide {
	curve := userCurve
	if by == byProgram {
		curve = programCurve
	}
	return &glide{from: from, to: to, start: -1, dur: glideTime(by, to.Sub(from)), by: by, curve: curve}
}

// longest is delta's longer side, its sign kept, as Chromium's
// MaximumDimension.
func longest(delta geom.Vec2) float32 {
	if abs32(delta.X) > abs32(delta.Y) {
		return delta.X
	}
	return delta.Y
}

// glideTime is how long Chrome takes to glide by delta when by moves it.
func glideTime(by motion, delta geom.Vec2) float32 {
	const frame = 1.0 / 60
	d := float64(abs32(longest(delta)))
	switch by {
	case byKeys:
		return 9 * frame
	case byWheel:
		// 12 frames up to 120 points, 6 from 480, a line between
		return float32(min(max(14-d/60, 6), 12) * frame)
	case byProgram:
	}
	return float32(math.Min(math.Sqrt(d), 90) * frame)
}

// at is where g is at now on the Document's clock.
func (g *glide) at(now float32) geom.Vec2 {
	if g.start < 0 {
		return g.from
	}
	t := float32(1)
	if g.dur > 0 {
		t = min(max((now-g.start)/g.dur, 0), 1)
	}
	return g.from.Add(g.to.Sub(g.from).Scale(g.curve(t)))
}

// velocity is how fast g moves along its longer side at now, in points
// a second.
func (g *glide) velocity(now float32) float32 {
	if g.start < 0 || g.dur <= 0 {
		return 0
	}
	const h = 1e-3
	t := min(max((now-g.start)/g.dur, h), 1-h)
	slope := (g.curve(t+h) - g.curve(t-h)) / (2 * h)
	return slope * longest(g.to.Sub(g.from)) / g.dur
}

// retarget sends g to to from where it is at now, as Chromium's
// UpdateTarget: the new leg keeps g's speed through the slope of its
// curve's start, and lasts no longer than that speed takes to get there,
// with some room for the ease out.
func (g *glide) retarget(now float32, to geom.Vec2) {
	const eps = 0.01
	if g.start < 0 {
		g.to, g.dur = to, glideTime(g.by, to.Sub(g.from))
		return
	}
	if abs32(longest(g.to.Sub(to))) < eps {
		g.to = to
		return
	}
	cur := g.at(now)
	delta := to.Sub(cur)
	v := g.velocity(now)
	dur := glideTime(g.by, delta)
	if d := longest(delta); abs32(d) < eps {
		dur = 0
	} else if abs32(v) >= eps {
		if bound := d / v * 2.5; bound >= 0 && bound < dur {
			dur = bound
		}
	}
	g.from, g.to, g.start = cur, to, now
	if dur < eps {
		g.from, g.dur = to, 0
		return
	}
	slope := float64(min(max(v*dur/longest(delta), -1000), 1000))
	g.curve, g.dur = css.CubicBezier(0.42, 0.42*slope, 0.58, 1), dur
}

// pageStep is how far a page key or a click on a track scrolls a
// snapport of length points, as Chrome's CalculatePageStep: seven
// eighths of it, on macOS at least all of it but 40 points.
func pageStep(length float32) float32 {
	l := float32(math.Floor(float64(length) + 0.5))
	step := float32(math.Trunc(float64(l * 0.875)))
	if runtime.GOOS == "darwin" {
		step = max(step, l-40)
	}
	return max(step, 1)
}

// snapportOf is the size of b's scrollport its scroll-padding leaves.
func snapportOf(b *box.Box) geom.Vec2 {
	bars := paint.BarsOf(b, origin(b))
	port := geom.V2(bars.View.W-b.Scrollbar.X, bars.View.H-b.Scrollbar.Y)
	if st := snapStyle(b); st != nil {
		p := st.Snap.Padding
		port.X -= p.Left.At(port.X) + p.Right.At(port.X)
		port.Y -= p.Top.At(port.Y) + p.Bottom.At(port.Y)
	}
	return port
}

// userScroll moves b, from where the user's glide under way is going, to
// to, already snapped: gliding by by unless the user's scrolls are
// instant, a glide under way taken on from where it is.
func (d *Document) userScroll(b *box.Box, to geom.Vec2, by motion) {
	s := d.scrollerOf(b.Node)
	s.wheeling = false
	d.noteScrolled(s, b.Scroll, to)
	if d.instantScrolls() {
		s.glide = nil
		d.setScroll(b, to)
		return
	}
	d.unsettled = true
	if g := s.glide; g != nil && g.by != byProgram {
		g.retarget(d.clock, to)
		return
	}
	if to == b.Scroll {
		s.glide = nil
		return
	}
	s.glide = newGlide(b.Scroll, to, by)
}

// userAim is where b's user scroll goes on from: the end of the user's
// glide under way, else where b is.
func (d *Document) userAim(b *box.Box) geom.Vec2 {
	if s := d.scrollers[b.Node]; s != nil && s.glide != nil && s.glide.by != byProgram {
		return s.glide.to
	}
	return b.Scroll
}

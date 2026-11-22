package css

import (
	"math"
	"slices"
	"strings"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ease"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// Transitions runs CSS transitions: Compute starts them where a style
// change is covered by the element's transition list, and Step advances
// them. Paint reads dom.Data(n).Shown, which is Style itself while nothing
// runs on n.
type Transitions struct {
	nodes map[*dom.Node]*nodeAnim
	root  *dom.Node
	// given is the sheets the last Compute had, all the same behind the
	// user agent's, kept so a frame like the last allocates nothing.
	given, all []*Sheet
	changed    func(n *dom.Node, old, next *style.Computed)
	m, pm      matcher
	// computed counts the elements restyled, for tests.
	computed int
	overlay  bool
	viewport bool
	browser  bool
	// media is what @media rules are answered for, off the rules of each
	// sheet they leave out, and ranks the sheets' cascade layers.
	media Media
	off   [][]bool
	ranks [][]int32
	// startOff is off with the rules of @starting-style on, when some
	// sheet has one (starts).
	startOff [][]bool
	starts   bool
	// stale is set once media has changed which rules apply, or what a
	// unit some length is in stands for: the next Compute restyles the
	// whole tree.
	stale bool
	// env is what lengths are measured against: the fonts, and media's
	// viewport, and the containers' boxes.
	env measures
	// queries are the sheets' @container rules, and queried the
	// viewport Requery last answered them in.
	queries []*containerRule
	queried [2]float32
	// anims are the animations of the elements whose style has an
	// animation list, events what they did since Events, and frames the
	// @keyframes rules by name, worked out again once rekey is set.
	anims  map[*dom.Node]*nodeAnims
	events []AnimationEvent
	frames map[string]*Keyframes
	rekey  bool
	// km and kpm match the elements whose keyframes are worked out,
	// probed counting them.
	km, kpm matcher
	probed  int
	// scratch is where a transition's ends are tried for interpolating.
	scratch style.Computed
}

// nodeAnim is one element's running transitions and the style it shows.
type nodeAnim struct {
	shown style.Computed
	// runs are the transitions by their longhand's place in longhands.
	runs map[int]*run
	// follows is set while the element shows an ancestor's colour
	// transition through inheritance, built once shown is.
	follows, touched, built bool
	// after is the style its transitions on Style run to, while ending
	// is set.
	after  style.Computed
	ending bool
}

// run is one transition, between the values of its longhand in two
// styles.
type run struct {
	from, to, reverseStart *style.Computed
	factor                 float32
	elapsed                float32
	delay, length          float32
	easing                 ease.Func
}

func NewTransitions() *Transitions {
	return &Transitions{nodes: map[*dom.Node]*nodeAnim{}, anims: map[*dom.Node]*nodeAnims{}}
}

// OverlayScrollbars has every box that shows scrollbars keep no gutter
// for them (ScrollbarWidth 0, ScrollbarOverlay set), from the next
// element styled: set it before the first Compute.
func (t *Transitions) OverlayScrollbars() { t.overlay = true }

// ScrollingViewport has the viewport take the root's overflow, or body's
// while the root's is visible, as a browser's does: that element's own
// Overflow is visible, and the root's Viewport (body's, when it gave its
// overflow) is the viewport's style, which scrolls unless the overflow it
// took is hidden. Set it before the first Compute.
func (t *Transitions) ScrollingViewport() { t.viewport = true }

// BrowserSheets has the sheets start from a browser's own
// (BrowserDefault after Default) and style attributes read as a
// browser's (Browser). Set it before the first Compute.
func (t *Transitions) BrowserSheets() { t.browser = true }

// SetMedia answers the sheets' @media rules for m from now on, and
// reports whether that changes which rules apply: the caller then flags
// the root DirtyStyle, and the next Compute restyles the whole tree. A
// viewport that crosses no rule's boundary costs only the queries.
func (t *Transitions) SetMedia(m Media) bool {
	if m == t.media {
		return false
	}
	t.media = m
	resized := t.env.width != m.Width || t.env.height != m.Height
	flipped := t.env.dark != m.Dark
	rescaled := t.env.dppx != m.Resolution
	t.env.width, t.env.height, t.env.dark, t.env.dppx = m.Width, m.Height, m.Dark, m.Resolution
	if t.all == nil {
		return false
	}
	if off := mediaOff(t.all, m); !sameOff(off, t.off) {
		t.off, t.stale, t.rekey = off, true, true
		t.startOff = rulesOff(t.all, m, true)
	} else if resized && t.env.used&viewportUnits != 0 || flipped && t.env.prefersRead || rescaled {
		t.stale = true
	}
	return t.stale
}

// MeasureFonts has the units that read a font (ex, ch, cap, ic, lh and
// the root's) ask font what they stand for; without it they are CSS's
// fallbacks off the font size. Set it before the first Compute.
func (t *Transitions) MeasureFonts(font func(style.Font) FontUnits) { t.env.font = font }

// FontsChanged tells that what MeasureFonts answers may have changed, a
// font having landed, and reports whether some length reads it: the
// caller then flags the root DirtyStyle, and the next Compute restyles
// the whole tree.
func (t *Transitions) FontsChanged() bool {
	if t.env.used&fontUnits == 0 {
		return false
	}
	t.stale = true
	return true
}

// SetSheets has the cascade read sheets, the author's, from now on:
// what Matched and Computed answer from at once, and what the next
// Compute styles the whole tree again from when they are not those it
// had. Compute calls it with its own.
func (t *Transitions) SetSheets(sheets []*Sheet) {
	if slices.Equal(t.given, sheets) && t.all != nil {
		return
	}
	t.stale = t.stale || t.all != nil
	t.rekey = true
	t.given = slices.Clone(sheets)
	agent := userAgent()
	if t.browser {
		agent = browserAgent()
	}
	t.all = append([]*Sheet{agent}, sheets...)
	t.off, t.ranks, t.queries = mediaOff(t.all, t.media), layerRanks(t.all), containerQueries(t.all)
	t.starts = startingRules(t.all)
	t.startOff = rulesOff(t.all, t.media, true)
}

// Compute restyles root as css.Compute does, starting a transition for
// each changed property its new style's transitions cover.
// Nothing is done while root and its tree are clean.
func (t *Transitions) Compute(root *dom.Node, sheets []*Sheet) []Warning {
	t.root = root
	dirty := dom.DirtyStyle | dom.DirtyTree | dom.DirtyState | dom.DirtyDescendant
	if d := dom.Data(root); d.Style != nil && d.Dirty&dirty == 0 && !t.stale {
		return nil
	}
	t.SetSheets(sheets)
	t.m.off, t.pm.off, t.m.ranks, t.pm.ranks, t.m.env, t.pm.env = t.off, t.off, t.ranks, t.ranks, &t.env, &t.env
	if t.changed == nil {
		t.changed = t.restyled
	}
	r := restyler{sheets: t.all, changed: t.changed, m: &t.m, pm: &t.pm, over: t.overrides, transit: t.transition,
		transits: t.Running, afterChange: t.after, computed: &t.computed, overlay: t.overlay, viewport: t.viewport,
		browser: t.browser, env: &t.env}
	r.run(root, t.stale)
	t.stale = false
	if (len(t.anims) > 0 || t.rekey) && t.animate() {
		r.run(root, false)
	}
	t.refresh()
	return r.warns
}

// Restyled is how many elements Compute has worked the style of out so
// far: what a change cost, for tests and tools.
func (t *Transitions) Restyled() int { return t.computed }

// Step advances every transition and animation by dt seconds and writes
// their painted values into Shown. One that moves what Style holds flags
// its element DirtyStyle for the next Compute. It reports whether any
// still runs.
func (t *Transitions) Step(dt float32) bool {
	running := false
	for n, a := range t.anims {
		if !t.live(n) {
			t.cancel(n, a.list)
			delete(t.anims, n)
			continue
		}
		for _, an := range a.list {
			if !an.Paused {
				an.elapsed += dt
			}
		}
		t.update(n, a)
		running = running || slices.ContainsFunc(a.list, (*animation).running)
	}
	for n, a := range t.nodes {
		if !t.attached(n) {
			delete(t.nodes, n)
			continue
		}
		for i, r := range a.runs {
			r.elapsed += dt
			switch {
			case longhands[i].prop.anim.kind != painted:
				restyle(n)
			case r.over():
				delete(a.runs, i)
			}
		}
	}
	t.refresh()
	for _, a := range t.nodes {
		running = running || len(a.runs) > 0
	}
	return running
}

// Running reports whether n shows a transition.
func (t *Transitions) Running(n *dom.Node) bool { return t.nodes[n] != nil }

func (t *Transitions) attached(n *dom.Node) bool {
	for ; n != nil; n = n.Parent() {
		if n == t.root {
			return true
		}
	}
	return false
}

// restyled watches n's animation list, and shows next while nothing runs
// on n.
func (t *Transitions) restyled(n *dom.Node, old, next *style.Computed) {
	t.watch(n, next)
	if old == nil || t.nodes[n] == nil {
		setShown(n, next)
	}
}

// transition starts, retargets or cancels n's transitions as its style
// goes to next, worked out with its animations and without its
// transitions, and returns what those running on Style set on it now.
func (t *Transitions) transition(n *dom.Node, next *style.Computed) []override {
	old := dom.Data(n).Style
	a := t.nodes[n]
	if old == nil || old.Display == style.DisplayNone {
		// An element that showed no box starts from its starting style,
		// and transitions nothing without one.
		if a != nil {
			delete(t.nodes, n)
			a = nil
		}
		if !t.starts || next.Display == style.DisplayNone || !anyTransition(next.Transitions) {
			return nil
		}
		old = t.probeIn(n, nil, t.startOff)
	}
	if a == nil && !anyTransition(next.Transitions) {
		return nil
	}
	shown := old
	if a != nil && a.built {
		shown = &a.shown
	}
	var to *style.Computed
	for _, i := range moving {
		if t.animated(n, i) {
			continue
		}
		var r *run
		if a != nil {
			r = a.runs[i]
		}
		nr, keep := t.retarget(i, r, shown, next, &to)
		switch {
		case keep:
		case nr == nil:
			delete(a.runs, i)
		default:
			if a == nil {
				a = &nodeAnim{runs: map[int]*run{}}
				t.nodes[n] = a
			}
			a.runs[i] = nr
		}
	}
	if a == nil {
		return nil
	}
	var over []override
	for _, i := range moving {
		r := a.runs[i]
		if r == nil || longhands[i].prop.anim.kind == painted {
			continue
		}
		over = append(over, r.override(i))
		if r.over() {
			delete(a.runs, i)
		}
	}
	if a.ending = len(over) > 0; a.ending {
		a.after = *next
	}
	return over
}

// after is the style n's transitions on Style run to, nil while none
// runs.
func (t *Transitions) after(n *dom.Node) *style.Computed {
	if a := t.nodes[n]; a != nil && a.ending {
		return &a.after
	}
	return nil
}

// retarget is the transition of the longhand at place i of longhands as
// the style goes to next from shown, r running on it: keep when r runs
// on, or none starts or stops; else the one that starts, nil when r stops.
// to is next's copy the transitions that start end at, made by the first.
func (t *Transitions) retarget(i int, r *run, shown, next *style.Computed, to **style.Computed) (*run, bool) {
	l := &longhands[i]
	an := &l.prop.anim
	tr, ok := transitionFor(next.Transitions, l.name)
	switch {
	case r == nil && (!ok || an.kind == discrete && !tr.Discrete):
		return nil, true
	case r != nil && t.same(i, r.to, next):
		return r, true
	case !ok:
		return nil, false
	case r == nil && t.same(i, shown, next):
		return nil, true
	}
	from := shown
	if r != nil {
		from = r.now(i, shown)
	}
	if !tr.Discrete && !t.interpolates(i, from, next) {
		return nil, r == nil
	}
	if *to == nil {
		copied := *next
		*to = &copied
	}
	nr := &run{from: from, to: *to, reverseStart: from, factor: 1,
		delay: tr.Delay, length: tr.Duration, easing: easing(tr.Easing)}
	if r != nil && t.same(i, r.reverseStart, next) {
		nr.reverseStart = r.to
		nr.factor = clamp01f(r.easing(r.progress())*r.factor + 1 - r.factor)
		nr.length *= nr.factor
		if nr.delay < 0 {
			nr.delay *= nr.factor
		}
	}
	if nr.delay+nr.length <= 0 {
		return nil, r == nil
	}
	return nr, false
}

// same reports whether a and b hold the same value of the longhand at
// place i of longhands.
func (t *Transitions) same(i int, a, b *style.Computed) bool {
	p := &longhands[i].prop
	if p.anim.equal != nil {
		return p.anim.equal(a, b)
	}
	t.scratch = *b
	p.inherit(&t.scratch, a)
	return t.scratch.Equal(b)
}

// interpolates reports whether the longhand at place i of longhands moves
// between a's value and b's, where it would otherwise flip.
func (t *Transitions) interpolates(i int, a, b *style.Computed) bool {
	an := &longhands[i].prop.anim
	return an.kind != discrete && an.lerp != nil && an.lerp(&t.scratch, a, b, 0.5)
}

// anyTransition reports whether list runs a transition on some property.
func anyTransition(list []style.Transition) bool {
	return slices.ContainsFunc(list, func(t style.Transition) bool { return t.Duration+t.Delay > 0 })
}

// refresh rebuilds every animated element's Shown, then carries colour
// transitions down to descendants whose colour is inherited.
func (t *Transitions) refresh() {
	for _, a := range t.nodes {
		a.touched = false
	}
	for n, a := range t.anims {
		if len(a.paints) > 0 && t.nodes[n] == nil {
			t.nodes[n] = &nodeAnim{runs: map[int]*run{}}
		}
	}
	for n, a := range t.nodes {
		if t.own(n, a) {
			t.build(n, a)
		}
	}
	for n, a := range t.nodes {
		if t.ownColor(n, a) && !a.follows {
			t.carry(n, a.shown.Color)
		}
	}
	for n, a := range t.nodes {
		if !a.touched && (a.follows || !t.own(n, a)) {
			delete(t.nodes, n)
			setShown(n, dom.Data(n).Style)
		}
	}
}

// own reports whether n shows a transition or an animation of its own.
func (t *Transitions) own(n *dom.Node, a *nodeAnim) bool {
	return len(a.runs) > 0 || t.anims[n] != nil && len(t.anims[n].paints) > 0
}

// ownColor reports whether n shows a colour transition or animation of
// its own.
func (t *Transitions) ownColor(n *dom.Node, a *nodeAnim) bool {
	_, ok := a.runs[colorIndex]
	return ok || t.animated(n, colorIndex)
}

// build shows n's style with its animations' painted values over it,
// then its transitions' over those, as CSS's cascade has them: a colour
// first, which the others given as currentColor follow.
func (t *Transitions) build(n *dom.Node, a *nodeAnim) {
	a.shown = *dom.Data(n).Style
	var paints []paintValue
	if an := t.anims[n]; an != nil {
		paints = an.paints
	}
	var current *color.Color
	if t.ownColor(n, a) {
		for _, p := range paints {
			if p.index == colorIndex {
				p.on(&a.shown, nil)
				current = &a.shown.Color
			}
		}
		if r, ok := a.runs[colorIndex]; ok {
			r.on(&a.shown, colorIndex)
		}
		a.shown.FollowColor(a.shown.Color)
	}
	for _, p := range paints {
		if p.index != colorIndex {
			p.on(&a.shown, current)
		}
	}
	for i, r := range a.runs {
		if i != colorIndex && longhands[i].prop.anim.kind == painted {
			r.on(&a.shown, i)
		}
	}
	a.touched, a.built = true, true
	setShown(n, &a.shown)
}

// carry shows c as the colour of n's children that inherit it, and of
// theirs.
func (t *Transitions) carry(n *dom.Node, c color.Color) {
	for _, k := range n.Flow() {
		if k.IsText() {
			continue
		}
		s := dom.Data(k).Style
		if s == nil || !s.ColorInherited {
			continue
		}
		a := t.nodes[k]
		if a == nil {
			a = &nodeAnim{runs: map[int]*run{}}
			t.nodes[k] = a
		}
		if t.ownColor(k, a) {
			continue
		}
		if !a.touched {
			t.build(k, a)
		}
		a.follows = !t.own(k, a)
		a.shown.Color = c
		a.shown.FollowColor(c)
		t.carry(k, c)
	}
}

// setShown points n and its text nodes at s.
func setShown(n *dom.Node, s *style.Computed) {
	dom.Data(n).Shown = s
	for _, k := range n.Children() {
		if k.IsText() {
			dom.Data(k).Shown = s
		}
	}
}

// over reports whether r has run its course. One on Style ends at the
// restyle after, which sets its last value: the value the next change
// starts from.
func (r *run) over() bool { return r.elapsed >= r.delay+r.length }

func (r *run) progress() float32 {
	if r.length <= 0 {
		if r.elapsed >= r.delay {
			return 1
		}
		return 0
	}
	return clamp01f((r.elapsed - r.delay) / r.length)
}

// on writes the value r gives the longhand i now into dst: its start
// while it waits out its delay.
func (r *run) on(dst *style.Computed, i int) {
	p := &longhands[i].prop
	if r.elapsed < r.delay {
		p.inherit(dst, r.from)
		return
	}
	p.blend(dst, r.from, r.to, r.easing(r.progress()))
}

// now is under with the value r gives the longhand i now.
func (r *run) now(i int, under *style.Computed) *style.Computed {
	s := *under
	r.on(&s, i)
	return &s
}

// override is what r sets on Style now.
func (r *run) override(i int) override {
	if r.elapsed < r.delay {
		return override{index: i, src: r.from}
	}
	return override{index: i, src: r.from, to: r.to, t: r.easing(r.progress())}
}

func clamp01f(v float32) float32 { return min(max(v, 0), 1) }

// transitionFor finds the transition the list gives property name: the
// last entry naming it, a shorthand of it or all.
func transitionFor(list []style.Transition, name string) (style.Transition, bool) {
	for i := len(list) - 1; i >= 0; i-- {
		p := list[i].Property
		if p == name || p == kwAll || slices.Contains(shorthands[p], name) {
			return list[i], list[i].Duration+list[i].Delay > 0
		}
	}
	return style.Transition{}, false
}

//nolint:gochecknoglobals // derived from the longhands
var (
	// colorIndex is color's place in longhands.
	colorIndex = orderIndex["color"]
	// displayIndex is display's.
	displayIndex = orderIndex[propDisplay]
	// moving are the places in longhands of those that transition, in
	// their order: the discrete ones under allow-discrete.
	moving = func() []int {
		var out []int
		for i, l := range longhands {
			if l.prop.anim.moves() || l.prop.anim.kind == discrete {
				out = append(out, i)
			}
		}
		return out
	}()
)

// identity is the neutral function of op's kind.
func identity(op style.TransformOp) style.TransformOp {
	switch op.Kind {
	case style.Translate:
		return style.TransformOp{Kind: style.Translate, X: style.Px(0), Y: style.Px(0)}
	case style.ScaleOp:
		return style.TransformOp{Kind: style.ScaleOp, A: [6]float32{1, 1}}
	case style.Matrix:
		return style.TransformOp{Kind: style.Matrix, A: [6]float32{1, 0, 0, 1, 0, 0}}
	case style.Rotate, style.Skew:
		// zero angles, as below
	}
	return style.TransformOp{Kind: op.Kind}
}

// lerpTransform interpolates function by function when both lists have
// the same functions (none stands for the other's identities), and
// otherwise as decomposed matrices. A list it can't make a matrix of
// (a percentage translation) switches at the end.
func lerpTransform(a, b []style.TransformOp, t float32) []style.TransformOp {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	if len(a) == 0 {
		a = identities(b)
	}
	if len(b) == 0 {
		b = identities(a)
	}
	if sameKinds(a, b) {
		out := make([]style.TransformOp, len(a))
		for i := range a {
			out[i] = lerpOp(a[i], b[i], t)
		}
		return out
	}
	ma, okA := toMatrix(a)
	mb, okB := toMatrix(b)
	if !okA || !okB {
		if t < 1 {
			return a
		}
		return b
	}
	m := recompose(decompose(ma).lerp(decompose(mb), t))
	return []style.TransformOp{{Kind: style.Matrix, A: m}}
}

func identities(ops []style.TransformOp) []style.TransformOp {
	out := make([]style.TransformOp, len(ops))
	for i, op := range ops {
		out[i] = identity(op)
	}
	return out
}

func sameKinds(a, b []style.TransformOp) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Kind != b[i].Kind {
			return false
		}
	}
	return true
}

func lerpOp(a, b style.TransformOp, t float32) style.TransformOp {
	switch a.Kind {
	case style.Translate:
		x, _ := lerpLength(a.X, b.X, t, false)
		y, _ := lerpLength(a.Y, b.Y, t, false)
		return style.TransformOp{Kind: style.Translate, X: x, Y: y}
	case style.Matrix:
		return style.TransformOp{Kind: style.Matrix, A: recompose(decompose(a.A).lerp(decompose(b.A), t))}
	case style.ScaleOp, style.Rotate, style.Skew:
		// argument by argument, as below
	}
	out := style.TransformOp{Kind: a.Kind}
	for i := range out.A {
		out.A[i] = lerpf(a.A[i], b.A[i], t)
	}
	return out
}

// toMatrix multiplies a list into one matrix(a, b, c, d, e, f), false
// when a translation has a percentage.
func toMatrix(ops []style.TransformOp) ([6]float32, bool) {
	m := [6]float64{1, 0, 0, 1, 0, 0}
	for _, op := range ops {
		a := func(i int) float64 { return float64(op.A[i]) }
		var n [6]float64
		switch op.Kind {
		case style.Translate:
			if op.X.Percent != 0 || op.Y.Percent != 0 {
				return [6]float32{}, false
			}
			n = [6]float64{1, 0, 0, 1, float64(op.X.Px), float64(op.Y.Px)}
		case style.ScaleOp:
			n = [6]float64{a(0), 0, 0, a(1), 0, 0}
		case style.Rotate:
			n = [6]float64{math.Cos(a(0)), math.Sin(a(0)), -math.Sin(a(0)), math.Cos(a(0)), 0, 0}
		case style.Skew:
			n = [6]float64{1, math.Tan(a(1)), math.Tan(a(0)), 1, 0, 0}
		case style.Matrix:
			n = [6]float64{a(0), a(1), a(2), a(3), a(4), a(5)}
		}
		m = [6]float64{
			m[0]*n[0] + m[2]*n[1], m[1]*n[0] + m[3]*n[1],
			m[0]*n[2] + m[2]*n[3], m[1]*n[2] + m[3]*n[3],
			m[0]*n[4] + m[2]*n[5] + m[4], m[1]*n[4] + m[3]*n[5] + m[5],
		}
	}
	var out [6]float32
	for i, v := range m {
		out[i] = float32(v)
	}
	return out, true
}

// decomposed is a 2D matrix as Chrome decomposes it: translate · rotate
// (angle) · skewX (k, as a tangent) · scale(sx, sy).
type decomposed struct {
	tx, ty, angle, sx, sy, k float64
}

func decompose(m [6]float32) decomposed {
	a, b, c, d := float64(m[0]), float64(m[1]), float64(m[2]), float64(m[3])
	sx := math.Hypot(a, b)
	angle := math.Atan2(b, a)
	cs, sn := math.Cos(angle), math.Sin(angle)
	shear, sy := cs*c+sn*d, -sn*c+cs*d
	k := 0.0
	if sy != 0 {
		k = shear / sy
	}
	return decomposed{tx: float64(m[4]), ty: float64(m[5]), angle: angle, sx: sx, sy: sy, k: k}
}

func (x decomposed) lerp(y decomposed, t float32) decomposed {
	if d := y.angle - x.angle; d > math.Pi {
		x.angle += 2 * math.Pi
	} else if d < -math.Pi {
		y.angle += 2 * math.Pi
	}
	f := func(a, b float64) float64 { return a + (b-a)*float64(t) }
	return decomposed{f(x.tx, y.tx), f(x.ty, y.ty), f(x.angle, y.angle), f(x.sx, y.sx), f(x.sy, y.sy), f(x.k, y.k)}
}

func recompose(x decomposed) [6]float32 {
	cs, sn := math.Cos(x.angle), math.Sin(x.angle)
	shear := x.k * x.sy
	return [6]float32{
		float32(cs * x.sx), float32(sn * x.sx),
		float32(cs*shear - sn*x.sy), float32(sn*shear + cs*x.sy),
		float32(x.tx), float32(x.ty),
	}
}

//nolint:gochecknoglobals // CSS's named easing curves, fixed
var (
	easeCurve      = cubicBezier(0.25, 0.1, 0.25, 1)
	easeInCurve    = cubicBezier(0.42, 0, 1, 1)
	easeOutCurve   = cubicBezier(0, 0, 0.58, 1)
	easeInOutCurve = cubicBezier(0.42, 0, 0.58, 1)
)

// easing reads a transition-timing-function as written in the computed
// style.
func easing(src string) ease.Func {
	cs := items(componentsOf(src))
	if len(cs) != 1 {
		return easeCurve
	}
	c := cs[0]
	switch {
	case isWord(c, "linear"):
		return ease.Linear
	case isWord(c, "ease-in"):
		return easeInCurve
	case isWord(c, "ease-out"):
		return easeOutCurve
	case isWord(c, "ease-in-out"):
		return easeInOutCurve
	case isWord(c, "step-start"):
		return steps(1, kwStart)
	case isWord(c, "step-end"):
		return steps(1, kwEnd)
	case c.Kind == Function:
		args := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
		switch strings.ToLower(c.Text) {
		case "cubic-bezier":
			if len(args) == 4 {
				return cubicBezier(args[0][0].Num, args[1][0].Num, args[2][0].Num, args[3][0].Num)
			}
		case "steps":
			pos := kwEnd
			if len(args) == 2 {
				pos = strings.ToLower(args[1][0].Text)
			}
			return steps(int(args[0][0].Num), pos)
		}
	}
	return easeCurve
}

// cubicBezier is CSS's cubic-bezier(x1, y1, x2, y2): y for the curve's
// point at x = t.
func cubicBezier(x1, y1, x2, y2 float64) ease.Func {
	bez := func(p1, p2, s float64) float64 {
		return 3*(1-s)*(1-s)*s*p1 + 3*(1-s)*s*s*p2 + s*s*s
	}
	slope := func(p1, p2, s float64) float64 {
		return 3*(1-s)*(1-s)*p1 + 6*(1-s)*s*(p2-p1) + 3*s*s*(1-p2)
	}
	return func(t float32) float32 {
		x := float64(t)
		if x <= 0 || x >= 1 {
			return t
		}
		s := x
		for range 8 {
			d := slope(x1, x2, s)
			if math.Abs(d) < 1e-7 {
				break
			}
			s -= (bez(x1, x2, s) - x) / d
		}
		if math.Abs(bez(x1, x2, s)-x) > 1e-6 {
			lo, hi := 0.0, 1.0
			s = x
			for range 40 {
				if bez(x1, x2, s) < x {
					lo = s
				} else {
					hi = s
				}
				s = (lo + hi) / 2
			}
		}
		return float32(bez(y1, y2, s))
	}
}

// steps is CSS's steps(n, position).
func steps(n int, pos string) ease.Func {
	jumps := n
	switch pos {
	case "jump-both":
		jumps = n + 1
	case "jump-none":
		jumps = max(n-1, 1)
	}
	start := pos == kwStart || pos == "jump-start" || pos == "jump-both"
	return func(t float32) float32 {
		step := math.Floor(float64(t) * float64(n))
		if start {
			step++
		}
		if t >= 0 && step < 0 {
			step = 0
		}
		return float32(min(step, float64(jumps)) / float64(jumps))
	}
}

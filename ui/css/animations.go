package css

import (
	"math"
	"slices"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ease"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// AnimationEventKind is what an AnimationEvent reports, as the DOM's
// animationstart, animationiteration, animationend and animationcancel.
type AnimationEventKind uint8

const (
	AnimationStart AnimationEventKind = iota
	AnimationIteration
	AnimationEnd
	AnimationCancel
)

// AnimationEvent is an animation of Node reaching a point: Elapsed is its
// active time then, in seconds, as the DOM event's elapsedTime.
type AnimationEvent struct {
	Node    *dom.Node
	Name    string
	Kind    AnimationEventKind
	Elapsed float32
}

// nodeAnims is one element's animations, in its animation list's order.
type nodeAnims struct {
	list []*animation
	// paints are the values they show through Shown, over sets on Style.
	paints []paintValue
	over   []override
	// restyled is set when the element's style changed since its
	// animations were matched to it, and hidden when they were matched
	// under display: none.
	restyled, hidden bool
}

type animation struct {
	style.Animation
	elapsed float32
	frames  *Keyframes
	tracks  []track
	// phase and iteration are where the last update found it, for its
	// events: none before the first.
	phase     phase
	iteration float64
}

type phase int8

const (
	idle phase = iota
	before
	active
	after
)

type trackKind uint8

const (
	paintTrack trackKind = iota
	lengthTrack
	discreteTrack
)

// track is one longhand an animation moves, with the keyframes that set
// it, an implicit one at 0 and 1 where none does.
type track struct {
	name  string
	kind  trackKind
	index int
	keys  []key
}

// key is a keyframe's style for a track: the element's style with the
// keyframe's declarations over it, and the easing of the interval it
// begins.
type key struct {
	at     float32
	easing ease.Func
	s      *style.Computed
}

// paintValue is an animatable property's value now, v, between the
// styles of two keyframes.
type paintValue struct {
	index    int
	from, to *style.Computed
	e        float32
	v        value
}

// override is a value an animation gives Style: src's value of the
// longhand name, or for a length, src's and to's at t between.
type override struct {
	name    string
	src, to *style.Computed
	t       float32
	length  int
}

func (o override) apply(c *style.Computed) {
	if o.length >= 0 {
		get := lengthProps[o.length].get
		*get(c) = lerpLength(*get(o.src), *get(o.to), o.t)
		return
	}
	props[o.name].inherit(c, o.src)
}

type lengthProp struct {
	name string
	get  func(*style.Computed) *style.Length
}

// lengthProps are the longhands of a length that animations interpolate
// on Style, where layout reads them: between two lengths, as calc() of
// px and %, and as discrete between an auto or a none and a length.
//
//nolint:gochecknoglobals,goconst // the interpolated lengths' table
var lengthProps = func() []lengthProp {
	out := []lengthProp{
		{propWidth, func(c *style.Computed) *style.Length { return &c.Size.Width }},
		{propHeight, func(c *style.Computed) *style.Length { return &c.Size.Height }},
		{"min-width", func(c *style.Computed) *style.Length { return &c.MinSize.Width }},
		{"min-height", func(c *style.Computed) *style.Length { return &c.MinSize.Height }},
		{"max-width", func(c *style.Computed) *style.Length { return &c.MaxSize.Width }},
		{"max-height", func(c *style.Computed) *style.Length { return &c.MaxSize.Height }},
	}
	sides := func(prefix string, get func(*style.Computed) *style.Sides[style.Length]) {
		for i, side := range []string{kwTop, kwRight, kwBottom, kwLeft} {
			out = append(out, lengthProp{prefix + side, func(c *style.Computed) *style.Length { return sideOf(get(c), i) }})
		}
	}
	sides("", func(c *style.Computed) *style.Sides[style.Length] { return &c.Inset })
	sides("margin-", func(c *style.Computed) *style.Sides[style.Length] { return &c.Margin })
	sides("padding-", func(c *style.Computed) *style.Sides[style.Length] { return &c.Padding })
	return out
}()

// Events drains the animation events since the last call, in the order
// they happened.
func (t *Transitions) Events() []AnimationEvent {
	ev := t.events
	t.events = nil
	return ev
}

// overrides is what n's animations set on its Style.
func (t *Transitions) overrides(n *dom.Node) []override {
	if len(t.anims) == 0 {
		return nil
	}
	if a := t.anims[n]; a != nil {
		return a.over
	}
	return nil
}

// watch notes that n's style changed: its animation list is matched again
// at the next animate.
func (t *Transitions) watch(n *dom.Node, next *style.Computed) {
	a := t.anims[n]
	if a == nil {
		if next.Animations == nil {
			return
		}
		a = &nodeAnims{}
		t.anims[n] = a
	}
	a.restyled = true
}

// animated reports whether an animation of n shows the animatable
// property at index i.
func (t *Transitions) animated(n *dom.Node, i int) bool {
	if a := t.anims[n]; a != nil {
		return slices.ContainsFunc(a.paints, func(p paintValue) bool { return p.index == i })
	}
	return false
}

// animate matches every watched element's animation list to the
// animations it runs, starting, updating and cancelling them, then works
// out their values. It reports whether that changed a Style, whose
// element it flags DirtyStyle.
func (t *Transitions) animate() bool {
	if t.rekey {
		t.frames = nil
	}
	dirty := false
	for n, a := range t.anims {
		if !t.live(n) {
			t.cancel(n, a.list)
			delete(t.anims, n)
			continue
		}
		hidden := hiddenNode(n)
		if a.restyled || t.rekey || hidden != a.hidden {
			t.matchAnimations(n, a, hidden)
		}
		if len(a.list) == 0 && dom.Data(n).Style.Animations == nil {
			delete(t.anims, n)
			if len(a.over) > 0 {
				restyle(n)
				dirty = true
			}
			continue
		}
		if t.update(n, a) {
			dirty = true
		}
	}
	t.rekey = false
	return dirty
}

// matchAnimations pairs n's animation list with its animations by name, the first
// of a name with the first running, and works their keyframes out again.
func (t *Transitions) matchAnimations(n *dom.Node, a *nodeAnims, hidden bool) {
	a.restyled, a.hidden = false, hidden
	var list []style.Animation
	if !hidden {
		list = dom.Data(n).Style.Animations
	}
	var next []*animation
	old := a.list
	for _, spec := range list {
		frames := t.keyframes(spec.Name)
		if frames == nil {
			continue
		}
		i := slices.IndexFunc(old, func(an *animation) bool { return an != nil && an.Name == spec.Name })
		an := &animation{}
		if i >= 0 {
			an, old[i] = old[i], nil
		}
		an.Animation, an.frames = spec, frames
		next = append(next, an)
	}
	t.cancel(n, old)
	a.list = next
	for _, an := range next {
		t.resolve(n, an)
	}
}

// cancel tells of the animations of list that were running.
func (t *Transitions) cancel(n *dom.Node, list []*animation) {
	for _, an := range list {
		if an != nil && (an.phase == before || an.phase == active) {
			t.events = append(t.events, AnimationEvent{Node: n, Name: an.Name, Kind: AnimationCancel,
				Elapsed: max(an.elapsed-an.Delay, 0)})
		}
	}
}

// keyframes is the @keyframes rule the name runs: the last of the
// sheets' whose @media rules hold.
func (t *Transitions) keyframes(name string) *Keyframes {
	if t.frames == nil {
		t.frames = map[string]*Keyframes{}
		for _, s := range t.given {
			for i := range s.Keyframes {
				k := &s.Keyframes[i]
				if !slices.ContainsFunc(k.media, func(m int) bool { return !s.media[m].matches(t.media) }) {
					t.frames[k.Name] = k
				}
			}
		}
	}
	return t.frames[name]
}

// resolve works out an's tracks on n: each keyframe's declarations
// computed over n's own style, as Chrome does, so var(), em and
// currentColor in a keyframe are n's, and missing 0% and 100% keyframes
// take n's own value.
func (t *Transitions) resolve(n *dom.Node, an *animation) {
	base := t.probe(n, nil)
	frames := slices.Clone(an.frames.Frames)
	slices.SortStableFunc(frames, func(a, b Keyframe) int {
		switch {
		case a.At < b.At:
			return -1
		case a.At > b.At:
			return 1
		}
		return 0
	})
	defaultEasing := easing(an.Easing)
	an.tracks = an.tracks[:0]
	for _, f := range frames {
		var decls []Declaration
		fe := defaultEasing
		for _, d := range f.Decls {
			if d.Name == propAnimationTiming {
				fe = easing(join(d.Value))
				continue
			}
			decls = append(decls, d)
		}
		if len(decls) == 0 {
			continue
		}
		s := t.probe(n, decls)
		for _, d := range decls {
			for _, name := range longhandsOf(d.Name) {
				an.add(name, key{at: f.At, easing: fe, s: s})
			}
		}
	}
	for i := range an.tracks {
		tr := &an.tracks[i]
		if tr.keys[0].at > 0 {
			tr.keys = slices.Insert(tr.keys, 0, key{easing: defaultEasing, s: base})
		}
		if tr.keys[len(tr.keys)-1].at < 1 {
			tr.keys = append(tr.keys, key{at: 1, easing: defaultEasing, s: base})
		}
	}
}

func longhandsOf(name string) []string {
	if long, ok := shorthands[name]; ok {
		return long
	}
	return []string{name}
}

// add puts k on the track of the longhand name, over a key of the same
// offset, as a later keyframe of it wins.
func (an *animation) add(name string, k key) {
	i := slices.IndexFunc(an.tracks, func(tr track) bool { return tr.name == name })
	if i < 0 {
		tr := track{name: name, kind: discreteTrack, index: -1}
		if p := slices.IndexFunc(animatable, func(p animProp) bool { return p.name == name }); p >= 0 {
			tr.kind, tr.index = paintTrack, p
		} else if l := slices.IndexFunc(lengthProps, func(l lengthProp) bool { return l.name == name }); l >= 0 {
			tr.kind, tr.index = lengthTrack, l
		}
		an.tracks = append(an.tracks, tr)
		i = len(an.tracks) - 1
	}
	tr := &an.tracks[i]
	if last := len(tr.keys) - 1; last >= 0 && tr.keys[last].at == k.at {
		tr.keys[last] = k
		return
	}
	tr.keys = append(tr.keys, k)
}

// probe is n's style with decls over its cascade, under its !important
// declarations, as an animation's are; its own style with none.
func (t *Transitions) probe(n *dom.Node, decls []Declaration) *style.Computed {
	t.km.off, t.kpm.off, t.km.ranks, t.kpm.ranks = t.off, t.off, t.ranks, t.ranks
	r := restyler{sheets: t.all, m: &t.km, pm: &t.kpm, computed: &t.probed, overlay: t.overlay,
		viewport: t.viewport, browser: t.browser, env: &t.env, keyframe: decls}
	if p := n.Pseudo(); p != dom.NoPseudo {
		origin := n.Parent()
		if s := r.generated(origin, dom.Data(origin).Style, t.rem(), p.String()); s != nil {
			return s
		}
		return dom.Data(n).Style
	}
	if n == t.root {
		return r.compute(n, nil, 0)
	}
	return r.compute(n, dom.Data(n.Parent()).Style, t.rem())
}

func (t *Transitions) rem() float32 { return dom.Data(t.root).Style.Font.Size }

// update works out the values of n's animations now, and tells of where
// they got to. It reports whether what they set on Style changed, and
// flags n DirtyStyle then.
func (t *Transitions) update(n *dom.Node, a *nodeAnims) bool {
	a.paints = a.paints[:0]
	over := make([]override, 0, len(a.over))
	for _, an := range a.list {
		p, ok := an.progress()
		t.events = an.advance(n, t.events)
		if !ok {
			continue
		}
		for i := range an.tracks {
			tr := &an.tracks[i]
			from, to, e := tr.at(p)
			switch tr.kind {
			case paintTrack:
				a.paints = setPaint(a.paints, paintValue{index: tr.index, from: from.s, to: to.s, e: e})
			case lengthTrack:
				get := lengthProps[tr.index].get
				if f, g := get(from.s), get(to.s); f.Kind == style.Fixed && g.Kind == style.Fixed {
					over = setOverride(over, override{name: tr.name, src: from.s, to: to.s, t: e, length: tr.index})
					continue
				}
				over = setOverride(over, discrete(tr.name, from.s, to.s, e))
			case discreteTrack:
				over = setOverride(over, discrete(tr.name, from.s, to.s, e))
			}
		}
	}
	a.paint()
	if slices.Equal(over, a.over) {
		return false
	}
	a.over = over
	restyle(n)
	return true
}

// paint works out the paints' values, colour first: a colour a keyframe
// gives as currentColor, its own or the element's, is the animated
// colour, as in Chrome.
func (a *nodeAnims) paint() {
	var current *color.Color
	if i := slices.IndexFunc(a.paints, func(p paintValue) bool { return p.index == 0 }); i >= 0 {
		p := &a.paints[i]
		p.v = animatable[0].get(p.from).lerp(animatable[0].get(p.to), p.e)
		current = &p.v.c
	}
	for i := range a.paints {
		p := &a.paints[i]
		if p.index == 0 {
			continue
		}
		prop := animatable[p.index]
		f, g := prop.get(p.from), prop.get(p.to)
		if flag := currentFlags[prop.name]; current != nil && flag != 0 {
			if p.from.CurrentColor&flag != 0 {
				f.c = *current
			}
			if p.to.CurrentColor&flag != 0 {
				g.c = *current
			}
		}
		p.v = f.lerp(g, p.e)
	}
}

//nolint:gochecknoglobals,goconst // the colours that can be currentColor's
var currentFlags = map[string]style.CurrentColorFields{
	"border-top-color": style.BorderTopCurrent, "border-right-color": style.BorderRightCurrent,
	"border-bottom-color": style.BorderBottomCurrent, "border-left-color": style.BorderLeftCurrent,
	"outline-color": style.OutlineCurrent,
}

func setPaint(list []paintValue, p paintValue) []paintValue {
	if i := slices.IndexFunc(list, func(q paintValue) bool { return q.index == p.index }); i >= 0 {
		list[i] = p
		return list
	}
	return append(list, p)
}

func setOverride(list []override, o override) []override {
	if i := slices.IndexFunc(list, func(q override) bool { return q.name == o.name }); i >= 0 {
		list[i] = o
		return list
	}
	return append(list, o)
}

// discrete is a value that can't interpolate: from's before half the
// interval, to's from there; visibility is visible all along when one end
// is.
func discrete(name string, from, to *style.Computed, e float32) override {
	src := from
	if e >= 0.5 {
		src = to
	}
	if name == propVisibility && from.Visible != to.Visible && e > 0 && e < 1 {
		src = from
		if to.Visible {
			src = to
		}
	}
	return override{name: name, src: src, length: -1}
}

// at is the keyframes of the interval p falls in and how far along it,
// eased.
func (tr *track) at(p float32) (key, key, float32) {
	i := 0
	for i+2 < len(tr.keys) && tr.keys[i+1].at <= p {
		i++
	}
	from, to := tr.keys[i], tr.keys[i+1]
	if to.at <= from.at {
		return from, to, 1
	}
	return from, to, from.easing((p - from.at) / (to.at - from.at))
}

// activeDuration is how long an runs, all its iterations.
func (an *animation) activeDuration() float64 {
	if an.Duration <= 0 || an.Iterations <= 0 {
		return 0
	}
	return float64(an.Duration) * float64(an.Iterations)
}

// phaseNow is an's phase at its time now, from its delay.
func (an *animation) phaseNow() phase {
	local := float64(an.elapsed - an.Delay)
	switch {
	case local < 0:
		return before
	case local >= an.activeDuration():
		return after
	}
	return active
}

// progress is an's iteration progress now, directed, as Web Animations
// works it out; false where it has no effect: in its delay or after its
// end without a fill.
func (an *animation) progress() (float32, bool) {
	local := float64(an.elapsed - an.Delay)
	var overall float64
	switch an.phaseNow() {
	case before:
		if an.Fill != style.FillBackwards && an.Fill != style.FillBoth {
			return 0, false
		}
	case after:
		if an.Fill != style.FillForwards && an.Fill != style.FillBoth {
			return 0, false
		}
		overall = float64(an.Iterations)
	case active:
		overall = local / float64(an.Duration)
	case idle:
	}
	iteration, simple := math.Floor(overall), overall-math.Floor(overall)
	if simple == 0 && overall != 0 && an.phaseNow() == after {
		simple, iteration = 1, iteration-1
	}
	odd := math.Mod(iteration, 2) == 1
	switch an.Direction {
	case style.AnimateReverse:
		simple = 1 - simple
	case style.AnimateAlternate:
		if odd {
			simple = 1 - simple
		}
	case style.AnimateAlternateReverse:
		if !odd {
			simple = 1 - simple
		}
	case style.AnimateNormal:
	}
	return float32(simple), true
}

// advance appends the events of an's move to where it is now.
func (an *animation) advance(n *dom.Node, events []AnimationEvent) []AnimationEvent {
	now := an.phaseNow()
	ev := func(kind AnimationEventKind, elapsed float64) {
		events = append(events, AnimationEvent{Node: n, Name: an.Name, Kind: kind, Elapsed: float32(elapsed)})
	}
	iteration := 0.0
	if now == active {
		iteration = math.Floor(float64(an.elapsed-an.Delay) / float64(an.Duration))
	}
	switch {
	case now == an.phase:
		if now == active && iteration != an.iteration {
			ev(AnimationIteration, iteration*float64(an.Duration))
		}
	case now == active && an.phase != after:
		ev(AnimationStart, max(float64(-an.Delay), 0))
	case now == after && an.phase != after:
		if an.phase != active {
			ev(AnimationStart, min(max(float64(-an.Delay), 0), an.activeDuration()))
		}
		ev(AnimationEnd, an.activeDuration())
	}
	an.phase, an.iteration = now, iteration
	return events
}

// running reports whether an still moves: before its end and not
// paused.
func (an *animation) running() bool {
	return !an.Paused && an.phaseNow() != after
}

// restyle flags n DirtyStyle, and its ancestors DirtyDescendant; a
// generated node's element, which styles it.
func restyle(n *dom.Node) {
	if n.Pseudo() != dom.NoPseudo {
		n = n.Parent()
	}
	dom.Data(n).Dirty |= dom.DirtyStyle
	for p := n.Parent(); p != nil; p = p.Parent() {
		dom.Data(p).Dirty |= dom.DirtyDescendant
	}
}

// hiddenNode reports whether n or an element above it is display: none.
func hiddenNode(n *dom.Node) bool {
	for ; n != nil; n = n.Parent() {
		if s := dom.Data(n).Style; s != nil && s.Display == style.DisplayNone {
			return true
		}
	}
	return false
}

// live reports whether n is in the tree under the root: a generated node
// while its element keeps it.
func (t *Transitions) live(n *dom.Node) bool {
	if p := n.Pseudo(); p != dom.NoPseudo {
		d := dom.Data(n.Parent())
		if d.Before != n && d.After != n {
			return false
		}
	}
	return t.attached(n)
}

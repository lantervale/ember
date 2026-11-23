package css

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"sync"
	"unique"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

//nolint:gochecknoglobals // the user agent sheet and initial style, built once and shared
var (
	agentOnce    sync.Once
	agentSheet   *Sheet
	browserOnce  sync.Once
	browserSheet *Sheet
	initial      = initialStyle()
)

// initialStyle is style.Initial with the initial values it leaves out:
// border widths are medium, used only under a border-style (finish),
// outline-width is medium whatever outline-style says, as in Chrome,
// corners are 0, and the transform and object origins are percentages.
func initialStyle() style.Computed {
	c := style.Initial()
	c.Border = style.Sides[float32]{Top: 3, Right: 3, Bottom: 3, Left: 3}
	c.OutlineWidth = 3
	c.ColumnRuleWidth = 3
	zero := style.Size[style.Length]{Width: style.Px(0), Height: style.Px(0)}
	c.Radius = style.Corners[style.Size[style.Length]]{TopLeft: zero, TopRight: zero, BottomRight: zero,
		BottomLeft: zero}
	half := style.Length{Kind: style.Fixed, Percent: 50, IsPercent: true}
	c.TransformOrigin = style.Size[style.Length]{Width: half, Height: half}
	c.AccentAuto, c.CaretAuto = true, true
	c.ObjectPosition = style.BackgroundPosition{X: half, Y: half}
	return c
}

func userAgent() *Sheet {
	agentOnce.Do(func() {
		s, warns := Parse("user-agent.css", Default)
		if len(warns) > 0 {
			panic(fmt.Sprint("css: the user-agent sheet: ", warns))
		}
		agentSheet = s
	})
	return agentSheet
}

// browserAgent is the user-agent sheet of a Document whose sheets are
// read as a browser's: Default, then BrowserDefault.
func browserAgent() *Sheet {
	browserOnce.Do(func() {
		s, warns := Parse("user-agent.css", Default+BrowserDefault)
		if len(warns) > 0 {
			panic(fmt.Sprint("css: the browser's user-agent sheet: ", warns))
		}
		browserSheet = s
	})
	return browserSheet
}

// Compute restyles the elements under root that changed since the last
// call: those flagged DirtyStyle, the subtrees under DirtyTree, and the
// descendants of any element whose style changed. Each gets
// dom.Data(n).Style, a text node its parent's, and loses both flags; an
// element whose style changed is flagged DirtyLayout. sheets are the
// author's, after Default. The warnings are the style attributes'. Their
// @media rules are answered for a viewport of no size: a Document's are
// answered for its own, through Transitions.
func Compute(root *dom.Node, sheets []*Sheet) []Warning {
	all := append([]*Sheet{userAgent()}, sheets...)
	env := &measures{}
	m := &matcher{off: mediaOff(all, Media{}), ranks: layerRanks(all), env: env}
	pm := &matcher{off: m.off, ranks: m.ranks, env: env}
	r := restyler{sheets: all, m: m, pm: pm, computed: new(int), env: env}
	r.run(root, false)
	return r.warns
}

type restyler struct {
	sheets []*Sheet
	// env is what lengths are measured against besides font sizes.
	env *measures
	// m matches and cascades elements, pm their pseudo-elements.
	m, pm *matcher
	// treeStates are the states some selector reads above its subject,
	// and besideStates those read before a sibling combinator: only a
	// change of those restyles an element's descendants, or the elements
	// after it and theirs, and of them the ones in reach.
	treeStates, besideStates dom.State
	// reach are the subjects of the selectors that read a state that
	// moved on an element above the one visited, or before one above.
	reach []*Compound
	// siblings is set when some selector has a sibling combinator: a
	// change of what an element reads there (movesSiblings) or of its
	// parent's children restyles the elements after it and their subtrees.
	siblings bool
	// anchors is set when some selector has a :has(), and hasBeside when
	// one reads after the element it is on: a change below or after an
	// element then flags it before the tree is visited (anchor).
	// hasStates are the states the :has() read, and empty is set when
	// some selector reads :empty, whose change on an element restyles it.
	anchors, hasBeside, empty bool
	hasStates                 dom.State
	// computed counts the elements whose style was worked out.
	computed *int
	// la and lb are scratch styles for comparing what layout or
	// inheritance reads of two.
	la, lb style.Computed
	warns  []Warning
	// changed, when set, is told of every element whose style changed
	// from an earlier one, and sets its Shown.
	changed func(n *dom.Node, old, next *style.Computed)
	// over, when set, is what an element's animations set on its style;
	// keyframe, when set, are a keyframe's declarations to compute over
	// the cascade, under its !important declarations.
	over     func(n *dom.Node) []override
	keyframe []Declaration
	// transit, when set, is told of the style worked out with its
	// animations of every element that declares transitions or runs some
	// (transits), and returns what its transitions set on it.
	transit  func(n *dom.Node, next *style.Computed) []override
	transits func(n *dom.Node) bool
	// afterChange, when set, is the style an element's transitions on Style run
	// to, nil while none runs: what its children's transitions read.
	afterChange func(n *dom.Node) *style.Computed
	// overlay has scrolling boxes keep no gutter for their bars
	// (Transitions.OverlayScrollbars).
	overlay bool
	// viewport has the root's overflow, or body's, go to the viewport
	// (Transitions.ScrollingViewport).
	viewport bool
	// browser reads style attributes as a browser does
	// (Transitions.BrowserSheets).
	browser bool
	// before, after, marker, firstLine and firstLetter are set when some
	// sheet has a rule for the pseudo-element: only then is one looked for.
	before, after, marker, firstLine, firstLetter bool
	// styles are the sheets' counter styles, worked out when first asked
	// for (counterStyles).
	styles *counterStyles
	// counts is set once an element counts: a numbered list item, a
	// counter property, or generated content of counters or quotes; and
	// recount when what the counters read may have changed.
	counts, recount bool
}

// run restyles the tree under root, all of it when stale, then counts
// its counters again where they may have moved.
// counterStyles are the counter styles of r's sheets.
func (r *restyler) counterStyles() *counterStyles {
	if r.styles == nil {
		r.styles = authorStyles(r.sheets, Media{})
	}
	return r.styles
}

func (r *restyler) run(root *dom.Node, stale bool) {
	for _, s := range r.sheets {
		x := s.indexed()
		r.before = r.before || x.pseudos["before"] != nil
		r.after = r.after || x.pseudos["after"] != nil
		r.marker = r.marker || x.pseudos["marker"] != nil
		r.firstLine = r.firstLine || x.pseudos["first-line"] != nil
		r.firstLetter = r.firstLetter || x.pseudos["first-letter"] != nil
		r.siblings = r.siblings || x.siblings
		r.treeStates |= x.aboveStates
		r.besideStates |= x.besideStates
		r.anchors = r.anchors || len(x.has) > 0
		r.hasBeside = r.hasBeside || slices.ContainsFunc(x.has, func(h hasRead) bool { return h.beside })
		r.hasStates |= x.hasStates
		r.empty = r.empty || x.empty
	}
	if r.env == nil {
		r.env = &measures{}
	}
	r.env.root = root
	if stale {
		r.env.used, r.env.prefersRead, r.env.titleBarRead, r.env.inactiveRead = 0, false, false, false
	} else if r.anchors && dom.Data(root).Style != nil {
		r.anchor(root)
	}
	r.visit(root, nil, 0, stale, stale)
	r.recount = r.recount || stale
	r.count(root)
	r.firsts(root)
}

// countsOn reports a change of an element, as dom.Pipeline.Changed names
// it, that moves the counters: a child added or removed, or one of the
// attributes a list counts from.
func countsOn(key string) bool {
	return key == dom.Children || key == "[start" || key == "[reversed" || key == "[value"
}

// visit restyles n where it needs it, and its subtree, and reports
// whether n or a node below needs layout. Only then does n keep
// DirtyDescendant: a change paint alone shows leaves layout clean. force
// restyles n; deep restyles its whole subtree, as a change of an id, a
// class or an attribute some selector reads above its subject asks (one
// no selector reads there restyles n alone). Children added or removed
// restyle the children only. A state that moved restyles, of the elements
// under n, those the selectors reading it on n can match (reach).
func (r *restyler) visit(n *dom.Node, parent *style.Computed, rem float32, force, deep bool) bool {
	data := dom.Data(n)
	if n.IsText() {
		data.Style = parent
		data.Shown = dom.Data(n.Parent()).Shown
		data.Dirty &^= dom.DirtyStyle | dom.DirtyTree
		return data.Dirty&dom.DirtyLayout != 0
	}
	dirty := data.Dirty
	moved := movedStates(n)
	inherits := false
	if force || deep || dirty&dom.DirtyStyle != 0 || data.Style == nil || r.reached(n) ||
		r.empty && moved&dom.Empty != 0 {
		inherits = r.restyle(n, data, parent, rem)
	}
	data.StyledState = n.State()
	data.Dirty &^= dom.DirtyStyle | dom.DirtyTree | dom.DirtyState
	if parent == nil {
		rem = data.Style.Font.Size
	}
	tree := dirty&dom.DirtyTree != 0
	if tree && slices.ContainsFunc(data.Changed, countsOn) {
		r.recount = true
	}
	kidsChanged := tree && slices.Contains(data.Changed, dom.Children)
	// A container Requery found resized restyles what it holds.
	requeried := r.env.requery[n]
	if requeried {
		delete(r.env.requery, n)
	}
	kidsDeep := deep || tree && r.readAbove(data.Changed) || kidsChanged && r.siblings || requeried
	// A details' content shows by its state (closedSlot), whatever the sheets read.
	slotted := moved&dom.Open != 0 && n.Tag() == dom.Details
	kidsForce := inherits || kidsDeep || kidsChanged || slotted
	data.Changed = data.Changed[:0]
	reach := len(r.reach)
	if moved&r.treeStates != 0 {
		r.reaches(n, moved, false)
	}
	if !kidsForce && dirty&dom.DirtyDescendant == 0 && !force && len(r.reach) == 0 {
		return data.Dirty&dom.DirtyLayout != 0
	}
	below := r.children(n, data.Style, rem, kidsForce, kidsDeep)
	r.reach = r.reach[:reach]
	if below {
		data.Dirty |= dom.DirtyDescendant
	} else {
		data.Dirty &^= dom.DirtyDescendant
	}
	return below || data.Dirty&dom.DirtyLayout != 0
}

// children visits n's children, whose parent's style is parent, and
// reports whether one or a node below needs layout. What a sibling
// combinator reads on one restyles those after it: all of them and
// their subtrees for an id, a class or an attribute, those in reach for
// a state.
func (r *restyler) children(n *dom.Node, parent *style.Computed, rem float32, force, deep bool) bool {
	below := false
	for _, c := range n.Children() {
		after := r.siblings && r.movesSiblings(c)
		beside := movedStates(c) & r.besideStates
		if r.visit(c, parent, rem, force, deep) {
			below = true
		}
		if after {
			force, deep = true, true
		}
		if beside != 0 {
			r.reaches(c, beside, true)
		}
	}
	return below
}

// restyle computes n's style again, and reports whether what its
// children inherit changed.
func (r *restyler) restyle(n *dom.Node, data *dom.Pipeline, parent *style.Computed, rem float32) bool {
	s := r.compute(n, parent, rem)
	*r.computed++
	if s.ContainerType != style.ContainerNormal {
		r.env.contain(n)
	}
	old := data.Style
	if old != nil && old.Equal(s) {
		r.generate(n, data, rem)
		return false
	}
	data.Style = s
	if r.changed != nil {
		r.changed(n, old, s)
	} else {
		data.Shown = s
	}
	inherits := old == nil || r.inheritedChanged(old, s) || old.OverflowPropagated != s.OverflowPropagated
	if old == nil || r.layoutChanged(old, s) {
		data.Dirty |= dom.DirtyLayout
		r.recount = true
	}
	r.generate(n, data, rem)
	return inherits
}

// layoutChanged reports whether a and b differ in what layout reads: not
// in colours, backgrounds, borders' looks, shadows, opacity or
// transforms, which paint reads through Shown.
func (r *restyler) layoutChanged(a, b *style.Computed) bool {
	r.la, r.lb = *a, *b
	paintOnly(&r.la)
	paintOnly(&r.lb)
	return !r.la.Equal(&r.lb)
}

func paintOnly(c *style.Computed) {
	c.Look = style.Look{}
	var zero color.Color
	c.Color, c.PlaceholderColor, c.SelectionColor, c.AccentColor, c.CaretColor = zero, zero, zero, zero, zero
	c.HasPlaceholder, c.HasSelection, c.AccentAuto, c.CaretAuto, c.ColorInherited = false, false, false, false, false
	c.SelectionText, c.HasSelectionText, c.UserSelect = zero, false, 0
	c.SelectionShadows, c.HasSelectionShadows = nil, false
	c.SelectionDecoration, c.HasSelectionDecoration = style.TextDecoration{}, false
	c.TextShadows, c.CurrentColor, c.Vars, c.Backdrop = nil, 0, nil, color.Color{}
	c.TextDecoration, c.Decorations = style.TextDecoration{}, nil
	c.ColorScheme, c.Dark = 0, false
}

// inheritedChanged reports whether a and b differ in what an element's
// children take from it: its inherited properties and custom properties,
// the longhands some declaration inherits explicitly, or anything at all
// where a shorthand does.
func (r *restyler) inheritedChanged(a, b *style.Computed) bool {
	if explicitInherit(r.sheets) {
		return true
	}
	r.la, r.lb = style.Computed{Vars: a.Vars}, style.Computed{Vars: b.Vars}
	r.la.Decorations, r.lb.Decorations = a.Decorations, b.Decorations
	for _, name := range inheritedNames {
		p := props[name]
		p.inherit(&r.la, a)
		p.inherit(&r.lb, b)
	}
	for _, s := range r.sheets {
		for _, name := range s.indexed().inherited {
			p := props[name]
			p.inherit(&r.la, a)
			p.inherit(&r.lb, b)
		}
	}
	return !r.la.Equal(&r.lb)
}

// inheritedNames are the inherited longhands.
//
//nolint:gochecknoglobals // derived from the properties
var inheritedNames = func() []string {
	var out []string
	for _, name := range order {
		if properties[name].inherited {
			out = append(out, name)
		}
	}
	return out
}()

// explicitInherit reports whether some declaration in sheets (the user
// agent's first) sets a shorthand or a custom property to inherit.
func explicitInherit(sheets []*Sheet) bool {
	return slices.ContainsFunc(sheets, func(s *Sheet) bool { return s.indexed().inherits })
}

// ctx is one element's computation.
type ctx struct {
	c, parent *style.Computed
	em, rem   float32
	// env is what the units of a font and of the viewport are measured
	// against, and u those of the length being read.
	env *measures
	u   Units
	// sized and lined are set once the element's font-size and its
	// line-height are known: until then the units of the font and lh read
	// the parent's. root is set for the root, whose root units are its own.
	sized, lined, root bool
	vars               map[string][]Component
	resolving          map[string]bool
	raw                map[string]*Declaration
	trans              [5][]Component
	anims              [8][]Component
	composite          []Component
	// over is what the element's animations set on its style; transit is
	// the restyler's, for moving, which runs transitions when transits is
	// set.
	over     []override
	transit  func(n *dom.Node, next *style.Computed) []override
	moving   *dom.Node
	transits bool
	// afterParent is the style the parent's transitions run to, while
	// some run on what its children inherit: their own transitions start
	// from the style worked out under it.
	afterParent *style.Computed
	// base is where the sheet the declaration being applied came from is.
	base     *origin
	expanded map[*Declaration]map[string][]Component
	// overlay is the restyler's: scrollbars keep no gutter.
	overlay bool
	// toViewport is set for the root and for its body where the viewport
	// scrolls: the two whose overflow it may take.
	toViewport bool
	// node is the element, which attr() reads, and self is set for its
	// pseudo-element, whose container may be the element itself.
	node *dom.Node
	self bool
	// rtl is the cascade's: the element's inline sides are mirrored.
	rtl bool
}

// leading are the longhands the others read, in the order they are
// applied before them: direction, the side inline-start and inline-end
// stand on, color-scheme, which resolves the colours, the
// font's family, style and weight, which its own units measure, then
// font-size (em), line-height (lh) and color (currentcolor).
//
//nolint:gochecknoglobals,goconst // a fixed list
var leading = []string{
	propDirection, propColorScheme, "font-family", "font-style", "font-weight", "font-stretch", propFontSize,
	"line-height", "color",
}

// colored are the longhands whose values take colours, where a system
// colour's name and light-dark() are resolved in the element's scheme.
//
//nolint:gochecknoglobals,goconst // a fixed set
var colored = map[string]bool{
	"color": true, "background-color": true, "background-image": true, "border-top-color": true,
	"border-right-color": true, "border-bottom-color": true, "border-left-color": true, "border-image-source": true,
	"outline-color": true, "text-decoration-color": true, "column-rule-color": true, "accent-color": true,
	"caret-color": true, "box-shadow": true, "text-shadow": true, "mask-image": true, "filter": true,
	"backdrop-filter": true, "-webkit-text-fill-color": true,
}

//nolint:gochecknoglobals // a fixed value every element shares
var canvasText = []Component{word("canvastext")}

// order puts the leading longhands first, then the rest.
//
//nolint:gochecknoglobals // derived from the properties
var order, orderIndex = func() ([]string, map[string]int) {
	names := make([]string, 0, len(properties))
	for n := range properties {
		if !slices.Contains(leading, n) {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	names = append(slices.Clone(leading), names...)
	index := make(map[string]int, len(names))
	for i, n := range names {
		index[n] = i
	}
	return names, index
}()

// longhand is one of order's longhands with what every element's
// computation reads of it, looked up once: how it is set and inherited,
// whether it is inherited, and whether its initial value is currentcolor.
type longhand struct {
	name                        string
	prop                        prop
	inherited, current, colored bool
}

// longhands are order's, in its order.
//
//nolint:gochecknoglobals // derived from the properties
var longhands = func() []longhand {
	out := make([]longhand, len(order))
	for i, name := range order {
		p := properties[name]
		out[i] = longhand{name: name, prop: props[name], inherited: p.inherited, current: p.initial == kwCurrentColor,
			colored: colored[name]}
	}
	return out
}()

//nolint:gochecknoglobals // a fixed value every element shares
var currentColor = []Component{word(kwCurrentColor)}

func (r *restyler) compute(n *dom.Node, parent *style.Computed, rem float32) *style.Computed {
	var inline []Declaration
	if src := n.Style(); src != "" {
		var warns []Warning
		p := parser{toks: Tokenize(src), browser: r.browser}
		inline, warns = p.inline()
		for _, w := range warns {
			w.File = describe(n) + " style"
			r.warns = append(r.warns, w)
		}
	}
	r.m.hints = hints(n)
	root := parent == nil
	if root {
		parent = &initial
		rem = initial.Font.Size
	}
	casc := r.m.cascade(r.m.matched(n, r.sheets, inline), parent.Dir)
	r.overKeyframe(casc)
	c := initial
	x := &ctx{c: &c, parent: parent, em: parent.Font.Size, rem: rem, env: r.env, root: root, overlay: r.overlay,
		toViewport: r.viewport && (root || rootBody(n)), node: n}
	if r.over != nil {
		x.over = r.over(n)
	}
	r.transitOf(x, n)
	x.cascaded(casc, root)
	if closedSlot(n) {
		c.Display = style.DisplayNone
	}
	c.EllipsisStart = slices.Contains(n.Classes(), "truncate-start")
	if _, shows := n.Attr("placeholder"); shows && r.may(n, "placeholder") {
		if ph, ok := x.pseudo(r.pm, n, r.sheets, "placeholder"); ok {
			c.PlaceholderColor, c.HasPlaceholder = fadeColor(ph.Color, ph.Opacity), true
		}
	}
	// A ::selection inherits from the parent's ::selection, property by
	// property, as Chrome's highlight pseudo-elements do.
	c.SelectionColor, c.HasSelection = parent.SelectionColor, parent.HasSelection
	c.SelectionText, c.HasSelectionText = parent.SelectionText, parent.HasSelectionText
	c.SelectionShadows, c.HasSelectionShadows = parent.SelectionShadows, parent.HasSelectionShadows
	c.SelectionDecoration, c.HasSelectionDecoration = parent.SelectionDecoration, parent.HasSelectionDecoration
	if r.may(n, "selection") {
		r.selection(x, n)
	}
	if n.Tag() == dom.Dialog && r.may(n, "backdrop") {
		if bd, ok := x.pseudo(r.pm, n, r.sheets, "backdrop"); ok {
			c.Backdrop = fadeColor(bd.Background, bd.Opacity)
		}
	}
	if c.ListItem && (c.ListStyle == style.ListCounter || c.ListStyle == style.ListString) ||
		len(c.CounterReset)+len(c.CounterIncrement)+len(c.CounterSet) > 0 {
		r.counts = true
	}
	return &c
}

// transitOf has x tell r's transit of n's style.
func (r *restyler) transitOf(x *ctx, n *dom.Node) {
	if r.transit != nil {
		x.transit, x.moving, x.transits = r.transit, n, r.transits(n)
		if p := n.Parent(); p != nil && r.afterChange != nil {
			x.afterParent = r.afterChange(p)
		}
	}
}

// overKeyframe puts the keyframe's declarations in casc where no
// !important one won, as animations stand in the cascade.
func (r *restyler) overKeyframe(casc cascaded) {
	for i := range r.keyframe {
		d := &r.keyframe[i]
		for _, name := range longhandsOf(d.Name) {
			if w := casc.longhands[orderIndex[name]]; w == nil || !w.Important {
				casc.longhands[orderIndex[name]] = d
			}
		}
	}
}

// selection gives x's style what n's own ::selection declares.
func (r *restyler) selection(x *ctx, n *dom.Node) {
	sel, ok := x.pseudo(r.pm, n, r.sheets, "selection")
	if !ok {
		return
	}
	if r.pm.declared("background-color") {
		x.c.SelectionColor, x.c.HasSelection = sel.Background, true
	}
	if r.pm.declared("color") {
		x.c.SelectionText, x.c.HasSelectionText = sel.Color, true
	}
	if r.pm.declared("text-shadow") {
		x.c.SelectionShadows, x.c.HasSelectionShadows = sel.TextShadows, true
	}
	if slices.ContainsFunc(shorthands[shortDecoration], r.pm.declared) {
		x.c.SelectionDecoration, x.c.HasSelectionDecoration = sel.TextDecoration, true
	}
}

// cascaded applies every longhand's winner, then what depends on several.
func (x *ctx) cascaded(casc cascaded, root bool) {
	x.rtl = casc.rtl
	x.customs(casc.custom)
	if !x.transitioning(casc) {
		x.longhands(casc, root, x.over)
		x.finish(root)
		return
	}
	start, c := *x, *x.c
	x.longhands(casc, root, x.over)
	x.finish(root)
	next := x.c
	if x.afterParent != nil {
		y, yc := start, c
		y.c, y.parent, y.em = &yc, x.afterParent, x.afterParent.Font.Size
		y.customs(casc.custom)
		y.longhands(casc, root, y.over)
		y.finish(root)
		next = y.c
	}
	if over := x.transit(x.moving, next); len(over) > 0 {
		*x, *x.c = start, c
		x.longhands(casc, root, append(slices.Clip(x.over), over...))
		x.finish(root)
	}
}

// transitioning reports whether x's element declares transitions or
// runs some.
func (x *ctx) transitioning(casc cascaded) bool {
	return x.transit != nil && (x.transits || slices.ContainsFunc(transitionLonghands, func(i int) bool {
		return casc.longhands[i] != nil
	}))
}

//nolint:gochecknoglobals // derived from the properties
var transitionLonghands = func() []int {
	var out []int
	for _, name := range shorthands[shortTransition] {
		out = append(out, orderIndex[name])
	}
	return out
}()

// longhands applies every longhand's winner, each with what over sets on
// it after it, so the units of the font and currentColor read the
// animated font and colour.
func (x *ctx) longhands(casc cascaded, root bool, over []override) {
	for i := range longhands {
		l := &longhands[i]
		x.apply(l, casc.longhands[i])
		for _, o := range over {
			if o.index == i {
				o.apply(x.c)
			}
		}
		switch l.name {
		case propFontSize:
			x.em, x.sized = x.c.Font.Size, true
			if root {
				x.rem = x.c.Font.Size
			}
		case "line-height":
			x.lined = true
		case propColorScheme:
			x.c.Dark = x.c.ColorScheme.Dark(x.env.dark)
			if x.c.ColorScheme&(style.SchemeLight|style.SchemeDark) == style.SchemeLight|style.SchemeDark {
				x.env.prefersRead = true
			}
		}
	}
}

func fadeColor(c color.Color, opacity float32) color.Color {
	c.A *= opacity
	return c
}

// may reports whether some rule of the sheets may style n's pseudo-element
// name: one names it, in a bucket n is in.
func (r *restyler) may(n *dom.Node, name string) bool {
	for _, s := range r.sheets {
		if b := s.indexed().pseudos[name]; b != nil && b.may(n) {
			return true
		}
	}
	return false
}

// pseudo computes the style of the element's pseudo-element name, one
// that draws with the element's box (::placeholder, ::selection,
// ::backdrop, ::marker): the element's own with the properties the
// pseudo-element takes set by its rules, inheriting from the element;
// false when no rule sets any.
func (x *ctx) pseudo(m *matcher, n *dom.Node, sheets []*Sheet, name string) (*style.Computed, bool) {
	casc := m.cascadeFor(m.matchedFor(n, sheets, nil, name), name, x.c.Dir)
	if !slices.ContainsFunc(casc.longhands, func(d *Declaration) bool { return d != nil }) && len(casc.custom) == 0 {
		return nil, false
	}
	ps := *x.c
	ps.Opacity, ps.Content, ps.Background = 1, style.Content{}, color.Color{}
	ps.TextDecoration, ps.CurrentColor = style.TextDecoration{}, ps.CurrentColor|style.DecorationCurrent
	px := &ctx{c: &ps, parent: x.c, em: x.em, rem: x.rem, env: x.env, node: n, rtl: casc.rtl}
	px.customs(casc.custom)
	for i := range longhands {
		l := &longhands[i]
		if d := casc.longhands[i]; d != nil {
			px.apply(l, d)
		}
		switch l.name {
		case propFontSize:
			px.em, px.sized = ps.Font.Size, true
		case "line-height":
			px.lined = true
		}
	}
	ps.FollowColor(ps.Color)
	ps.TextShadows = followColor(ps.TextShadows, ps.Color)
	return &ps, true
}

func describe(n *dom.Node) string {
	s := "<" + n.Tag().String()
	if id := n.ID(); id != "" {
		s += "#" + id
	}
	for _, c := range n.Classes() {
		s += "." + c
	}
	return s + ">"
}

// parsedVar is an inherited custom property's text as components. The
// same few texts come back for every element, so they are parsed once;
// the components are shared and never changed.
func parsedVar(text string) []Component {
	varsMu.Lock()
	defer varsMu.Unlock()
	if v, ok := varsParsed[text]; ok {
		return v
	}
	v := componentsOf(text)
	if len(varsParsed) >= maxParsedVars {
		clear(varsParsed)
	}
	varsParsed[text] = v
	return v
}

// maxParsedVars bounds parsedVar's cache, which a page writing ever new
// values into style attributes would otherwise grow.
const maxParsedVars = 4096

//nolint:gochecknoglobals // the var() parse cache every Document shares
var (
	varsMu     sync.Mutex
	varsParsed = map[string][]Component{}
)

// customs sets the element's custom properties: the parent's, then its
// own, each with its var() references substituted. A cycle or a missing
// reference without fallback leaves the property unset.
func (x *ctx) customs(decls []*Declaration) {
	x.c.Vars = x.parent.Vars
	if len(decls) == 0 {
		return
	}
	x.raw = map[string]*Declaration{}
	for _, d := range decls {
		x.raw[d.Name] = d
	}
	x.vars = map[string][]Component{}
	x.resolving = map[string]bool{}
	vars := make(map[string]string, len(x.parent.Vars)+len(decls))
	for k, v := range x.parent.Vars {
		vars[k] = v
	}
	for name := range x.raw {
		if v, ok := x.lookup(name); ok {
			vars[name] = join(v)
		} else {
			delete(vars, name)
		}
	}
	x.c.Vars = vars
}

func (x *ctx) lookup(name string) ([]Component, bool) {
	if v, ok := x.vars[name]; ok {
		return v, v != nil
	}
	d, own := x.raw[name]
	if !own {
		if text, ok := x.parent.Vars[name]; ok {
			return parsedVar(text), true
		}
		return nil, false
	}
	if x.resolving[name] {
		return nil, false
	}
	x.resolving[name] = true
	v, ok := x.substitute(d.Value)
	x.resolving[name] = false
	if !ok || globalKeyword(v) != "" {
		v, ok = x.customKeyword(name, v, ok)
	}
	x.vars[name] = v
	return v, ok
}

// customKeyword handles inherit, initial and unset on a custom property.
func (x *ctx) customKeyword(name string, v []Component, ok bool) ([]Component, bool) {
	if ok && globalKeyword(v) == "initial" {
		return nil, false
	}
	if text, has := x.parent.Vars[name]; has {
		return parsedVar(text), true
	}
	return nil, false
}

// substitute replaces every var() in v.
func (x *ctx) substitute(v []Component) ([]Component, bool) {
	if !hasVar(v) {
		return v, true
	}
	out := make([]Component, 0, len(v))
	for _, c := range v {
		switch {
		case c.Kind == Function && strings.EqualFold(c.Text, "var"):
			args := trim(c.Args)
			if len(args) == 0 || args[0].Kind != Ident || !strings.HasPrefix(args[0].Text, "--") {
				return nil, false
			}
			if val, ok := x.lookup(args[0].Text); ok {
				out = append(out, val...)
				continue
			}
			rest := trim(args[1:])
			if len(rest) == 0 || rest[0].Kind != Comma {
				return nil, false
			}
			fb, ok := x.substitute(trim(rest[1:]))
			if !ok {
				return nil, false
			}
			out = append(out, fb...)
		case c.Kind == Function && strings.EqualFold(c.Text, "env"):
			val, ok := x.envValue(c)
			if !ok {
				return nil, false
			}
			out = append(out, val...)
		case len(c.Args) > 0:
			args, ok := x.substitute(c.Args)
			if !ok {
				return nil, false
			}
			c.Args = args
			out = append(out, c)
		default:
			out = append(out, c)
		}
	}
	return out, true
}

// envValue is what env() c stands for: its variable, else its fallback,
// false when it has neither.
func (x *ctx) envValue(c Component) ([]Component, bool) {
	args := trim(c.Args)
	if len(args) == 0 || args[0].Kind != Ident {
		return nil, false
	}
	rest := trim(args[1:])
	if len(rest) > 0 && rest[0].Kind != Comma {
		return nil, false
	}
	if val, ok := x.env.variable(args[0].Text); ok {
		return val, true
	}
	if len(rest) == 0 {
		return nil, false
	}
	return x.substitute(trim(rest[1:]))
}

// apply computes one longhand from its winning declaration, or from
// inheritance or its initial value when it has none.
func (x *ctx) apply(l *longhand, d *Declaration) {
	name, p := l.name, l.prop
	if d == nil {
		if l.inherited {
			x.inherit(l)
			x.c.ColorInherited = x.c.ColorInherited || name == "color"
		} else {
			x.reset(l)
		}
		return
	}
	x.base = d.base
	v, ok := x.valueOf(name, d)
	if ok {
		v = items(v)
		ok = len(v) > 0
	}
	kw := ""
	if ok {
		kw = globalKeyword(v)
	}
	if ok && kw == "" && hasVar(d.Value) {
		if chk := checks[name]; chk != nil && chk(v) != nil {
			ok = false
		}
	}
	if !ok {
		kw = kwUnset
	}
	if name == "color" {
		x.c.ColorInherited = kw == kwInherit || kw == kwUnset || kw == "" && isWord(v[0], kwCurrentColor)
	}
	switch kw {
	case kwInherit:
		x.inherit(l)
	case "initial":
		x.reset(l)
	case kwUnset:
		if l.inherited {
			x.inherit(l)
		} else {
			x.reset(l)
		}
	default:
		if l.colored {
			v = inScheme(v, x.c.Dark)
		}
		p.set(x, v)
	}
}

// valueOf is the value d gives the longhand name, its var()s substituted;
// false when they leave none.
func (x *ctx) valueOf(name string, d *Declaration) ([]Component, bool) {
	key := x.keyOf(name, d)
	switch {
	case d.longhands != nil:
		return d.longhands[key], true
	case d.Name != key:
		long, done := x.expanded[d]
		if !done {
			if sub, subOK := x.substitute(d.Value); subOK {
				long, _ = expand(d.Name, items(sub))
			}
			if x.expanded == nil {
				x.expanded = map[*Declaration]map[string][]Component{}
			}
			x.expanded[d] = long
		}
		return long[key], long != nil
	case d.vals != nil:
		return d.vals, true
	default:
		return x.substitute(d.Value)
	}
}

// keyOf is the longhand d gives name the value of: the other side's
// when d is of an inline side and the element's direction rtl.
func (x *ctx) keyOf(name string, d *Declaration) string {
	if x.rtl && d.inline {
		return mirrored(name)
	}
	return name
}

// inherit gives a longhand its parent's value; the root's is its initial
// value.
func (x *ctx) inherit(l *longhand) {
	if x.root {
		x.reset(l)
		return
	}
	l.prop.inherit(x.c, x.parent)
}

// reset gives a longhand its initial value; currentcolor is the
// element's colour, and canvastext is its scheme's.
func (x *ctx) reset(l *longhand) {
	switch {
	case l.current:
		l.prop.set(x, currentColor)
	case l.name == "color":
		l.prop.set(x, inScheme(canvasText, x.c.Dark))
	default:
		l.prop.inherit(x.c, &initial)
	}
}

// rootBody reports whether n is the root's first body element, whose
// overflow the viewport takes when the root's is visible.
func rootBody(n *dom.Node) bool {
	p := n.Parent()
	if n.Tag() != dom.Body || p == nil || p.Parent() != nil {
		return false
	}
	for _, c := range p.Children() {
		if !c.IsText() && c.Tag() == dom.Body {
			return c == n
		}
	}
	return false
}

// propagateOverflow hands the element's overflow to the viewport, as CSS
// does the root's, or body's while the root's is visible: the viewport's
// style is the element's Viewport, where visible is auto and clip hidden,
// and the element's own overflow is visible. A root with a visible
// overflow still holds the style of a viewport that scrolls when its
// content overflows; a body's is the root's then.
func (x *ctx) propagateOverflow(root bool) {
	c := x.c
	own := c.Overflow != style.Size[style.Overflow]{}
	if !root && (!own || x.parent.OverflowPropagated) {
		return
	}
	of := func(o style.Overflow) style.Overflow {
		switch o {
		case style.Visible:
			return style.OverflowAuto
		case style.Clip:
			return style.Hidden
		case style.Hidden, style.Scroll, style.OverflowAuto:
		}
		return o
	}
	v := initial
	v.Display, v.Font = style.Block, c.Font
	v.Overflow = style.Size[style.Overflow]{Width: of(c.Overflow.Width), Height: of(c.Overflow.Height)}
	v.ScrollbarNone = c.ScrollbarNone
	if bars := v.Overflow.Width != style.Hidden || v.Overflow.Height != style.Hidden; bars && !v.ScrollbarNone {
		if v.ScrollbarOverlay = x.overlay; !x.overlay {
			v.ScrollbarWidth = 15
		}
	}
	c.Viewport = &v
	if own {
		c.OverflowPropagated = true
		c.Overflow = style.Size[style.Overflow]{}
		c.ScrollbarWidth, c.ScrollbarOverlay = 0, false
	}
}

// finish applies what depends on several properties.
func (x *ctx) finish(root bool) {
	c := x.c
	none := func(s style.BorderStyle) bool { return s == style.BorderNone }
	if none(c.BorderStyle.Top) {
		c.Border.Top = 0
	}
	if none(c.BorderStyle.Right) {
		c.Border.Right = 0
	}
	if none(c.BorderStyle.Bottom) {
		c.Border.Bottom = 0
	}
	if none(c.BorderStyle.Left) {
		c.Border.Left = 0
	}
	x.snapWidths()
	ox, oy := c.Overflow.Width, c.Overflow.Height
	scrolls := func(o style.Overflow) bool {
		return o == style.Hidden || o == style.Scroll || o == style.OverflowAuto
	}
	fix := func(o style.Overflow) style.Overflow {
		if o == style.Visible {
			return style.OverflowAuto
		}
		return o
	}
	if (c.Overflow.Width == style.Scroll || c.Overflow.Width == style.OverflowAuto ||
		c.Overflow.Height == style.Scroll || c.Overflow.Height == style.OverflowAuto) && !c.ScrollbarNone {
		if c.ScrollbarOverlay = x.overlay; !x.overlay {
			c.ScrollbarWidth = 15
		}
	}
	if scrolls(ox) && !scrolls(oy) {
		c.Overflow.Height = fix(oy)
	} else if scrolls(oy) && !scrolls(ox) {
		c.Overflow.Width = fix(ox)
	}
	if x.toViewport {
		x.propagateOverflow(root)
	}
	x.display(root)
	x.transitions()
	x.animations()
	c.FollowColor(c.Color)
	c.TextShadows = followColor(c.TextShadows, c.Color)
	c.Decorations = x.decorations()
	c.Shadows = followColor(c.Shadows, c.Color)
	c.Filters = followFilterColor(c.Filters, c.Color)
	c.BackdropFilters = followFilterColor(c.BackdropFilters, c.Color)
	if c.AccentAuto {
		c.AccentColor = c.Color
	}
	if c.CaretAuto {
		c.CaretColor = c.Color
	}
}

// snapWidths snaps the widths of borders, the outline and column rules,
// outline-offset and border-spacing to device pixels, as Chrome does.
func (x *ctx) snapWidths() {
	c := x.c
	for i := range 4 {
		w := sideOf(&c.Border, i)
		*w = x.env.snap(*w, true)
	}
	c.OutlineWidth = x.env.snap(c.OutlineWidth, true)
	c.ColumnRuleWidth = x.env.snap(c.ColumnRuleWidth, true)
	c.OutlineOffset = x.env.snap(c.OutlineOffset, false)
	c.BorderSpacing = [2]float32{x.env.snap(c.BorderSpacing[0], false), x.env.snap(c.BorderSpacing[1], false)}
}

// display is the box the element's display makes where it stands: a block for the root,
// a flex or grid item, a float or a box out of the flow.
func (x *ctx) display(root bool) {
	c := x.c
	x.webkitBox()
	switch x.parent.Display {
	case style.FlexDisplay, style.InlineFlex, style.GridDisplay, style.InlineGrid:
		if !root {
			c.Display, c.TablePart = blockified(c.Display), style.NoTablePart
			c.Float = style.FloatNone
		}
	case style.Inline, style.Block, style.InlineBlock, style.DisplayNone, style.Contents:
		// not a flex or grid container
	}
	outside := c.Position == style.Absolute || c.Position == style.FixedPosition
	c.StaticInline = !root && outside && blockified(c.Display) != c.Display
	if root || outside {
		c.Display, c.TablePart = blockified(c.Display), style.NoTablePart
		c.Float = style.FloatNone
	}
	if c.Float != style.FloatNone {
		c.Display, c.TablePart = blockified(c.Display), style.NoTablePart
	}
}

// webkitBox lays display: -webkit-box out as Chrome does: a vertical one
// clamping its lines as flow-root, any other as flex along its orient.
func (x *ctx) webkitBox() {
	c := x.c
	switch {
	case !c.WebkitBox:
	case c.BoxVertical && c.WebkitLineClamp > 0:
		if c.Display == style.InlineFlex {
			c.Display = style.InlineBlock
		} else {
			c.Display = style.Block
		}
		c.FlowRoot = true
	case c.BoxVertical:
		c.Direction = style.Column
	default:
		c.Direction = style.Row
	}
}

// clampOf is a line-clamp's count, 0 for none.
func clampOf(_ *ctx, v []Component) int {
	if v[0].Kind == Number {
		return int(v[0].Num)
	}
	return 0
}

func blockified(d style.Display) style.Display {
	switch d {
	case style.Inline, style.InlineBlock:
		return style.Block
	case style.InlineFlex:
		return style.FlexDisplay
	case style.InlineGrid:
		return style.GridDisplay
	case style.Block, style.FlexDisplay, style.GridDisplay, style.DisplayNone, style.Contents:
		// already block-level, or no box
	}
	return d
}

func (x *ctx) transitions() {
	if !slices.ContainsFunc(x.trans[:], func(v []Component) bool { return v != nil }) {
		x.c.Transitions = initialTransitions()
		return
	}
	x.c.Transitions = transitionList(x.trans)
}

// animations is the animation list from the animation longhands' values,
// unset ones initial: one entry for each name, none where there is no
// name, the other lists repeated or cut to its length.
func (x *ctx) animations() {
	names := x.anims[len(x.anims)-1]
	if names == nil {
		return
	}
	var lists [8][][]Component
	for i, v := range x.anims {
		if v == nil {
			v = initialValue(shorthands[shortAnimation][i])
		}
		lists[i] = splitOn(v, func(c Component) bool { return c.Kind == Comma })
	}
	at := func(l, i int) Component { return lists[l][i%len(lists[l])][0] }
	var composites [][]Component
	if x.composite != nil {
		composites = splitOn(x.composite, func(c Component) bool { return c.Kind == Comma })
	}
	var out []style.Animation
	for i, n := range lists[7] {
		if isWord(n[0], kwNone) {
			continue
		}
		a := style.Animation{Name: n[0].Text, Easing: at(1, i).String(), Iterations: float32(math.Inf(1))}
		dur, _ := duration(at(0, i))
		delay, _ := duration(at(2, i))
		a.Duration, a.Delay = float32(max(dur, 0)), float32(delay)
		if c := at(3, i); c.Kind == Number {
			a.Iterations = float32(c.Num)
		}
		for d := style.AnimateNormal; d <= style.AnimateAlternateReverse; d++ {
			if isWord(at(4, i), animationDirections[d]) {
				a.Direction = d
			}
		}
		for f := style.FillNone; f <= style.FillBoth; f++ {
			if isWord(at(5, i), animationFills[f]) {
				a.Fill = f
			}
		}
		a.Paused = isWord(at(6, i), kwPaused)
		if len(composites) > 0 {
			a.Composite = compositeOf(composites[i%len(composites)][0])
		}
		out = append(out, a)
	}
	x.c.Animations = out
}

// compositeOf reads an animation-composition keyword.
func compositeOf(c Component) style.Composite {
	switch {
	case isWord(c, "add"):
		return style.ComposeAdd
	case isWord(c, "accumulate"):
		return style.ComposeAccumulate
	}
	return style.ComposeReplace
}

// transitionList is the transition list from the transition longhands'
// values, unset ones initial.
func transitionList(trans [5][]Component) []style.Transition {
	lists := make([][][]Component, len(trans))
	for i, v := range trans {
		if v == nil {
			v = initialValue(shorthands["transition"][i])
		}
		lists[i] = splitOn(v, func(c Component) bool { return c.Kind == Comma })
	}
	props := lists[0]
	if len(props) == 1 && len(props[0]) == 1 && isWord(props[0][0], kwNone) {
		return nil
	}
	var out []style.Transition
	for i, p := range props {
		t := style.Transition{Property: strings.ToLower(p[0].Text)}
		dur, _ := duration(lists[1][i%len(lists[1])][0])
		t.Duration = float32(max(dur, 0))
		t.Easing = lists[2][i%len(lists[2])][0].String()
		delay, _ := duration(lists[3][i%len(lists[3])][0])
		t.Delay = float32(delay)
		t.Discrete = isWord(lists[4][i%len(lists[4])][0], kwAllowDiscrete)
		out = append(out, t)
	}
	return out
}

// initialTransitions is the transition list of an element that sets
// none, worked out once and shared.
//
//nolint:gochecknoglobals // worked out once
var initialTransitions = sync.OnceValue(func() []style.Transition {
	return transitionList([5][]Component{})
})

// prop is how one longhand is set from its value, copied when inherited,
// and animated.
type prop struct {
	set     func(x *ctx, v []Component)
	inherit func(dst, src *style.Computed)
	anim    anim
}

// painted has p interpolate in Shown alone.
func (p prop) painted() prop {
	p.anim.kind = painted
	return p
}

func field[T any](get func(*style.Computed) *T, set func(x *ctx, v []Component) T, how interp[T]) prop {
	return prop{
		set:     func(x *ctx, v []Component) { *get(x.c) = set(x, v) },
		inherit: func(dst, src *style.Computed) { *get(dst) = *get(src) },
		anim:    how.on(get),
	}
}

func keyword[T any](get func(*style.Computed) *T, m map[string]T) prop {
	return field(get, func(_ *ctx, v []Component) T { return m[strings.ToLower(v[0].Text)] }, flips[T]())
}

func lengthField(get func(*style.Computed) *style.Length, how interp[style.Length]) prop {
	return field(get, func(x *ctx, v []Component) style.Length { return x.length(v[0]) }, how)
}

func colorField(get func(*style.Computed) *color.Color, how interp[color.Color]) prop {
	return field(get, func(x *ctx, v []Component) color.Color { return x.color(v[0]) }, how)
}

// currentField is a colour that may be currentColor, marked by f: two
// currentColors are the same value, whatever colour each stands for.
func currentField(f style.CurrentColorFields, get func(*style.Computed) *color.Color) prop {
	p := colorField(get, colors)
	return prop{
		set: func(x *ctx, v []Component) {
			p.set(x, v)
			x.c.CurrentColor &^= f
			if isWord(v[0], kwCurrentColor) {
				x.c.CurrentColor |= f
			}
		},
		inherit: func(dst, src *style.Computed) {
			p.inherit(dst, src)
			dst.CurrentColor = dst.CurrentColor&^f | src.CurrentColor&f
		},
		anim: anim{kind: interpolated, current: f,
			lerp: func(dst, a, b *style.Computed, t float32) bool {
				*get(dst) = lerpColor(*get(a), *get(b), t)
				dst.CurrentColor &^= f
				return true
			},
			equal: func(a, b *style.Computed) bool {
				ca, cb := a.CurrentColor&f != 0, b.CurrentColor&f != 0
				return ca == cb && (ca || *get(a) == *get(b))
			},
			add: func(dst, a, b *style.Computed, _ bool) bool {
				*get(dst) = addColor(*get(a), *get(b))
				dst.CurrentColor &^= f
				return true
			},
		},
	}
}

func (x *ctx) length(c Component) style.Length {
	if c.Kind == Ident {
		switch strings.ToLower(c.Text) {
		case kwMinContent:
			return style.Length{Kind: style.MinContent}
		case kwMaxContent, kwContent:
			return style.Length{Kind: style.MaxContent}
		case kwFitContent:
			return style.Length{Kind: style.FitContent}
		case kwStretch:
			return style.Length{Kind: style.Stretch}
		case kwNone:
			return style.Length{Kind: style.None}
		}
		return style.Length{}
	}
	if c.Kind == Function && strings.EqualFold(c.Text, kwFitContent) {
		l := x.length(items(c.Args)[0])
		l.Kind = style.FitContent
		return l
	}
	d, _ := parseDim(c)
	if d.IsFlex() {
		return style.Length{Kind: style.Fr, Px: float32(d.Fr)}
	}
	d = d.Absolute(x.units(&d))
	if len(d.terms) > 0 {
		return style.Length{Kind: style.Fixed, Px: style.Finite(d.Resolve(0), style.MaxLength), IsPercent: true,
			Calc: calcOf(d)}
	}
	return style.Length{Kind: style.Fixed, Px: style.Finite(d.value(), style.MaxLength),
		Percent: style.Finite(d.Pct, style.MaxLength), IsPercent: d.HasPercent()}
}

// number is c as a number on the element, a percentage of 1, against
// basis when what is a number depends on a percentage.
func (x *ctx) number(c Component, basis float64) float32 {
	d, _ := parseDim(c)
	return style.Finite(d.Absolute(x.units(&d)).Resolve(basis), math.MaxFloat32)
}

func (x *ctx) num(c Component) float32 { return x.number(c, 1) }

// angle is c in degrees on the element.
func (x *ctx) angle(c Component) float64 {
	if c.Kind == Number {
		return 0
	}
	d, _ := parseDim(c)
	return float64(style.Finite(d.Absolute(x.units(&d)).Resolve(0), math.MaxFloat32))
}

// units are what d's relative units stand for on the element.
func (x *ctx) units(d *Dim) *Units {
	u := &x.u
	u.Em, u.Rem = float64(x.em), float64(x.rem)
	if d.rel == 0 {
		return u
	}
	if x.env != nil {
		x.env.used |= d.rel
	}
	for i := range relUnits {
		if d.rel&(1<<i) != 0 {
			u.Rel[i] = float64(x.unit(i))
		}
	}
	return u
}

// unit is what one of u stands for on the element: a unit of the font in
// the element's own, or its parent's while its font-size is being read
// (lh its parent's line height while its line-height is), a root unit in
// the root's, a viewport unit in the viewport.
func (x *ctx) unit(u relUnit) float32 {
	if u >= unitCqw {
		return x.env.containerUnit(x.node, x.self, u) / 100
	}
	if u >= unitVw {
		w, h := x.env.viewport()
		return [...]float32{w, h, min(w, h), max(w, h)}[u-unitVw] / 100
	}
	st, sized, lined := x.c, x.sized, x.lined
	if u >= unitRex {
		u -= unitRex
		if root := x.env.rootStyle(); root != nil && !x.root {
			st, sized, lined = root, true, true
		}
	}
	if !sized || u == unitLh && !lined {
		st = x.parent
	}
	in := x.env.fontUnits(st.Font)
	switch u {
	case unitEx:
		return in.Ex
	case unitCh:
		return in.Ch
	case unitCap:
		return in.Cap
	case unitIc:
		return in.Ic
	case unitLh, unitRex, unitRch, unitRcap, unitRic, unitRlh, unitVw, unitVh, unitVmin, unitVmax, unitCqw, unitCqh,
		unitCqmin, unitCqmax, relUnits:
	}
	switch lh := st.LineHeight; {
	case lh.Normal:
		return in.Line
	case lh.Factor != 0:
		return lh.Factor * st.Font.Size
	default:
		return lh.Px
	}
}

// spacing is a letter-spacing or a word-spacing: a calc() resolved, a
// percentage kept to resolve against the font size of each element that
// inherits it.
func (x *ctx) spacing(v []Component) style.Spacing {
	l := x.length(v[0])
	if l.Calc != (unique.Handle[style.Calc]{}) {
		return style.Spacing{Px: l.At(x.em)}
	}
	return style.Spacing{Px: l.Px, Percent: l.Percent}
}

func (x *ctx) px(c Component) float32 {
	return x.length(c).Px
}

func (x *ctx) color(c Component) color.Color {
	if isWord(c, kwCurrentColor) {
		return x.c.Color
	}
	return linear(c, x.c.Color)
}

// linear is the colour c, currentcolor in it standing for cur.
func linear(c Component, cur color.Color) color.Color {
	var s Color
	if c.Kind == Function {
		in := cur.SRGB()
		s, _ = colorIn(c, &Color{float64(in[0]), float64(in[1]), float64(in[2]), float64(cur.A)})
	} else {
		s, _ = parseColor(c)
	}
	l := color.FromSRGB([3]float32{float32(s.R), float32(s.G), float32(s.B)})
	l.A = float32(s.A)
	return l
}

//nolint:gochecknoglobals // lookup table
var verticalAligns = map[string]style.VerticalAlign{
	kwBaseline: style.AlignBaselineText, "sub": style.AlignSub, "super": style.AlignSuper,
	"text-top": style.AlignTextTop, "text-bottom": style.AlignTextBottom, kwMiddle: style.AlignMiddle,
	kwTop: style.AlignTop, kwBottom: style.AlignBottom,
}

//nolint:gochecknoglobals,goconst // lookup table
var displays = map[string]style.Display{
	"inline": style.Inline, "block": style.Block, "inline-block": style.InlineBlock,
	"flex": style.FlexDisplay, "inline-flex": style.InlineFlex, "grid": style.GridDisplay,
	"inline-grid": style.InlineGrid, "none": style.DisplayNone, "contents": style.Contents,
	"list-item": style.Block, "flow-root": style.Block, "table": style.Block, "inline-table": style.InlineBlock,
	"table-row": style.Block, "table-cell": style.Block, "table-row-group": style.Block,
	"table-header-group": style.Block, "table-footer-group": style.Block, "table-column": style.Block,
	"table-column-group": style.Block, "table-caption": style.Block,
	"-webkit-box": style.FlexDisplay, "-webkit-inline-box": style.InlineFlex,
}

//nolint:gochecknoglobals // lookup table
var tableParts = map[string]style.TablePart{
	"table-row": style.TableRow, "table-cell": style.TableCell, "table-row-group": style.TableRowGroup,
	"table-header-group": style.TableHeaderGroup, "table-footer-group": style.TableFooterGroup,
	"table-column": style.TableColumn, "table-column-group": style.TableColumnGroup,
	"table-caption": style.TableCaption,
}

//nolint:gochecknoglobals,goconst // lookup tables
var (
	alignWords4 = map[string]style.Align{
		"normal": style.AlignNormal, "auto": style.AlignAuto, "stretch": style.AlignStretch,
		"start": style.AlignStart, "end": style.AlignEnd, "flex-start": style.AlignFlexStart,
		"flex-end": style.AlignFlexEnd, "center": style.AlignCenter, "baseline": style.AlignBaseline,
		"space-between": style.AlignSpaceBetween, "space-around": style.AlignSpaceAround,
		"space-evenly": style.AlignSpaceEvenly, "self-start": style.AlignStart, "self-end": style.AlignEnd,
		"left": style.AlignStart, "right": style.AlignEnd, "legacy": style.AlignNormal,
	}
	borderStyles = map[string]style.BorderStyle{
		"none": style.BorderNone, "hidden": style.BorderNone, "solid": style.BorderSolid,
		"dashed": style.BorderDashed, "dotted": style.BorderDotted,
		"inset": style.BorderInset, "outset": style.BorderOutset,
		"groove": style.BorderGroove, "ridge": style.BorderRidge, "double": style.BorderDouble,
	}
	overflows = map[string]style.Overflow{
		"visible": style.Visible, "hidden": style.Hidden, "clip": style.Clip,
		"scroll": style.Scroll, "auto": style.OverflowAuto,
	}
)

//nolint:gochecknoglobals,goconst // a keyword table
var breaks = map[string]style.Break{
	"avoid": style.BreakAvoid, "avoid-column": style.BreakAvoid, "column": style.BreakColumn,
}

//nolint:gochecknoglobals // lookup tables
var (
	directions   = map[string]style.Direction{dirLTR: style.LTR, dirRTL: style.RTL}
	unicodeBidis = map[string]style.UnicodeBidi{"normal": style.BidiNormal, "embed": style.BidiEmbed,
		"isolate": style.BidiIsolate, "bidi-override": style.BidiOverride, "isolate-override": style.BidiIsolateOverride,
		"plaintext": style.BidiPlaintext}
)

// sided is the keyword c of m, inline-start and inline-end the left or
// the right side as the element's direction has them.
func sided[T any](x *ctx, c Component, m map[string]T, left, right T) T {
	switch strings.ToLower(c.Text) {
	case "inline-start":
		if x.c.Dir == style.RTL {
			return right
		}
		return left
	case "inline-end":
		if x.c.Dir == style.RTL {
			return left
		}
		return right
	}
	return m[strings.ToLower(c.Text)]
}

func borderWidthOf(x *ctx, v []Component) float32 {
	switch {
	case isWord(v[0], "thin"):
		return 1
	case isWord(v[0], "medium"):
		return 3
	case isWord(v[0], "thick"):
		return 5
	}
	return x.px(v[0])
}

func cornerOf(x *ctx, v []Component) style.Size[style.Length] {
	l := x.length(v[0])
	return style.Size[style.Length]{Width: l, Height: l}
}

func inheritDisplay(dst, src *style.Computed) {
	dst.Display, dst.ListItem, dst.FlowRoot, dst.Table = src.Display, src.ListItem, src.FlowRoot, src.Table
	dst.TablePart, dst.WebkitBox = src.TablePart, src.WebkitBox
}

// boxed reports whether display at t between a's and b's is b's: as
// any discrete value from halfway, but all inside the interval when a's is
// none and b's is not, and none of it when b's is none and a's is not, so
// the element shows a box all along.
func boxed(a, b *style.Computed, t float32) bool {
	switch none := style.DisplayNone; {
	case t <= 0:
		return false
	case t >= 1:
		return true
	case a.Display == none && b.Display != none:
		return true
	case b.Display == none && a.Display != none:
		return false
	}
	return t >= 0.5
}

func sideProps(prefix, suffix string, how interp[style.Length],
	get func(*style.Computed) *style.Sides[style.Length]) map[string]prop {
	return map[string]prop{
		prefix + kwTop + suffix:    lengthField(func(c *style.Computed) *style.Length { return &get(c).Top }, how),
		prefix + kwRight + suffix:  lengthField(func(c *style.Computed) *style.Length { return &get(c).Right }, how),
		prefix + kwBottom + suffix: lengthField(func(c *style.Computed) *style.Length { return &get(c).Bottom }, how),
		prefix + kwLeft + suffix:   lengthField(func(c *style.Computed) *style.Length { return &get(c).Left }, how),
	}
}

//nolint:gochecknoglobals,goconst // the longhands' table
var props = func() map[string]prop {
	m := map[string]prop{
		"display": {
			set: func(x *ctx, v []Component) {
				x.c.Display = displays[strings.ToLower(v[0].Text)]
				x.c.ListItem = isWord(v[0], "list-item")
				x.c.FlowRoot = isWord(v[0], "flow-root")
				x.c.Table = isWord(v[0], "table") || isWord(v[0], "inline-table")
				x.c.TablePart = tableParts[strings.ToLower(v[0].Text)]
				x.c.WebkitBox = isWord(v[0], "-webkit-box", "-webkit-inline-box")
			},
			inherit: inheritDisplay,
			anim: anim{kind: discrete, lerp: func(dst, a, b *style.Computed, t float32) bool {
				src := a
				if boxed(a, b, t) {
					src = b
				}
				inheritDisplay(dst, src)
				return true
			}},
		},
		"float": field(func(c *style.Computed) *style.Float { return &c.Float },
			func(x *ctx, v []Component) style.Float {
				return sided(x, v[0], map[string]style.Float{"none": style.FloatNone, "left": style.FloatLeft,
					"right": style.FloatRight}, style.FloatLeft, style.FloatRight)
			}, flips[style.Float]()),
		"direction": {
			set:     func(x *ctx, v []Component) { x.c.Dir = directions[strings.ToLower(v[0].Text)] },
			inherit: func(dst, src *style.Computed) { dst.Dir = src.Dir },
			anim:    anim{kind: still},
		},
		"unicode-bidi": {
			set:     func(x *ctx, v []Component) { x.c.Bidi = unicodeBidis[strings.ToLower(v[0].Text)] },
			inherit: func(dst, src *style.Computed) { dst.Bidi = src.Bidi },
			anim:    anim{kind: still},
		},
		"vertical-align": {
			set: func(x *ctx, v []Component) {
				x.c.VerticalShift = style.Length{}
				if v[0].Kind == Ident {
					x.c.VerticalAlign = verticalAligns[strings.ToLower(v[0].Text)]
					return
				}
				x.c.VerticalAlign, x.c.VerticalShift = style.AlignLength, x.length(v[0])
			},
			inherit: func(dst, src *style.Computed) {
				dst.VerticalAlign, dst.VerticalShift = src.VerticalAlign, src.VerticalShift
			},
			anim: anim{kind: interpolated,
				lerp: func(dst, a, b *style.Computed, t float32) bool {
					if a.VerticalAlign != style.AlignLength || b.VerticalAlign != style.AlignLength {
						return false
					}
					l, ok := lerpLength(a.VerticalShift, b.VerticalShift, t, false)
					if ok {
						dst.VerticalAlign, dst.VerticalShift = style.AlignLength, l
					}
					return ok
				},
				equal: func(a, b *style.Computed) bool {
					return a.VerticalAlign == b.VerticalAlign && a.VerticalShift == b.VerticalShift
				},
			},
		},
		"clear": field(func(c *style.Computed) *style.Clear { return &c.Clear },
			func(x *ctx, v []Component) style.Clear {
				return sided(x, v[0], map[string]style.Clear{"none": style.ClearNone, "left": style.ClearLeft,
					"right": style.ClearRight, "both": style.ClearBoth}, style.ClearLeft, style.ClearRight)
			}, flips[style.Clear]()),
		"container-type": keyword(func(c *style.Computed) *style.ContainerType { return &c.ContainerType },
			map[string]style.ContainerType{"normal": style.ContainerNormal, "size": style.ContainerSize,
				"inline-size": style.ContainerInlineSize}),
		"container-name": field(func(c *style.Computed) *[]string { return &c.ContainerNames },
			func(_ *ctx, v []Component) []string {
				if isWord(v[0], kwNone) {
					return nil
				}
				names := make([]string, len(v))
				for i, c := range v {
					names[i] = c.Text
				}
				return names
			}, flips[[]string]()),
		"box-sizing": keyword(func(c *style.Computed) *style.BoxSizing { return &c.BoxSizing },
			map[string]style.BoxSizing{"border-box": style.BorderBox, "content-box": style.ContentBox}),
		"position": keyword(func(c *style.Computed) *style.Position { return &c.Position }, map[string]style.Position{
			"static": style.Static, "relative": style.Relative, "absolute": style.Absolute,
			"fixed": style.FixedPosition, "sticky": style.Sticky,
		}),
		"width":      lengthField(func(c *style.Computed) *style.Length { return &c.Size.Width }, sizeLength),
		"height":     lengthField(func(c *style.Computed) *style.Length { return &c.Size.Height }, sizeLength),
		"min-width":  lengthField(func(c *style.Computed) *style.Length { return &c.MinSize.Width }, sizeLength),
		"min-height": lengthField(func(c *style.Computed) *style.Length { return &c.MinSize.Height }, sizeLength),
		"max-width":  lengthField(func(c *style.Computed) *style.Length { return &c.MaxSize.Width }, sizeLength),
		"max-height": lengthField(func(c *style.Computed) *style.Length { return &c.MaxSize.Height }, sizeLength),
		"aspect-ratio": field(
			func(c *style.Computed) *float32 { return &c.AspectRatio },
			func(_ *ctx, v []Component) float32 {
				switch {
				case len(v) == 3 && v[2].Num != 0:
					return float32(v[0].Num / v[2].Num)
				case len(v) == 1 && v[0].Kind == Number:
					return float32(v[0].Num)
				}
				return 0
			}, ratios,
		),
		"overflow-x": keyword(func(c *style.Computed) *style.Overflow { return &c.Overflow.Width }, overflows),
		"overflow-y": keyword(func(c *style.Computed) *style.Overflow { return &c.Overflow.Height }, overflows),
		"z-index": {
			set: func(x *ctx, v []Component) {
				x.c.ZAuto = isWord(v[0], "auto")
				x.c.ZIndex = 0
				if !x.c.ZAuto {
					x.c.ZIndex = int(v[0].Num)
				}
			},
			inherit: func(dst, src *style.Computed) { dst.ZIndex, dst.ZAuto = src.ZIndex, src.ZAuto },
			anim: anim{kind: interpolated,
				lerp: func(dst, a, b *style.Computed, t float32) bool {
					if a.ZAuto || b.ZAuto {
						return false
					}
					dst.ZIndex, _ = integers(math.MinInt).lerp(a.ZIndex, b.ZIndex, t)
					dst.ZAuto = false
					return true
				},
				equal: func(a, b *style.Computed) bool { return a.ZIndex == b.ZIndex && a.ZAuto == b.ZAuto },
			},
		},

		"flex-direction": keyword(func(c *style.Computed) *style.FlexDirection { return &c.Direction },
			map[string]style.FlexDirection{"row": style.Row, "row-reverse": style.RowReverse,
				"column": style.Column, "column-reverse": style.ColumnReverse}),
		"flex-wrap": keyword(func(c *style.Computed) *style.FlexWrap { return &c.Wrap }, map[string]style.FlexWrap{
			"nowrap": style.NoWrap, "wrap": style.Wrap, "wrap-reverse": style.WrapReverse,
		}),
		"flex-grow": field(func(c *style.Computed) *float32 { return &c.Grow },
			func(x *ctx, v []Component) float32 { return x.num(v[0]) }, numbers(0, math.MaxFloat32)),
		"flex-shrink": field(func(c *style.Computed) *float32 { return &c.Shrink },
			func(x *ctx, v []Component) float32 { return x.num(v[0]) }, numbers(0, math.MaxFloat32)),
		"flex-basis":      lengthField(func(c *style.Computed) *style.Length { return &c.Basis }, sizeLength),
		"row-gap":         lengthField(func(c *style.Computed) *style.Length { return &c.Gap.Height }, sizeLength),
		"column-gap":      lengthField(func(c *style.Computed) *style.Length { return &c.Gap.Width }, sizeLength),
		"align-items":     keyword(func(c *style.Computed) *style.Align { return &c.AlignItems }, alignWords4),
		"align-self":      keyword(func(c *style.Computed) *style.Align { return &c.AlignSelf }, alignWords4),
		"align-content":   keyword(func(c *style.Computed) *style.Align { return &c.AlignContent }, alignWords4),
		"justify-content": keyword(func(c *style.Computed) *style.Align { return &c.JustifyContent }, alignWords4),
		"justify-items":   keyword(func(c *style.Computed) *style.Align { return &c.JustifyItems }, alignWords4),
		"justify-self":    keyword(func(c *style.Computed) *style.Align { return &c.JustifySelf }, alignWords4),
		"order": field(func(c *style.Computed) *int { return &c.Order },
			func(_ *ctx, v []Component) int { return int(v[0].Num) }, integers(math.MinInt)),

		"column-count": field(func(c *style.Computed) *int { return &c.ColumnCount },
			func(_ *ctx, v []Component) int { return int(v[0].Num) }, counts),
		"column-width": lengthField(func(c *style.Computed) *style.Length { return &c.ColumnWidth }, sizeLength),
		"column-fill": field(func(c *style.Computed) *bool { return &c.ColumnFillAuto },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "auto") }, flips[bool]()),
		"column-span": field(func(c *style.Computed) *bool { return &c.ColumnSpanAll },
			func(_ *ctx, v []Component) bool { return isWord(v[0], kwAll) }, flips[bool]()),
		"column-rule-width": field(func(c *style.Computed) *float32 { return &c.ColumnRuleWidth }, borderWidthOf, widths),
		"column-rule-style": keyword(func(c *style.Computed) *style.BorderStyle { return &c.ColumnRuleStyle }, borderStyles),
		"column-rule-color": currentField(style.ColumnRuleCurrent,
			func(c *style.Computed) *color.Color { return &c.ColumnRuleColor }),
		"break-inside": field(func(c *style.Computed) *bool { return &c.BreakInsideAvoid },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "avoid", "avoid-column") }, flips[bool]()),
		"break-before": keyword(func(c *style.Computed) *style.Break { return &c.BreakBefore }, breaks),
		"break-after":  keyword(func(c *style.Computed) *style.Break { return &c.BreakAfter }, breaks),
		"line-clamp":   field(func(c *style.Computed) *int { return &c.LineClamp }, clampOf, flips[int]()),
		"-webkit-line-clamp": field(func(c *style.Computed) *int { return &c.WebkitLineClamp }, clampOf,
			flips[int]()),
		"-webkit-box-orient": field(func(c *style.Computed) *bool { return &c.BoxVertical },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "vertical", "block-axis") }, flips[bool]()),
		"orphans": field(func(c *style.Computed) *int { return &c.Orphans },
			func(_ *ctx, v []Component) int { return int(v[0].Num) }, integers(1)),
		"widows": field(func(c *style.Computed) *int { return &c.Widows },
			func(_ *ctx, v []Component) int { return int(v[0].Num) }, integers(1)),

		"grid-template-columns": templateTracks(func(c *style.Computed) (*[]style.TrackRepeat, *bool) {
			return &c.Columns, &c.Subgrid.Width
		}),
		"grid-template-rows": templateTracks(func(c *style.Computed) (*[]style.TrackRepeat, *bool) {
			return &c.Rows, &c.Subgrid.Height
		}),
		"grid-template-areas": field(func(c *style.Computed) *[]style.GridArea { return &c.Areas },
			templateAreas, flips[[]style.GridArea]()),
		"grid-auto-columns": field(func(c *style.Computed) *[]style.Track { return &c.AutoColumns },
			(*ctx).autoTracks, flips[[]style.Track]()),
		"grid-auto-rows": field(func(c *style.Computed) *[]style.Track { return &c.AutoRows },
			(*ctx).autoTracks, flips[[]style.Track]()),
		"grid-auto-flow": field(
			func(c *style.Computed) *style.GridFlow { return &c.Flow },
			func(_ *ctx, v []Component) style.GridFlow {
				column, dense := false, false
				for _, c := range v {
					column = column || isWord(c, "column")
					dense = dense || isWord(c, "dense")
				}
				switch {
				case column && dense:
					return style.FlowColumnDense
				case column:
					return style.FlowColumn
				case dense:
					return style.FlowRowDense
				}
				return style.FlowRow
			}, flips[style.GridFlow](),
		),
		"grid-column-start": field(func(c *style.Computed) *style.GridLine { return &c.Column[0] },
			gridLine, flips[style.GridLine]()),
		"grid-column-end": field(func(c *style.Computed) *style.GridLine { return &c.Column[1] },
			gridLine, flips[style.GridLine]()),
		"grid-row-start": field(func(c *style.Computed) *style.GridLine { return &c.Row[0] },
			gridLine, flips[style.GridLine]()),
		"grid-row-end": field(func(c *style.Computed) *style.GridLine { return &c.Row[1] },
			gridLine, flips[style.GridLine]()),

		"font-family": field(
			func(c *style.Computed) *[]string { return &c.Font.Family },
			func(_ *ctx, v []Component) []string {
				var out []string
				for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
					var names []string
					for _, c := range part {
						names = append(names, c.Text)
					}
					out = append(out, strings.Join(names, " "))
				}
				return out
			}, flips[[]string](),
		),
		"font-size": field(func(c *style.Computed) *float32 { return &c.Font.Size },
			(*ctx).fontSize, numbers(0, math.MaxFloat32)),
		"font-weight": field(func(c *style.Computed) *float32 { return &c.Font.Weight }, (*ctx).fontWeight, weights),
		"font-style": field(func(c *style.Computed) *bool { return &c.Font.Italic }, func(_ *ctx, v []Component) bool {
			return !isWord(v[0], "normal")
		}, flips[bool]()),
		"font-stretch": field(func(c *style.Computed) *float32 { return &c.Font.Stretch },
			func(_ *ctx, v []Component) float32 {
				if w, ok := fontStretch(v[0]); ok {
					return w
				}
				return 100
			}, numbers(0, math.MaxFloat32)),
		"font-synthesis": field(func(c *style.Computed) *style.FontSynthesis { return &c.Font.Synthesis },
			func(_ *ctx, v []Component) style.FontSynthesis {
				no := style.NoSyntheticBold | style.NoSyntheticItalic
				for _, c := range v {
					switch {
					case isWord(c, "weight"):
						no &^= style.NoSyntheticBold
					case isWord(c, "style"):
						no &^= style.NoSyntheticItalic
					}
				}
				return no
			}, flips[style.FontSynthesis]()),
		"line-height": field(
			func(c *style.Computed) *style.LineHeight { return &c.LineHeight },
			func(x *ctx, v []Component) style.LineHeight {
				switch {
				case isWord(v[0], "normal"):
					return style.LineHeight{Normal: true}
				case v[0].Kind == Number:
					return style.LineHeight{Factor: float32(v[0].Num)}
				case v[0].Kind == Function:
					d, _ := parseDim(v[0])
					d = signPercents(d)
					v := style.Finite(d.Absolute(x.units(&d)).Resolve(float64(x.em)), math.MaxFloat32)
					if d.typ == typeNumber {
						return style.LineHeight{Factor: v}
					}
					return style.LineHeight{Px: max(v, 0)}
				}
				l := x.length(v[0])
				return style.LineHeight{Px: l.At(x.em)}
			}, lineHeights,
		),
		propCaps: keyword(func(c *style.Computed) *style.FontCaps { return &c.Font.Caps }, capsWords),
		propLigatures: field(func(c *style.Computed) *string { return &c.FontVariant.Ligatures },
			variantField(propLigatures), flips[string]()),
		propNumeric: field(func(c *style.Computed) *string { return &c.FontVariant.Numeric },
			variantField(propNumeric), flips[string]()),
		propEastAsian: field(func(c *style.Computed) *string { return &c.FontVariant.EastAsian },
			variantField(propEastAsian), flips[string]()),
		propAlternates: field(func(c *style.Computed) *string { return &c.FontVariant.Alternates },
			variantField(propAlternates), flips[string]()),
		propPosition: field(func(c *style.Computed) *string { return &c.FontVariant.Position },
			variantField(propPosition), flips[string]()),
		"letter-spacing": field(func(c *style.Computed) *style.Spacing { return &c.LetterSpacing },
			(*ctx).spacing, textSpacings),
		"word-spacing": field(func(c *style.Computed) *style.Spacing { return &c.WordSpacing },
			(*ctx).spacing, textSpacings),
		"color": field(
			func(c *style.Computed) *color.Color { return &c.Color },
			func(x *ctx, v []Component) color.Color {
				if isWord(v[0], "currentcolor") {
					return x.parent.Color
				}
				return linear(v[0], x.parent.Color)
			}, colors,
		).painted(),
		propColorScheme: field(func(c *style.Computed) *style.ColorScheme { return &c.ColorScheme },
			func(_ *ctx, v []Component) style.ColorScheme { return colorScheme(v) }, flips[style.ColorScheme]()),
		"-webkit-font-smoothing": {
			set: func(x *ctx, v []Component) {
				kw := strings.ToLower(v[0].Text)
				x.c.PixelText, x.c.ExactText = kw == kwNone, kw == "antialiased"
			},
			inherit: func(dst, src *style.Computed) { dst.PixelText, dst.ExactText = src.PixelText, src.ExactText },
			anim:    anim{kind: discrete},
		},
		"text-transform": keyword(func(c *style.Computed) *style.TextTransform { return &c.TextTransform },
			map[string]style.TextTransform{"none": style.NoTransform, "uppercase": style.Uppercase,
				"lowercase": style.Lowercase, "capitalize": style.Capitalize}),
		"text-align": keyword(func(c *style.Computed) *style.TextAlign { return &c.TextAlign },
			map[string]style.TextAlign{"start": style.Start, "end": style.End, "left": style.Left, "right": style.Right,
				"center": style.Center, "justify": style.Justify, "-webkit-left": style.BlockLeft,
				"-webkit-right": style.BlockRight, "-webkit-center": style.BlockCenter}),
		"text-indent": lengthField(func(c *style.Computed) *style.Length { return &c.TextIndent }, anyLength),
		"caption-side": keyword(func(c *style.Computed) *style.CaptionSide { return &c.CaptionSide },
			map[string]style.CaptionSide{"top": style.CaptionTop, "bottom": style.CaptionBottom}),
		"empty-cells": field(func(c *style.Computed) *bool { return &c.EmptyCellsHide },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "hide") }, flips[bool]()),
		"border-collapse": field(func(c *style.Computed) *bool { return &c.BorderCollapse },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "collapse") }, flips[bool]()),
		"table-layout": keyword(func(c *style.Computed) *style.TableLayout { return &c.TableLayout },
			map[string]style.TableLayout{"auto": style.TableAuto, "fixed": style.TableFixed}),
		"white-space": keyword(func(c *style.Computed) *style.WhiteSpace { return &c.WhiteSpace },
			map[string]style.WhiteSpace{"normal": style.Normal, "nowrap": style.NoWrapSpace, "pre": style.Pre,
				"pre-wrap": style.PreWrap, "pre-line": style.PreLine}),
		"border-spacing": {
			set: func(x *ctx, v []Component) {
				v = items(v)
				h := x.px(v[0])
				x.c.BorderSpacing = [2]float32{h, h}
				if len(v) > 1 {
					x.c.BorderSpacing[1] = x.px(v[1])
				}
			},
			inherit: func(dst, src *style.Computed) { dst.BorderSpacing = src.BorderSpacing },
			anim:    spacings.on(func(c *style.Computed) *[2]float32 { return &c.BorderSpacing }),
		},
		"overflow-wrap": {
			set: func(x *ctx, v []Component) {
				x.c.BreakWord = !isWord(v[0], "normal")
				x.c.BreakAnywhere = isWord(v[0], "anywhere")
			},
			inherit: func(dst, src *style.Computed) { dst.BreakWord, dst.BreakAnywhere = src.BreakWord, src.BreakAnywhere },
			anim:    anim{kind: discrete},
		},
		"text-decoration-skip-ink": keyword(func(c *style.Computed) *bool { return &c.InkThrough },
			map[string]bool{"auto": false, "all": false, "none": true}),
		"tab-size": field(func(c *style.Computed) *style.TabSize { return &c.TabSize },
			func(x *ctx, v []Component) style.TabSize {
				if d, err := parseDim(v[0]); err == nil && d.typ == typeNumber {
					return style.TabSize{Spaces: x.number(v[0], 0)}
				}
				return style.TabSize{Px: x.px(v[0]), Length: true}
			}, tabSizes),
		"word-break": keyword(func(c *style.Computed) *style.WordBreak { return &c.WordBreak },
			map[string]style.WordBreak{"normal": style.WordBreakNormal, "break-all": style.BreakAll,
				"keep-all": style.KeepAll, "break-word": style.WordBreakBreakWord}),
		"line-break": keyword(func(c *style.Computed) *style.LineBreak { return &c.LineBreak },
			map[string]style.LineBreak{"auto": style.LineBreakAuto, "loose": style.LineBreakLoose,
				"normal": style.LineBreakNormal, "strict": style.LineBreakStrict, "anywhere": style.LineBreakAnywhere}),
		"scrollbar-width": field(func(c *style.Computed) *bool { return &c.ScrollbarNone },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "none") }, flips[bool]()),
		"text-overflow": field(func(c *style.Computed) *bool { return &c.Ellipsis }, func(_ *ctx, v []Component) bool {
			return isWord(v[0], "ellipsis")
		}, flips[bool]()),
		"list-style-image": field(func(c *style.Computed) *string { return &c.ListImage },
			func(x *ctx, v []Component) string {
				if isWord(v[0], kwNone) {
					return ""
				}
				return imageSource(x, v)
			}, flips[string]()),
		"list-style-type": {
			set:     listStyle,
			inherit: func(dst, src *style.Computed) { dst.ListStyle, dst.ListName = src.ListStyle, src.ListName },
			anim:    anim{kind: discrete},
		},
		"visibility": {
			set: func(x *ctx, v []Component) {
				x.c.Visible, x.c.Collapse = isWord(v[0], "visible"), isWord(v[0], "collapse")
			},
			inherit: func(dst, src *style.Computed) { dst.Visible, dst.Collapse = src.Visible, src.Collapse },
			anim:    visibility.on(func(c *style.Computed) *bool { return &c.Visible }),
		},
		"pointer-events": field(func(c *style.Computed) *bool { return &c.PointerNone },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "none") }, flips[bool]()),
		"app-region": keyword(func(c *style.Computed) *style.AppRegion { return &c.AppRegion },
			map[string]style.AppRegion{"drag": style.AppRegionDrag, "no-drag": style.AppRegionNoDrag,
				"none": style.AppRegionNoDrag}),
		"user-select": field(func(c *style.Computed) *style.UserSelect { return &c.UserSelect },
			func(_ *ctx, v []Component) style.UserSelect {
				switch strings.ToLower(v[0].Text) {
				case "text":
					return style.SelectText
				case "none":
					return style.SelectNone
				case kwAll:
					return style.SelectAll
				}
				// auto set on an element is text in Chrome, whatever its parent
				return style.SelectText
			}, flips[style.UserSelect]()),
		"cursor": field(func(c *style.Computed) *string { return &c.Cursor }, func(_ *ctx, v []Component) string {
			if isWord(v[0], "auto") {
				return ""
			}
			return strings.ToLower(v[0].Text)
		}, flips[string]()),

		"background-color": colorField(func(c *style.Computed) *color.Color { return &c.Background }, colors).painted(),
		"background-image": layered(func(c *style.Computed) *[]style.Image { return &c.BackgroundImages },
			backgroundImage, flips[[]style.Image]()),
		"background-size": layered(func(c *style.Computed) *[]style.BackgroundSize { return &c.BackgroundSizes },
			(*ctx).backgroundSize, repeatable(backgroundSizes, style.BackgroundSize{})),
		"background-repeat": layered(func(c *style.Computed) *[]style.BackgroundRepeat { return &c.BackgroundRepeats },
			tiling, flips[[]style.BackgroundRepeat]()),
		"background-position": layered(
			func(c *style.Computed) *[]style.BackgroundPosition { return &c.BackgroundPositions }, (*ctx).position,
			repeatable(positions, style.TopLeft)),
		"background-attachment": layered(
			func(c *style.Computed) *[]style.BackgroundAttachment { return &c.BackgroundAttachments },
			wordOf(map[string]style.BackgroundAttachment{
				"scroll": style.AttachScroll, "fixed": style.AttachFixed, "local": style.AttachLocal,
			}), flips[[]style.BackgroundAttachment]()),
		"background-origin": layered(func(c *style.Computed) *[]style.BackgroundBox { return &c.BackgroundOrigins },
			wordOf(map[string]style.BackgroundBox{
				"padding-box": style.InitialBox, "border-box": style.UnderBorder, "content-box": style.UnderContent,
			}), flips[[]style.BackgroundBox]()),
		"background-clip": layered(func(c *style.Computed) *[]style.BackgroundBox { return &c.BackgroundClips },
			wordOf(map[string]style.BackgroundBox{
				"border-box": style.InitialBox, "padding-box": style.UnderPadding, "content-box": style.UnderContent,
				"text": style.UnderText,
			}), flips[[]style.BackgroundBox]()),
		"mask-image": layered(func(c *style.Computed) *[]style.Image { return &c.MaskImages },
			backgroundImage, flips[[]style.Image]()),
		"mask-size": layered(func(c *style.Computed) *[]style.BackgroundSize { return &c.MaskSizes },
			(*ctx).backgroundSize, repeatable(backgroundSizes, style.BackgroundSize{})),
		"mask-repeat": layered(func(c *style.Computed) *[]style.BackgroundRepeat { return &c.MaskRepeats },
			tiling, flips[[]style.BackgroundRepeat]()),
		"mask-position": layered(func(c *style.Computed) *[]style.BackgroundPosition { return &c.MaskPositions },
			(*ctx).position, repeatable(positions, style.TopLeft)),
		"mask-origin": layered(func(c *style.Computed) *[]style.BackgroundBox { return &c.MaskOrigins },
			wordOf(maskBoxes), flips[[]style.BackgroundBox]()),
		"mask-clip": layered(func(c *style.Computed) *[]style.BackgroundBox { return &c.MaskClips },
			wordOf(maskBoxes), flips[[]style.BackgroundBox]()),
		"mask-mode": layered(func(c *style.Computed) *[]style.MaskMode { return &c.MaskModes },
			wordOf(map[string]style.MaskMode{
				"match-source": style.MaskMatchSource, "alpha": style.MaskAlpha, "luminance": style.MaskLuminance,
			}), flips[[]style.MaskMode]()),
		"mask-composite": layered(func(c *style.Computed) *[]style.MaskComposite { return &c.MaskComposites },
			wordOf(map[string]style.MaskComposite{
				"add": style.CompositeAdd, "subtract": style.CompositeSubtract, "intersect": style.CompositeIntersect,
				"exclude": style.CompositeExclude,
			}), flips[[]style.MaskComposite]()),
		"clip-path": field(func(c *style.Computed) *style.ClipPath { return &c.ClipPath }, clipPath,
			clipPaths).painted(),
		"text-shadow": field(func(c *style.Computed) *[]style.Shadow { return &c.TextShadows }, (*ctx).shadows, shadows),
		"text-decoration-line": field(func(c *style.Computed) *style.DecorationLine { return &c.TextDecoration.Line },
			decorationLine, flips[style.DecorationLine]()),
		"text-decoration-style": keyword(func(c *style.Computed) *style.DecorationStyle { return &c.TextDecoration.Style },
			decorationStyles),
		"text-decoration-color": currentField(style.DecorationCurrent,
			func(c *style.Computed) *color.Color { return &c.TextDecoration.Color }),
		"text-decoration-thickness": lengthField(
			func(c *style.Computed) *style.Length { return &c.TextDecoration.Thickness }, anyLength),
		"image-rendering": keyword(func(c *style.Computed) *style.ImageRendering { return &c.ImageRendering },
			map[string]style.ImageRendering{"auto": style.RenderAuto, "pixelated": style.RenderPixelated}),
		"accent-color": autoColor(func(c *style.Computed) (*color.Color, *bool) { return &c.AccentColor, &c.AccentAuto }),
		"caret-color":  autoColor(func(c *style.Computed) (*color.Color, *bool) { return &c.CaretColor, &c.CaretAuto }),

		"-webkit-text-fill-color": textFill(),

		"object-position": field(func(c *style.Computed) *style.BackgroundPosition { return &c.ObjectPosition },
			(*ctx).position, positions),
		"object-fit": keyword(func(c *style.Computed) *style.ObjectFit { return &c.ObjectFit },
			map[string]style.ObjectFit{"fill": style.FitFill, "contain": style.FitContain, "cover": style.FitCover,
				"none": style.FitNone, "scale-down": style.FitScaleDown}),
		"border-image-source": field(func(c *style.Computed) *string { return &c.BorderImage.Source },
			imageSource, flips[string]()),
		"border-image-slice": filled(field(
			func(c *style.Computed) *style.Sides[float32] { return &c.BorderImage.Slice },
			func(_ *ctx, v []Component) style.Sides[float32] {
				if len(v) == 1 && v[0].Kind == Percentage && v[0].Num == 100 {
					return style.Sides[float32]{}
				}
				var n []float32
				for _, c := range v {
					if !isWord(c, "fill") {
						n = append(n, float32(c.Num))
					}
				}
				pick := [4][4]int{{0, 0, 0, 0}, {0, 1, 0, 1}, {0, 1, 2, 1}, {0, 1, 2, 3}}[len(n)-1]
				return style.Sides[float32]{Top: n[pick[0]], Right: n[pick[1]], Bottom: n[pick[2]], Left: n[pick[3]]}
			}, imageSlices,
		)),
		"border-image-width": field(
			func(c *style.Computed) *style.Sides[style.ImageWidth] { return &c.BorderImageWidth },
			func(x *ctx, v []Component) style.Sides[style.ImageWidth] {
				var w []style.ImageWidth
				for _, c := range v {
					switch {
					case isWord(c, "auto"):
						w = append(w, style.ImageWidth{Kind: style.WidthAuto})
					case c.Kind == Number:
						w = append(w, style.ImageWidth{Kind: style.WidthTimes, V: float32(c.Num)})
					case c.Kind == Percentage:
						w = append(w, style.ImageWidth{Kind: style.WidthPercent, V: float32(c.Num)})
					default:
						w = append(w, style.ImageWidth{Kind: style.WidthLength, V: x.px(c)})
					}
				}
				pick := [4][4]int{{0, 0, 0, 0}, {0, 1, 0, 1}, {0, 1, 2, 1}, {0, 1, 2, 3}}[len(w)-1]
				return style.Sides[style.ImageWidth]{Top: w[pick[0]], Right: w[pick[1]], Bottom: w[pick[2]], Left: w[pick[3]]}
			}, imageWidths,
		),
		"border-image-repeat": field(
			func(c *style.Computed) *[2]style.ImageRepeat { return &c.BorderImageRepeat },
			func(_ *ctx, v []Component) [2]style.ImageRepeat {
				of := func(c Component) style.ImageRepeat {
					switch {
					case isWord(c, kwRepeat):
						return style.RepeatTile
					case isWord(c, "round"):
						return style.RepeatRound
					case isWord(c, "space"):
						return style.RepeatSpace
					}
					return style.RepeatStretch
				}
				across := of(v[0])
				down := across
				if len(v) > 1 {
					down = of(v[1])
				}
				return [2]style.ImageRepeat{across, down}
			}, flips[[2]style.ImageRepeat](),
		),
		"box-shadow": field(func(c *style.Computed) *[]style.Shadow { return &c.Shadows },
			(*ctx).shadows, shadows).painted(),
		"outline-width": field(func(c *style.Computed) *float32 { return &c.OutlineWidth }, borderWidthOf, widths),
		"outline-style": keyword(func(c *style.Computed) *style.BorderStyle { return &c.OutlineStyle }, borderStyles),
		"outline-color": currentField(style.OutlineCurrent,
			func(c *style.Computed) *color.Color { return &c.OutlineColor }).painted(),
		"outline-offset": field(func(c *style.Computed) *float32 { return &c.OutlineOffset },
			func(x *ctx, v []Component) float32 { return x.px(v[0]) }, numbers(-math.MaxFloat32, math.MaxFloat32)),
		"opacity": field(func(c *style.Computed) *float32 { return &c.Opacity }, func(x *ctx, v []Component) float32 {
			return float32(clamp01(float64(x.num(v[0]))))
		}, numbers(0, 1)).painted(),
		"transform": field(func(c *style.Computed) *[]style.TransformOp { return &c.Transform },
			(*ctx).transform, transforms).painted(),
		"filter": field(func(c *style.Computed) *[]style.Filter { return &c.Filters }, (*ctx).filters,
			filters).painted(),
		"backdrop-filter": field(func(c *style.Computed) *[]style.Filter { return &c.BackdropFilters }, (*ctx).filters,
			filters).painted(),
		"mix-blend-mode": keyword(func(c *style.Computed) *style.BlendMode { return &c.BlendMode }, blendModes),
		"isolation": keyword(func(c *style.Computed) *bool { return &c.Isolate },
			map[string]bool{"auto": false, "isolate": true}),
		"background-blend-mode": layered(func(c *style.Computed) *[]style.BlendMode { return &c.BackgroundBlends },
			wordOf(blendModes), flips[[]style.BlendMode]()),
		"transform-origin": field(func(c *style.Computed) *style.Size[style.Length] { return &c.TransformOrigin },
			(*ctx).origin, pairs(anyLength)),
		"appearance": {set: func(*ctx, []Component) {}, inherit: func(_, _ *style.Computed) {}, anim: anim{kind: discrete}},
		"resize":     {set: func(*ctx, []Component) {}, inherit: func(_, _ *style.Computed) {}, anim: anim{kind: discrete}},

		"border-top-left-radius": field(func(c *style.Computed) *style.Size[style.Length] { return &c.Radius.TopLeft },
			cornerOf, pairs(sizeLength)),
		"border-top-right-radius": field(func(c *style.Computed) *style.Size[style.Length] { return &c.Radius.TopRight },
			cornerOf, pairs(sizeLength)),
		"border-bottom-right-radius": field(
			func(c *style.Computed) *style.Size[style.Length] { return &c.Radius.BottomRight },
			cornerOf, pairs(sizeLength),
		),
		"border-bottom-left-radius": field(func(c *style.Computed) *style.Size[style.Length] { return &c.Radius.BottomLeft },
			cornerOf, pairs(sizeLength)),
	}
	boxes := map[string]func(c *style.Computed) *style.Sides[style.Length]{
		"margin-":  func(c *style.Computed) *style.Sides[style.Length] { return &c.Margin },
		"padding-": func(c *style.Computed) *style.Sides[style.Length] { return &c.Padding },
		"":         func(c *style.Computed) *style.Sides[style.Length] { return &c.Inset },
	}
	how := map[string]interp[style.Length]{"margin-": anyLength, "padding-": sizeLength, "": anyLength}
	for prefix, get := range boxes {
		maps.Copy(m, sideProps(prefix, "", how[prefix], get))
	}
	for i, side := range []string{"top", "right", "bottom", "left"} {
		m["border-"+side+"-width"] = field(func(c *style.Computed) *float32 { return sideOf(&c.Border, i) }, borderWidthOf,
			widths)
		m["border-"+side+"-style"] = prop{
			set: func(x *ctx, v []Component) {
				*sideOf(&x.c.BorderStyle, i) = borderStyles[strings.ToLower(v[0].Text)]
				*sideOf(&x.c.BorderHidden, i) = isWord(v[0], "hidden")
			},
			inherit: func(dst, src *style.Computed) {
				*sideOf(&dst.BorderStyle, i) = *sideOf(&src.BorderStyle, i)
				*sideOf(&dst.BorderHidden, i) = *sideOf(&src.BorderHidden, i)
			},
			anim: anim{kind: discrete},
		}
		m["border-"+side+"-color"] = currentField(style.BorderTopCurrent<<i,
			func(c *style.Computed) *color.Color { return sideOf(&c.BorderColor, i) }).painted()
	}
	noop := prop{set: func(*ctx, []Component) {}, inherit: func(_, _ *style.Computed) {}, anim: anim{kind: discrete}}
	m["border-image-outset"] = noop
	maps.Copy(m, contentProps())
	for i, n := range shorthands["transition"] {
		m[n] = prop{
			set:     func(x *ctx, v []Component) { x.trans[i] = v },
			inherit: func(dst, src *style.Computed) { dst.Transitions = src.Transitions },
			anim:    anim{kind: still},
		}
	}
	for i, n := range shorthands[shortAnimation] {
		m[n] = prop{
			set:     func(x *ctx, v []Component) { x.anims[i] = v },
			inherit: func(dst, src *style.Computed) { dst.Animations = src.Animations },
			anim:    anim{kind: still},
		}
	}
	m[propAnimationComposition] = prop{
		set:     func(x *ctx, v []Component) { x.composite = v },
		inherit: func(dst, src *style.Computed) { dst.Animations = src.Animations },
		anim:    anim{kind: still},
	}
	return m
}()

// followColor gives the shadows given as currentColor the element's colour c, in a copy
// when one changes: inherited shadows are the parent's slice.
func followColor(shadows []style.Shadow, c color.Color) []style.Shadow {
	var out []style.Shadow
	for i, sh := range shadows {
		if sh.CurrentColor && sh.Color != c {
			if out == nil {
				out = slices.Clone(shadows)
			}
			out[i].Color = c
		}
	}
	if out == nil {
		return shadows
	}
	return out
}

// followFilterColor gives the drop shadows given as currentColor the
// element's colour c, in a copy when one changes.
func followFilterColor(filters []style.Filter, c color.Color) []style.Filter {
	var out []style.Filter
	for i, f := range filters {
		if f.Kind == style.DropShadow && f.Shadow.CurrentColor && f.Shadow.Color != c {
			if out == nil {
				out = slices.Clone(filters)
			}
			out[i].Shadow.Color = c
		}
	}
	if out == nil {
		return filters
	}
	return out
}

// autoColor is a colour whose auto (and currentColor) follow the element's own
// color, inherited as the keyword.
func autoColor(get func(*style.Computed) (*color.Color, *bool)) prop {
	return prop{
		set: func(x *ctx, v []Component) {
			c, auto := get(x.c)
			*auto = isWord(v[0], "auto", kwCurrentColor)
			if !*auto {
				*c = linear(v[0], x.c.Color)
			}
		},
		inherit: func(dst, src *style.Computed) {
			dc, da := get(dst)
			sc, sa := get(src)
			*dc, *da = *sc, *sa
		},
		anim: anim{kind: interpolated,
			lerp: func(dst, a, b *style.Computed, t float32) bool {
				ca, aa := get(a)
				cb, ab := get(b)
				if *aa || *ab {
					return false
				}
				dc, da := get(dst)
				*dc, *da = lerpColor(*ca, *cb, t), false
				return true
			},
			equal: func(a, b *style.Computed) bool {
				ca, aa := get(a)
				cb, ab := get(b)
				return *aa == *ab && (*aa || *ca == *cb)
			},
		},
	}
}

// filled is border-image-slice's p that also reads its fill word, which
// flips with the slices.
func filled(p prop) prop {
	set, inherit := p.set, p.inherit
	p.set = func(x *ctx, v []Component) {
		set(x, v)
		x.c.BorderImage.Fill = slices.ContainsFunc(v, func(c Component) bool { return isWord(c, "fill") })
	}
	p.inherit = func(dst, src *style.Computed) {
		inherit(dst, src)
		dst.BorderImage.Fill = src.BorderImage.Fill
	}
	return p
}

// textFill is -webkit-text-fill-color: a colour, or currentColor, which
// leaves the glyphs to color, as the element's own, inherited or not.
func textFill() prop {
	return prop{
		set: func(x *ctx, v []Component) {
			if x.c.TextFilled = !isWord(v[0], kwCurrentColor); x.c.TextFilled {
				x.c.TextFill = linear(v[0], x.c.Color)
			}
		},
		inherit: func(dst, src *style.Computed) { dst.TextFill, dst.TextFilled = src.TextFill, src.TextFilled },
		anim: anim{kind: interpolated,
			lerp: func(dst, a, b *style.Computed, t float32) bool {
				dst.TextFill, dst.TextFilled = lerpColor(a.Fill(), b.Fill(), t), true
				return true
			},
			equal: func(a, b *style.Computed) bool {
				return a.TextFilled == b.TextFilled && (!a.TextFilled || a.TextFill == b.TextFill)
			},
		},
	}
}

func sideOf[T any](s *style.Sides[T], i int) *T {
	switch i {
	case 0:
		return &s.Top
	case 1:
		return &s.Right
	case 2:
		return &s.Bottom
	}
	return &s.Left
}

// imageSource is an image's url: kept as written when it has a scheme (ember:, data:),
// joined to the sheet's directory otherwise.
func (x *ctx) position(v []Component) style.BackgroundPosition {
	p, _ := parsePosition(v)
	return style.BackgroundPosition{X: x.length(p.x), Y: x.length(p.y), FromRight: p.right, FromBottom: p.bot}
}

func imageSource(x *ctx, v []Component) string {
	src := ""
	if v[0].Kind == URL {
		src = v[0].Text
	} else if a := items(v[0].Args); v[0].Kind == Function && len(a) > 0 {
		src = a[0].Text
	}
	if x == nil {
		return src
	}
	return x.base.resolve(src)
}

func (x *ctx) backgroundSize(v []Component) style.BackgroundSize {
	v = items(v)
	switch {
	case isWord(v[0], "cover"):
		return style.BackgroundSize{Scale: style.ScaleCover}
	case isWord(v[0], "contain"):
		return style.BackgroundSize{Scale: style.ScaleContain}
	}
	size := style.BackgroundSize{Width: x.length(v[0])}
	if len(v) > 1 {
		size.Height = x.length(v[1])
	}
	return size
}

// layered is a background longhand: the list of its layers' values, each
// read by each from its part of the value.
func layered[T any](get func(*style.Computed) *[]T, each func(x *ctx, v []Component) T, how interp[[]T]) prop {
	return field(get, func(x *ctx, v []Component) []T {
		parts := splitOn(items(v), func(c Component) bool { return c.Kind == Comma })
		out := make([]T, len(parts))
		for i, part := range parts {
			out[i] = each(x, part)
		}
		return out
	}, how)
}

func wordOf[T any](m map[string]T) func(*ctx, []Component) T {
	return func(_ *ctx, v []Component) T { return m[strings.ToLower(v[0].Text)] }
}

// maskBoxes are the boxes mask-origin and mask-clip name; the border box
// is both one's initial.
//
//nolint:gochecknoglobals // lookup table
var maskBoxes = map[string]style.BackgroundBox{
	"border-box": style.InitialBox, "padding-box": style.UnderPadding, "content-box": style.UnderContent,
	"fill-box": style.FillBox, "stroke-box": style.StrokeBox, "view-box": style.ViewBox, "no-clip": style.NoClip,
}

//nolint:gochecknoglobals // lookup table
var tilings = map[string]style.Tiling{
	kwRepeat: style.TileRepeat, "no-repeat": style.TileNoRepeat, "space": style.TileSpace, "round": style.TileRound,
}

// tiling is a layer of background-repeat, or mask-repeat: how the
// picture fills each axis.
func tiling(_ *ctx, v []Component) style.BackgroundRepeat {
	if len(v) == 2 {
		return style.BackgroundRepeat{X: tilings[strings.ToLower(v[0].Text)], Y: tilings[strings.ToLower(v[1].Text)]}
	}
	switch strings.ToLower(v[0].Text) {
	case "repeat-x":
		return style.BackgroundRepeat{X: style.TileRepeat, Y: style.TileNoRepeat}
	case "repeat-y":
		return style.BackgroundRepeat{X: style.TileNoRepeat, Y: style.TileRepeat}
	}
	t := tilings[strings.ToLower(v[0].Text)]
	return style.BackgroundRepeat{X: t, Y: t}
}

func gridLine(_ *ctx, v []Component) style.GridLine {
	var g style.GridLine
	span, n := false, 0
	for _, c := range v {
		switch {
		case isWord(c, "auto"):
		case isWord(c, "span"):
			span = true
		case c.Kind == Ident:
			g.Name = c.Text
		default:
			n = int(c.Num)
		}
	}
	if span {
		g.Span = max(n, 1)
	} else {
		g.Line = n
	}
	return g
}

//nolint:gochecknoglobals,goconst // lookup table
var fontSizes = map[string]float32{
	"xx-small": 9, "x-small": 10, "small": 13, "medium": 16, "large": 18, "x-large": 24, "xx-large": 32,
}

func (x *ctx) fontSize(v []Component) float32 {
	c := v[0]
	if c.Kind == Ident {
		switch w := strings.ToLower(c.Text); w {
		case "smaller":
			return x.parent.Font.Size / 1.2
		case "larger":
			return x.parent.Font.Size * 1.2
		default:
			return fontSizes[w]
		}
	}
	l := x.length(c)
	return l.Px + l.Percent*x.parent.Font.Size/100
}

func (x *ctx) fontWeight(v []Component) float32 {
	p := x.parent.Font.Weight
	switch {
	case isWord(v[0], "normal"):
		return 400
	case isWord(v[0], "bold"):
		return 700
	case isWord(v[0], "bolder"):
		switch {
		case p < 350:
			return 400
		case p < 550:
			return 700
		case p < 900:
			return 900
		}
		return p
	case isWord(v[0], "lighter"):
		switch {
		case p < 100:
			return p
		case p < 550:
			return 100
		case p < 750:
			return 400
		}
		return 700
	}
	return float32(math.Round(v[0].Num))
}

func (x *ctx) track(c Component) style.Track {
	if c.Kind == Function && strings.EqualFold(c.Text, "minmax") {
		args := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
		return style.Track{Min: x.length(args[0][0]), Max: x.length(args[1][0])}
	}
	l := x.length(c)
	return style.Track{Min: l, Max: l}
}

// templateTracks is grid-template-columns or -rows, whether subgrid
// beside its tracks: two track lists interpolate as trackLists, a
// subgrid flips.
func templateTracks(get func(*style.Computed) (*[]style.TrackRepeat, *bool)) prop {
	return prop{
		set: func(x *ctx, v []Component) {
			tracks, sub := get(x.c)
			*sub = isWord(v[0], kwSubgrid)
			if *sub {
				*tracks = subgridNames(v[1:])
				return
			}
			*tracks = x.tracks(v)
		},
		inherit: func(dst, src *style.Computed) {
			dt, ds := get(dst)
			st, ss := get(src)
			*dt, *ds = *st, *ss
		},
		anim: anim{kind: interpolated,
			lerp: func(dst, a, b *style.Computed, t float32) bool {
				at, as := get(a)
				bt, bs := get(b)
				if *as || *bs {
					return false
				}
				v, ok := trackLists.lerp(*at, *bt, t)
				if ok {
					dt, ds := get(dst)
					*dt, *ds = v, false
				}
				return ok
			},
			equal: func(a, b *style.Computed) bool {
				at, as := get(a)
				bt, bs := get(b)
				return *as == *bs && trackLists.equal(*at, *bt)
			},
		},
	}
}

// subgridNames are a subgrid's names of its lines, an entry a bracket or a
// repeat() of them.
func subgridNames(v []Component) []style.TrackRepeat {
	var out []style.TrackRepeat
	for _, c := range v {
		if c.Kind == LBracket {
			out = append(out, style.TrackRepeat{Count: 1, Names: [][]string{lineNames(c)}})
			continue
		}
		args := items(c.Args)
		r := style.TrackRepeat{Count: int(args[0].Num)}
		if isWord(args[0], "auto-fill") {
			r.Count = style.AutoFill
		}
		for _, n := range args[2:] {
			r.Names = append(r.Names, lineNames(n))
		}
		out = append(out, r)
	}
	return out
}

func (x *ctx) tracks(v []Component) []style.TrackRepeat {
	if isWord(v[0], kwNone) {
		return nil
	}
	var out []style.TrackRepeat
	var before []string
	for _, c := range v {
		switch {
		case c.Kind == LBracket:
			before = lineNames(c)
			continue
		case c.Kind != Function || !strings.EqualFold(c.Text, kwRepeat):
			out = append(out, style.TrackRepeat{Count: 1, Tracks: []style.Track{x.track(c)}, Before: before})
			before = nil
			continue
		}
		args := items(c.Args)
		r := style.TrackRepeat{Count: int(args[0].Num), Before: before}
		before = nil
		switch {
		case isWord(args[0], "auto-fill"):
			r.Count = style.AutoFill
		case isWord(args[0], "auto-fit"):
			r.Count = style.AutoFit
		}
		var names []string
		named := false
		for _, t := range args[2:] {
			if t.Kind == LBracket {
				names, named = lineNames(t), true
				continue
			}
			r.Names = append(r.Names, names)
			r.Tracks = append(r.Tracks, x.track(t))
			names = nil
		}
		if r.Names = append(r.Names, names); !named {
			r.Names = nil
		}
		out = append(out, r)
	}
	out[len(out)-1].After = before
	return out
}

// lineNames are the names in a bracket, nil for none.
func lineNames(c Component) []string {
	var out []string
	for _, n := range items(c.Args) {
		out = append(out, n.Text)
	}
	return out
}

func (x *ctx) autoTracks(v []Component) []style.Track {
	out := make([]style.Track, len(v))
	for i, c := range v {
		out[i] = x.track(c)
	}
	return out
}

func (x *ctx) shadows(v []Component) []style.Shadow {
	if isWord(v[0], kwNone) {
		return nil
	}
	var out []style.Shadow
	for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		s := style.Shadow{Color: x.c.Color, CurrentColor: true}
		var ls []float32
		for _, c := range part {
			switch {
			case isWord(c, kwInset):
				s.Inset = true
			case length(c) == nil:
				ls = append(ls, x.px(c))
			default:
				s.Color = x.color(c)
				s.CurrentColor = isWord(c, kwCurrentColor)
			}
		}
		ls = append(ls, 0, 0)
		s.X, s.Y, s.Blur, s.Spread = ls[0], ls[1], ls[2], ls[3]
		out = append(out, s)
	}
	return out
}

//nolint:gochecknoglobals // lookup table
var blendModes = map[string]style.BlendMode{
	"normal": style.BlendNormal, "multiply": style.BlendMultiply, "screen": style.BlendScreen,
	"overlay": style.BlendOverlay, "darken": style.BlendDarken, "lighten": style.BlendLighten,
	"color-dodge": style.BlendColorDodge, "color-burn": style.BlendColorBurn, "hard-light": style.BlendHardLight,
	"soft-light": style.BlendSoftLight, "difference": style.BlendDifference, "exclusion": style.BlendExclusion,
	"hue": style.BlendHue, "saturation": style.BlendSaturation, "color": style.BlendColor,
	"luminosity": style.BlendLuminosity,
}

//nolint:gochecknoglobals // lookup table
var filterKinds = map[string]style.FilterKind{
	"blur": style.Blur, "brightness": style.Brightness, "contrast": style.Contrast, "grayscale": style.Grayscale,
	"hue-rotate": style.HueRotate, "invert": style.Invert, "opacity": style.OpacityFilter, "saturate": style.Saturate,
	"sepia": style.Sepia, "drop-shadow": style.DropShadow,
}

// filters are filter's functions; grayscale, invert, opacity and sepia
// past 100% are 100%, as Chrome computes them.
func (x *ctx) filters(v []Component) []style.Filter {
	if isWord(v[0], kwNone) {
		return nil
	}
	out := make([]style.Filter, 0, len(v))
	for _, c := range v {
		f := style.Filter{Kind: filterKinds[strings.ToLower(c.Text)], Amount: 1}
		args := items(c.Args)
		switch f.Kind {
		case style.Blur, style.HueRotate:
			f.Amount = 0
			if len(args) > 0 && f.Kind == style.Blur {
				f.Amount = x.px(args[0])
			} else if len(args) > 0 {
				d, _ := angle(args[0])
				f.Amount = float32(d)
			}
		case style.DropShadow:
			f.Amount = 0
			f.Shadow = style.Shadow{Color: x.c.Color, CurrentColor: true}
			var ls []float32
			for _, a := range args {
				if length(a) == nil {
					ls = append(ls, x.px(a))
				} else {
					f.Shadow.Color, f.Shadow.CurrentColor = x.color(a), isWord(a, kwCurrentColor)
				}
			}
			ls = append(ls, 0)
			f.Shadow.X, f.Shadow.Y, f.Shadow.Blur = ls[0], ls[1], ls[2]
		case style.Grayscale, style.Invert, style.OpacityFilter, style.Sepia:
			if len(args) > 0 {
				f.Amount = min(x.num(args[0]), 1)
			}
		case style.Brightness, style.Contrast, style.Saturate:
			if len(args) > 0 {
				f.Amount = x.num(args[0])
			}
		}
		out = append(out, f)
	}
	return out
}

func (x *ctx) transform(v []Component) []style.TransformOp {
	if isWord(v[0], kwNone) {
		return nil
	}
	var out []style.TransformOp
	for _, c := range v {
		args := items(c.Args)
		var a []Component
		for _, arg := range args {
			if arg.Kind != Comma {
				a = append(a, arg)
			}
		}
		out = append(out, x.transformOp(strings.ToLower(c.Text), a))
	}
	return out
}

// transformOp is the transform function name with its arguments a.
func (x *ctx) transformOp(name string, a []Component) style.TransformOp {
	rad := func(i int) float32 { return float32(x.angle(a[i]) * math.Pi / 180) }
	var op style.TransformOp
	switch name {
	case fnTranslate, fnTranslateX, fnTranslateY:
		op.Kind = style.Translate
		op.X, op.Y = style.Px(0), style.Px(0)
		if name == fnTranslateY {
			op.Y = x.length(a[0])
		} else {
			op.X = x.length(a[0])
			if len(a) > 1 {
				op.Y = x.length(a[1])
			}
		}
	case fnScale, fnScaleX, fnScaleY:
		op.Kind = style.ScaleOp
		op.A[0], op.A[1] = 1, 1
		switch name {
		case fnScaleX:
			op.A[0] = x.num(a[0])
		case fnScaleY:
			op.A[1] = x.num(a[0])
		default:
			op.A[0], op.A[1] = x.num(a[0]), x.num(a[0])
			if len(a) > 1 {
				op.A[1] = x.num(a[1])
			}
		}
	case fnRotate:
		op.Kind = style.Rotate
		op.A[0] = rad(0)
	case fnSkew, fnSkewX, fnSkewY:
		op.Kind = style.Skew
		switch name {
		case fnSkewY:
			op.A[1] = rad(0)
		default:
			op.A[0] = rad(0)
			if len(a) > 1 {
				op.A[1] = rad(1)
			}
		}
	case fnMatrix:
		op.Kind = style.Matrix
		for i := range 6 {
			op.A[i] = float32(a[i].Num)
		}
	}
	return op
}

func (x *ctx) origin(v []Component) style.Size[style.Length] {
	pos := map[string]float32{kwLeft: 0, kwTop: 0, kwCenter: 50, kwRight: 100, kwBottom: 100}
	at := func(c Component) style.Length {
		if c.Kind == Ident {
			l := style.Percent(pos[strings.ToLower(c.Text)])
			l.IsPercent = true
			return l
		}
		return x.length(c)
	}
	half := style.Length{Kind: style.Fixed, Percent: 50, IsPercent: true}
	o := style.Size[style.Length]{Width: half, Height: half}
	if len(v) == 1 {
		if isWord(v[0], kwTop, kwBottom) {
			o.Height = at(v[0])
		} else {
			o.Width = at(v[0])
		}
		return o
	}
	x0, y0 := v[0], v[1]
	if isWord(x0, kwTop, kwBottom) || isWord(y0, kwLeft, kwRight) {
		x0, y0 = y0, x0
	}
	o.Width, o.Height = at(x0), at(y0)
	return o
}

// movedStates are the states of n that changed since css last styled it.
func movedStates(n *dom.Node) dom.State {
	data := dom.Data(n)
	if data.Dirty&dom.DirtyState == 0 {
		return 0
	}
	return n.State() ^ data.StyledState
}

// reaches adds to r.reach the subjects of the selectors that read one of
// the states moved on n, before a sibling combinator (beside) or above
// their subject, where n has what the compounds reading it name and no
// @media rule has the selector's rule off.
func (r *restyler) reaches(n *dom.Node, moved dom.State, beside bool) {
	for si, s := range r.sheets {
		var off []bool
		if r.m.off != nil {
			off = r.m.off[si]
		}
		x := s.indexed()
		for i := range x.reads {
			rd := &x.reads[i]
			if rd.states&moved != 0 && rd.beside == beside && (off == nil || !off[rd.rule]) && rd.names(n) {
				r.reach = append(r.reach, rd.subject)
			}
		}
	}
}

// anchor flags, before the tree under n is styled, the elements whose
// :has() a change may turn: round an element whose class, id, attribute
// or state some :has() reads changed, and round one whose children
// changed (flagAround). An element never styled is left with what is
// under it: it is styled whole, and its parent's children changed.
func (r *restyler) anchor(n *dom.Node) {
	data := dom.Data(n)
	if data.Style == nil {
		return
	}
	tree := data.Dirty&dom.DirtyTree != 0
	if tree && slices.Contains(data.Changed, dom.Children) {
		r.flagAround(n, true)
	}
	if movedStates(n)&r.hasStates != 0 || tree && slices.ContainsFunc(data.Changed, r.readInHas) {
		r.flagAround(n, false)
	}
	if data.Dirty&dom.DirtyDescendant == 0 {
		return
	}
	for _, c := range n.Children() {
		if !c.IsText() {
			r.anchor(c)
		}
	}
}

// flagAround flags for their :has() the ancestors of e, and when some
// :has() reads after its element, the elements before e and before each
// ancestor; when e's children changed (kids), e itself too, and its
// children.
func (r *restyler) flagAround(e *dom.Node, kids bool) {
	if kids {
		r.flag(e, false)
		if r.hasBeside {
			for _, c := range e.Children() {
				if !c.IsText() {
					r.flag(c, true)
				}
			}
		}
	}
	for x := e; x != nil; x = x.Parent() {
		if x != e {
			r.flag(x, false)
		}
		if r.hasBeside {
			for _, b := range elementsBefore(x) {
				if !b.IsText() {
					r.flag(b, true)
				}
			}
		}
	}
}

// flag has a restyled for its :has() when a has what one reading below
// it (beside unset), or after it, is on, in a rule no @media rule has
// off.
func (r *restyler) flag(a *dom.Node, beside bool) {
	for si, s := range r.sheets {
		var off []bool
		if r.m.off != nil {
			off = r.m.off[si]
		}
		x := s.indexed()
		for i := range x.has {
			h := &x.has[i]
			if h.beside == beside && (off == nil || !off[h.rule]) && h.names(a) {
				dom.Invalidate(a, ":has")
				return
			}
		}
	}
}

// readInHas reports whether some :has() reads the class, id or attribute
// a dom.Pipeline.Changed key names.
func (r *restyler) readInHas(key string) bool {
	return slices.ContainsFunc(r.sheets, func(s *Sheet) bool { return s.indexed().hasKeys[key] })
}

// reached reports whether n may be the subject of a selector in r.reach.
func (r *restyler) reached(n *dom.Node) bool {
	return slices.ContainsFunc(r.reach, func(c *Compound) bool { return c.names(n) })
}

// movesSiblings reports whether a class, an id or an attribute that
// changed on c, not restyled yet, is read before a sibling combinator.
func (r *restyler) movesSiblings(c *dom.Node) bool {
	if c.IsText() {
		return false
	}
	data := dom.Data(c)
	if data.Dirty&dom.DirtyTree == 0 {
		return false
	}
	for _, k := range data.Changed {
		for _, sh := range r.sheets {
			if sh.indexed().beside[k] {
				return true
			}
		}
	}
	return false
}

// readAbove reports whether some sheet reads one of changed in a compound
// above a selector's subject, or it is an attribute an element's
// descendants are hinted by.
func (r *restyler) readAbove(changed []string) bool {
	for _, k := range changed {
		for _, sh := range r.sheets {
			if sh.indexed().above[k] {
				return true
			}
		}
		if hintedBelow[k] {
			return true
		}
	}
	return false
}

// closedSlot reports whether n is held by a closed details other than as
// its first summary: Chrome shows none of that, whatever its display.
func closedSlot(n *dom.Node) bool {
	p := n.Parent()
	if p == nil || p.Tag() != dom.Details {
		return false
	}
	if _, open := p.Attr("open"); open || p.State().Has(dom.Open) {
		return false
	}
	for _, c := range p.Children() {
		if !c.IsText() && c.Tag() == dom.Summary {
			return c != n
		}
	}
	return true
}

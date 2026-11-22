package css

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"sync"

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
// corners are 0, and the transform origin and mask position are
// percentages.
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
	c.MaskPosition = style.TopLeft
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
	// overlay has scrolling boxes keep no gutter for their bars
	// (Transitions.OverlayScrollbars).
	overlay bool
	// viewport has the root's overflow, or body's, go to the viewport
	// (Transitions.ScrollingViewport).
	viewport bool
	// browser reads style attributes as a browser does
	// (Transitions.BrowserSheets).
	browser bool
	// before, after and marker are set when some sheet has a rule for the
	// pseudo-element: only then is one looked for.
	before, after, marker bool
	// counts is set once an element counts: a numbered list item, a
	// counter property, or generated content of counters or quotes; and
	// recount when what the counters read may have changed.
	counts, recount bool
}

// run restyles the tree under root, all of it when stale, then counts
// its counters again where they may have moved.
func (r *restyler) run(root *dom.Node, stale bool) {
	for _, s := range r.sheets {
		x := s.indexed()
		r.before = r.before || x.pseudos["before"] != nil
		r.after = r.after || x.pseudos["after"] != nil
		r.marker = r.marker || x.pseudos["marker"] != nil
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
		r.env.used = 0
	} else if r.anchors && dom.Data(root).Style != nil {
		r.anchor(root)
	}
	r.visit(root, nil, 0, stale, stale)
	r.count(root)
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
	c.TextShadows, c.CurrentColor, c.Vars, c.Backdrop = nil, 0, nil, color.Color{}
	c.TextDecoration, c.Decorations = style.TextDecoration{}, nil
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
	trans              [4][]Component
	anims              [8][]Component
	// over is what the element's animations set on its style.
	over []override
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
}

// leading are the longhands the others read, in the order they are
// applied before them: the font's family, style and weight, which its
// own units measure, then font-size (em), line-height (lh) and color
// (currentcolor).
//
//nolint:gochecknoglobals,goconst // a fixed list
var leading = []string{"font-family", "font-style", "font-weight", propFontSize, "line-height", "color"}

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
	name               string
	prop               prop
	inherited, current bool
}

// longhands are order's, in its order.
//
//nolint:gochecknoglobals // derived from the properties
var longhands = func() []longhand {
	out := make([]longhand, len(order))
	for i, name := range order {
		p := properties[name]
		out[i] = longhand{name: name, prop: props[name], inherited: p.inherited, current: p.initial == kwCurrentColor}
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
	casc := r.m.cascade(r.m.matched(n, r.sheets, inline))
	r.overKeyframe(casc)
	root := parent == nil
	if root {
		parent = &initial
		rem = initial.Font.Size
	}
	c := initial
	x := &ctx{c: &c, parent: parent, em: parent.Font.Size, rem: rem, env: r.env, root: root, overlay: r.overlay,
		toViewport: r.viewport && (root || rootBody(n)), node: n}
	if r.over != nil {
		x.over = r.over(n)
	}
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
	if r.may(n, "selection") {
		r.selection(x, n)
	}
	if n.Tag() == dom.Dialog && r.may(n, "backdrop") {
		if bd, ok := x.pseudo(r.pm, n, r.sheets, "backdrop"); ok {
			c.Backdrop = fadeColor(bd.Background, bd.Opacity)
		}
	}
	if c.ListItem && c.ListStyle.Numeric() || len(c.CounterReset)+len(c.CounterIncrement)+len(c.CounterSet) > 0 {
		r.counts = true
	}
	return &c
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
}

// cascaded applies every longhand's winner, then what depends on several.
func (x *ctx) cascaded(casc cascaded, root bool) {
	x.customs(casc.custom)
	for i := range longhands {
		l := &longhands[i]
		x.apply(l, casc.longhands[i])
		switch l.name {
		case propFontSize:
			x.em, x.sized = x.c.Font.Size, true
			if root {
				x.rem = x.c.Font.Size
			}
		case "line-height":
			x.lined = true
		}
	}
	for _, o := range x.over {
		o.apply(x.c)
	}
	x.finish(root)
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
	casc := m.cascadeFor(m.matchedFor(n, sheets, nil, name), name)
	if !slices.ContainsFunc(casc.longhands, func(d *Declaration) bool { return d != nil }) && len(casc.custom) == 0 {
		return nil, false
	}
	ps := *x.c
	ps.Opacity, ps.Content, ps.Background = 1, style.Content{}, color.Color{}
	px := &ctx{c: &ps, parent: x.c, em: x.em, rem: x.rem, env: x.env, node: n}
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

// apply computes one longhand from its winning declaration, or from
// inheritance or its initial value when it has none.
func (x *ctx) apply(l *longhand, d *Declaration) {
	name, p := l.name, l.prop
	if d == nil {
		if l.inherited {
			p.inherit(x.c, x.parent)
			x.c.ColorInherited = x.c.ColorInherited || name == "color"
		} else {
			x.reset(l)
		}
		return
	}
	x.base = d.base
	var v []Component
	ok := true
	switch {
	case d.longhands != nil:
		v = d.longhands[name]
	case d.Name != name:
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
		v, ok = long[name], long != nil
	case d.vals != nil:
		v = d.vals
	default:
		v, ok = x.substitute(d.Value)
	}
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
		p.inherit(x.c, x.parent)
	case "initial":
		x.reset(l)
	case kwUnset:
		if l.inherited {
			p.inherit(x.c, x.parent)
		} else {
			x.reset(l)
		}
	default:
		p.set(x, v)
	}
}

// reset gives a longhand its initial value; currentcolor is the
// element's colour.
func (x *ctx) reset(l *longhand) {
	if l.current {
		l.prop.set(x, currentColor)
		return
	}
	l.prop.inherit(x.c, &initial)
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
	switch x.parent.Display {
	case style.FlexDisplay, style.InlineFlex, style.GridDisplay, style.InlineGrid:
		if !root {
			c.Display, c.TablePart = blockified(c.Display), style.NoTablePart
			c.Float = style.FloatNone
		}
	case style.Inline, style.Block, style.InlineBlock, style.DisplayNone, style.Contents:
		// not a flex or grid container
	}
	if root || c.Position == style.Absolute || c.Position == style.FixedPosition {
		c.Display, c.TablePart = blockified(c.Display), style.NoTablePart
		c.Float = style.FloatNone
	}
	if c.Float != style.FloatNone {
		c.Display, c.TablePart = blockified(c.Display), style.NoTablePart
	}
	x.transitions()
	x.animations()
	c.FollowColor(c.Color)
	c.TextShadows = followColor(c.TextShadows, c.Color)
	c.Decorations = x.decorations()
	c.Shadows = followColor(c.Shadows, c.Color)
	if c.AccentAuto {
		c.AccentColor = c.Color
	}
	if c.CaretAuto {
		c.CaretColor = c.Color
	}
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
	if x.trans[0] == nil && x.trans[1] == nil && x.trans[2] == nil && x.trans[3] == nil {
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
	var out []style.Animation
	for i, n := range lists[7] {
		if isWord(n[0], kwNone) {
			continue
		}
		a := style.Animation{Name: n[0].Text, Easing: at(1, i).String(), Iterations: float32(math.Inf(1))}
		dur, _ := duration(at(0, i))
		delay, _ := duration(at(2, i))
		a.Duration, a.Delay = float32(dur), float32(delay)
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
		out = append(out, a)
	}
	x.c.Animations = out
}

// transitionList is the transition list from the transition longhands'
// values, unset ones initial.
func transitionList(trans [4][]Component) []style.Transition {
	lists := make([][][]Component, 4)
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
		t.Duration = float32(dur)
		t.Easing = lists[2][i%len(lists[2])][0].String()
		delay, _ := duration(lists[3][i%len(lists[3])][0])
		t.Delay = float32(delay)
		out = append(out, t)
	}
	return out
}

// initialTransitions is the transition list of an element that sets
// none, worked out once and shared.
//
//nolint:gochecknoglobals // worked out once
var initialTransitions = sync.OnceValue(func() []style.Transition {
	return transitionList([4][]Component{})
})

// prop is how one longhand is set from its value and copied when
// inherited.
type prop struct {
	set     func(x *ctx, v []Component)
	inherit func(dst, src *style.Computed)
}

func field[T any](get func(*style.Computed) *T, set func(x *ctx, v []Component) T) prop {
	return prop{
		set:     func(x *ctx, v []Component) { *get(x.c) = set(x, v) },
		inherit: func(dst, src *style.Computed) { *get(dst) = *get(src) },
	}
}

func keyword[T any](get func(*style.Computed) *T, m map[string]T) prop {
	return field(get, func(_ *ctx, v []Component) T { return m[strings.ToLower(v[0].Text)] })
}

func lengthField(get func(*style.Computed) *style.Length) prop {
	return field(get, func(x *ctx, v []Component) style.Length { return x.length(v[0]) })
}

func colorField(get func(*style.Computed) *color.Color) prop {
	return field(get, func(x *ctx, v []Component) color.Color { return x.color(v[0]) })
}

// currentField is a colour that may be currentColor, marked by f.
func currentField(f style.CurrentColorFields, get func(*style.Computed) *color.Color) prop {
	p := colorField(get)
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
	l := style.Length{Kind: style.Fixed, Px: float32(d.Px), Percent: float32(d.Pct), IsPercent: d.HasPercent()}
	if len(d.terms) > 0 {
		l.Px, l.Percent = float32(d.Resolve(0)), 0
		l.Calc, l.IsPercent = calcOf(d)
	}
	return l
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

func (x *ctx) px(c Component) float32 {
	return x.length(c).Px
}

func (x *ctx) color(c Component) color.Color {
	if isWord(c, kwCurrentColor) {
		return x.c.Color
	}
	return linear(c)
}

func linear(c Component) color.Color {
	s, _ := parseColor(c)
	l := color.FromSRGB([3]float32{float32(s.R), float32(s.G), float32(s.B)})
	l.A = float32(s.A)
	return l
}

func num(c Component) float32 {
	d, _ := parseDim(c)
	if d.HasPercent() {
		return float32(d.Pct / 100)
	}
	return float32(d.Num)
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
		"groove": style.BorderGroove, "ridge": style.BorderRidge,
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

func sideProps(prefix, suffix string, mk func(get func(*style.Computed) *style.Length) prop,
	get func(*style.Computed) *style.Sides[style.Length]) map[string]prop {
	return map[string]prop{
		prefix + kwTop + suffix:    mk(func(c *style.Computed) *style.Length { return &get(c).Top }),
		prefix + kwRight + suffix:  mk(func(c *style.Computed) *style.Length { return &get(c).Right }),
		prefix + kwBottom + suffix: mk(func(c *style.Computed) *style.Length { return &get(c).Bottom }),
		prefix + kwLeft + suffix:   mk(func(c *style.Computed) *style.Length { return &get(c).Left }),
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
			},
			inherit: func(dst, src *style.Computed) {
				dst.Display, dst.ListItem, dst.FlowRoot, dst.Table = src.Display, src.ListItem, src.FlowRoot, src.Table
				dst.TablePart = src.TablePart
			},
		},
		"float": keyword(func(c *style.Computed) *style.Float { return &c.Float }, map[string]style.Float{
			"none": style.FloatNone, "left": style.FloatLeft, "right": style.FloatRight,
			"inline-start": style.FloatLeft, "inline-end": style.FloatRight,
		}),
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
		},
		"clear": keyword(func(c *style.Computed) *style.Clear { return &c.Clear }, map[string]style.Clear{
			"none": style.ClearNone, "left": style.ClearLeft, "right": style.ClearRight, "both": style.ClearBoth,
			"inline-start": style.ClearLeft, "inline-end": style.ClearRight,
		}),
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
			}),
		"box-sizing": keyword(func(c *style.Computed) *style.BoxSizing { return &c.BoxSizing },
			map[string]style.BoxSizing{"border-box": style.BorderBox, "content-box": style.ContentBox}),
		"position": keyword(func(c *style.Computed) *style.Position { return &c.Position }, map[string]style.Position{
			"static": style.Static, "relative": style.Relative, "absolute": style.Absolute,
			"fixed": style.FixedPosition, "sticky": style.Sticky,
		}),
		"width":      lengthField(func(c *style.Computed) *style.Length { return &c.Size.Width }),
		"height":     lengthField(func(c *style.Computed) *style.Length { return &c.Size.Height }),
		"min-width":  lengthField(func(c *style.Computed) *style.Length { return &c.MinSize.Width }),
		"min-height": lengthField(func(c *style.Computed) *style.Length { return &c.MinSize.Height }),
		"max-width":  lengthField(func(c *style.Computed) *style.Length { return &c.MaxSize.Width }),
		"max-height": lengthField(func(c *style.Computed) *style.Length { return &c.MaxSize.Height }),
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
			},
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
		},

		"flex-direction": keyword(func(c *style.Computed) *style.FlexDirection { return &c.Direction },
			map[string]style.FlexDirection{"row": style.Row, "row-reverse": style.RowReverse,
				"column": style.Column, "column-reverse": style.ColumnReverse}),
		"flex-wrap": keyword(func(c *style.Computed) *style.FlexWrap { return &c.Wrap }, map[string]style.FlexWrap{
			"nowrap": style.NoWrap, "wrap": style.Wrap, "wrap-reverse": style.WrapReverse,
		}),
		"flex-grow": field(func(c *style.Computed) *float32 { return &c.Grow },
			func(_ *ctx, v []Component) float32 { return num(v[0]) }),
		"flex-shrink": field(func(c *style.Computed) *float32 { return &c.Shrink },
			func(_ *ctx, v []Component) float32 { return num(v[0]) }),
		"flex-basis":      lengthField(func(c *style.Computed) *style.Length { return &c.Basis }),
		"row-gap":         lengthField(func(c *style.Computed) *style.Length { return &c.Gap.Height }),
		"column-gap":      lengthField(func(c *style.Computed) *style.Length { return &c.Gap.Width }),
		"align-items":     keyword(func(c *style.Computed) *style.Align { return &c.AlignItems }, alignWords4),
		"align-self":      keyword(func(c *style.Computed) *style.Align { return &c.AlignSelf }, alignWords4),
		"align-content":   keyword(func(c *style.Computed) *style.Align { return &c.AlignContent }, alignWords4),
		"justify-content": keyword(func(c *style.Computed) *style.Align { return &c.JustifyContent }, alignWords4),
		"justify-items":   keyword(func(c *style.Computed) *style.Align { return &c.JustifyItems }, alignWords4),
		"justify-self":    keyword(func(c *style.Computed) *style.Align { return &c.JustifySelf }, alignWords4),
		"order": field(func(c *style.Computed) *int { return &c.Order },
			func(_ *ctx, v []Component) int { return int(v[0].Num) }),

		"column-count": field(func(c *style.Computed) *int { return &c.ColumnCount },
			func(_ *ctx, v []Component) int { return int(v[0].Num) }),
		"column-width": lengthField(func(c *style.Computed) *style.Length { return &c.ColumnWidth }),
		"column-fill": field(func(c *style.Computed) *bool { return &c.ColumnFillAuto },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "auto") }),
		"column-span":       {set: func(*ctx, []Component) {}, inherit: func(_, _ *style.Computed) {}},
		"column-rule-width": field(func(c *style.Computed) *float32 { return &c.ColumnRuleWidth }, borderWidthOf),
		"column-rule-style": keyword(func(c *style.Computed) *style.BorderStyle { return &c.ColumnRuleStyle }, borderStyles),
		"column-rule-color": currentField(style.ColumnRuleCurrent,
			func(c *style.Computed) *color.Color { return &c.ColumnRuleColor }),
		"break-inside": field(func(c *style.Computed) *bool { return &c.BreakInsideAvoid },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "avoid", "avoid-column") }),
		"break-before": keyword(func(c *style.Computed) *style.Break { return &c.BreakBefore }, breaks),
		"break-after":  keyword(func(c *style.Computed) *style.Break { return &c.BreakAfter }, breaks),
		"orphans": field(func(c *style.Computed) *int { return &c.Orphans },
			func(_ *ctx, v []Component) int { return int(v[0].Num) }),
		"widows": field(func(c *style.Computed) *int { return &c.Widows },
			func(_ *ctx, v []Component) int { return int(v[0].Num) }),

		"grid-template-columns": field(func(c *style.Computed) *[]style.TrackRepeat { return &c.Columns }, (*ctx).tracks),
		"grid-template-rows":    field(func(c *style.Computed) *[]style.TrackRepeat { return &c.Rows }, (*ctx).tracks),
		"grid-template-areas":   field(func(c *style.Computed) *[]style.GridArea { return &c.Areas }, templateAreas),
		"grid-auto-columns":     field(func(c *style.Computed) *[]style.Track { return &c.AutoColumns }, (*ctx).autoTracks),
		"grid-auto-rows":        field(func(c *style.Computed) *[]style.Track { return &c.AutoRows }, (*ctx).autoTracks),
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
			},
		),
		"grid-column-start": field(func(c *style.Computed) *style.GridLine { return &c.Column[0] }, gridLine),
		"grid-column-end":   field(func(c *style.Computed) *style.GridLine { return &c.Column[1] }, gridLine),
		"grid-row-start":    field(func(c *style.Computed) *style.GridLine { return &c.Row[0] }, gridLine),
		"grid-row-end":      field(func(c *style.Computed) *style.GridLine { return &c.Row[1] }, gridLine),

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
			},
		),
		"font-size":   field(func(c *style.Computed) *float32 { return &c.Font.Size }, (*ctx).fontSize),
		"font-weight": field(func(c *style.Computed) *int { return &c.Font.Weight }, (*ctx).fontWeight),
		"font-style": field(func(c *style.Computed) *bool { return &c.Font.Italic }, func(_ *ctx, v []Component) bool {
			return !isWord(v[0], "normal")
		}),
		"line-height": field(
			func(c *style.Computed) *style.LineHeight { return &c.LineHeight },
			func(x *ctx, v []Component) style.LineHeight {
				switch {
				case isWord(v[0], "normal"):
					return style.LineHeight{Normal: true}
				case v[0].Kind == Number:
					return style.LineHeight{Factor: float32(v[0].Num)}
				case v[0].Kind == Function:
					if d, _ := parseDim(v[0]); d.IsNumber() {
						return style.LineHeight{Factor: float32(d.Num)}
					}
				}
				l := x.length(v[0])
				return style.LineHeight{Px: l.Px + l.Percent*x.em/100}
			},
		),
		"letter-spacing": field(func(c *style.Computed) *float32 { return &c.LetterSpacing },
			func(x *ctx, v []Component) float32 { return x.px(v[0]) }),
		"word-spacing": field(func(c *style.Computed) *style.WordSpacing { return &c.WordSpacing },
			func(x *ctx, v []Component) style.WordSpacing {
				l := x.length(v[0])
				return style.WordSpacing{Px: l.Px, Percent: l.Percent}
			}),
		"color": field(
			func(c *style.Computed) *color.Color { return &c.Color },
			func(x *ctx, v []Component) color.Color {
				if isWord(v[0], "currentcolor") {
					return x.parent.Color
				}
				return linear(v[0])
			},
		),
		"-webkit-font-smoothing": {
			set: func(x *ctx, v []Component) {
				kw := strings.ToLower(v[0].Text)
				x.c.PixelText, x.c.ExactText = kw == kwNone, kw == "antialiased"
			},
			inherit: func(dst, src *style.Computed) { dst.PixelText, dst.ExactText = src.PixelText, src.ExactText },
		},
		"text-transform": keyword(func(c *style.Computed) *style.TextTransform { return &c.TextTransform },
			map[string]style.TextTransform{"none": style.NoTransform, "uppercase": style.Uppercase,
				"lowercase": style.Lowercase, "capitalize": style.Capitalize}),
		"text-align": keyword(func(c *style.Computed) *style.TextAlign { return &c.TextAlign },
			map[string]style.TextAlign{"start": style.Start, "end": style.End, "left": style.Left, "right": style.Right,
				"center": style.Center, "justify": style.Justify, "-webkit-left": style.BlockLeft,
				"-webkit-right": style.BlockRight, "-webkit-center": style.BlockCenter}),
		"text-indent": lengthField(func(c *style.Computed) *style.Length { return &c.TextIndent }),
		"caption-side": keyword(func(c *style.Computed) *style.CaptionSide { return &c.CaptionSide },
			map[string]style.CaptionSide{"top": style.CaptionTop, "bottom": style.CaptionBottom}),
		"border-collapse": field(func(c *style.Computed) *bool { return &c.BorderCollapse },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "collapse") }),
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
		},
		"overflow-wrap": {
			set: func(x *ctx, v []Component) {
				x.c.BreakWord = !isWord(v[0], "normal")
				x.c.BreakAnywhere = isWord(v[0], "anywhere")
			},
			inherit: func(dst, src *style.Computed) { dst.BreakWord, dst.BreakAnywhere = src.BreakWord, src.BreakAnywhere },
		},
		"scrollbar-width": field(func(c *style.Computed) *bool { return &c.ScrollbarNone },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "none") }),
		"text-overflow": field(func(c *style.Computed) *bool { return &c.Ellipsis }, func(_ *ctx, v []Component) bool {
			return isWord(v[0], "ellipsis")
		}),
		"list-style-type": keyword(func(c *style.Computed) *style.ListStyle { return &c.ListStyle }, listStyles),
		"visibility": field(func(c *style.Computed) *bool { return &c.Visible }, func(_ *ctx, v []Component) bool {
			return isWord(v[0], "visible")
		}),
		"pointer-events": field(func(c *style.Computed) *bool { return &c.PointerNone },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "none") }),
		"app-region": field(func(c *style.Computed) *bool { return &c.AppDrag },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "drag") }),
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
			}),
		"cursor": field(func(c *style.Computed) *string { return &c.Cursor }, func(_ *ctx, v []Component) string {
			if isWord(v[0], "auto") {
				return ""
			}
			return strings.ToLower(v[0].Text)
		}),

		"background-color": colorField(func(c *style.Computed) *color.Color { return &c.Background }),
		"background-image": layered(func(c *style.Computed) *[]style.Image { return &c.BackgroundImages },
			backgroundImage),
		"background-size": layered(func(c *style.Computed) *[]style.BackgroundSize { return &c.BackgroundSizes },
			(*ctx).backgroundSize),
		"background-repeat": layered(func(c *style.Computed) *[]style.BackgroundRepeat { return &c.BackgroundRepeats },
			tiling),
		"background-position": layered(
			func(c *style.Computed) *[]style.BackgroundPosition { return &c.BackgroundPositions }, (*ctx).position),
		"background-attachment": layered(
			func(c *style.Computed) *[]style.BackgroundAttachment { return &c.BackgroundAttachments },
			wordOf(map[string]style.BackgroundAttachment{
				"scroll": style.AttachScroll, "fixed": style.AttachFixed, "local": style.AttachLocal,
			})),
		"background-origin": layered(func(c *style.Computed) *[]style.BackgroundBox { return &c.BackgroundOrigins },
			wordOf(map[string]style.BackgroundBox{
				"padding-box": style.InitialBox, "border-box": style.UnderBorder, "content-box": style.UnderContent,
			})),
		"background-clip": layered(func(c *style.Computed) *[]style.BackgroundBox { return &c.BackgroundClips },
			wordOf(map[string]style.BackgroundBox{
				"border-box": style.InitialBox, "padding-box": style.UnderPadding, "content-box": style.UnderContent,
			})),
		"mask-image": field(func(c *style.Computed) *style.Image { return &c.MaskImage },
			func(x *ctx, v []Component) style.Image { return style.Image{Source: imageSource(x, v)} }),
		"mask-size":   field(func(c *style.Computed) *style.BackgroundSize { return &c.MaskSize }, (*ctx).backgroundSize),
		"mask-repeat": field(func(c *style.Computed) *style.BackgroundRepeat { return &c.MaskRepeat }, tiling),
		"mask-position": field(func(c *style.Computed) *style.BackgroundPosition { return &c.MaskPosition },
			(*ctx).position),
		"text-shadow": field(func(c *style.Computed) *[]style.Shadow { return &c.TextShadows }, (*ctx).shadows),
		"text-decoration-line": field(func(c *style.Computed) *style.DecorationLine { return &c.TextDecoration.Line },
			decorationLine),
		"text-decoration-style": keyword(func(c *style.Computed) *style.DecorationStyle { return &c.TextDecoration.Style },
			decorationStyles),
		"text-decoration-color": currentField(style.DecorationCurrent,
			func(c *style.Computed) *color.Color { return &c.TextDecoration.Color }),
		"text-decoration-thickness": lengthField(
			func(c *style.Computed) *style.Length { return &c.TextDecoration.Thickness }),
		"image-rendering": keyword(func(c *style.Computed) *style.ImageRendering { return &c.ImageRendering },
			map[string]style.ImageRendering{"auto": style.RenderAuto, "pixelated": style.RenderPixelated}),
		"accent-color": autoColor(func(c *style.Computed) (*color.Color, *bool) { return &c.AccentColor, &c.AccentAuto }),
		"caret-color":  autoColor(func(c *style.Computed) (*color.Color, *bool) { return &c.CaretColor, &c.CaretAuto }),
		"object-position": field(func(c *style.Computed) *style.BackgroundPosition { return &c.ObjectPosition },
			(*ctx).position),
		"object-fit": keyword(func(c *style.Computed) *style.ObjectFit { return &c.ObjectFit },
			map[string]style.ObjectFit{"fill": style.FitFill, "contain": style.FitContain, "cover": style.FitCover,
				"none": style.FitNone, "scale-down": style.FitScaleDown}),
		"border-image-source": field(func(c *style.Computed) *string { return &c.BorderImage.Source }, imageSource),
		"border-image-slice": field(
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
			},
		),
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
			},
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
			},
		),
		"box-shadow":    field(func(c *style.Computed) *[]style.Shadow { return &c.Shadows }, (*ctx).shadows),
		"outline-width": field(func(c *style.Computed) *float32 { return &c.OutlineWidth }, borderWidthOf),
		"outline-style": keyword(func(c *style.Computed) *style.BorderStyle { return &c.OutlineStyle }, borderStyles),
		"outline-color": currentField(style.OutlineCurrent, func(c *style.Computed) *color.Color { return &c.OutlineColor }),
		"outline-offset": field(func(c *style.Computed) *float32 { return &c.OutlineOffset },
			func(x *ctx, v []Component) float32 { return x.px(v[0]) }),
		"opacity": field(func(c *style.Computed) *float32 { return &c.Opacity }, func(_ *ctx, v []Component) float32 {
			return float32(clamp01(float64(num(v[0]))))
		}),
		"transform": field(func(c *style.Computed) *[]style.TransformOp { return &c.Transform }, (*ctx).transform),
		"transform-origin": field(func(c *style.Computed) *style.Size[style.Length] { return &c.TransformOrigin },
			(*ctx).origin),
		"appearance": {set: func(*ctx, []Component) {}, inherit: func(_, _ *style.Computed) {}},
		"resize":     {set: func(*ctx, []Component) {}, inherit: func(_, _ *style.Computed) {}},

		"border-top-left-radius": field(func(c *style.Computed) *style.Size[style.Length] { return &c.Radius.TopLeft },
			cornerOf),
		"border-top-right-radius": field(func(c *style.Computed) *style.Size[style.Length] { return &c.Radius.TopRight },
			cornerOf),
		"border-bottom-right-radius": field(
			func(c *style.Computed) *style.Size[style.Length] { return &c.Radius.BottomRight },
			cornerOf,
		),
		"border-bottom-left-radius": field(func(c *style.Computed) *style.Size[style.Length] { return &c.Radius.BottomLeft },
			cornerOf),
	}
	boxes := map[string]func(c *style.Computed) *style.Sides[style.Length]{
		"margin-":  func(c *style.Computed) *style.Sides[style.Length] { return &c.Margin },
		"padding-": func(c *style.Computed) *style.Sides[style.Length] { return &c.Padding },
		"":         func(c *style.Computed) *style.Sides[style.Length] { return &c.Inset },
	}
	for prefix, get := range boxes {
		maps.Copy(m, sideProps(prefix, "", lengthField, get))
	}
	for i, side := range []string{"top", "right", "bottom", "left"} {
		m["border-"+side+"-width"] = field(func(c *style.Computed) *float32 { return sideOf(&c.Border, i) }, borderWidthOf)
		m["border-"+side+"-style"] = prop{
			set: func(x *ctx, v []Component) {
				*sideOf(&x.c.BorderStyle, i) = borderStyles[strings.ToLower(v[0].Text)]
				*sideOf(&x.c.BorderHidden, i) = isWord(v[0], "hidden")
			},
			inherit: func(dst, src *style.Computed) {
				*sideOf(&dst.BorderStyle, i) = *sideOf(&src.BorderStyle, i)
				*sideOf(&dst.BorderHidden, i) = *sideOf(&src.BorderHidden, i)
			},
		}
		m["border-"+side+"-color"] = currentField(style.BorderTopCurrent<<i,
			func(c *style.Computed) *color.Color { return sideOf(&c.BorderColor, i) })
	}
	noop := prop{set: func(*ctx, []Component) {}, inherit: func(_, _ *style.Computed) {}}
	m["border-image-outset"] = noop
	maps.Copy(m, contentProps())
	for i, n := range shorthands["transition"] {
		m[n] = prop{
			set:     func(x *ctx, v []Component) { x.trans[i] = v },
			inherit: func(dst, src *style.Computed) { dst.Transitions = src.Transitions },
		}
	}
	for i, n := range shorthands[shortAnimation] {
		m[n] = prop{
			set:     func(x *ctx, v []Component) { x.anims[i] = v },
			inherit: func(dst, src *style.Computed) { dst.Animations = src.Animations },
		}
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

// autoColor is a colour whose auto (and currentColor) follow the element's own
// color, inherited as the keyword.
func autoColor(get func(*style.Computed) (*color.Color, *bool)) prop {
	return prop{
		set: func(x *ctx, v []Component) {
			c, auto := get(x.c)
			*auto = isWord(v[0], "auto", kwCurrentColor)
			if !*auto {
				*c = linear(v[0])
			}
		},
		inherit: func(dst, src *style.Computed) {
			dc, da := get(dst)
			sc, sa := get(src)
			*dc, *da = *sc, *sa
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
func layered[T any](get func(*style.Computed) *[]T, each func(x *ctx, v []Component) T) prop {
	return field(get, func(x *ctx, v []Component) []T {
		parts := splitOn(items(v), func(c Component) bool { return c.Kind == Comma })
		out := make([]T, len(parts))
		for i, part := range parts {
			out[i] = each(x, part)
		}
		return out
	})
}

func wordOf[T any](m map[string]T) func(*ctx, []Component) T {
	return func(_ *ctx, v []Component) T { return m[strings.ToLower(v[0].Text)] }
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

func (x *ctx) fontWeight(v []Component) int {
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
	return int(math.Round(v[0].Num))
}

func (x *ctx) track(c Component) style.Track {
	if c.Kind == Function && strings.EqualFold(c.Text, "minmax") {
		args := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
		return style.Track{Min: x.length(args[0][0]), Max: x.length(args[1][0])}
	}
	l := x.length(c)
	return style.Track{Min: l, Max: l}
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
	rad := func(i int) float32 { d, _ := angle(a[i]); return float32(d * math.Pi / 180) }
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
			op.A[0] = num(a[0])
		case fnScaleY:
			op.A[1] = num(a[0])
		default:
			op.A[0], op.A[1] = num(a[0]), num(a[0])
			if len(a) > 1 {
				op.A[1] = num(a[1])
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

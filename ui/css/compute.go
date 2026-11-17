package css

import (
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"strconv"
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
// outline-width is medium whatever outline-style says, as in Chrome,
// corners are 0, and the transform origin and background position are
// percentages.
func initialStyle() style.Computed {
	c := style.Initial()
	c.OutlineWidth = 3
	c.ColumnRuleWidth = 3
	zero := style.Size[style.Length]{Width: style.Px(0), Height: style.Px(0)}
	c.Radius = style.Corners[style.Size[style.Length]]{TopLeft: zero, TopRight: zero, BottomRight: zero,
		BottomLeft: zero}
	half := style.Length{Kind: style.Fixed, Percent: 50, IsPercent: true}
	c.TransformOrigin = style.Size[style.Length]{Width: half, Height: half}
	c.AccentAuto, c.CaretAuto = true, true
	none := style.Length{Kind: style.Fixed, IsPercent: true}
	c.BackgroundPosition = style.BackgroundPosition{X: none, Y: none}
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
	m := &matcher{off: mediaOff(all, Media{}), ranks: layerRanks(all)}
	pm := &matcher{off: m.off, ranks: m.ranks}
	r := restyler{sheets: all, m: m, pm: pm, treeStates: treeStates(all), computed: new(int)}
	r.run(root, false)
	return r.warns
}

type restyler struct {
	sheets []*Sheet
	// m matches and cascades elements, pm their pseudo-elements.
	m, pm *matcher
	// treeStates are the states some selector reads above its subject:
	// only a change of those restyles an element's descendants.
	treeStates dom.State
	// computed counts the elements whose style was worked out.
	computed *int
	// la and lb are scratch styles for comparing what layout or
	// inheritance reads of two.
	la, lb style.Computed
	warns  []Warning
	// changed, when set, is told of every element whose style changed
	// from an earlier one, and sets its Shown.
	changed func(n *dom.Node, old, next *style.Computed)
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
// no selector reads there restyles n alone). Children added or removed,
// or a tree state that moved, restyle the children only, as the subset's
// state selectors read their parent (tr:hover > td).
func (r *restyler) visit(n *dom.Node, parent *style.Computed, rem float32, force, deep bool) bool {
	data := dom.Data(n)
	if n.IsText() {
		data.Style = parent
		data.Shown = dom.Data(n.Parent()).Shown
		data.Dirty &^= dom.DirtyStyle | dom.DirtyTree
		return data.Dirty&dom.DirtyLayout != 0
	}
	dirty := data.Dirty
	inherits := false
	if force || deep || dirty&dom.DirtyStyle != 0 || data.Style == nil {
		inherits = r.restyle(n, data, parent, rem)
	}
	stateMoved := dirty&dom.DirtyState != 0 && (n.State()^data.StyledState)&r.treeStates != 0
	data.StyledState = n.State()
	data.Dirty &^= dom.DirtyStyle | dom.DirtyTree | dom.DirtyState
	if parent == nil {
		rem = data.Style.Font.Size
	}
	tree := dirty&dom.DirtyTree != 0
	if tree && slices.ContainsFunc(data.Changed, countsOn) {
		r.recount = true
	}
	kidsDeep := deep || stateMoved || tree && r.readAbove(data.Changed)
	kidsForce := inherits || kidsDeep || tree && slices.Contains(data.Changed, dom.Children)
	data.Changed = data.Changed[:0]
	if !kidsForce && dirty&dom.DirtyDescendant == 0 && !force {
		return data.Dirty&dom.DirtyLayout != 0
	}
	below := false
	for _, c := range n.Children() {
		if r.visit(c, data.Style, rem, kidsForce, kidsDeep) {
			below = true
		}
	}
	if below {
		data.Dirty |= dom.DirtyDescendant
	} else {
		data.Dirty &^= dom.DirtyDescendant
	}
	return below || data.Dirty&dom.DirtyLayout != 0
}

// restyle computes n's style again, and reports whether what its
// children inherit changed.
func (r *restyler) restyle(n *dom.Node, data *dom.Pipeline, parent *style.Computed, rem float32) bool {
	s := r.compute(n, parent, rem)
	*r.computed++
	old := data.Style
	if old != nil && reflect.DeepEqual(old, s) {
		r.generate(n, data, rem)
		return false
	}
	data.Style = s
	if r.changed != nil && old != nil {
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
	return !reflect.DeepEqual(&r.la, &r.lb)
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
// or anything at all where some declaration inherits explicitly.
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
	return !reflect.DeepEqual(&r.la, &r.lb)
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
// agent's first) sets a property that isn't inherited to inherit.
func explicitInherit(sheets []*Sheet) bool {
	return slices.ContainsFunc(sheets, func(s *Sheet) bool { return s.indexed().inherits })
}

// ctx is one element's computation.
type ctx struct {
	c, parent *style.Computed
	em, rem   float32
	vars      map[string][]Component
	resolving map[string]bool
	raw       map[string]*Declaration
	trans     [4][]Component
	// base is where the sheet the declaration being applied came from is.
	base     *origin
	expanded map[*Declaration]map[string][]Component
	// overlay is the restyler's: scrollbars keep no gutter.
	overlay bool
	// toViewport is set for the root and for its body where the viewport
	// scrolls: the two whose overflow it may take.
	toViewport bool
	// node is the element, which attr() reads.
	node *dom.Node
}

// order puts font-size first (em needs it), then color (currentcolor
// needs it), then the rest.
//
//nolint:gochecknoglobals,goconst // derived from the properties
var order, orderIndex = func() ([]string, map[string]int) {
	names := make([]string, 0, len(properties))
	for n := range properties {
		if n != "font-size" && n != "color" {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	names = append([]string{"font-size", "color"}, names...)
	index := make(map[string]int, len(names))
	for i, n := range names {
		index[n] = i
	}
	return names, index
}()

//nolint:gochecknoglobals // a fixed value every element shares
var currentColor = []Component{word(kwCurrentColor)}

// hints are the declarations an img's width and height attributes stand
// for, as HTML maps them: a number of pixels or a percentage, whatever
// follows it.
func hints(n *dom.Node) []Declaration {
	if n.Tag() != dom.Img {
		return nil
	}
	var out []Declaration
	for _, name := range [...]string{propWidth, propHeight} {
		v, ok := n.Attr(name)
		if !ok {
			continue
		}
		v = strings.TrimLeft(v, " \t\r\n\f")
		end := 0
		for end < len(v) && (v[end] >= '0' && v[end] <= '9' || v[end] == '.') {
			end++
		}
		f, err := strconv.ParseFloat(v[:end], 32)
		if err != nil {
			continue
		}
		unit := "px"
		if strings.HasPrefix(v[end:], "%") {
			unit = "%"
		}
		decls, _ := ParseInline(name + ": " + strconv.FormatFloat(f, 'f', -1, 32) + unit)
		out = append(out, decls...)
	}
	return out
}

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
	root := parent == nil
	if root {
		parent = &initial
		rem = initial.Font.Size
	}
	c := initial
	x := &ctx{c: &c, parent: parent, em: parent.Font.Size, rem: rem, overlay: r.overlay,
		toViewport: r.viewport && (root || rootBody(n)), node: n}
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
	for i, name := range order {
		x.apply(name, casc.longhands[i])
		if name == "font-size" {
			x.em = x.c.Font.Size
			if root {
				x.rem = x.c.Font.Size
			}
		}
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
	px := &ctx{c: &ps, parent: x.c, em: x.em, rem: x.rem, node: n}
	px.customs(casc.custom)
	for i, prop := range order {
		if d := casc.longhands[i]; d != nil {
			px.apply(prop, d)
			if prop == "font-size" {
				px.em = ps.Font.Size
			}
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
func (x *ctx) apply(name string, d *Declaration) {
	p := props[name]
	if d == nil {
		if properties[name].inherited {
			p.inherit(x.c, x.parent)
			x.c.ColorInherited = x.c.ColorInherited || name == "color"
		} else {
			x.reset(name, p)
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
		x.c.ColorInherited = kw == "inherit" || kw == kwUnset || kw == "" && isWord(v[0], kwCurrentColor)
	}
	switch kw {
	case "inherit":
		p.inherit(x.c, x.parent)
	case "initial":
		x.reset(name, p)
	case kwUnset:
		if properties[name].inherited {
			p.inherit(x.c, x.parent)
		} else {
			x.reset(name, p)
		}
	default:
		p.set(x, v)
	}
}

// reset gives a longhand its initial value; currentcolor is the
// element's colour.
func (x *ctx) reset(name string, p prop) {
	if properties[name].initial == kwCurrentColor {
		p.set(x, currentColor)
		return
	}
	p.inherit(x.c, &initial)
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
			c.Display = blockified(c.Display)
			c.Float = style.FloatNone
		}
	case style.Inline, style.Block, style.InlineBlock, style.DisplayNone, style.Contents:
		// not a flex or grid container
	}
	if root || c.Position == style.Absolute || c.Position == style.FixedPosition {
		c.Display = blockified(c.Display)
		c.Float = style.FloatNone
	}
	if c.Float != style.FloatNone {
		c.Display = blockified(c.Display)
	}
	x.transitions()
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
	d = d.Absolute(float64(x.em), float64(x.rem))
	return style.Length{Kind: style.Fixed, Px: float32(d.Px), Percent: float32(d.Pct), IsPercent: d.HasPercent()}
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

//nolint:gochecknoglobals,goconst // lookup table
var displays = map[string]style.Display{
	"inline": style.Inline, "block": style.Block, "inline-block": style.InlineBlock,
	"flex": style.FlexDisplay, "inline-flex": style.InlineFlex, "grid": style.GridDisplay,
	"inline-grid": style.InlineGrid, "none": style.DisplayNone, "contents": style.Contents,
	"list-item": style.Block, "flow-root": style.Block,
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
			},
			inherit: func(dst, src *style.Computed) {
				dst.Display, dst.ListItem, dst.FlowRoot = src.Display, src.ListItem, src.FlowRoot
			},
		},
		"float": keyword(func(c *style.Computed) *style.Float { return &c.Float }, map[string]style.Float{
			"none": style.FloatNone, "left": style.FloatLeft, "right": style.FloatRight,
			"inline-start": style.FloatLeft, "inline-end": style.FloatRight,
		}),
		"vertical-align": keyword(func(c *style.Computed) *style.VerticalAlign { return &c.VerticalAlign },
			map[string]style.VerticalAlign{"baseline": style.AlignBaselineText, "sub": style.AlignSub,
				"super": style.AlignSuper}),
		"clear": keyword(func(c *style.Computed) *style.Clear { return &c.Clear }, map[string]style.Clear{
			"none": style.ClearNone, "left": style.ClearLeft, "right": style.ClearRight, "both": style.ClearBoth,
			"inline-start": style.ClearLeft, "inline-end": style.ClearRight,
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
				}
				l := x.length(v[0])
				return style.LineHeight{Px: l.Px + l.Percent*x.em/100}
			},
		),
		"letter-spacing": field(func(c *style.Computed) *float32 { return &c.LetterSpacing },
			func(x *ctx, v []Component) float32 { return x.px(v[0]) }),
		"color": field(
			func(c *style.Computed) *color.Color { return &c.Color },
			func(x *ctx, v []Component) color.Color {
				if isWord(v[0], "currentcolor") {
					return x.parent.Color
				}
				return linear(v[0])
			},
		),
		"-webkit-font-smoothing": keyword(func(c *style.Computed) *bool { return &c.PixelText },
			map[string]bool{"none": true}),
		"text-transform": keyword(func(c *style.Computed) *style.TextTransform { return &c.TextTransform },
			map[string]style.TextTransform{"none": style.NoTransform, "uppercase": style.Uppercase,
				"lowercase": style.Lowercase, "capitalize": style.Capitalize}),
		"text-align": keyword(func(c *style.Computed) *style.TextAlign { return &c.TextAlign },
			map[string]style.TextAlign{"start": style.Start, "end": style.End, "left": style.Left, "right": style.Right,
				"center": style.Center, "justify": style.Justify}),
		"text-indent": lengthField(func(c *style.Computed) *style.Length { return &c.TextIndent }),
		"caption-side": keyword(func(c *style.Computed) *style.CaptionSide { return &c.CaptionSide },
			map[string]style.CaptionSide{"top": style.CaptionTop, "bottom": style.CaptionBottom}),
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
		"background-image": field(func(c *style.Computed) *style.Image { return &c.BackgroundImage }, backgroundImage),
		"background-size": field(func(c *style.Computed) *style.BackgroundSize { return &c.BackgroundSize },
			func(x *ctx, v []Component) style.BackgroundSize {
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
			}),
		"background-repeat": {
			set: func(x *ctx, v []Component) {
				x.c.BackgroundNoRepeat, x.c.BackgroundRepeat = isWord(v[0], "no-repeat"), style.RepeatBoth
				switch {
				case isWord(v[0], "repeat-x"):
					x.c.BackgroundRepeat = style.RepeatX
				case isWord(v[0], "repeat-y"):
					x.c.BackgroundRepeat = style.RepeatY
				}
			},
			inherit: func(dst, src *style.Computed) {
				dst.BackgroundNoRepeat, dst.BackgroundRepeat = src.BackgroundNoRepeat, src.BackgroundRepeat
			},
		},
		"background-position": field(func(c *style.Computed) *style.BackgroundPosition { return &c.BackgroundPosition },
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
					case isWord(c, "repeat"):
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
		m["border-"+side+"-style"] = keyword(func(c *style.Computed) *style.BorderStyle { return sideOf(&c.BorderStyle, i) },
			borderStyles)
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

func gridLine(_ *ctx, v []Component) style.GridLine {
	for i, c := range v {
		if isWord(c, "span") {
			return style.GridLine{Span: int(v[1-i].Num)}
		}
	}
	if isWord(v[0], "auto") {
		return style.GridLine{}
	}
	return style.GridLine{Line: int(v[0].Num)}
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
	for _, c := range v {
		if c.Kind != Function || !strings.EqualFold(c.Text, "repeat") {
			out = append(out, style.TrackRepeat{Count: 1, Tracks: []style.Track{x.track(c)}})
			continue
		}
		args := items(c.Args)
		r := style.TrackRepeat{Count: int(args[0].Num)}
		switch {
		case isWord(args[0], "auto-fill"):
			r.Count = style.AutoFill
		case isWord(args[0], "auto-fit"):
			r.Count = style.AutoFit
		}
		for _, t := range args[2:] {
			r.Tracks = append(r.Tracks, x.track(t))
		}
		out = append(out, r)
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

// readAbove reports whether some sheet reads one of changed in a compound
// above a selector's subject.
func (r *restyler) readAbove(changed []string) bool {
	for _, k := range changed {
		for _, sh := range r.sheets {
			if sh.indexed().above[k] {
				return true
			}
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

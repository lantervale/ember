// Package paint draws a laid-out box tree onto a twod canvas, as CSS
// paints: stacking contexts by z-index, backgrounds, borders, shadows,
// text, outlines, opacity and 2D transforms, clipped where overflow says,
// with the old ember/ui's scrollbars on scroll containers.
//
// One simplification holds for now: opacity multiplies the alpha of
// everything under it rather than compositing the subtree as one picture.
// The README lists what else is approximate.
package paint

import (
	"math"
	"slices"
	"sort"
	"sync"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/cull"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/fx"
	"github.com/lantervale/ember/ui/internal/style"
)

// Surface is what paint draws on; *twod.Canvas is one.
type Surface interface {
	Rect(r geom.Rect, c color.Color)
	RoundRect(r geom.Rect, radius float32, c color.Color)
	RoundRectCorners(r geom.Rect, radii [4]float32, c color.Color)
	BorderCorners(r geom.Rect, radii [4]float32, thickness float32, c color.Color)
	Border(r geom.Rect, radius, thickness float32, c color.Color)
	Glow(r geom.Rect, radius, spread float32, c color.Color)
	Line(a, b geom.Vec2, width float32, c color.Color)
	Circle(center geom.Vec2, radius float32, c color.Color)
	Triangles(points []geom.Vec2, indices []int, c color.Color)
	TrianglesColored(points []geom.Vec2, colors []color.Color, indices []int)
	Text(at geom.Vec2, text string, opts ...twod.TextOption) geom.Vec2
	Image(r geom.Rect, tex *twod.Texture, uv geom.Rect, tint color.Color)
	SmoothImage(r geom.Rect, tex *twod.Texture, uv geom.Rect, tint color.Color)
	NinePatch(r geom.Rect, tex *twod.Texture, left, top, right, bottom float32, tint color.Color)
	PushClip(r geom.Rect)
	PushClipRounded(r geom.Rect, radii [4]float32)
	PushClipMask(r geom.Rect, mask *twod.Texture)
	PopClip()
	Transform() geom.Transform2D
	SetTransform(t geom.Transform2D)
}

// Option sets what paint knows of the game's fonts and images.
type Option func(p *painter)

// Faces picks the twod face a font draws with; twod.Regular() without it.
func Faces(face func(f style.Font) *twod.Face) Option { return func(p *painter) { p.face = face } }

// Fonts gives the metrics and advances layout used, for what paint lays
// out itself: an open select's rows. Text sits on layout's baselines.
func Fonts(f box.Fonts) Option { return func(p *painter) { p.fonts = f } }

// Controls finds the live control behind a node (a *control.Checkbox,
// *control.Range, *control.Text…), for the marks that show its state;
// without one, a control is drawn from its attributes and state.
func Controls(find func(n *dom.Node) any) Option { return func(p *painter) { p.control = find } }

// Time is the time in seconds, unscaled, that blinks the caret and slides
// an indeterminate progress; without it the caret stays lit and the
// progress shows its empty bar.
func Time(seconds float32) Option {
	return func(p *painter) { p.time, p.timed = seconds, true }
}

// Viewport is the canvas the document is drawn on: the window, or the
// area of the element hosting it. The root's background covers it.
func Viewport(r geom.Rect) Option { return func(p *painter) { p.viewport, p.hosted = r, true } }

// Scale is how many device pixels make a point: the edges of boxes,
// their borders, shadows and outlines are snapped to them, as Chrome
// snaps them, so a box at a fractional place shows no half-lit seam.
// Text is never snapped. Without it nothing is.
func Scale(pixelsPerPoint float32) Option { return func(p *painter) { p.scale = pixelsPerPoint } }

// Lists keeps an open select's list inside within, in the root's space,
// rather than the viewport: a Document in an iframe has its list shown
// whole over the page round it, in the window (decision 0166). Apart,
// the paint leaves the list out, for DrawLists to draw over that page.
func Lists(within geom.Rect, apart bool) Option {
	return func(p *painter) { p.lists, p.listsIn, p.listsApart = true, within, apart }
}

// Frame is what changes from one paint to the next: the clock, the live
// set and whether its parts are kept apart, and the box to Keep. A host
// keeps one and passes Each(&frame) once, so a frame's paint makes no
// options.
type Frame struct {
	Time  float32
	Live  *LiveSet
	Apart bool
	Keep  *box.Box
	// Selection is the text selected outside the fields, nil for none.
	Selection Selection
	// Inactive is the window not being the active one: a selection no
	// ::selection colours is drawn in the inactive colours, as Chrome's.
	Inactive bool
	// Highlight is the system's selection colours, which a selection no
	// ::selection colours is drawn in.
	Highlight Highlight
	// Lists, when set, is where an open select's list is kept, and
	// ListsApart whether it is left out, as the Lists option says.
	Lists      *geom.Rect
	ListsApart bool
}

// Highlight is the colours a system selects text in, as
// platform.Highlight has them: a colour of no alpha is one it does not
// say, Chrome's on macOS in its place.
type Highlight struct {
	Background, Inactive, Text color.Color
}

// Selection is the text a Document has selected, as paint draws it.
type Selection interface {
	// Parts are the rectangles of a text run the selection covers, in the
	// run's space as its fragments' are, or of a br the mark of the line
	// it holds alone, in its box's space.
	Parts(run *box.Box) []Selected
	// Holds reports an element wholly inside the selection: a picture
	// there is tinted.
	Holds(n *dom.Node) bool
}

// Selected is the part of a text run's fragment Line a selection covers:
// Line is -1 for the mark of an empty line, which no fragment holds.
type Selected struct {
	Line int
	Rect geom.Rect
}

// TransformOf is the transform b's own style turns it and what it holds
// by, in the root's space, false when it has none.
func TransformOf(b *box.Box) (geom.Transform2D, bool) {
	st := shown(b)
	if b.Kind != box.Element || st == nil || !st.Transformed() {
		return geom.Transform2D{}, false
	}
	r := b.Rect()
	return st.TransformAt(geom.V2(r.X, r.Y), b.Size), true
}

// Each has every paint take f as it stands when the paint starts, as
// Time, Live and Keep would.
func Each(f *Frame) Option {
	return func(p *painter) {
		p.time, p.timed, p.keep, p.sel, p.inactive, p.system = f.Time, true, f.Keep, f.Selection, f.Inactive,
			f.Highlight
		if f.Live != nil {
			f.Live.reset()
			p.live, p.apart = f.Live, f.Apart
		}
		if f.Lists != nil {
			p.lists, p.listsIn, p.listsApart = true, *f.Lists, f.ListsApart
		}
	}
}

// Keep names a box paint always reaches, though rows around it are out
// of view: the focused one, whose outline may reach into view.
func Keep(b *box.Box) Option { return func(p *painter) { p.keep = b } }

// Images finds background and border images by name; without it they
// are skipped. A texture a game gives here filters as CSS's only when made
// twod.Encoded() (decision 0056), as the Document's own are.
func Images(image func(name string) *twod.Texture) Option {
	return func(p *painter) { p.image = image }
}

// Vector is a picture drawn from shapes, with no pixels of its own: what
// it asks to be sized by, and its texture drawn at a size.
type Vector interface {
	Intrinsic() box.Intrinsic
	// Texture is the picture drawn w×h device pixels large, scale of
	// them to a point. It may be one of another size, to stretch over
	// the same box, while the picture is being drawn or its size is
	// changing, and nil when there is none yet.
	Texture(w, h int, scale float32) *twod.Texture
}

// Filters finds the filter element an element's filter: url(#id) names,
// read for an element of size points, nil for none: one in an svg element
// of the Document.
func Filters(graph func(id string, size geom.Vec2) *fx.Graph) Option {
	return func(p *painter) { p.graph = graph }
}

// Vectors finds by name the pictures Images has no texture for because
// they are drawn from shapes, nil for any other: each is drawn at the
// size it is shown at, in device pixels.
func Vectors(vector func(name string) Vector) Option {
	return func(p *painter) { p.vector = vector }
}

type painter struct {
	s       Surface
	keep    *box.Box
	face    func(style.Font) *twod.Face
	fonts   box.Fonts
	image   func(string) *twod.Texture
	vector  func(string) Vector
	graph   func(id string, size geom.Vec2) *fx.Graph
	control func(*dom.Node) any
	sel     Selection
	// around is the text round the fragment drawn, which its words are
	// shaped between (box.Fragment's Before and After), and amid the text
	// beside it they are shaped as one with (its Lead and Trail).
	around, amid [2]string
	// inactive and system are Frame's Inactive and Highlight.
	inactive bool
	system   Highlight
	time     float32
	timed    bool
	scale    float32
	// popups are the open selects met, drawn over everything at the end.
	popups []popup
	// lists is set when the open selects' lists are kept in listsIn,
	// left out of the paint when listsApart.
	lists, listsApart bool
	listsIn           geom.Rect
	// top is the top layer: the modal dialogs met, painted over the page
	// in the order they were shown modal.
	top []entry
	// viewport is the canvas: the Viewport option's, or the root's border
	// box. The root's background covers it, a modal dialog's backdrop too,
	// and an open select's list stays in it. canvas is the box whose
	// background it took.
	viewport geom.Rect
	hosted   bool
	canvas   *box.Box
	// found is the last element box a hit test's walk met holding point,
	// and probe its surface.
	found *box.Box
	probe *probe
	// slabs hold the walk's clips, reused paint after paint; used of them
	// are taken.
	slabs [][]clip
	used  int
	// clip is the clip chain pushed on the surface now, held of them,
	// those of floor's chain left out: the clips a filtered element's
	// picture is shown in, not drawn in.
	clip  *clip
	held  int
	floor *clip
	// pictures is how many pictures paint has begun and not ended, paged
	// whether one of them is the whole page's.
	pictures int
	paged    bool
	// spread is how far the filters of the elements painted round what
	// is painted now draw it past where it is: so much more of it shows.
	spread float32
	// scratch holds pieces' parts.
	scratch []piece
	// standings are the styles of glyphs standing in vertical lines, by their own.
	standings map[*style.Computed]*style.Computed
	// base is the surface's transform as painting starts: the root's
	// space, where the viewport culls.
	base geom.Transform2D
	// whole walks every box, and reached counts those the walk came to
	// under the root.
	whole   bool
	reached int
	// probing is a scan's walk, which tells hit only of the boxes whose
	// ink holds point, in the root's space, and of none when pointless.
	probing, pointless bool
	point              geom.Vec2
	// live, if set, keeps the live parts drawn, apart from the paint when
	// apart.
	live  *LiveSet
	apart bool
	// chain is use's room for a clip chain.
	chain []*clip
	// glyphsOnly draws text as the shape of its glyphs and decorations,
	// opaque white: the picture background-clip: text shows through.
	glyphsOnly bool
	// lineDecorated is set while the parts of a vertical line are drawn,
	// whose text decorations are drawn along the whole line instead.
	lineDecorated bool
	// lineSelecting is set while the parts of a vertical line are drawn, whose
	// selection's background is drawn along the whole line: lineParts is what of the
	// part drawn it selects, in that part's space.
	lineSelecting bool
	lineParts     []Selected
	// lastText is the last text run's options.
	lastText struct {
		key  textKey
		opts []twod.TextOption
	}
}

// slab is how many clips a slab holds.
const slab = 64

// newClip is c kept in the painter's slabs, which never move, so the
// clip chains pointing into them hold for the walk.
func (p *painter) newClip(c clip) *clip {
	i, j := p.used/slab, p.used%slab
	if i == len(p.slabs) {
		p.slabs = append(p.slabs, make([]clip, slab))
	}
	p.used++
	p.slabs[i][j] = c
	return &p.slabs[i][j]
}

// clip is one overflow clip inside its parent's, in the space of xf: the
// transform of the stacking context it was met in. round and radii, when
// radii aren't all zero, are the curve it also cuts to (decision 0058):
// the padding box rounded by its inner radii, while rect stops at the
// scrollbars' gutter. by is the box whose overflow it is. A clip-path's
// clip has its shape instead, and rect its bounds (decision 0119).
type clip struct {
	rect   geom.Rect
	round  geom.Rect
	radii  radii
	xf     geom.Transform2D
	parent *clip
	by     *box.Box
	shape  *clipShape
}

// Paint draws t onto s.
func Paint(s Surface, t *box.Tree, opts ...Option) {
	if t == nil || t.Root == nil {
		return
	}
	mode := cull.Now()
	if mode == cull.Check {
		checkPaint(s, t, opts)
	}
	paintOn(s, t, opts, mode == cull.Off)
}

// paintOn draws t onto s, walking every box when whole and else only
// those whose ink shows (decision 0101), and returns how many boxes under
// the root the walk reached.
func paintOn(s Surface, t *box.Tree, opts []Option, whole bool) int {
	inked(t)
	p := pooled[painter](&painters)
	defer painters.Put(p)
	*p = painter{
		s: s, base: s.Transform(), whole: whole,
		scratch: p.scratch[:0], chain: p.chain[:0], popups: p.popups[:0], top: p.top[:0], slabs: p.slabs,
	}
	for _, o := range opts {
		o(p)
	}
	p.start(t.Root)
	on, grouped := p.s.(Pictures)
	if grouped = grouped && t.Grouped; grouped {
		on.BeginPicture()
		p.pictures++
		p.paged = true
	}
	p.canvasBackground(t.Root)
	p.page(t.Root)
	p.use(nil)
	if grouped {
		p.pictures--
		on.DrawPicture(on.EndPicture(0))
	}
	p.drawPopups()
	return p.reached
}

// page paints root's stacking context, then the top layer over it, each
// modal dialog with its backdrop as a stacking context of its own, free of
// its ancestors' clips, transforms and opacity. A modal met inside one
// joins the top layer in its place by order.
func (p *painter) page(root *box.Box) {
	p.context(entry{b: root, at: root.Pos}, 1)
	for len(p.top) > 0 {
		i := 0
		for j := range p.top {
			if p.layer(p.top[j].b) < p.layer(p.top[i].b) {
				i = j
			}
		}
		e := p.top[i]
		p.top = slices.Delete(p.top, i, i+1)
		saved := p.s.Transform()
		p.s.SetTransform(p.base)
		p.context(e, 1)
		p.s.SetTransform(saved)
	}
}

// start takes the viewport from the Viewport option, or else the root's
// border box.
func (p *painter) start(root *box.Box) {
	if !p.hosted {
		p.viewport = geom.Rect{X: root.Pos.X, Y: root.Pos.Y, W: root.Size.X, H: root.Size.Y}
	}
}

// canvasBackground paints the root's background over the whole viewport,
// or body's when the root has none, as CSS's canvas does; that box then
// leaves it out of its own decoration. Without either, a root drawn in
// the dark scheme has the canvas Chrome's dark Canvas colour, where a
// light one leaves the window's own showing. Its picture is placed in the
// root element's box its origin names, as Chrome places it, so in a
// viewport that scrolls it scrolls with the page, unless it is fixed.
func (p *painter) canvasBackground(root *box.Box) {
	at := root.Pos
	if root.Kind == box.Viewport {
		if len(root.Children) == 0 {
			return
		}
		at = root.Pos.Sub(root.Scroll)
		root = root.Children[0]
		at = at.Add(root.Pos)
	}
	border := geom.Rect{X: at.X, Y: at.Y, W: root.Size.X, H: root.Size.Y}
	src := root
	if !hasBackground(shown(root)) {
		src = nil
		for _, c := range root.Children {
			if c.Kind == box.Element && c.Node != nil && c.Node.Tag() == dom.Body {
				src = c
				break
			}
		}
	}
	if src == nil || !hasBackground(shown(src)) {
		if st := shown(root); st != nil && st.Dark {
			p.s.Rect(p.viewport, darkCanvas)
		}
		return
	}
	p.canvas = src
	st := shown(src)
	mixing := p.beginLayers(st)
	defer mixing.end()
	if st.Background.A > 0 {
		p.s.Rect(p.viewport, st.Background)
	}
	for i := st.BackgroundLayers() - 1; i >= 0; i-- {
		l := st.BackgroundLayer(i)
		area, _ := backgroundBox(root, border, root.Border, radii{}, or(l.Origin, style.UnderPadding))
		if l.Attachment == style.AttachFixed {
			area = p.viewport
		}
		mixing.layer(l.Blend, func() { p.backgroundImage(p.viewport, area, radii{}, &l, st, 1) })
	}
}

func hasBackground(st *style.Computed) bool { return st.Background.A > 0 || st.HasBackgroundImage() }

// entry is a box to paint, its border box's top-left in the root's space
// and the clips of its ancestors within its stacking context.
type entry struct {
	b    *box.Box
	at   geom.Vec2
	clip *clip
	// pseudo marks a positioned box with z-index auto: painted in its
	// place as if it made a stacking context, though its positioned
	// descendants and the contexts in it belong to the context around it.
	pseudo bool
	// atomic marks an inline-level box laid out whole, an inline-block or
	// an inline table, flex or grid: in the flow, painted whole with its
	// line's text, as if it made a stacking context.
	atomic bool
}

// layers are a stacking context's descendants in paint order.
type layers struct {
	negative, flow, floats, positioned, positive []entry
	// scrollers have their bars drawn once everything else is.
	scrollers []entry
}

// spareLayers keeps layers between frames, so a frame's are made once.
//
//nolint:gochecknoglobals // a sync.Pool every paint shares
var spareLayers = sync.Pool{New: func() any { return new(layers) }}

func newLayers() *layers { return pooled[layers](&spareLayers) }

// pooled is a *T from pool, which holds only *T: new when it held none.
func pooled[T any](pool *sync.Pool) *T {
	if v, ok := pool.Get().(*T); ok {
		return v
	}
	return new(T)
}

// free empties l, keeping its room, for the next stack.
func (l *layers) free() {
	for _, s := range []*[]entry{&l.negative, &l.flow, &l.floats, &l.positioned, &l.positive, &l.scrollers} {
		clear(*s)
		*s = (*s)[:0]
	}
	spareLayers.Put(l)
}

// context paints a box that starts a stacking context, and everything in
// it: its own decoration, negative z-indexes, the flow's decorations then
// its text and outlines, positioned boxes and zero z-indexes, then
// positive ones, as Chrome does.
func (p *painter) context(e entry, alpha float32) { p.stack(e, alpha, false) }

// stack paints e and what stacks with it: everything in it when it is a
// stacking context, or only its flow when pseudo, the rest having been
// hoisted into the context around it.
// inFlow paints e's in-flow children in their passes: their boxes, the
// borders tables collapse over them, then
// the floats, each whole, then their text, markers and inline-blocks,
// each whole, then e's and their control marks, then e's and
// their outlines.
func (p *painter) inFlow(e entry, l *layers, alpha float32) {
	p.flowBoxes(e, l, alpha)
	for _, c := range l.floats {
		p.stack(c, alpha, true)
	}
	// A control laid out round its parts draws them under its generated
	// content, as Chrome's shadow tree.
	if marked(e.b) && parted(e.b) {
		p.use(e.clip)
		p.marks(e, alpha)
	}
	for _, c := range l.flow {
		if marked(c.b) && parted(c.b) && !c.atomic {
			p.use(c.clip)
			p.marks(c, alpha)
		}
	}
	for _, c := range l.flow {
		switch c.b.Kind {
		case box.TextRun:
			p.use(c.clip)
			p.text(c, alpha)
		case box.Marker:
			p.use(c.clip)
			p.marker(c, alpha)
		case box.Element:
			if c.atomic {
				p.stack(c, alpha, true)
			} else if p.sel != nil && c.b.Node != nil && c.b.Node.Tag() == dom.Br {
				p.use(c.clip)
				p.lineBreak(c, alpha)
			}
		case box.AnonymousBlock, box.Viewport, box.ControlPart:
		}
	}
	if marked(e.b) && !parted(e.b) {
		p.use(e.clip)
		p.marks(e, alpha)
	}
	for _, c := range l.flow {
		if marked(c.b) && !parted(c.b) && !c.atomic {
			p.use(c.clip)
			p.marks(c, alpha)
		}
	}
	p.use(e.clip)
	p.outline(e, alpha)
	for _, c := range l.flow {
		if outlined(c.b) && !c.atomic {
			p.use(c.clip)
			p.outline(c, alpha)
		}
	}
}

// flowBoxes paints the boxes of e's flow, but those painted whole with
// their line, then the borders tables collapse over them.
func (p *painter) flowBoxes(e entry, l *layers, alpha float32) {
	for _, c := range l.flow {
		if c.b.Kind == box.Element && !c.atomic {
			p.use(c.clip)
			p.decorate(c, alpha)
		}
	}
	if e.b.Collapsed != nil {
		p.use(e.clip)
		p.collapsedBorders(e, alpha)
	}
	for _, c := range l.flow {
		if c.b.Collapsed != nil && !c.atomic {
			p.use(c.clip)
			p.collapsedBorders(c, alpha)
		}
	}
}

// shaped is e cut by its clip-path, false where the clip-path leaves
// nothing, and on a scan by its border box where it is masked.
func (p *painter) shaped(e entry, st *style.Computed) (entry, bool) {
	b := e.b
	if s, ok := clipShapeOf(b, e.at, st); ok {
		if s.bounds.W <= 0 || s.bounds.H <= 0 {
			return e, false
		}
		e.clip = p.newClip(clip{rect: s.bounds, xf: p.s.Transform(), parent: e.clip, shape: &s})
	}
	if p.probing && b.Kind == box.Element && st.Masked() {
		// Chrome hits a masked element's content in its border box alone,
		// wherever its mask is clear or reaches.
		e.clip = p.newClip(clip{rect: geom.Rect{X: e.at.X, Y: e.at.Y, W: b.Size.X, H: b.Size.Y}, xf: p.s.Transform(),
			parent: e.clip})
	}
	return e, true
}

func (p *painter) stack(e entry, alpha float32, pseudo bool) {
	b, st := e.b, shown(e.b)
	whole, in := e, alpha
	if b.Kind == box.Element {
		alpha *= st.Opacity * (1 - b.Dim)
	}
	if alpha <= 0 && !p.probing {
		return
	}
	from, gapped := p.stackGap()
	if p.modal(b) {
		p.use(nil)
		if p.probing {
			p.probed(b, p.viewport)
		} else if st.Backdrop.A > 0 {
			p.s.Rect(p.viewport, fade(st.Backdrop, alpha))
		}
	} else if p.topLayer(b) && st.Backdrop.A > 0 && !p.probing {
		p.use(nil)
		p.s.Rect(p.viewport, fade(st.Backdrop, alpha))
	}
	saved, around := p.s.Transform(), e.clip
	if b.Kind == box.Element && st.Transformed() {
		p.s.SetTransform(saved.Mul(st.TransformAt(e.at, b.Size)))
	}
	e, shows := p.shaped(e, st)
	if !shows {
		p.s.SetTransform(saved)
		return
	}
	group := p.beginGroup(e, st, &alpha)
	var mask *masking
	var filters *filtering
	if b.Kind == box.Element {
		mask = p.beginMask(e, st, around, group, &alpha)
		p.showBackdrop(group, e, st)
		filters = p.beginFilters(e, st)
	}
	p.use(e.clip)
	p.decorate(e, alpha)
	l, hoisted := newLayers(), newLayers()
	defer l.free()
	defer hoisted.free()
	if pseudo {
		p.collect(e, l, hoisted)
	} else {
		p.collect(e, l, l)
	}
	for _, c := range sortZ(l.negative) {
		p.context(c, alpha)
	}
	p.inFlow(e, l, alpha)
	for _, c := range l.positioned {
		p.stack(c, alpha, c.pseudo)
	}
	for _, c := range sortZ(l.positive) {
		p.context(c, alpha)
	}
	p.scrollbars(l.scrollers, alpha)
	p.endFilters(filters, e.clip)
	p.endMask(mask, e, st)
	p.endGroup(group, e.clip)
	p.use(around)
	p.s.SetTransform(saved)
	if gapped {
		p.endStackGap(from, whole, saved, in, pseudo)
	}
}

// scrollbars paints the bars of the scrollers in view.
func (p *painter) scrollbars(scrollers []entry, alpha float32) {
	for _, c := range scrollers {
		if p.missed(c, geom.Rect{X: c.at.X, Y: c.at.Y, W: c.b.Size.X, H: c.b.Size.Y}) {
			continue
		}
		p.use(c.clip)
		p.bars(BarsOf(c.b, c.at), alpha)
	}
}

// collect sorts e's descendants, but those whose ink and all they hold
// is out of view or off a scan's point (sight): its flow into own, down
// to but not into
// the stacking contexts it holds, and those contexts and the positioned
// boxes into outer, the layers of the context they stack in. A positioned
// box with z-index auto goes into outer's positioned layer in tree order,
// and its own positioned descendants and contexts after it into outer too.
// A float goes into own's floats, painted whole between the flow's boxes and
// its text, its positioned descendants and contexts into outer likewise.
func (p *painter) collect(e entry, own, outer *layers) {
	b := e.b
	inner := e.clip
	if r, ok := clipRect(b, e.at); ok {
		round, rr := clipRound(b, e.at)
		inner = p.newClip(clip{rect: r, round: round, radii: rr, xf: p.s.Transform(), parent: e.clip, by: b})
		if st := shown(b); scrolls(st) || st.Resizable() {
			own.scrollers = append(own.scrollers, e)
		}
	}
	origin := e.at.Sub(b.Scroll)
	run, extra := p.reachable(b, origin, inner)
	if extra != nil {
		run = append(run[:len(run):len(run)], extra)
	}
	see := p.sight(inner)
	items := b.Kind == box.Element && holdsItems(shown(b))
	for _, c := range run {
		at := origin.Add(c.Pos)
		if see.misses(c, at) {
			continue
		}
		p.reached++
		st := shown(c)
		ce := entry{b: c, at: at, clip: placedClip(c, st, inner)}
		if c.Discarded {
			p.collectPositioned(ce, outer)
			continue
		}
		switch {
		case p.topLayer(c):
			p.top = append(p.top, entry{b: c, at: ce.at})
		case c.Kind != box.Element:
			own.flow = append(own.flow, ce)
			p.collect(ce, own, outer)
		case c.Dim > 0 && !startsContext(st):
			// A block a faded inline element holds stacks with it, as that element does.
			outer.positioned = append(outer.positioned, ce)
		case startsContext(st) && st.ZIndex < 0 && !st.ZAuto:
			outer.negative = append(outer.negative, ce)
		case startsContext(st) && st.ZIndex > 0 && !st.ZAuto:
			outer.positive = append(outer.positive, ce)
		case startsContext(st):
			outer.positioned = append(outer.positioned, ce)
		case st.Position != style.Static:
			ce.pseudo = true
			outer.positioned = append(outer.positioned, ce)
			p.hoist(ce, outer)
		case st.Float != style.FloatNone:
			ce.pseudo = true
			own.floats = append(own.floats, ce)
			p.hoist(ce, outer)
		default:
			ce.atomic = items || atomicInline(st)
			own.flow = append(own.flow, ce)
			if ce.atomic {
				p.hoist(ce, outer)
			} else {
				p.collect(ce, own, outer)
			}
		}
	}
}

// hoist collects into outer the positioned boxes and stacking contexts
// under e, a box painted whole in its place, which collects its flow
// itself.
func (p *painter) hoist(e entry, outer *layers) {
	var flow layers
	p.collect(e, &flow, outer)
}

// atomicInline reports a box laid out whole on a line, painted with its
// line's text as CSS 2's appendix E paints an inline-block.
func atomicInline(st *style.Computed) bool {
	return st.Display == style.InlineBlock || st.Display == style.InlineFlex || st.Display == style.InlineGrid
}

// holdsItems reports a flex or grid container, whose items paint as
// inline-blocks do.
func holdsItems(st *style.Computed) bool {
	switch st.Display {
	case style.FlexDisplay, style.InlineFlex, style.GridDisplay, style.InlineGrid:
		return true
	case style.Inline, style.Block, style.InlineBlock, style.DisplayNone, style.Contents:
	}
	return false
}

// collectPositioned collects into outer the positioned boxes under e, a box past a
// line-clamp's last line: e and the rest it holds are never drawn, as Chrome's.
func (p *painter) collectPositioned(e entry, outer *layers) {
	var own, held layers
	p.collect(e, &own, &held)
	keep := func(to *[]entry, from []entry) {
		for _, c := range from {
			if shown(c.b).Position != style.Static {
				*to = append(*to, c)
			}
		}
	}
	keep(&outer.negative, held.negative)
	keep(&outer.positioned, held.positioned)
	keep(&outer.positive, held.positive)
}

// placedClip is the clip of c, a child of the box inner is the clip in: inner, less,
// for an absolutely or fixed positioned box, the clips of the boxes under its
// containing block, whose overflow it lies outside.
func placedClip(c *box.Box, st *style.Computed, inner *clip) *clip {
	if c.Kind != box.Element || st.Position != style.Absolute && st.Position != style.FixedPosition {
		return inner
	}
	for a := c.Parent; inner != nil && a != nil && a.Parent != nil; a = a.Parent {
		if as := shown(a); as.HoldsFixed() || st.Position == style.Absolute && as.Position != style.Static {
			break
		}
		if inner.by == a {
			inner = inner.parent
		}
	}
	return inner
}

// reachable is what of b's children, placed from origin, can show inside
// c: the run of them whose rows, widened by the ink they draw past their
// boxes, c's rect reaches, found by binary search, and the one holding the
// box Keep names when it falls outside the run. That holds when the
// children stack down in block order, each in flow, untransformed, drawing
// nothing outside its box but ink and holding no box a containing block
// above placed, as a long list's or table's rows do; otherwise, or under a
// transform, all of them show.
func (p *painter) reachable(b *box.Box, origin geom.Vec2, c *clip) ([]*box.Box, *box.Box) {
	kids := b.Children
	if len(kids) < 32 || c == nil || c.xf != geom.Identity2D() || p.s.Transform() != geom.Identity2D() {
		return kids, nil
	}
	prev, reach := float32(math.Inf(-1)), float32(0)
	for _, k := range kids {
		st := shown(k)
		if k.Kind != box.Element || k.Escapes || k.Pos.Y < prev || st.Position != style.Static || st.Transformed() ||
			k.Content.X > k.Size.X || k.Content.Y > k.Size.Y {
			return kids, nil
		}
		prev = k.Pos.Y + k.Size.Y
		reach = max(reach, inkReach(st))
	}
	reach = max(reach, inkMargin) + p.spread
	top, bottom := c.rect.Y-origin.Y-reach, c.rect.Y+c.rect.H-origin.Y+reach
	from := sort.Search(len(kids), func(i int) bool { return kids[i].Pos.Y+kids[i].Size.Y > top })
	to := max(from, sort.Search(len(kids), func(i int) bool { return kids[i].Pos.Y >= bottom }))
	var extra *box.Box
	for k := p.keep; k != nil; k = k.Parent {
		if k.Parent == b {
			if i := slices.Index(kids, k); i >= 0 && (i < from || i >= to) {
				extra = k
			}
			break
		}
	}
	return kids[from:to], extra
}

// inkMargin is how far, at least, the rows around the view are walked for
// ink their descendants draw past them, unseen until walked: enough for
// kit's largest glow, a 24-point blur spread by 4.
const inkMargin = 48

// inkReach is how far what a box of style st draws may reach past it: its
// shadows, its outline, and its text's glyphs and shadows.
func inkReach(st *style.Computed) float32 {
	r := max(shadowReach(st.Shadows), st.Font.Size+shadowReach(st.TextShadows)) + filterReach(st.Filters)
	if st.OutlineStyle != style.BorderNone {
		r = max(r, outlineReach(st)-st.OutlineOffset+max(st.OutlineOffset, 0))
	}
	return r
}

// startsContext reports whether a box paints as a stacking context: a
// positioned one with a z-index, or one less than opaque, transformed,
// clipped by a clip-path, filtered, masked, mixed or isolated.
func startsContext(st *style.Computed) bool {
	return st.Position != style.Static && !st.ZAuto || st.Opacity < 1 || st.Transformed() ||
		!st.ClipPath.None() || len(st.Filters) > 0 || len(st.BackdropFilters) > 0 || st.Masked() ||
		st.BlendMode != style.BlendNormal || st.Isolate
}

func sortZ(es []entry) []entry {
	slices.SortStableFunc(es, func(a, b entry) int { return shown(a.b).ZIndex - shown(b.b).ZIndex })
	return es
}

// use puts c's chain on the surface, popping what is not in it.
func (p *painter) use(c *clip) {
	if c == p.clip {
		return
	}
	for range p.held {
		p.s.PopClip()
	}
	p.clip, p.held = c, 0
	p.chain = p.chain[:0]
	for k := c; k != nil && k != p.floor; k = k.parent {
		p.chain = append(p.chain, k)
	}
	xf := p.s.Transform()
	for i := len(p.chain) - 1; i >= 0; i-- {
		p.s.SetTransform(p.chain[i].xf)
		if k := p.chain[i]; k.shape != nil {
			if s, ok := p.s.(shaped); ok {
				s.pushShape(k.shape)
			} else {
				p.s.PushClipMask(k.rect, masks.mask(k.shape, p.maskScale()))
			}
			p.held++
			continue
		}
		if k := p.chain[i]; k.radii != (radii{}) {
			p.s.PushClipRounded(k.round, k.radii)
			p.held++
		}
		p.s.PushClip(p.chain[i].rect)
		p.held++
	}
	p.s.SetTransform(xf)
}

// unbounded stands for an axis overflow leaves visible.
const unbounded = 1e6

// clipRect is the scrollport b clips its content to, if its overflow
// clips on either axis or a collapsed row or column cut it: its padding
// box less its scrollbars' gutter.
func clipRect(b *box.Box, at geom.Vec2) (geom.Rect, bool) {
	o := shown(b).Overflow
	if b.Cut {
		o = style.Size[style.Overflow]{Width: style.Clip, Height: style.Clip}
	}
	if b.Kind != box.Element && b.Kind != box.Viewport || o.Width == style.Visible && o.Height == style.Visible {
		return geom.Rect{}, false
	}
	r := padding(b, at)
	bar := b.Scrollbar
	// A stable gutter no bar stands in clips nothing: the content shows
	// under it, as in Chrome.
	if oy := shown(b).Overflow.Height; b.Kind == box.Element && oy != style.Scroll &&
		(oy != style.OverflowAuto || b.Content.Y <= r.H-bar.Y) {
		bar.X = 0
	}
	r.W, r.H = max(r.W-bar.X, 0), max(r.H-bar.Y, 0)
	if shown(b).Dir == style.RTL {
		r.X += bar.X
	}
	if o.Width == style.Visible {
		r.X, r.W = -unbounded, 2*unbounded
	}
	if o.Height == style.Visible {
		r.Y, r.H = -unbounded, 2*unbounded
	}
	return r, true
}

// clipRound is the curve b's clip also cuts to, when its overflow clips
// on both axes and it has rounded corners: its padding box and the inner
// radii its background is painted with, as Chrome rounds an overflow clip.
// With an axis left visible its corners don't cut, and it has none.
func clipRound(b *box.Box, at geom.Vec2) (geom.Rect, radii) {
	st := shown(b)
	if st.Overflow.Width == style.Visible || st.Overflow.Height == style.Visible {
		return geom.Rect{}, radii{}
	}
	return padding(b, at), radiiOf(st, b.Size).inside(b.Border)
}

func scrolls(st *style.Computed) bool {
	is := func(o style.Overflow) bool { return o == style.Scroll || o == style.OverflowAuto }
	return is(st.Overflow.Width) || is(st.Overflow.Height)
}

func padding(b *box.Box, at geom.Vec2) geom.Rect {
	return geom.Rect{
		X: at.X + b.Border.Left, Y: at.Y + b.Border.Top,
		W: b.Size.X - b.Border.Horizontal(), H: b.Size.Y - b.Border.Vertical(),
	}
}

// shown is the style b is painted with: its node's Shown, which holds
// running transitions' values, or for an anonymous box the nearest
// node's; its Style when no node has one, and for a viewport, whose node
// is the root element.
func shown(b *box.Box) *style.Computed {
	if b.Kind == box.Viewport {
		return b.Style
	}
	for a := b; a != nil; a = a.Parent {
		if a.Node != nil {
			if s := dom.Data(a.Node).Shown; s != nil {
				return s
			}
			break
		}
	}
	return b.Style
}

// snap is r with its edges on device pixels, unless no Scale was given or
// what is drawn is transformed past the paint's base, a Document's zoom,
// where pixels don't lie along r's edges.
func (p *painter) snap(r geom.Rect) geom.Rect {
	if p.snapScale() <= 0 {
		return r
	}
	xf := p.s.Transform()
	// k is device pixels per point of the space the base draws into: the
	// window's, when the base is a Document's zoom.
	k := p.scale / xf.X.X
	round := func(v, origin float32) float32 {
		return (float32(math.Round(float64((v*xf.X.X+origin)*k)))/k - origin) / xf.X.X
	}
	x0, y0 := round(r.X, xf.Origin.X), round(r.Y, xf.Origin.Y)
	return geom.Rect{X: x0, Y: y0, W: round(r.X+r.W, xf.Origin.X) - x0, H: round(r.Y+r.H, xf.Origin.Y) - y0}
}

// snapScale is how many device pixels a point of what is drawn takes, 0
// where snap leaves things as they are.
func (p *painter) snapScale() float32 {
	xf := p.s.Transform()
	if p.scale <= 0 || xf != p.base || xf.X.Y != 0 || xf.Y.X != 0 || xf.X.X <= 0 || xf.X.X != xf.Y.Y {
		return 0
	}
	return p.scale
}

func fade(c color.Color, alpha float32) color.Color { c.A *= alpha; return c }

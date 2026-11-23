package paint

import (
	"math"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var whole = geom.Rect{W: 1, H: 1}

// piece is one part of an element's border box: the whole box, or one
// line's fragment of a split inline, with the border edges it carries.
type piece struct {
	rect  geom.Rect
	edges box.Edges
}

// pieces are b's parts in the root's space, b's border box at at: one per
// line of an inline split over several, the left edge on the first and
// the right edge on the last; one per column of a block columns cut, the
// top edge on the first and the bottom edge on the last, as Chrome slices
// it; else the box itself. Lines past a line-clamp's last are left out, the
// last shown with no right edge, as Chrome draws them.
// They go in p's scratch, valid until the next call.
func (p *painter) pieces(b *box.Box, at geom.Vec2) []piece {
	shown := len(b.Fragments) - b.Hidden
	if b.Hidden > 0 && shown <= 0 {
		p.scratch = p.scratch[:0]
		return p.scratch
	}
	if len(b.Fragments) <= 1 {
		s := framed(b)
		p.scratch = append(p.scratch[:0], piece{geom.Rect{X: at.X, Y: at.Y + b.Wrap.Top, W: s.X, H: s.Y}, b.Border})
		return p.scratch
	}
	out := slices.Grow(p.scratch[:0], shown)[:shown]
	p.scratch = out
	for i, f := range b.Fragments[:shown] {
		e := b.Border
		switch first, last := i == 0, i == len(b.Fragments)-1; {
		case b.Broken:
			if !first {
				e.Top = 0
			}
			if !last {
				e.Bottom = 0
			}
		default:
			if !first {
				e.Left = 0
			}
			if !last {
				e.Right = 0
			}
		}
		out[i] = piece{geom.Rect{X: at.X + f.X, Y: at.Y + f.Y, W: f.W, H: f.H}, e}
	}
	return out
}

// legendGap is where a fieldset's border stops for the legend set on it, in the root's
// space.
func legendGap(b *box.Box, at geom.Vec2) (geom.Rect, bool) {
	if b.Wrap.Top <= 0 || b.Node == nil || b.Node.Tag() != dom.Fieldset {
		return geom.Rect{}, false
	}
	for _, c := range b.Children {
		if c.Kind == box.Element && c.Node != nil && c.Node.Tag() == dom.Legend {
			return geom.Rect{X: at.X + c.Pos.X, Y: at.Y + c.Pos.Y, W: c.Size.X, H: c.Size.Y}, true
		}
	}
	return geom.Rect{}, false
}

// around is r without the columns of gap above gap's bottom: what is left of it, right
// of it and below it.
func around(r, gap geom.Rect) []geom.Rect {
	right, bottom := gap.X+gap.W, gap.Y+gap.H
	return []geom.Rect{
		{X: r.X, Y: r.Y, W: gap.X - r.X, H: r.H},
		{X: right, Y: r.Y, W: r.X + r.W - right, H: r.H},
		{X: gap.X, Y: bottom, W: gap.W, H: r.Y + r.H - bottom},
	}
}

// rowBackground paints a table row's background behind each of its own
// cells, over a cell's whole box when it spans rows, as Chrome paints
// rows (CSS 2.1 §17.5.1): a row below never paints over a cell spanning
// into it from above, and the gaps border-spacing leaves show what's
// under the row.
func (p *painter) rowBackground(row *box.Box, at geom.Vec2, c color.Color) {
	for _, cell := range row.Children {
		p.fill(p.snap(geom.Rect{X: at.X + cell.Pos.X, Y: at.Y + cell.Pos.Y, W: cell.Size.X, H: cell.Size.Y}), radii{}, c)
	}
}

// groupBackground paints a row group's background behind the cells of
// its rows, as a row's.
func (p *painter) groupBackground(group *box.Box, at geom.Vec2, c color.Color) {
	for _, row := range group.Children {
		p.rowBackground(row, at.Add(row.Pos), c)
	}
}

// columnBackground paints the background of a table's column, or of a
// group of them, behind what its cells have between its sides: the
// layer under the rows' (CSS 2.1 §17.5.1). at is the column's place.
func (p *painter) columnBackground(col *box.Box, at geom.Vec2, c color.Color) {
	table, origin := col.Parent, at.Sub(col.Pos)
	if table != nil && table.Style != nil && table.Style.TablePart == style.TableColumnGroup {
		table, origin = table.Parent, origin.Sub(table.Pos)
	}
	if table == nil {
		return
	}
	cells := func(row *box.Box, rowAt geom.Vec2) {
		for _, cell := range row.Children {
			l, r := max(rowAt.X+cell.Pos.X, at.X), min(rowAt.X+cell.Pos.X+cell.Size.X, at.X+col.Size.X)
			if r > l {
				p.fill(p.snap(geom.Rect{X: l, Y: rowAt.Y + cell.Pos.Y, W: r - l, H: cell.Size.Y}), radii{}, c)
			}
		}
	}
	for _, k := range table.Children {
		switch partOf(k) {
		case style.TableRowGroup, style.TableHeaderGroup, style.TableFooterGroup:
			for _, row := range k.Children {
				cells(row, origin.Add(k.Pos).Add(row.Pos))
			}
		case style.TableRow, style.NoTablePart:
			cells(k, origin.Add(k.Pos))
		case style.TableCell, style.TableColumn, style.TableColumnGroup, style.TableCaption:
		}
	}
}

// hiddenEmpty reports a cell that empty-cells: hide leaves undrawn: its borders apart
// and nothing in it, white space aside; an element, even an empty one, is something.
func hiddenEmpty(b *box.Box, st *style.Computed) bool {
	if !st.EmptyCellsHide || partOf(b) != style.TableCell || sharesBorders(b) {
		return false
	}
	for _, c := range b.Children {
		if c.Kind != box.TextRun || len(c.Lines) > 0 {
			return false
		}
	}
	return b.Node == nil || !slices.ContainsFunc(b.Node.Children(), func(c *dom.Node) bool { return !c.IsText() })
}

// sharesBorders reports a table that collapses its borders, or a cell of one: the table
// draws them all, over its cells.
func sharesBorders(b *box.Box) bool {
	if b.Collapsed != nil {
		return true
	}
	if partOf(b) != style.TableCell {
		return false
	}
	for a := b.Parent; a != nil && a.Style != nil; a = a.Parent {
		if a.Style.Table {
			return a.Collapsed != nil
		}
	}
	return false
}

// collapsedBorders draws the borders of a table that collapses them (CSS 2.1 §17.6.2):
// each edge of its grid once, in the style that won it. A groove or an outset is dark
// then light across its width, a ridge or an inset light then dark, a double two lines
// as a box's.
func (p *painter) collapsedBorders(e entry, alpha float32) {
	if len(e.b.Collapsed) == 0 || p.probing || !shown(e.b).Visible {
		return
	}
	for _, ed := range e.b.Collapsed {
		r := p.snap(geom.Rect{X: e.at.X + ed.Rect.X, Y: e.at.Y + ed.Rect.Y, W: ed.Rect.W, H: ed.Rect.H})
		c := fade(ed.Color, alpha)
		if r.W <= 0 || r.H <= 0 || c.A <= 0 {
			continue
		}
		switch ed.Style {
		case style.BorderDashed, style.BorderDotted:
			p.dashes(r, ed.Across, ed.Style == style.BorderDotted, c)
		case style.BorderDouble:
			w := r.W
			if ed.Across {
				w = r.H
			}
			outer, inner, ok := doubleCut(w, p.maskScale())
			if !ok {
				p.s.Rect(r, c)
				continue
			}
			a, b := r, r
			if ed.Across {
				a.H, b.Y, b.H = outer, r.Y+inner, r.H-inner
			} else {
				a.W, b.X, b.W = outer, r.X+inner, r.W-inner
			}
			p.s.Rect(a, c)
			p.s.Rect(b, c)
		case style.BorderGroove, style.BorderOutset, style.BorderRidge, style.BorderInset:
			first, second := shades(c)
			if ed.Style == style.BorderRidge || ed.Style == style.BorderInset {
				first, second = second, first
			}
			a, b := r, r
			if ed.Across {
				a.H = r.H / 2
				b.Y, b.H = r.Y+a.H, r.H-a.H
			} else {
				a.W = r.W / 2
				b.X, b.W = r.X+a.W, r.W-a.W
			}
			p.s.Rect(a, first)
			p.s.Rect(b, second)
		case style.BorderNone, style.BorderAuto, style.BorderSolid:
			p.s.Rect(r, c)
		}
	}
}

// partOf is the part of a table b's element is shown as: none for an
// anonymous box, as the anonymous rows of a table are.
func partOf(b *box.Box) style.TablePart {
	if b.Kind != box.Element || b.Style == nil {
		return style.NoTablePart
	}
	return b.Style.TablePart
}

// background paints e's background colour, its row's for a table row,
// then its layers from the last to the first, over r rounded by radius
// with borders of edges: placed in strip, r itself but for a line of an
// inline, whose box is its lines end to end. A colour painted in the box
// and the layers over it share one clip to r, so those showing in all of
// it draw as one batch.
func (p *painter) background(e entry, r, strip geom.Rect, edges box.Edges, radius radii, st *style.Computed,
	alpha float32) {
	b := e.b
	n := st.BackgroundLayers()
	if st.Background.A > 0 && !tracked(b) && !paintsCells(st) && st.HasBackgroundImage() {
		p.s.PushClip(r)
		defer p.s.PopClip()
	}
	var text *twod.Picture
	if clipsText(st) {
		text = p.textMask(e)
	}
	mixing := p.beginLayers(st)
	defer mixing.end()
	if st.Background.A > 0 && !tracked(b) {
		switch {
		case st.TablePart == style.TableRow:
			p.rowBackground(b, e.at, fade(st.Background, alpha))
		case st.TablePart == style.TableRowGroup || st.TablePart == style.TableHeaderGroup ||
			st.TablePart == style.TableFooterGroup:
			p.groupBackground(b, e.at, fade(st.Background, alpha))
		case st.TablePart == style.TableColumn || st.TablePart == style.TableColumnGroup:
			p.columnBackground(b, e.at, fade(st.Background, alpha))
		case silhouetted(b, st):
			p.maskedFill(b, r, edges, radius, st, alpha)
		default:
			which := or(st.BackgroundLayer(n-1).Clip, style.UnderBorder)
			clip, rounding := backgroundBox(b, r, edges, radius, which)
			p.throughText(text, which, func() { p.fill(clip, rounding, fade(st.Background, alpha)) })
		}
	}
	for i := n - 1; i >= 0; i-- {
		if l := st.BackgroundLayer(i); l.Image.Source != "" || l.Image.Gradient != nil {
			mixing.layer(l.Blend, func() {
				p.throughText(text, l.Clip, func() { p.backgroundLayer(b, r, strip, edges, radius, &l, st, alpha) })
			})
		}
	}
}

// paintsCells reports whether st's element paints its background colour
// behind its cells, out of its own box: a table's row, row group, column
// or column group.
func paintsCells(st *style.Computed) bool {
	switch st.TablePart {
	case style.TableRow, style.TableRowGroup, style.TableHeaderGroup, style.TableFooterGroup, style.TableColumn,
		style.TableColumnGroup:
		return true
	case style.NoTablePart, style.TableCell, style.TableCaption:
	}
	return false
}

// backgroundLayer paints l, a layer of b's background, as background does.
func (p *painter) backgroundLayer(b *box.Box, r, strip geom.Rect, edges box.Edges, radius radii,
	l *style.BackgroundLayer, st *style.Computed, alpha float32) {
	view, rounding := backgroundBox(b, r, edges, radius, or(l.Clip, style.UnderBorder))
	area, _ := backgroundBox(b, strip, edges, radius, or(l.Origin, style.UnderPadding))
	if strip != r {
		area, _ = backgroundBox(b, strip, b.Border, radius, or(l.Origin, style.UnderPadding))
	}
	switch l.Attachment {
	case style.AttachFixed:
		if !transformed(b) {
			area = p.viewport
		}
	case style.AttachLocal:
		pad := inside(r, edges)
		view = intersect(view, pad)
		area.X, area.Y = area.X-b.Scroll.X, area.Y-b.Scroll.Y
		area.W, area.H = area.W+max(b.Content.X-pad.W, 0), area.H+max(b.Content.Y-pad.H, 0)
	case style.AttachScroll:
	}
	p.backgroundImage(view, area, rounding, l, st, alpha)
}

// transformed reports whether b or a box round it is turned, moved or
// scaled by its style, where Chrome places a fixed background in b's box
// as a scrolling one.
func transformed(b *box.Box) bool {
	for ; b != nil; b = b.Parent {
		if st := shown(b); b.Kind == box.Element && st != nil && len(st.Transform) > 0 {
			return true
		}
	}
	return false
}

func or(b, initial style.BackgroundBox) style.BackgroundBox {
	if b == style.InitialBox {
		return initial
	}
	return b
}

// backgroundBox is the box of b, whose border box is r with borders of
// edges and corners of radius, that which names, and its corners.
func backgroundBox(
	b *box.Box, r geom.Rect, edges box.Edges, radius radii, which style.BackgroundBox,
) (geom.Rect, radii) {
	switch which {
	case style.UnderPadding:
		return inside(r, edges), radius.inside(edges)
	case style.UnderContent:
		in := box.Edges{
			Top: edges.Top + b.Padding.Top, Right: edges.Right + b.Padding.Right,
			Bottom: edges.Bottom + b.Padding.Bottom, Left: edges.Left + b.Padding.Left,
		}
		return inside(r, in), radius.inside(in)
	case style.InitialBox, style.UnderBorder, style.FillBox, style.StrokeBox, style.ViewBox, style.NoClip,
		style.UnderText:
	}
	return r, radius
}

// decorate paints an element's shadows, background and border, on each
// of its pieces; a split inline's are square.
func (p *painter) decorate(e entry, alpha float32) {
	b, st := e.b, shown(e.b)
	if b.Kind == box.Viewport && p.probing {
		// The whole viewport takes the pointer, as a browser's canvas does
		// for the root element, so the wheel scrolls the page from anywhere.
		p.probed(b, geom.Rect{X: e.at.X, Y: e.at.Y, W: b.Size.X, H: b.Size.Y})
		return
	}
	if b.Kind != box.Element || !st.Visible || p.culled(e, inkOf(e, st)) {
		return
	}
	ps := p.pieces(b, e.at)
	if p.probing {
		for _, pc := range ps {
			p.probed(b, p.snap(pc.rect))
		}
		return
	}
	if hiddenEmpty(b, st) {
		return
	}
	radius := radiiOf(st, framed(b))
	if len(b.Fragments) > 1 {
		radius = radii{}
	}
	// The lines of an inline are slices of one box, its lines end to end, and its
	// background's picture is placed in that box: laid is how much of it the lines
	// before took.
	total, laid := float32(0), float32(0)
	if len(ps) > 1 && !b.Broken {
		for _, pc := range ps {
			total += pc.rect.W
		}
	}
	for _, pc := range ps {
		r := p.snap(pc.rect)
		for i := len(st.Shadows) - 1; i >= 0; i-- {
			if sh := st.Shadows[i]; !sh.Inset {
				p.shadow(r, radius, sh, alpha)
			}
		}
		if b != p.canvas {
			strip := r
			if total > 0 {
				strip = geom.Rect{X: r.X - laid, Y: r.Y, W: total, H: r.H}
				laid += pc.rect.W
			}
			p.background(e, r, strip, pc.edges, radius, st, alpha)
		}
		for i := len(st.Shadows) - 1; i >= 0; i-- {
			if sh := st.Shadows[i]; sh.Inset {
				p.insetShadow(inside(r, pc.edges), radius.inside(pc.edges), sh, alpha)
			}
		}
		if tex, k := p.borderImage(st.BorderImage, st); tex != nil {
			s := st.BorderImage.Slice
			p.picture(r, tex, style.Sides[float32]{Top: s.Top * k, Right: s.Right * k, Bottom: s.Bottom * k, Left: s.Left * k},
				imageWidths(r, st.BorderImageWidth, st.Border, s), st.BorderImageRepeat, st.BorderImage.Fill, alpha)
		} else if gap, ok := legendGap(b, e.at); ok {
			for _, clip := range around(r, p.snap(gap)) {
				p.s.PushClip(clip)
				p.border(r, pc.edges, radius, st, alpha)
				p.s.PopClip()
			}
		} else if !sharesBorders(b) {
			p.border(r, pc.edges, radius, st, alpha)
		}
	}
	p.columnRules(e, st, alpha)
	p.selection(e, alpha)
}

// columnRules draws a multi-column container's column-rule in the middle
// of each gap between two columns holding content, as high as the
// columns, row by row, as a border of its style would be drawn.
func (p *painter) columnRules(e entry, st *style.Computed, alpha float32) {
	c := e.b.Columns
	w := st.ColumnRuleWidth
	if c == nil || st.ColumnRuleStyle == style.BorderNone || w <= 0 {
		return
	}
	var rule style.Computed
	rule.BorderStyle.Left, rule.BorderColor.Left = st.ColumnRuleStyle, st.ColumnRuleColor
	switch st.ColumnRuleStyle {
	case style.BorderInset:
		rule.BorderStyle.Left = style.BorderRidge
	case style.BorderOutset:
		rule.BorderStyle.Left = style.BorderGroove
	case style.BorderNone, style.BorderAuto, style.BorderSolid, style.BorderDashed, style.BorderDotted, style.BorderGroove,
		style.BorderRidge, style.BorderDouble:
	}
	b := e.b
	left, top := e.at.X+b.Border.Left+b.Padding.Left, e.at.Y+b.Border.Top+b.Padding.Top
	right := e.at.X + b.Size.X - b.Border.Right - b.Padding.Right
	for _, r := range c.Rows {
		for k := 1; k < r.Count; k++ {
			x := left + float32(k)*(c.Width+c.Gap) - (c.Gap+w)/2
			if st.Dir == style.RTL {
				x = right - float32(k)*(c.Width+c.Gap) + (c.Gap-w)/2
			}
			p.border(p.snap(geom.Rect{X: x, Y: top + r.Top, W: w, H: r.Height}), box.Edges{Left: w}, radii{}, &rule, alpha)
		}
	}
}

// find is img's texture through Images, its pixelated copy when st's
// image-rendering says so.
func (p *painter) find(img style.Image, st *style.Computed) *twod.Texture {
	if img.Source == "" {
		return nil
	}
	if v := p.shapes(img.Source); v != nil {
		w, h := pixels(v.Intrinsic().Natural(geom.V2(defaultWidth, defaultHeight)), 1)
		return v.Texture(w, h, 1)
	}
	if p.image == nil {
		return nil
	}
	tex := p.image(img.Source)
	if tex == nil || st.ImageRendering != style.RenderPixelated {
		return tex
	}
	return pixelated(tex)
}

// borderImage is img's picture, a border image of st, and how many of
// its pixels make one of the picture's own: a picture of shapes is drawn
// at the screen's density, as Chrome rasterises it, its slices scaled to
// match.
func (p *painter) borderImage(img style.Image, st *style.Computed) (*twod.Texture, float32) {
	v := p.shapes(img.Source)
	if v == nil {
		return p.find(img, st), 1
	}
	natural := v.Intrinsic().Natural(geom.V2(defaultWidth, defaultHeight))
	tex := p.drawn(v, natural)
	if tex == nil || natural.X <= 0 {
		return nil, 1
	}
	return tex, tex.Size().X / natural.X
}

// defaultWidth and defaultHeight are the size CSS gives a picture that
// asks for none where no area sizes it.
const defaultWidth, defaultHeight = 300, 150

// shapes is the picture of shapes name names, nil for any other.
func (p *painter) shapes(name string) Vector {
	if name == "" || p.vector == nil {
		return nil
	}
	return p.vector(name)
}

// drawn is v's texture for a box of size points: as many device pixels
// as the box takes, so the picture is drawn once, at the size it shows.
func (p *painter) drawn(v Vector, size geom.Vec2) *twod.Texture {
	scale := p.scale
	if scale <= 0 {
		scale = 1
	}
	w, h := pixels(size, scale)
	return v.Texture(w, h, scale)
}

// pixels is size points in whole pixels at scale pixels a point, never
// under one.
func pixels(size geom.Vec2, scale float32) (int, int) {
	side := func(v float32) int { return max(int(math.Round(float64(v*scale))), 1) }
	return side(size.X), side(size.Y)
}

// inside is r less edges: a border box's padding box.
func inside(r geom.Rect, e box.Edges) geom.Rect {
	return geom.Rect{X: r.X + e.Left, Y: r.Y + e.Top, W: r.W - e.Horizontal(), H: r.H - e.Vertical()}
}

// maxTiles bounds how many copies of a repeated background image are
// drawn, so a tiny image over a big box can't flood the draw list.
const maxTiles = 4096

// backgroundImage draws l's picture, a picture or a gradient, at its
// background-size in area, the padding box rounded by radius: placed by
// background-position, tiled from there as background-repeat says,
// clipped to view. view is area but for the canvas of a viewport that
// scrolls, whose image is placed in the root element's box and tiles over
// the canvas.
func (p *painter) backgroundImage(view, area geom.Rect, radius radii, l *style.BackgroundLayer, st *style.Computed,
	alpha float32) {
	p.tiled(view, area, radius, l, st, alpha, fade(color.White, alpha), false)
}

// tiled draws l's picture tinted by tint, as its shape alone when shape
// is set: its alpha, whatever its colours. st is its element's style.
func (p *painter) tiled(view, area geom.Rect, radius radii, l *style.BackgroundLayer, st *style.Computed,
	alpha float32, tint color.Color, shape bool) {
	g := l.Image.Gradient
	var tex *twod.Texture
	var natural box.Intrinsic
	vec := p.shapes(l.Image.Source)
	switch {
	case g != nil:
	case vec != nil:
		natural = vec.Intrinsic()
	default:
		if tex = p.find(l.Image, st); tex == nil {
			return
		}
		if natural = box.Sized(tex.Size()); natural.Ratio <= 0 {
			return
		}
	}
	if area.W <= 0 || area.H <= 0 || view.W <= 0 || view.H <= 0 {
		return
	}
	size := tileSize(l.Size, natural, geom.V2(area.W, area.H), p.snapScale())
	if size.X <= 0 || size.Y <= 0 {
		return
	}
	rep := l.Repeat
	size = roundTile(size, rep, l.Size, area)
	pos := l.Position
	first := geom.Rect{
		X: place(pos.X, pos.FromRight, area.X, area.W, size.X),
		Y: place(pos.Y, pos.FromBottom, area.Y, area.H, size.Y),
		W: size.X, H: size.Y,
	}
	var gap geom.Vec2
	gap.X, rep.X, first.X = spaceTiles(rep.X, area.X, area.W, first.X, size.X)
	gap.Y, rep.Y, first.Y = spaceTiles(rep.Y, area.Y, area.H, first.Y, size.Y)
	if g != nil && first == area && view == area {
		p.gradient(area, radius, g, alpha)
		return
	}
	// Along an axis it doesn't repeat on, Chrome starts an image placed at
	// or after the area's start on a device pixel; one that doesn't repeat
	// at all also ends on the pixel nearest its length from there. One
	// starting before the area, or repeating, keeps its place.
	at := p.snap(first)
	alone := rep == style.NoRepeat
	if alone {
		at = p.snap(geom.Rect{X: at.X, Y: at.Y, W: first.W, H: first.H})
	} else {
		at.W, at.H = first.W, first.H
	}
	if rep.X == style.TileNoRepeat && first.X >= area.X {
		first.X, first.W = at.X, at.W
	}
	if rep.Y == style.TileNoRepeat && first.Y >= area.Y {
		first.Y, first.H = at.Y, at.H
	}
	if vec != nil {
		if tex = p.drawn(vec, geom.V2(first.W, first.H)); tex == nil {
			return
		}
	}
	if radius.zero() || alone && insideAll(corners(first), view, radius) {
		p.s.PushClip(view)
	} else {
		p.s.PushClipRounded(view, radius)
	}
	defer p.s.PopClip()
	if shape {
		tex = silhouette(tex)
	}
	backgroundTiles(view, first, rep, gap, func(tile geom.Rect) {
		if g != nil {
			p.gradient(tile, radii{}, g, alpha)
			return
		}
		p.s.Image(tile, tex, texels(tex), tint)
	})
}

// roundTile is size made a whole number of copies of the area along each
// axis rep rounds, the nearest; when one axis alone is, the other side
// follows to keep the tile's proportions unless background-size bs gives
// it.
func roundTile(size geom.Vec2, rep style.BackgroundRepeat, bs style.BackgroundSize, area geom.Rect) geom.Vec2 {
	whole := func(extent, tile float32) float32 {
		return extent / max(float32(math.Round(float64(extent/tile))), 1)
	}
	out := size
	if rep.X == style.TileRound {
		out.X = whole(area.W, size.X)
	}
	if rep.Y == style.TileRound {
		out.Y = whole(area.H, size.Y)
	}
	scaled := bs.Scale != style.ScaleNone
	switch {
	case rep.X == style.TileRound && rep.Y != style.TileRound && (scaled || bs.Height.Kind == style.Auto):
		out.Y = size.Y * out.X / size.X
	case rep.Y == style.TileRound && rep.X != style.TileRound && (scaled || bs.Width.Kind == style.Auto):
		out.X = size.X * out.Y / size.Y
	}
	return out
}

// spaceTiles lays copies of a tile along an axis of an area from origin of
// length extent as tiling t says, the first placed at start: for space,
// as many as fit whole, the first and the last at its ends, and it
// returns the room left between them, the tiling and the first's start;
// with fewer than two the picture is placed alone, as by no-repeat. Any
// other tiling is returned as it is.
func spaceTiles(t style.Tiling, origin, extent, start, tile float32) (float32, style.Tiling, float32) {
	if t != style.TileSpace {
		return 0, t, start
	}
	n := float32(math.Floor(float64(extent / tile)))
	if n < 2 {
		return 0, style.TileNoRepeat, start
	}
	return (extent - n*tile) / (n - 1), style.TileSpace, origin
}

// tileSize is the size a background image is drawn at in a padding box
// of size area, as background-size gives it. natural is what the picture
// asks to be sized by: its size and proportions for one of pixels,
// nothing for a gradient, any of a width, a height and proportions for
// one drawn from shapes. A side it doesn't ask for comes from the other
// through its proportions, else is the area's; with proportions alone
// auto is contain, and with nothing cover and contain are the area.
// Chrome rounds the side contain scales to a whole device pixel, px to a
// point, and so does this when px is known.
func tileSize(bs style.BackgroundSize, natural box.Intrinsic, area geom.Vec2, px float32) geom.Vec2 {
	whole := func(v float32) float32 {
		if px <= 0 {
			return v
		}
		return max(float32(math.Round(float64(v*px))), 1) / px
	}
	shape := natural.Proportions()
	contain := func(scale style.BackgroundScale) geom.Vec2 {
		kx, ky := area.X/shape.X, area.Y/shape.Y
		if (kx < ky) == (scale == style.ScaleContain) {
			h := shape.Y * kx
			if scale == style.ScaleContain {
				h = whole(h)
			}
			return geom.V2(area.X, h)
		}
		w := shape.X * ky
		if scale == style.ScaleContain {
			w = whole(w)
		}
		return geom.V2(w, area.Y)
	}
	shaped := shape.X > 0 && shape.Y > 0
	switch bs.Scale {
	case style.ScaleCover, style.ScaleContain:
		if !shaped {
			return area
		}
		return contain(bs.Scale)
	case style.ScaleNone:
	}
	w, fixedW := bs.Width.Resolve(area.X)
	h, fixedH := bs.Height.Resolve(area.Y)
	w, h = max(w, 0), max(h, 0)
	switch {
	case fixedW && fixedH:
	case fixedW && shaped:
		h = shape.Y * w / shape.X
	case fixedW:
		h = natural.Natural(area).Y
	case fixedH && shaped:
		w = shape.X * h / shape.Y
	case fixedH:
		w = natural.Natural(area).X
	case shaped && natural.Width <= 0 && natural.Height <= 0:
		return contain(style.ScaleContain)
	default:
		return natural.Natural(area)
	}
	return geom.V2(w, h)
}

// corners are r's four corners.
func corners(r geom.Rect) []geom.Vec2 {
	return []geom.Vec2{{X: r.X, Y: r.Y}, {X: r.X + r.W, Y: r.Y}, {X: r.X + r.W, Y: r.Y + r.H}, {X: r.X, Y: r.Y + r.H}}
}

// backgroundTiles calls draw with each copy of a background image over
// view: first alone when rep doesn't repeat, else it and its copies
// before and after, gap apart, along the axes rep repeats, up to
// maxTiles.
func backgroundTiles(view, first geom.Rect, rep style.BackgroundRepeat, gap geom.Vec2, draw func(tile geom.Rect)) {
	if rep == style.NoRepeat {
		draw(first)
		return
	}
	stepX, stepY := first.W+gap.X, first.H+gap.Y
	x0 := first.X - float32(math.Ceil(float64((first.X-view.X-seam)/stepX)))*stepX
	y0 := first.Y - float32(math.Ceil(float64((first.Y-view.Y-seam)/stepY)))*stepY
	x1, y1 := view.X+view.W, view.Y+view.H
	if rep.X == style.TileNoRepeat {
		x0, x1 = first.X, first.X+first.W
	}
	if rep.Y == style.TileNoRepeat {
		y0, y1 = first.Y, first.Y+first.H
	}
	n := 0
	for j := float32(0); y0+j*stepY < y1-seam; j++ {
		for i := float32(0); x0+i*stepX < x1-seam; i++ {
			if n++; n > maxTiles {
				return
			}
			draw(geom.Rect{X: x0 + i*stepX, Y: y0 + j*stepY, W: first.W, H: first.H})
		}
	}
}

// seam is how far short of a view's end a tile may start and not be
// drawn: the rounding a tile size that divides its area leaves.
const seam = 1e-3

// place is where an image of length img starts along an area from origin
// of length extent, at l from the start, or from the end when fromEnd: a
// percentage puts the image's point at the area's point.
func place(l style.Length, fromEnd bool, origin, extent, img float32) float32 {
	off := l.At(extent - img)
	if fromEnd {
		return origin + extent - img - off
	}
	return origin + off
}

// picture stretches tex over r, or when slice is set draws it cut by
// slice, its sides widths thick, its edges and centre laid by repeat.
func (p *painter) picture(
	r geom.Rect, tex *twod.Texture, slice, widths style.Sides[float32], repeat [2]style.ImageRepeat, fill bool,
	alpha float32,
) {
	tint := fade(color.White, alpha)
	if slice == (style.Sides[float32]{}) {
		p.s.Image(r, tex, whole, tint)
		return
	}
	p.ninePatch(r, tex, slice, widths, repeat, tint, fill)
}

// imageWidths is how thick a border image's sides are drawn over r, as
// border-image-width w gives them: times the border's width, a length, a
// share of r's side, or the slice's own size. When two opposite sides
// would overlap, all four shrink by the same factor until they meet.
func imageWidths(
	r geom.Rect, w style.Sides[style.ImageWidth], border, slice style.Sides[float32],
) style.Sides[float32] {
	side := func(iw style.ImageWidth, b, s, extent float32) float32 {
		switch iw.Kind {
		case style.WidthLength:
			return iw.V
		case style.WidthPercent:
			return iw.V * extent / 100
		case style.WidthAuto:
			return s
		case style.WidthTimes:
		}
		return iw.V * b
	}
	out := style.Sides[float32]{
		Top:    side(w.Top, border.Top, slice.Top, r.H),
		Right:  side(w.Right, border.Right, slice.Right, r.W),
		Bottom: side(w.Bottom, border.Bottom, slice.Bottom, r.H),
		Left:   side(w.Left, border.Left, slice.Left, r.W),
	}
	f := float32(1)
	if across := out.Left + out.Right; across > 0 {
		f = min(f, r.W/across)
	}
	if down := out.Top + out.Bottom; down > 0 {
		f = min(f, r.H/down)
	}
	if f < 1 {
		out = style.Sides[float32]{Top: out.Top * f, Right: out.Right * f, Bottom: out.Bottom * f, Left: out.Left * f}
	}
	return out
}

func (p *painter) fill(r geom.Rect, radius radii, c color.Color) {
	switch {
	case radius.zero():
		p.s.Rect(r, c)
	case radius.uniform():
		p.s.RoundRect(r, radius[0], c)
	default:
		p.s.RoundRectCorners(r, radius, c)
	}
}

// shadow paints an outer box-shadow: the box moved by its offset and
// grown by its spread, sharp, or blurred as CSS blurs it, a Gaussian of
// sigma half the blur on a mesh three sigmas round it; either way
// clipped to outside the border box r, as CSS draws it, so a
// translucent box never shows its own shadow.
func (p *painter) shadow(r geom.Rect, radius radii, sh style.Shadow, alpha float32) {
	c := fade(sh.Color, alpha)
	if c.A <= 0 {
		return
	}
	s := geom.Rect{X: r.X + sh.X - sh.Spread, Y: r.Y + sh.Y - sh.Spread, W: r.W + 2*sh.Spread, H: r.H + 2*sh.Spread}
	grown := radius.grown(sh.Spread)
	if sh.Blur <= 0 {
		if radius.zero() {
			for _, part := range outside(s, r) {
				p.s.Rect(part, c)
			}
			return
		}
		p.fill(s, grown, c)
		return
	}
	m := shadowMesh(shadowKey{
		size: geom.V2(s.W, s.H), rounding: grown, shadow: style.Shadow{Blur: sh.Blur}, color: c, outer: true,
		box: geom.Rect{X: r.X - s.X, Y: r.Y - s.Y, W: r.W, H: r.H}, boxRounding: radius,
	})
	xf := p.s.Transform()
	p.s.SetTransform(xf.Mul(geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: geom.V2(s.X, s.Y)}))
	p.s.TrianglesColored(m.points, m.colors, m.indices)
	p.s.SetTransform(xf)
}

// outside is what of a lies outside b, in up to four rectangles: the
// bands above and below b, then those beside it.
func outside(a, b geom.Rect) []geom.Rect {
	ix0, iy0 := max(a.X, b.X), max(a.Y, b.Y)
	ix1, iy1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if ix1 <= ix0 || iy1 <= iy0 {
		return []geom.Rect{a}
	}
	var out []geom.Rect
	for _, r := range []geom.Rect{
		{X: a.X, Y: a.Y, W: a.W, H: iy0 - a.Y},
		{X: a.X, Y: iy1, W: a.W, H: a.Y + a.H - iy1},
		{X: a.X, Y: iy0, W: ix0 - a.X, H: iy1 - iy0},
		{X: ix1, Y: iy0, W: a.X + a.W - ix1, H: iy1 - iy0},
	} {
		if r.W > 0 && r.H > 0 {
			out = append(out, r)
		}
	}
	return out
}

// gridMesh is a mesh over the grid of xs by ys, each point c with alpha
// times its colour's.
func gridMesh(xs, ys []float32, c color.Color, alpha func(x, y float32) float32) *mesh {
	m := &mesh{points: make([]geom.Vec2, 0, len(xs)*len(ys)), colors: make([]color.Color, 0, len(xs)*len(ys))}
	for _, y := range ys {
		for _, x := range xs {
			k := c
			k.A = c.A * alpha(x, y)
			m.points, m.colors = append(m.points, geom.V2(x, y)), append(m.colors, k)
		}
	}
	n := len(xs)
	m.indices = make([]int, 0, 6*(len(xs)-1)*(len(ys)-1))
	blank := func(t ...int) bool {
		for _, i := range t {
			if m.colors[i].A > 0 {
				return false
			}
		}
		return true
	}
	for j := 0; j+1 < len(ys); j++ {
		for i := 0; i+1 < n; i++ {
			a := j*n + i
			if !blank(a, a+1, a+n+1) {
				m.indices = append(m.indices, a, a+1, a+n+1)
			}
			if !blank(a, a+n+1, a+n) {
				m.indices = append(m.indices, a, a+n+1, a+n)
			}
		}
	}
	m.compact()
	return m
}

// compact keeps only the points m's triangles use, in their order.
func (m *mesh) compact() {
	at := make([]int, len(m.points))
	for i := range at {
		at[i] = -1
	}
	var points []geom.Vec2
	var colors []color.Color
	for j, i := range m.indices {
		if at[i] < 0 {
			at[i] = len(points)
			points, colors = append(points, m.points[i]), append(colors, m.colors[i])
		}
		m.indices[j] = at[i]
	}
	m.points, m.colors = points, colors
}

// insetShadow paints an inset box-shadow inside the padding box pad: all
// of it but a hole, pad moved by the offset and shrunk by the spread. A
// blurred one is CSS's Gaussian blur of that shape (sigma half the blur),
// exact because a rectangle's blur separates: 1 - gx(x)·gy(y), drawn as a
// mesh dense round the hole's edges. A rounded box's goes through
// roundedInset.
func (p *painter) insetShadow(pad geom.Rect, rounding radii, sh style.Shadow, alpha float32) {
	c := fade(sh.Color, alpha)
	if c.A <= 0 || pad.W <= 0 || pad.H <= 0 {
		return
	}
	if !rounding.zero() {
		p.roundedInset(pad, rounding, sh, c)
		return
	}
	hole := geom.Rect{
		X: pad.X + sh.X + sh.Spread, Y: pad.Y + sh.Y + sh.Spread,
		W: pad.W - 2*sh.Spread, H: pad.H - 2*sh.Spread,
	}
	hole = grow(hole, 0)
	p.s.PushClip(pad)
	defer p.s.PopClip()
	sigma := max(sh.Blur, 0) / 2
	if sigma <= 0 {
		for _, band := range []geom.Rect{
			{X: pad.X, Y: pad.Y, W: pad.W, H: hole.Y - pad.Y},
			{X: pad.X, Y: hole.Y + hole.H, W: pad.W, H: pad.Y + pad.H - hole.Y - hole.H},
			{X: pad.X, Y: hole.Y, W: hole.X - pad.X, H: hole.H},
			{X: hole.X + hole.W, Y: hole.Y, W: pad.X + pad.W - hole.X - hole.W, H: hole.H},
		} {
			if band.W > 0 && band.H > 0 {
				p.s.Rect(band, c)
			}
		}
		return
	}
	xs := blurNodes(pad.X, pad.X+pad.W, hole.X, hole.X+hole.W, sigma)
	ys := blurNodes(pad.Y, pad.Y+pad.H, hole.Y, hole.Y+hole.H, sigma)
	m := gridMesh(xs, ys, c, func(x, y float32) float32 {
		return 1 - blurredInterval(x, hole.X, hole.X+hole.W, sigma)*blurredInterval(y, hole.Y, hole.Y+hole.H, sigma)
	})
	p.s.TrianglesColored(m.points, m.colors, m.indices)
}

// blurOffsets are where, in sigmas from an edge, a blurred edge is sampled.
//
//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var blurOffsets = []float32{-3, -2, -1.5, -1, -0.5, 0, 0.5, 1, 1.5, 2, 3}

// blurNodes are the mesh's coordinates from lo to hi along one axis:
// its ends, and samples round the hole's edges a and b.
func blurNodes(lo, hi, a, b, sigma float32) []float32 {
	out := []float32{lo, hi}
	for _, e := range []float32{a, b} {
		for _, t := range blurOffsets {
			if v := e + t*sigma; v > lo && v < hi {
				out = append(out, v)
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// blurredInterval is the Gaussian blur, by sigma, of the interval [a, b]
// seen at x: 1 well inside it, 0 well outside.
func blurredInterval(x, a, b, sigma float32) float32 {
	cdf := func(v float32) float64 { return 0.5 * math.Erfc(-float64(v)/(float64(sigma)*math.Sqrt2)) }
	return float32(cdf(x-a) - cdf(x-b))
}

// roundedInset paints an inset shadow in a rounded padding box: its hole
// rounded by the box's radii less the spread, each point shaded by the
// Gaussian of its distance to the hole, sigma half the blur (a quarter
// point at least, an antialiased edge where there is no blur), on a mesh
// dense round the hole's edges and through the corners, its cells
// clipped to the rounded box so nothing shows past the corners.
func (p *painter) roundedInset(pad geom.Rect, rounding radii, sh style.Shadow, c color.Color) {
	m := shadowMesh(shadowKey{size: geom.V2(pad.W, pad.H), rounding: rounding, shadow: sh, color: c})
	if len(m.indices) == 0 {
		return
	}
	xf := p.s.Transform()
	p.s.SetTransform(xf.Mul(geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: geom.V2(pad.X, pad.Y)}))
	p.s.TrianglesColored(m.points, m.colors, m.indices)
	p.s.SetTransform(xf)
}

// shadowKey is what a blurred shadow's mesh depends on: its box's size,
// not where it is, so a moved or scrolled box keeps its mesh; outer for
// an outer shadow, whose box is the spread one.
type shadowKey struct {
	size     geom.Vec2
	rounding radii
	shadow   style.Shadow
	color    color.Color
	outer    bool
	// box and boxRounding are an outer shadow's border box, in its own
	// space, which clips it.
	box         geom.Rect
	boxRounding radii
}

type mesh struct {
	points  []geom.Vec2
	colors  []color.Color
	indices []int
}

// shadowMeshes keeps the meshes made, up to maxMeshes, then starts over.
//
//nolint:gochecknoglobals // a cache every paint shares, under its mutex
var (
	shadowMeshes   = map[shadowKey]*mesh{}
	shadowMeshesMu sync.Mutex
)

const maxMeshes = 256

// shadowMesh is the mesh of k's shadow in its box's own space, made once.
func shadowMesh(k shadowKey) *mesh {
	shadowMeshesMu.Lock()
	defer shadowMeshesMu.Unlock()
	if m, ok := shadowMeshes[k]; ok {
		return m
	}
	if len(shadowMeshes) >= maxMeshes {
		clear(shadowMeshes)
	}
	m := buildInsetMesh(k)
	if k.outer {
		m = buildOuterMesh(k)
	}
	shadowMeshes[k] = m
	return m
}

// buildOuterMesh shades each point by the Gaussian blur of the spread
// box: separable for a square one; for a rounded one exact across each
// row, its span's two edges, and summed down the rows within four
// sigmas, an eighth of a sigma apart. Each point is then cleared under
// the border box. Its nodes run three sigmas round the box, every sigma
// (a point and a half at least) through the shadow's corners and four
// sigmas past them, at the border box's edges and through its rounded
// corners every boxStep; nodes closer than minNodeGap merge, and the
// points no triangle uses go.
func buildOuterMesh(k shadowKey) *mesh {
	w, h := k.size.X, k.size.Y
	sigma := k.shadow.Blur / 2
	corner := max(k.rounding[0], k.rounding[1], k.rounding[2], k.rounding[3]) + 4*sigma
	step := max(sigma, 1.5)
	boxCorner := max(k.boxRounding[0], k.boxRounding[1], k.boxRounding[2], k.boxRounding[3])
	nodes := func(length, b0, b1 float32) []float32 {
		lo, hi := -3*sigma, length+3*sigma
		out := blurNodes(lo, hi, 0, length, sigma)
		if !k.rounding.zero() {
			for d := float32(0); d < corner; d += step {
				out = append(out, min(lo+d, hi), max(hi-d, lo))
			}
		}
		for d := boxStep(boxCorner); d < boxCorner; d += boxStep(boxCorner) {
			out = append(out, b0+d, b1-d)
		}
		out = slices.DeleteFunc(out, func(v float32) bool { return v < lo || v > hi })
		slices.Sort(out)
		out = thin(out, minNodeGap)
		out = append(out, b0-clipEdge, b0, b1, b1+clipEdge)
		out = slices.DeleteFunc(out, func(v float32) bool { return v < lo || v > hi })
		slices.Sort(out)
		return slices.Compact(out)
	}
	blur := func(x, y float32) float32 {
		return blurredInterval(x, 0, w, sigma) * blurredInterval(y, 0, h, sigma)
	}
	if !k.rounding.zero() {
		blur = func(x, y float32) float32 {
			lo, hi := max(y-4*sigma, 0), min(y+4*sigma, h)
			if hi <= lo {
				return 0
			}
			n := int(math.Ceil(float64((hi - lo) / (sigma / 8))))
			dt := (hi - lo) / float32(n)
			var sum float64
			for i := range n {
				t := lo + (float32(i)+0.5)*dt
				x0, x1 := span(t, w, h, k.rounding)
				g := float64((t - y) / sigma)
				sum += math.Exp(-g*g/2) * float64(blurredInterval(x, x0, x1, sigma))
			}
			return float32(sum * float64(dt/sigma) / math.Sqrt(2*math.Pi))
		}
	}
	xs, ys := nodes(w, k.box.X, k.box.X+k.box.W), nodes(h, k.box.Y, k.box.Y+k.box.H)
	return gridMesh(xs, ys, k.color, func(x, y float32) float32 {
		clip := min(max(roundedDistance(geom.V2(x, y), k.box, k.boxRounding)/clipEdge, 0), 1)
		if clip == 0 {
			return 0
		}
		return clip * blur(x, y)
	})
}

// minNodeGap is how close two mesh nodes along an axis may be, but for
// the border box's edges, kept exact: closer ones add vertices a linear
// blend doesn't need.
const minNodeGap = 0.75

// boxStep is the gap between nodes through a border box's rounded corner
// of radius r: a point and a half at most, about eight along the arc.
func boxStep(r float32) float32 { return min(max(r/8, 1), 1.5) }

// thin drops nodes closer than gap to the one kept before them, keeping
// the first and the last.
func thin(nodes []float32, gap float32) []float32 {
	if len(nodes) < 3 {
		return nodes
	}
	out := nodes[:1]
	for _, v := range nodes[1 : len(nodes)-1] {
		if v-out[len(out)-1] >= gap {
			out = append(out, v)
		}
	}
	return append(out, nodes[len(nodes)-1])
}

// clipEdge is how far past the border box an outer shadow's clip fades
// in: none to speak of, as Chrome's clip is hard on a box's pixel-aligned
// sides; along a rounded corner the nodes a point apart soften it.
const clipEdge = 1.0 / 1024

// span is where row y of a w×h box rounded by r lies across.
func span(y, w, h float32, r radii) (float32, float32) {
	inset := func(rad, d float32) float32 {
		rad = min(rad, w/2, h/2)
		if d >= rad {
			return 0
		}
		return rad - float32(math.Sqrt(float64(rad*rad-(rad-d)*(rad-d))))
	}
	x0 := max(inset(r[0], y), inset(r[3], h-y))
	x1 := w - max(inset(r[1], y), inset(r[2], h-y))
	return x0, x1
}

func buildInsetMesh(k shadowKey) *mesh {
	pad, rounding, sh, c := geom.Rect{W: k.size.X, H: k.size.Y}, k.rounding, k.shadow, k.color
	hole := grow(geom.Rect{
		X: pad.X + sh.X + sh.Spread, Y: pad.Y + sh.Y + sh.Spread,
		W: pad.W - 2*sh.Spread, H: pad.H - 2*sh.Spread,
	}, 0)
	holeRadii := rounding.grown(-sh.Spread)
	sigma := max(sh.Blur/2, 0.25)
	outer := max(rounding[0], rounding[1], rounding[2], rounding[3])
	inner := max(holeRadii[0], holeRadii[1], holeRadii[2], holeRadii[3]) + 2*sigma
	xs := cornerNodes(blurNodes(pad.X, pad.X+pad.W, hole.X, hole.X+hole.W, sigma),
		pad.X, pad.X+pad.W, outer, hole.X, hole.X+hole.W, inner)
	ys := cornerNodes(blurNodes(pad.Y, pad.Y+pad.H, hole.Y, hole.Y+hole.H, sigma),
		pad.Y, pad.Y+pad.H, outer, hole.Y, hole.Y+hole.H, inner)
	edge := roundedPolygon(pad, rounding)
	shade := func(q geom.Vec2) color.Color {
		k := c
		k.A = c.A * float32(0.5*math.Erfc(-float64(roundedDistance(q, hole, holeRadii)/sigma)/math.Sqrt2))
		return k
	}
	m := &mesh{}
	for j := 0; j+1 < len(ys); j++ {
		for i := 0; i+1 < len(xs); i++ {
			cell := []geom.Vec2{{X: xs[i], Y: ys[j]}, {X: xs[i+1], Y: ys[j]}, {X: xs[i+1], Y: ys[j+1]}, {X: xs[i], Y: ys[j+1]}}
			if !insideAll(cell, pad, rounding) {
				cell = clipConvex(cell, edge)
			}
			if len(cell) < 3 {
				continue
			}
			first := len(m.points)
			for _, q := range cell {
				m.points, m.colors = append(m.points, q), append(m.colors, shade(q))
			}
			for t := 1; t+1 < len(cell); t++ {
				m.indices = append(m.indices, first, first+t, first+t+1)
			}
		}
	}
	return m
}

// inside are the radii of the box edges e inside a box rounded by r: each
// corner less the wider of the two edges it joins, never below 0.
func (r radii) inside(e box.Edges) radii {
	widths := [4]float32{max(e.Left, e.Top), max(e.Right, e.Top), max(e.Right, e.Bottom), max(e.Left, e.Bottom)}
	for i := range r {
		r[i] = max(r[i]-widths[i], 0)
	}
	return r
}

// cornerNodes adds to nodes a point every point of length through the
// corner zones: within outer of lo and hi, and within inner of the hole's
// a and b.
func cornerNodes(nodes []float32, lo, hi, outer, a, b, inner float32) []float32 {
	for _, zone := range [][2]float32{{lo, lo + outer}, {hi - outer, hi}, {a, a + inner}, {b - inner, b}} {
		for v := zone[0]; v <= zone[1]; v++ {
			if v > lo && v < hi {
				nodes = append(nodes, v)
			}
		}
	}
	slices.Sort(nodes)
	return slices.Compact(nodes)
}

// roundedDistance is how far q lies outside r rounded by its radii, less
// than 0 inside.
func roundedDistance(q geom.Vec2, r geom.Rect, rounding radii) float32 {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	k := 0
	switch {
	case q.X >= cx && q.Y < cy:
		k = 1
	case q.X >= cx && q.Y >= cy:
		k = 2
	case q.X < cx && q.Y >= cy:
		k = 3
	}
	rad := min(rounding[k], r.W/2, r.H/2)
	d := geom.V2(geom.Abs(q.X-cx)-(r.W/2-rad), geom.Abs(q.Y-cy)-(r.H/2-rad))
	return geom.V2(max(d.X, 0), max(d.Y, 0)).Len() + min(max(d.X, d.Y), 0) - rad
}

// insideAll reports whether every point of cell lies in r rounded by its
// radii.
func insideAll(cell []geom.Vec2, r geom.Rect, rounding radii) bool {
	for _, q := range cell {
		if roundedDistance(q, r, rounding) > 0 {
			return false
		}
	}
	return true
}

// roundedPolygon is r's outline rounded by its radii, clockwise on screen,
// each arc in segments of about a point and a half.
func roundedPolygon(r geom.Rect, rounding radii) []geom.Vec2 {
	var out []geom.Vec2
	centres := [4]geom.Vec2{
		{X: r.X + rounding[0], Y: r.Y + rounding[0]},
		{X: r.X + r.W - rounding[1], Y: r.Y + rounding[1]},
		{X: r.X + r.W - rounding[2], Y: r.Y + r.H - rounding[2]},
		{X: r.X + rounding[3], Y: r.Y + r.H - rounding[3]},
	}
	for i, from := range [4]float64{math.Pi, 1.5 * math.Pi, 0, 0.5 * math.Pi} {
		rad := rounding[i]
		n := max(int(math.Ceil(float64(rad)/1.5)), 1)
		if rad <= 0 {
			n = 0
		}
		for k := 0; k <= n; k++ {
			a := from + math.Pi/2*float64(k)/float64(max(n, 1))
			out = append(out, geom.V2(centres[i].X+rad*float32(math.Cos(a)), centres[i].Y+rad*float32(math.Sin(a))))
		}
	}
	return out
}

// clipConvex is poly clipped to the convex polygon edge, both clockwise on
// screen.
func clipConvex(poly, edge []geom.Vec2) []geom.Vec2 {
	in := func(p, a, b geom.Vec2) bool { return (b.X-a.X)*(p.Y-a.Y)-(b.Y-a.Y)*(p.X-a.X) >= 0 }
	cross := func(p, q, a, b geom.Vec2) geom.Vec2 {
		d1, d2 := q.Sub(p), b.Sub(a)
		den := d1.X*d2.Y - d1.Y*d2.X
		if den == 0 {
			return p
		}
		t := ((a.X-p.X)*d2.Y - (a.Y-p.Y)*d2.X) / den
		return p.Add(d1.Scale(t))
	}
	out := poly
	for i := range edge {
		a, b := edge[i], edge[(i+1)%len(edge)]
		if a == b || len(out) == 0 {
			continue
		}
		src := out
		out = nil
		for j := range src {
			p, q := src[j], src[(j+1)%len(src)]
			switch pin, qin := in(p, a, b), in(q, a, b); {
			case pin && qin:
				out = append(out, q)
			case pin:
				out = append(out, cross(p, q, a, b))
			case qin:
				out = append(out, cross(p, q, a, b), q)
			}
		}
	}
	return out
}

// grow is r grown by d each side, or shrunk when d is negative, never
// past its centre.
func grow(r geom.Rect, d float32) geom.Rect {
	g := geom.Rect{X: r.X - d, Y: r.Y - d, W: r.W + 2*d, H: r.H + 2*d}
	if g.W < 0 {
		g.X, g.W = r.X+r.W/2, 0
	}
	if g.H < 0 {
		g.Y, g.H = r.Y+r.H/2, 0
	}
	return g
}

// border paints the border: one stroke when the box is rounded and its
// sides solid or double alike, any other rounded border side by side as
// Chrome draws it, else each side square, solid or dashed or dotted, solid
// sides of more than one colour mitered into each other.
func (p *painter) border(r geom.Rect, w box.Edges, radius radii, st *style.Computed, alpha float32) {
	side := func(width float32, s style.BorderStyle, c color.Color) (float32, color.Color) {
		if s == style.BorderNone || width <= 0 {
			return 0, color.Color{}
		}
		return width, fade(c, alpha)
	}
	tw, tc := side(w.Top, st.BorderStyle.Top, st.BorderColor.Top)
	rw, rc := side(w.Right, st.BorderStyle.Right, st.BorderColor.Right)
	bw, bc := side(w.Bottom, st.BorderStyle.Bottom, st.BorderColor.Bottom)
	lw, lc := side(w.Left, st.BorderStyle.Left, st.BorderColor.Left)
	styles := [4]style.BorderStyle{st.BorderStyle.Top, st.BorderStyle.Right, st.BorderStyle.Bottom, st.BorderStyle.Left}
	if !radius.zero() && !oneRing([4]float32{tw, rw, bw, lw}, [4]color.Color{tc, rc, bc, lc}, styles) {
		p.roundedBorder(r, [4]float32{tw, rw, bw, lw}, [4]color.Color{tc, rc, bc, lc}, styles, radius)
		return
	}
	if p.doubleBorder(r, [4]float32{tw, rw, bw, lw}, [4]color.Color{tc, rc, bc, lc}, radius, st.BorderStyle) {
		return
	}
	if beveled(st.BorderStyle) {
		// A groove or a ridge is two bevels, each half the width: the outer one, then
		// the inner one the other way.
		outer, inner, split := halves(st.BorderStyle)
		if !split {
			p.bevel(r, [4]float32{tw, rw, bw, lw}, [4]color.Color{tc, rc, bc, lc}, st.BorderStyle)
			return
		}
		half, rest := [4]float32{tw, rw, bw, lw}, [4]float32{}
		for i, s := range styles {
			if s == style.BorderGroove || s == style.BorderRidge {
				half[i] /= 2
				rest[i] = half[i]
			}
		}
		p.bevel(r, half, [4]color.Color{tc, rc, bc, lc}, outer)
		in := geom.Rect{X: r.X + half[3], Y: r.Y + half[0], W: r.W - half[1] - half[3], H: r.H - half[0] - half[2]}
		p.bevel(in, rest, [4]color.Color{tc, rc, bc, lc}, inner)
		return
	}
	if !radius.zero() {
		if tw > 0 && tc.A > 0 {
			p.ring(r, radius, tw, tc)
		}
		return
	}
	if mitered([4]float32{tw, rw, bw, lw}, [4]color.Color{tc, rc, bc, lc}, st.BorderStyle) {
		p.bevel(r, [4]float32{tw, rw, bw, lw}, [4]color.Color{tc, rc, bc, lc}, st.BorderStyle)
		return
	}
	for _, s := range []struct {
		rect   geom.Rect
		c      color.Color
		style  style.BorderStyle
		across bool
	}{
		{geom.Rect{X: r.X, Y: r.Y, W: r.W, H: tw}, tc, st.BorderStyle.Top, true},
		{geom.Rect{X: r.X + r.W - rw, Y: r.Y, W: rw, H: r.H}, rc, st.BorderStyle.Right, false},
		{geom.Rect{X: r.X, Y: r.Y + r.H - bw, W: r.W, H: bw}, bc, st.BorderStyle.Bottom, true},
		{geom.Rect{X: r.X, Y: r.Y, W: lw, H: r.H}, lc, st.BorderStyle.Left, false},
	} {
		if s.rect.W <= 0 || s.rect.H <= 0 || s.c.A <= 0 {
			continue
		}
		switch s.style {
		case style.BorderDashed, style.BorderDotted:
			p.dashes(s.rect, s.across, s.style == style.BorderDotted, s.c)
		case style.BorderNone, style.BorderAuto, style.BorderSolid, style.BorderInset, style.BorderOutset, style.BorderGroove,
			style.BorderRidge, style.BorderDouble:
			p.s.Rect(s.rect, s.c)
		}
	}
}

// mitered reports solid sides that must not lie over each other's
// corners: of more than one colour, or seen through.
func mitered(w [4]float32, c [4]color.Color, s style.Sides[style.BorderStyle]) bool {
	var first color.Color
	differ, sides := false, 0
	for i, x := range [4]style.BorderStyle{s.Top, s.Right, s.Bottom, s.Left} {
		if w[i] <= 0 {
			continue
		}
		if x == style.BorderDashed || x == style.BorderDotted {
			return false
		}
		if sides == 0 {
			first = c[i]
		}
		differ = differ || c[i] != first || c[i].A < 1
		sides++
	}
	return differ && sides > 1
}

// beveled reports whether a side is inset, outset, groove or ridge.
func beveled(s style.Sides[style.BorderStyle]) bool {
	for _, x := range [4]style.BorderStyle{s.Top, s.Right, s.Bottom, s.Left} {
		if x == style.BorderInset || x == style.BorderOutset || x == style.BorderGroove || x == style.BorderRidge {
			return true
		}
	}
	return false
}

// halves are the styles of a border's outer and inner halves, split when a side is a
// groove or a ridge: a groove is inset outside and outset inside, a ridge the other way.
func halves(s style.Sides[style.BorderStyle]) (style.Sides[style.BorderStyle], style.Sides[style.BorderStyle], bool) {
	var outer, inner style.Sides[style.BorderStyle]
	split := false
	half := func(x style.BorderStyle) (style.BorderStyle, style.BorderStyle) {
		switch x {
		case style.BorderGroove:
			split = true
			return style.BorderInset, style.BorderOutset
		case style.BorderRidge:
			split = true
			return style.BorderOutset, style.BorderInset
		case style.BorderNone, style.BorderAuto, style.BorderSolid, style.BorderDashed, style.BorderDotted, style.BorderInset,
			style.BorderOutset, style.BorderDouble:
		}
		return x, x
	}
	outer.Top, inner.Top = half(s.Top)
	outer.Right, inner.Right = half(s.Right)
	outer.Bottom, inner.Bottom = half(s.Bottom)
	outer.Left, inner.Left = half(s.Left)
	return outer, inner, split
}

// bevel paints the sides as Chrome does when one is inset or outset: each
// side a trapezoid mitered into its neighbours, an inset's top and left
// and an outset's bottom and right in their colour's darker shade, its
// other sides in the lighter one; any other style is drawn solid for now.
func (p *painter) bevel(r geom.Rect, w [4]float32, c [4]color.Color, st style.Sides[style.BorderStyle]) {
	styles := [4]style.BorderStyle{st.Top, st.Right, st.Bottom, st.Left}
	var whole [4][]stripe
	for i := range 4 {
		whole[i] = []stripe{{0, w[i]}}
		if styles[i] == style.BorderInset || styles[i] == style.BorderOutset {
			darker, lighter := shades(c[i])
			c[i] = lighter
			if (styles[i] == style.BorderInset) == (i == 0 || i == 3) {
				c[i] = darker
			}
		}
	}
	p.mitres(r, w, c, whole)
}

// shades are the colours Chromium gives an inset or outset side of colour
// c, its Color::Dark and Color::Light worked in encoded sRGB: scaled by
// (v - 0.33) / v and by min(1, v + 0.33) / v, v its brightest channel;
// black's are #545454 and #a8a8a8. A colour nearer white than #ebebeb is
// its own light shade, as the #eeeeee of an iframe's default border.
func shades(c color.Color) (color.Color, color.Color) {
	s := c.SRGB()
	v := max(s[0], s[1], s[2])
	if v <= 0 {
		return color.FromSRGB([3]float32{84.0 / 255, 84.0 / 255, 84.0 / 255}).Alpha(c.A),
			color.FromSRGB([3]float32{168.0 / 255, 168.0 / 255, 168.0 / 255}).Alpha(c.A)
	}
	scaled := func(k float32) color.Color {
		return color.FromSRGB([3]float32{s[0] * k, s[1] * k, s[2] * k}).Alpha(c.A)
	}
	light := scaled(min(1, v+0.33) / v)
	const lightest = 3 * (1 - 235.0/255) * (1 - 235.0/255)
	if d := (1-s[0])*(1-s[0]) + (1-s[1])*(1-s[1]) + (1-s[2])*(1-s[2]); d <= lightest {
		light = c
	}
	return scaled(max(0, (v-0.33)/v)), light
}

// dashes draws a dashed or dotted side over r, along X when across, else
// along Y, as Chrome strokes it, in device pixels: dashes 2 widths long
// with gaps of a width (3 and 2 below 3 pixels wide), as many as leave a
// gap nearest that, one flush at each end, at fractions of a pixel and
// smoothed there; two shortened alike on a side too short for more, and a
// solid side when even those don't fit. Dots up to 3 pixels wide are
// squares, one at each end and those between a width apart, centred;
// wider ones round, spread as dashes are. Sides run corner to corner, so
// corners join.
func (p *painter) dashes(r geom.Rect, across, dotted bool, c color.Color) {
	length, w := r.H, r.W
	if across {
		length, w = r.W, r.H
	}
	k := p.maskScale()
	pieces, round, ok := dashPattern(float32(math.Round(float64(length*k))), float32(math.Round(float64(w*k))), dotted)
	if !ok {
		p.s.Rect(r, c)
		return
	}
	for _, d := range pieces {
		if !across && !round {
			// Skia smooths a fill's edges across the scanlines at four
			// samples a pixel: a dash down a side starts and ends at a
			// quarter of one.
			end := quarter(d[0] + d[1])
			d[0] = quarter(d[0])
			d[1] = end - d[0]
		}
		at, n := d[0]/k, d[1]/k
		if round {
			centre := geom.V2(r.X+w/2, r.Y+at)
			if across {
				centre = geom.V2(r.X+at, r.Y+w/2)
			}
			p.s.Circle(centre, w/2, c)
			continue
		}
		piece := geom.Rect{X: r.X, Y: r.Y + at, W: w, H: n}
		if across {
			piece = geom.Rect{X: r.X + at, Y: r.Y, W: n, H: w}
		}
		p.s.RoundRect(piece, 0, c)
	}
}

// dashPattern is the dashes Chrome strokes along a side length long and
// w wide, both in whole device pixels: each one's start and length, or
// for round dots their centres; false for a solid side.
func dashPattern(length, w float32, dotted bool) ([][2]float32, bool, bool) {
	var pieces [][2]float32
	if dotted && w > 3 {
		if length < 2*w {
			return [][2]float32{{w / 2, 0}}, true, true
		}
		const epsilon = 0.01
		step := bestDashGap(length, w, w) + w - epsilon
		for at := w / 2; at <= length-w/2+epsilon; at += step {
			pieces = append(pieces, [2]float32{at, 0})
		}
		return pieces, true, true
	}
	dash, gap := w, w
	if !dotted {
		dash, gap = 3*w, 2*w
		if w >= 3 {
			dash, gap = 2*w, w
		}
	}
	if length <= 2*dash {
		return nil, false, false
	}
	if dotted {
		// A square at each end, and between them a width apart as many as
		// leave a width at least beside the end ones, as Chrome lays them:
		// a whole pattern of dot and gap from where the remainder of the
		// room after the first square ends, half a width in when there is
		// none.
		inner := length - 2*w
		pieces = append(pieces, [2]float32{0, w})
		first := float32(math.Mod(float64(inner), float64(2*w)))
		if first == 0 {
			first = w / 2
		}
		for at := w + first; at+w <= length-w-w/2; at += 2 * w {
			pieces = append(pieces, [2]float32{at, w})
		}
		return append(pieces, [2]float32{length - w, w}), false, true
	}
	if two := 2*dash + gap; length <= two {
		d := dash * length / two
		return [][2]float32{{0, d}, {length - d, d}}, false, true
	}
	gap = bestDashGap(length, dash, gap)
	for at := float32(0); at < length; at += dash + gap {
		pieces = append(pieces, [2]float32{at, min(dash, length-at)})
	}
	return pieces, false, true
}

// quarter is the edge at v, in device pixels, as four samples a pixel
// at the centres of its quarters see it.
func quarter(v float32) float32 { return float32(math.Ceil(float64(4*v-0.5))) / 4 }

// bestDashGap is the gap between dashes of length dash along an open side
// length long, as Chrome picks it: of the whole numbers of dashes on
// either side of what gap would fit, the one whose gap comes nearer it.
func bestDashGap(length, dash, gap float32) float32 {
	fewer := float32(math.Floor(float64((length + gap) / (dash + gap))))
	more := fewer + 1
	few, many := (length-fewer*dash)/(fewer-1), (length-more*dash)/(more-1)
	if many <= 0 || abs(few-gap) < abs(many-gap) {
		return few
	}
	return many
}

// outline paints an element's outline outside its border box: a solid or
// double one as rings, any other as a border of its style all round would
// be drawn there.
func (p *painter) outline(e entry, alpha float32) {
	b, st := e.b, shown(e.b)
	ring := grow(inkOf(e, st), outlineReach(st))
	if !outlined(b) || p.missed(e, ring) {
		return
	}
	p.cover(b, ring, true)
	c := fade(st.OutlineColor, alpha)
	if c.A <= 0 {
		return
	}
	grow := st.OutlineOffset + st.OutlineWidth
	ps := p.pieces(b, e.at)
	radius := radiiOf(st, b.Size).grown(grow)
	switch {
	case len(ps) == 0:
		return
	case len(b.Fragments) > 1:
		radius = radii{}
	default:
		ps[0].rect = geom.Rect{X: e.at.X, Y: e.at.Y, W: b.Size.X, H: b.Size.Y}
	}
	w := st.OutlineWidth
	lined := style.Computed{BorderStyle: style.All(st.OutlineStyle), BorderColor: style.All(c)}
	for _, pc := range ps {
		r := p.snap(pc.rect)
		out := geom.Rect{X: r.X - grow, Y: r.Y - grow, W: r.W + 2*grow, H: r.H + 2*grow}
		switch st.OutlineStyle {
		case style.BorderAuto:
			p.focusRing(r, radius.grown(-grow), w, st.OutlineOffset, c)
		case style.BorderDouble:
			if outer, inner, ok := doubleCut(w, p.maskScale()); ok {
				p.ring(out, radius, outer, c)
				gap := box.Edges{Top: inner, Right: inner, Bottom: inner, Left: inner}
				p.ring(inside(out, gap), radius.inside(gap), w-inner, c)
				continue
			}
			p.ring(out, radius, w, c)
		case style.BorderNone, style.BorderSolid:
			p.ring(out, radius, w, c)
		case style.BorderDashed, style.BorderDotted, style.BorderInset, style.BorderOutset, style.BorderGroove,
			style.BorderRidge:
			p.border(out, box.Edges{Top: w, Right: w, Bottom: w, Left: w}, radius, &lined, 1)
		}
	}
}

func outlined(b *box.Box) bool {
	st := shown(b)
	return b.Kind == box.Element && st.Visible && st.OutlineStyle != style.BorderNone && st.OutlineWidth > 0
}

// radii are a box's corner radii, top-left, top-right, bottom-right,
// bottom-left: each corner the smaller of its two radii, all scaled down
// together when two on one side would overlap, as CSS does.
type radii [4]float32

// framed is the size b's background and border frame: its box less a
// table's caption room, which its outline goes round as Chrome's does.
func framed(b *box.Box) geom.Vec2 { return geom.V2(b.Size.X, b.Size.Y-b.Wrap.Vertical()) }

func radiiOf(st *style.Computed, size geom.Vec2) radii {
	corner := func(c style.Size[style.Length]) float32 {
		w, okW := c.Width.Resolve(size.X)
		h, okH := c.Height.Resolve(size.Y)
		if !okW || !okH {
			return 0
		}
		return max(min(w, h), 0)
	}
	rc := st.Radius
	r := radii{corner(rc.TopLeft), corner(rc.TopRight), corner(rc.BottomRight), corner(rc.BottomLeft)}
	f := float32(1)
	sides := [][3]float32{{size.X, r[0], r[1]}, {size.X, r[3], r[2]}, {size.Y, r[0], r[3]}, {size.Y, r[1], r[2]}}
	for _, side := range sides {
		if sum := side[1] + side[2]; sum > side[0] && sum > 0 {
			f = min(f, side[0]/sum)
		}
	}
	for i := range r {
		r[i] *= f
	}
	return r
}

func (r radii) zero() bool    { return r == radii{} }
func (r radii) uniform() bool { return r[0] == r[1] && r[1] == r[2] && r[2] == r[3] }

// grown is r with every rounded corner grown by d, never below 0; square
// corners stay square.
func (r radii) grown(d float32) radii {
	for i := range r {
		if r[i] > 0 {
			r[i] = max(r[i]+d, 0)
		}
	}
	return r
}

// text paints a text run's fragments: those of a first line a ::first-line
// styles in its style, the others in the run's.
func (p *painter) text(e entry, alpha float32) {
	b, st := e.b, shown(e.b)
	if b.Kind != box.TextRun || len(b.Lines) == 0 || p.culled(e, textInk(e, st)) {
		return
	}
	if st.Mode.Vertical() {
		p.verticalText(e, alpha)
		return
	}
	p.horizontalText(e, alpha)
}

// horizontalText is text for a run of horizontal lines.
func (p *painter) horizontalText(e entry, alpha float32) {
	b, st := e.b, shown(e.b)
	var parts []Selected
	if p.sel != nil {
		parts = p.sel.Parts(b)
	}
	first := 0
	for first < len(b.Lines) && b.Lines[first].Style != nil {
		first++
	}
	if first == 0 {
		p.textLines(e, st, parts, alpha)
		return
	}
	head, rest := *b, *b
	head.Lines, rest.Lines = b.Lines[:first], b.Lines[first:]
	var headParts, restParts []Selected
	for _, s := range parts {
		if s.Line < first {
			headParts = append(headParts, s)
		} else {
			s.Line -= first
			restParts = append(restParts, s)
		}
	}
	at := e
	at.b = &head
	p.textLines(at, b.Lines[0].Style, headParts, alpha)
	if len(rest.Lines) > 0 {
		at.b = &rest
		p.textLines(at, st, restParts, alpha)
	}
}

// textLines paints the fragments of e's box in st, parts of them selected.
func (p *painter) textLines(e entry, st *style.Computed, parts []Selected, alpha float32) {
	b := e.b
	if !st.Visible {
		return
	}
	if p.glyphsOnly {
		white := color.White
		p.decorationsIn(e, st, style.Underline|style.Overline|style.LineThrough, 1, geom.Vec2{}, &white)
		opts := p.textOptions(p.faceOf(st.Font), st.Font.Size, st.LetterGap(), color.White, smoothingOf(st))
		for _, f := range b.Lines {
			p.fragment(geom.V2(e.at.X+f.Rect.X, e.at.Y+f.Baseline), &f, st, opts)
		}
		return
	}
	c := fade(st.Fill(), alpha)
	if placeholder(b) {
		c = fade(c, placeholderAlpha)
		if st.HasPlaceholder {
			c = fade(st.PlaceholderColor, alpha)
		}
	}
	ink := c
	if len(parts) > 0 {
		ink = p.highlight(e, st, parts, c, alpha)
	} else if st.HasSelectionText {
		// a field's selection is drawn behind it, its text in the colour here
		if parts = p.fieldParts(e); len(parts) > 0 {
			ink = fade(st.SelectionText, alpha)
		}
	}
	p.decorations(e, st, style.Underline|style.Overline, alpha)
	p.selectionDecorations(e, st, parts, style.Underline|style.Overline, alpha)
	stroked := st.TextStroke.Width > 0 || len(parts) > 0 && st.HasSelectionStroke
	if c.A <= 0 && ink == c && !stroked {
		p.decorations(e, st, style.LineThrough, alpha)
		p.selectionDecorations(e, st, parts, style.LineThrough, alpha)
		return
	}
	face := p.faceOf(st.Font)
	if len(st.TextShadows) > 0 || len(parts) > 0 && len(st.SelectionShadows) > 0 {
		p.runShadows(e, st, face, parts, alpha)
	}
	opts := p.textOptions(face, st.Font.Size, st.LetterGap(), c, smoothingOf(st))
	if ink != c {
		p.recoloured(e, st, face, parts, c, ink)
	} else if c.A > 0 {
		for _, f := range b.Lines {
			p.fragment(geom.V2(e.at.X+f.Rect.X, e.at.Y+f.Baseline), &f, st, opts)
		}
	}
	if stroked {
		p.strokes(e, st, face, parts, alpha)
	}
	p.decorations(e, st, style.LineThrough, alpha)
	p.selectionDecorations(e, st, parts, style.LineThrough, alpha)
}

// runShadows draws the text shadows of e's run, the selected parts' the
// ::selection's where it sets them.
func (p *painter) runShadows(e entry, st *style.Computed, face *twod.Face, parts []Selected, alpha float32) {
	for i, f := range e.b.Lines {
		k := -1
		if st.HasSelectionShadows {
			k = slices.IndexFunc(parts, func(s Selected) bool { return s.Line == i })
		}
		if k < 0 {
			p.shadows(e, st, face, f, st.TextShadows, st.Color, alpha)
			continue
		}
		for j, clip := range p.partClips(e, st, f, parts[k]) {
			shadows, current := st.TextShadows, st.Color
			if j == 1 {
				shadows, current = st.SelectionShadows, color.Color{}
			}
			if clip.W <= 0 || len(shadows) == 0 {
				continue
			}
			p.s.PushClip(clip)
			p.shadows(e, st, face, f, shadows, current, alpha)
			p.s.PopClip()
		}
	}
}

// strokes draws -webkit-text-stroke over the fill of e's run: the
// outline of each glyph, the selected parts' as the ::selection strokes
// them.
func (p *painter) strokes(e entry, st *style.Computed, face *twod.Face, parts []Selected, alpha float32) {
	plain := st.TextStroke
	plain.Color = fade(plain.Color, alpha)
	selected := plain
	if len(parts) > 0 {
		selected = p.selectionStroke(st, alpha)
	}
	draw := func(at geom.Vec2, f box.Fragment, s style.TextStroke) {
		if s.Width > 0 && s.Color.A > 0 {
			o := p.textOptions(face, st.Font.Size, st.LetterGap(), s.Color, smoothingOf(st))
			p.fragment(at, &f, st, append(slices.Clip(o), twod.TextStroke(s.Width)))
		}
	}
	for i, f := range e.b.Lines {
		at := geom.V2(e.at.X+f.Rect.X, e.at.Y+f.Baseline)
		k := slices.IndexFunc(parts, func(s Selected) bool { return s.Line == i })
		if k < 0 || plain == selected {
			draw(at, f, plain)
			continue
		}
		for j, clip := range p.partClips(e, st, f, parts[k]) {
			s := plain
			if j == 1 {
				s = selected
			}
			if clip.W > 0 {
				p.s.PushClip(clip)
				draw(at, f, s)
				p.s.PopClip()
			}
		}
	}
}

// selectionStroke is the stroke of the selected text: the ::selection's
// where it sets one, else the text's own, in the selection's colour
// where the stroke's is currentColor.
func (p *painter) selectionStroke(st *style.Computed, alpha float32) style.TextStroke {
	s := st.TextStroke
	switch {
	case st.HasSelectionStroke:
		s = st.SelectionStroke
	case st.HasSelectionText && st.CurrentColor&style.StrokeCurrent != 0:
		s.Color = st.SelectionText
	}
	s.Color = fade(s.Color, alpha)
	return s
}

// shadows draws the text shadows of fragment f of e's run, the last
// first, those in currentColor in current, else in their own colour.
func (p *painter) shadows(e entry, st *style.Computed, face *twod.Face, f box.Fragment, shadows []style.Shadow,
	current color.Color, alpha float32) {
	for i := len(shadows) - 1; i >= 0; i-- {
		sh := shadows[i]
		sc := sh.Color
		if sh.CurrentColor && current.A > 0 {
			sc = current
		}
		for _, t := range taps(sh.Blur) {
			tc := fade(fade(sc, alpha), t.weight)
			if tc.A <= 0 {
				continue
			}
			o := p.textOptions(face, st.Font.Size, st.LetterGap(), tc, smoothingOf(st))
			p.fragment(geom.V2(e.at.X+f.Rect.X+sh.X+t.at.X, e.at.Y+f.Baseline+sh.Y+t.at.Y), &f, st, o)
		}
	}
}

// selectionDecorations draws the ::selection's text-decoration lines
// among lines on the selected parts of e's run, skipping ink and casting
// the selection's shadows as the text's own lines do.
func (p *painter) selectionDecorations(e entry, st *style.Computed, parts []Selected, lines style.DecorationLine,
	alpha float32) {
	if len(parts) == 0 || !st.HasSelectionDecoration || st.SelectionDecoration.Line&lines == 0 {
		return
	}
	sel := *st
	sel.Decorations = []style.Decoration{{TextDecoration: st.SelectionDecoration, Font: st.Font}}
	if st.HasSelectionShadows {
		sel.TextShadows = st.SelectionShadows
	}
	for _, part := range parts {
		if part.Line < 0 {
			continue
		}
		f := e.b.Lines[part.Line]
		run := *e.b
		run.Lines = e.b.Lines[part.Line : part.Line+1]
		one := e
		one.b = &run
		p.s.PushClip(p.partClips(e, st, f, part)[1])
		p.decorations(one, &sel, lines, alpha)
		p.s.PopClip()
	}
}

// words draws text from at as opts draw it, each word separator widened
// by st's word-spacing: whole without one, else word by word, each where
// layout measured it.
func (p *painter) words(at geom.Vec2, text string, st *style.Computed, opts []twod.TextOption) {
	f := st.Font
	if c, ok := p.fonts.(capsFonts); ok {
		f = c.Caps(f, text)
	}
	runs := f.SmallCaps(text)
	if runs == nil {
		p.wordsIn(at, text, st, f, withFeatures(opts, f))
		return
	}
	p.around = [2]string{}
	for _, r := range runs {
		o := opts
		if r.Font.Size != st.Font.Size {
			o = append(slices.Clip(opts), twod.FontSize(r.Font.Size))
		}
		at.X = p.wordsIn(at, r.Text, st, r.Font, withFeatures(o, r.Font))
	}
}

// fragment draws fragment f of a line at at with words: text shown
// right to left in the order it is read, shaped so and laid out from the
// right, each piece small capitals or a word gap cut placed from where
// the fragment ends.
func (p *painter) fragment(at geom.Vec2, f *box.Fragment, st *style.Computed, opts []twod.TextOption) {
	p.around = [2]string{f.Before, f.After}
	defer func() { p.around = [2]string{} }()
	if !f.RTL {
		if f.Unspaced {
			// The space starting its block's text takes no word-spacing.
			at.X -= st.WordGap()
		}
		p.words(at, f.Text, st, opts)
		return
	}
	opts = append(slices.Clip(opts), twod.RightToLeft())
	font := st.Font
	if c, ok := p.fonts.(capsFonts); ok {
		font = c.Caps(font, f.Text)
	}
	runs := font.SmallCaps(f.Text)
	if runs == nil {
		runs = []style.CapsRun{{Text: f.Text, Font: font}}
	} else {
		p.around = [2]string{}
	}
	type piece struct {
		text   string
		opts   []twod.TextOption
		w, gap float32
	}
	var pieces []piece
	var width float32
	gap := st.WordGap()
	for _, r := range runs {
		o := opts
		if r.Font.Size != st.Font.Size {
			o = append(slices.Clip(opts), twod.FontSize(r.Font.Size))
		}
		o = withFeatures(o, r.Font)
		for at := 0; at < len(r.Text); {
			text := r.Text[at:]
			i, after := len(text), float32(0)
			if k := strings.IndexFunc(text, style.WordSeparator); gap != 0 && k >= 0 {
				_, n := utf8.DecodeRuneInString(text[k:])
				i, after = k+n, gap
			}
			oi, ctx := p.between(o, r.Text, at, at+i)
			pc := piece{text[:i], oi, p.advance(st, r.Font, text[:i], oi, ctx) + after, after}
			pieces = append(pieces, pc)
			width += pc.w
			at += i
		}
	}
	x := at.X + width
	for _, pc := range pieces {
		x -= pc.w
		p.s.Text(geom.V2(x+pc.gap, at.Y), pc.text, pc.opts...)
	}
}

// capsFonts are fonts that know which capitals a face draws itself
// (fonts.Set.Caps).
type capsFonts interface {
	Caps(f style.Font, text string) style.Font
}

// withFeatures is opts drawing text shaped with f's OpenType features.
func withFeatures(opts []twod.TextOption, f style.Font) []twod.TextOption {
	if f.Features == "" {
		return opts
	}
	return append(slices.Clip(opts), twod.Features(f.Features))
}

// wordsIn is words in font f, a part of st's text small capitals cut, and
// returns where the text after it starts.
func (p *painter) wordsIn(at geom.Vec2, text string, st *style.Computed, f style.Font, opts []twod.TextOption) float32 {
	gap := st.WordGap()
	if gap == 0 {
		o, ctx := p.between(opts, text, 0, len(text))
		p.s.Text(at, text, o...)
		return at.X + p.advance(st, f, text, o, ctx)
	}
	for from := 0; from < len(text); {
		rest := text[from:]
		i := strings.IndexFunc(rest, style.WordSeparator)
		if i < 0 {
			o, ctx := p.between(opts, text, from, len(text))
			p.s.Text(at, rest, o...)
			return at.X + p.advance(st, f, rest, o, ctx)
		}
		if i > 0 {
			o, ctx := p.between(opts, text, from, from+i)
			p.s.Text(at, rest[:i], o...)
			at.X += p.advance(st, f, rest[:i], o, ctx)
		}
		_, n := utf8.DecodeRuneInString(rest[i:])
		at.X += p.advance(st, f, rest[i:i+n], opts, [2]string{}) + gap
		from += i + n
	}
	return at.X
}

// between is opts shaping text[a:b] between the rest of text and the
// text round the fragment drawn, where the fragment has any, and that
// text before and after it.
func (p *painter) between(opts []twod.TextOption, text string, a, b int) ([]twod.TextOption, [2]string) {
	if p.around == [2]string{} {
		return opts, [2]string{}
	}
	ctx := [2]string{p.around[0] + text[:a], text[b:] + p.around[1]}
	return append(slices.Clip(opts), twod.Between(ctx[0], ctx[1])), ctx
}

// advance is how far s in f, between the text ctx holds, takes the text
// after it, its letters spaced: by layout's fonts, or without them by the
// face opts draw it in.
func (p *painter) advance(st *style.Computed, f style.Font, s string, opts []twod.TextOption, ctx [2]string) float32 {
	if p.fonts == nil {
		return twod.Measure(s, opts...).X
	}
	w := p.fonts.Advance(f, s)
	if cf, ok := p.fonts.(box.ContextFonts); ok && ctx != [2]string{} {
		w = cf.AdvanceIn(f, s, ctx[0], ctx[1])
	}
	return w + st.LetterGap()*float32(utf8.RuneCountInString(s))
}

// selectionColors are the ::selection of the element st styles: its
// background-color and its color. A rule that sets either leaves the
// other as CSS has it, no background and the text's own colour; with no
// rule they are the system's, its background blended with white as Chrome
// on macOS blends it, else Chrome's there, greyed while the window is
// inactive, and the text keeps its colour where the system gives it none.
func (p *painter) selectionColors(st *style.Computed, text color.Color, alpha float32) (color.Color, color.Color) {
	if !st.HasSelection && !st.HasSelectionText {
		back, sys := pageSelection, p.system.Background
		if p.inactive {
			back, sys = inactiveSelection, p.system.Inactive
		}
		if sys.A > 0 {
			back = blendWithWhite(sys)
		}
		if p.system.Text.A > 0 {
			text = fade(p.system.Text, alpha)
		}
		return fade(back, alpha), text
	}
	var back color.Color
	if st.HasSelection {
		back = fade(st.SelectionColor, alpha)
	}
	if st.HasSelectionText {
		text = fade(st.SelectionText, alpha)
	}
	return back, text
}

// blendWithWhite is c made as translucent as it can be, from 60% to 80%
// opaque, and still show as c over white, as Chrome draws a system's
// selection colour (Color::BlendWithWhite): a colour too dark for that
// keeps 80%, its channels clamped. A translucent colour stays as it is.
func blendWithWhite(c color.Color) color.Color {
	if c.A < 1 {
		return c
	}
	srgb := c.SRGB()
	var ch [3]int
	alpha := 153
	for ; alpha <= 204; alpha += 17 {
		a := float32(alpha) / 255
		negative := false
		for i, v := range srgb {
			ch[i] = int(float32(int(v*255+0.5)-(255-alpha)) / a)
			negative = negative || ch[i] < 0
		}
		if !negative {
			break
		}
	}
	alpha = min(alpha, 204)
	return color.RGB8(clamp8(ch[0]), clamp8(ch[1]), clamp8(ch[2])).Alpha(float32(alpha) / 255)
}

func clamp8(v int) uint8 { return uint8(max(0, min(255, v))) }

// highlight draws the selection behind a text run's selected parts and
// returns the colour their text takes.
func (p *painter) highlight(e entry, st *style.Computed, parts []Selected, text color.Color,
	alpha float32) color.Color {
	back, ink := p.selectionColors(st, text, alpha)
	if back.A > 0 {
		for _, s := range parts {
			p.s.Rect(p.snap(geom.Rect{X: e.at.X + s.Rect.X, Y: e.at.Y + s.Rect.Y, W: s.Rect.W, H: s.Rect.H}), back)
		}
	}
	return ink
}

// recoloured draws a text run whose selected parts take the colour ink:
// each line's text in c outside its part and in ink inside it.
func (p *painter) recoloured(e entry, st *style.Computed, face *twod.Face, parts []Selected, c, ink color.Color) {
	plain := p.textOptions(face, st.Font.Size, st.LetterGap(), c, smoothingOf(st))
	for i, f := range e.b.Lines {
		at := geom.V2(e.at.X+f.Rect.X, e.at.Y+f.Baseline)
		k := slices.IndexFunc(parts, func(s Selected) bool { return s.Line == i })
		if k < 0 {
			p.fragment(at, &f, st, plain)
			continue
		}
		for j, clip := range p.partClips(e, st, f, parts[k]) {
			drawn := c
			if j == 1 {
				drawn = ink
			}
			if clip.W <= 0 || drawn.A <= 0 {
				continue
			}
			p.s.PushClip(clip)
			p.fragment(at, &f, st, p.textOptions(face, st.Font.Size, st.LetterGap(), drawn, smoothingOf(st)))
			p.s.PopClip()
		}
	}
}

// partClips are the clips the selected part of fragment f of e's run
// cuts its line in: before the part, the part, and after it, each with
// room above and below for what glyphs draw outside the line, the part
// taking what its glyphs draw past the fragment's start or end where it
// reaches them.
func (p *painter) partClips(e entry, st *style.Computed, f box.Fragment, part Selected) [3]geom.Rect {
	reach := st.Font.Size
	line := geom.Rect{X: e.at.X + f.Rect.X - reach, Y: e.at.Y + f.Rect.Y - reach, W: f.Rect.W + 2*reach,
		H: f.Rect.H + 2*reach}
	in := line
	const near = 0.01
	if left := e.at.X + part.Rect.X; left > e.at.X+f.Rect.X+near {
		in.W -= left - in.X
		in.X = left
	}
	if right := e.at.X + part.Rect.X + part.Rect.W; right < e.at.X+f.Rect.X+f.Rect.W-near {
		in.W = right - in.X
	}
	return [3]geom.Rect{
		{X: line.X, Y: line.Y, W: in.X - line.X, H: line.H},
		in,
		{X: in.X + in.W, Y: line.Y, W: line.X + line.W - in.X - in.W, H: line.H},
	}
}

// textKey is what a text run's drawing options depend on.
type textKey struct {
	face           *twod.Face
	size, tracking float32
	color          color.Color
	smoothing      smoothing
}

// smoothing is a text's -webkit-font-smoothing: its stems darkened as
// Chrome's by default, at their exact coverage, or unsmoothed.
type smoothing uint8

const (
	smoothDarkened smoothing = iota
	smoothExact
	smoothNone
)

func smoothingOf(st *style.Computed) smoothing {
	switch {
	case st.PixelText:
		return smoothNone
	case st.ExactText:
		return smoothExact
	}
	return smoothDarkened
}

// textOpts are the options of the texts drawn without a face. A face's
// are kept with the face (draw.Kept), each made once so a frame boxes
// none, up to maxTextOpts a face, then starting over: a Document's faces
// and their atlases go when it does, where one cache for every face held
// them.
//
//nolint:gochecknoglobals // a cache every paint shares, under its mutex
var (
	textOpts   any
	textOptsMu sync.Mutex
)

const maxTextOpts = 4096

// textOptions are textOptions', the last ones kept on p, as runs in a
// row mostly share their look.
func (p *painter) textOptions(face *twod.Face, size, tracking float32, c color.Color, how smoothing) []twod.TextOption {
	k := textKey{face, size, tracking, c, how}
	if p.lastText.opts == nil || p.lastText.key != k {
		p.lastText.key, p.lastText.opts = k, textOptions(face, size, tracking, c, how)
	}
	return p.lastText.opts
}

// textOptions are the options that draw text in face at size, tracked,
// in c, from its baseline, smoothed as how says.
func textOptions(face *twod.Face, size, tracking float32, c color.Color, how smoothing) []twod.TextOption {
	k := textKey{face, size, tracking, c, how}
	textOptsMu.Lock()
	defer textOptsMu.Unlock()
	kept := &textOpts
	if of := draw.Kept(face); of != nil {
		kept = of
	}
	made, _ := (*kept).(map[textKey][]twod.TextOption)
	if o, ok := made[k]; ok {
		return o
	}
	if made == nil || len(made) >= maxTextOpts {
		made = map[textKey][]twod.TextOption{}
		*kept = made
	}
	o := smoothed([]twod.TextOption{
		twod.Typeface(face), twod.FontSize(size), twod.Tracking(tracking), twod.Baseline(), twod.Color(c),
	}, how)
	made[k] = o
	return o
}

// smoothed is o drawing its glyphs as how says.
func smoothed(o []twod.TextOption, how smoothing) []twod.TextOption {
	switch how {
	case smoothNone:
		return append(o, twod.Pixelated())
	case smoothExact:
		return append(o, twod.ExactCoverage())
	case smoothDarkened:
	}
	return o
}

type tap struct {
	at     geom.Vec2
	weight float32
}

// taps share a text shadow's ink out over copies of the text: one for a
// sharp shadow, else a grid over ±1.5 sigma (half the blur) weighted by
// the Gaussian, so a blur is drawn without blurring glyphs.
func taps(blur float32) []tap {
	if blur <= 0 {
		return sharpTap
	}
	tapsMu.Lock()
	defer tapsMu.Unlock()
	if t, ok := blurTaps[blur]; ok {
		return t
	}
	t := blurredTaps(blur)
	blurTaps[blur] = t
	return t
}

// sharpTap is a sharp shadow's one tap; blurTaps keeps a blurred one's, by
// its blur.
//
//nolint:gochecknoglobals // a cache every paint shares, under its mutex
var (
	sharpTap = []tap{{weight: 1}}
	blurTaps = map[float32][]tap{}
	tapsMu   sync.Mutex
)

func blurredTaps(blur float32) []tap {
	sigma := blur / 2
	k := int(min(max(float32(math.Ceil(float64(sigma))), 1), 3))
	step := 1.5 * sigma / float32(k)
	var out []tap
	var sum float32
	for y := -k; y <= k; y++ {
		for x := -k; x <= k; x++ {
			at := geom.V2(float32(x)*step, float32(y)*step)
			w := geom.Exp(-(at.X*at.X + at.Y*at.Y) / (2 * sigma * sigma))
			out = append(out, tap{at, w})
			sum += w
		}
	}
	for i := range out {
		out[i].weight /= sum
	}
	return out
}

// faceOf is the face that draws f: Faces', else Go.
func (p *painter) faceOf(f style.Font) *twod.Face {
	if p.face != nil {
		if face := p.face(f); face != nil {
			return face
		}
	}
	return twod.Regular()
}

// placeholderAlpha is what a placeholder keeps of its colour when no
// ::placeholder rule gives it one.
const placeholderAlpha = 0.54

// placeholder reports whether run is the placeholder its element shows.
func placeholder(run *box.Box) bool {
	if run.Node == nil {
		return false
	}
	for a := run.Parent; a != nil; a = a.Parent {
		if a.Kind == box.Element && a.Node != nil {
			d := dom.Data(a.Node)
			return d.Placeholder && d.Shadow == run.Node
		}
	}
	return false
}

// transform is the element's transform in the root's space: its
// functions in order, about its origin.
func transform(st *style.Computed, at, size geom.Vec2) geom.Transform2D {
	ox, _ := st.TransformOrigin.Width.Resolve(size.X)
	oy, _ := st.TransformOrigin.Height.Resolve(size.Y)
	o := at.Add(geom.V2(ox, oy))
	m := geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: o}
	for _, op := range st.Transform {
		m = m.Mul(transformOp(op, size))
	}
	return m.Mul(geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: o.Scale(-1)})
}

func transformOp(op style.TransformOp, size geom.Vec2) geom.Transform2D {
	a := op.A
	switch op.Kind {
	case style.Translate:
		x, _ := op.X.Resolve(size.X)
		y, _ := op.Y.Resolve(size.Y)
		return geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: geom.V2(x, y)}
	case style.ScaleOp:
		return geom.Transform2D{X: geom.V2(a[0], 0), Y: geom.V2(0, a[1])}
	case style.Rotate:
		return geom.Transform2DFrom(geom.Vec2{}, a[0], 0, geom.V2(1, 1))
	case style.Skew:
		return geom.Transform2D{X: geom.V2(1, geom.Tan(a[1])), Y: geom.V2(geom.Tan(a[0]), 1)}
	case style.Matrix:
		return geom.Transform2D{X: geom.V2(a[0], a[1]), Y: geom.V2(a[2], a[3]), Origin: geom.V2(a[4], a[5])}
	}
	return geom.Identity2D()
}

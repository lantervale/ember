package ui

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/assets"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/fonts"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/layout"
	"github.com/lantervale/ember/ui/paint"
)

// Document is one window's tree, from its html element down. Added to a
// window, it lays out at the window's size and draws over the 2D world
// each frame.
type Document struct {
	twod.Item
	// zoom draws the Document that many times its size (Zoom); 0 is 1.
	zoom float32
	// drawn is the World frame d was last drawn in, and fresh whether this
	// frame is the first after frames it wasn't: keys pressed then were
	// pressed before it showed, so they are the game's (drawnIn).
	drawn uint64
	// typing is the text field the player has typed in since it last
	// changed or was submitted: SetValue leaves it be while it has the
	// focus, and sets it again once its text is committed.
	typing *Node
	// answering is the element whose callbacks are running, whose value a
	// SetValue sets though it is typed in.
	answering  *Node
	drawnOnce  bool
	fresh      bool
	html, body *Node
	tree       *box.Tree
	fonts      *fonts.Set
	faces      map[*fonts.Face]*twod.Face
	// authored are the stylesheets in cascade order, sheets what the
	// cascade reads of those not disabled, and resheet is set once one
	// changed: the next frame takes them again (take).
	authored []*Sheet
	sheets   []*css.Sheet
	resheet  bool
	warns    []error
	warned   map[string]bool
	// files are the stylesheets' file systems, where their images are;
	// images the textures decoded from them.
	files  []fs.FS
	images map[string]*twod.Texture
	// vectors are the pictures drawn from shapes among them (vector),
	// painted counts the paints, which they age their sizes by, and
	// unsettled is whether one asked the last paint for a size it waits
	// to see asked again before drawing it.
	vectors   map[string]*vector
	painted   uint64
	unsettled bool
	// fetch gets the pictures files don't hold (FetchImages); fetching
	// are those on their way.
	fetch    ImageFetcher
	fetching fetches
	choices  choices
	// playing are the animated pictures, by url.
	playing map[string]*playing
	// pageFonts are the fonts of its @font-face rules that are fetched
	// (FetchFonts).
	pageFonts pageFonts
	// pageSheets are the sheets its @import rules name (FetchSheets).
	pageSheets pageSheets
	trans      *css.Transitions
	emoji      *twod.EmojiSet
	// clock is the World's real time at the last Draw, -1 before it.
	clock float32
	// systemFonts is whether font-family names the machine's fonts
	// (SystemFonts).
	systemFonts bool
	// rightDragSelects is whether a drag of the right button selects text.
	rightDragSelects bool
	// prefs are the system's preferences the Document follows, scheme and
	// motion the program's answers over them, unknown for none, and touch
	// whether the pointer last used is a finger on the screen (media.go).
	prefs  *platform.Preferences
	scheme platform.Appearance
	motion platform.Motion
	touch  bool
	// overlayBars is whether scrollbars lie over the content
	// (OverlayScrollbars).
	overlayBars bool
	// scrollingViewport is whether the viewport scrolls
	// (ScrollingViewport).
	scrollingViewport bool
	// browserSheets is whether the sheets and style attributes are read
	// as a browser reads them (BrowserSheets), and loading the sheets
	// the options gave, read once every option has spoken.
	browserSheets bool
	loading       []func()
	// viewport is the canvas: the window, or the host's area.
	viewport geom.Rect
	// laidAt and laidScale are the viewport and font scale tree was laid
	// out at.
	laidAt    geom.Vec2
	laidScale float32
	// kept carries layout's nodes between layouts, so a change lays out only its path.
	kept layout.Retained
	interaction
	validity
	replay replay
	// placed counts the nodes Layout pointed at their boxes, for tests.
	placed int
	// translator shows the text, said remembers what it said, and relayout
	// lays everything out again after it changed.
	translator Translator
	said       map[string]string
	relayout   bool
	// tips are the tooltips the last layout showed; placedTips those
	// placeTips has looked at since they showed.
	tips       []*Node
	placedTips map[*Node]bool
	// followers are the nodes following an entity that the last layout
	// placed, looked for again once followings moved past followingsSeen.
	followers      []*Node
	followingsSeen uint64
	// framed is where d shows inside another Document, nil in a window;
	// frames are d's iframes that show a Document, framedAt the one each
	// showed at the last frame; shield is whether the pointer is kept
	// from d this frame, and idle the input it then reads (frame.go).
	framed   *framing
	frames   []*Node
	framedAt []*Document
	shield   bool
	idle     *input.State
	// wheel is how many times the wheel's turn d scrolls by (wheel.go).
	wheel float32
}

// DocumentOption configures a Document.
type DocumentOption func(*Document)

func NewDocument(opts ...DocumentOption) *Document {
	d := &Document{
		Item:  twod.Base(),
		html:  dom.NewElement(dom.HTML),
		body:  dom.NewElement(dom.Body),
		fonts: fonts.Builtin(),
		trans: css.NewTransitions(),
		clock: -1,
		wheel: 1,
	}
	d.html.Append(d.body)
	d.trans.MeasureFonts(d.fontUnits)
	twod.Build(d, "document", twod.Overlay())
	for _, o := range opts {
		o(d)
	}
	for _, load := range d.loading {
		load()
	}
	d.loading = nil
	return d
}

// BrowserSheets reads the Document's stylesheets and style attributes as
// a browser reads a page's, for sheets written for browsers, as the web
// explorer shows. Their @supports rules apply where ui supports what
// they ask, their @layer rules are cascade layers, a width is its
// content's (box-sizing: content-box) but for the controls a browser
// sizes by their border, and each vendor's prefixed names are read as
// Chrome reads them: a -webkit- name that is another name of a plain one
// is that one, where it stands in the cascade, and what Chrome drops
// (-moz-, -ms-, -o-, -khtml-) goes without a warning. The guide's CSS
// page has each vendor's table. What ui still can't read keeps its
// warning. Without the option each of these is a warning, so a game's
// own sheet says what it gets wrong, and a width is the border box's.
// It reads every sheet of the Document, wherever it stands among the
// options.
func BrowserSheets() DocumentOption {
	return func(d *Document) {
		d.browserSheets = true
		d.trans.BrowserSheets()
	}
}

// BrowserSheets reports whether the Document reads its sheets as a
// browser does.
func (d *Document) BrowserSheets() bool { return d.browserSheets }

// parsing are the css options the Document reads a sheet with.
func (d *Document) parsing(opts ...css.ParseOption) []css.ParseOption {
	if d.browserSheets {
		opts = append(opts, css.Browser())
	}
	return opts
}

// OverlayScrollbars draws every scrollbar of the Document over the
// content it scrolls, in no room of its own, as macOS and phones draw
// theirs: a box that scrolls keeps its whole width for its content,
// where a classic bar takes 15 points of it. The bars work as classic
// ones do, and show whenever the content overflows: a dark thumb with a
// light edge, seen on a page of any colour. It is for showing pages laid
// out for such bars, as the web explorer does; CSS has no word for it.
func OverlayScrollbars() DocumentOption {
	return func(d *Document) {
		d.overlayBars = true
		d.trans.OverlayScrollbars()
	}
}

// OverlayScrollbars reports whether scrollbars lie over the content.
func (d *Document) OverlayScrollbars() bool { return d.overlayBars }

// ScrollingViewport has the Document's viewport scroll, as a browser's
// does: html keeps its own height, the page's, and what of it overflows
// the window, or the area of the element hosting the Document, scrolls
// there by the wheel, the keys and a scrollbar, as in any box that
// scrolls. The viewport takes html's overflow, or body's while html's is
// visible, as CSS has it, so html { overflow: hidden } leaves it still
// and no bar. A position: fixed box stays put in the window, a sticky
// one sticks to it, and percentages of html and body are still of the
// window. The canvas background, html's or body's, covers the window,
// its picture placed in html's box, so it scrolls with the page. The
// whole viewport then takes the pointer. Without the option a window's
// Document never scrolls as a whole and html is the root box: a game's
// screen is as big as its window.
func ScrollingViewport() DocumentOption {
	return func(d *Document) {
		d.scrollingViewport = true
		d.trans.ScrollingViewport()
	}
}

// ScrollingViewport reports whether the Document's viewport scrolls.
func (d *Document) ScrollingViewport() bool { return d.scrollingViewport }

// Stylesheet adds the CSS file name in fsys, after those added before.
// Its @font-face files are read from fsys too. What it gets wrong is in
// Warnings.
func Stylesheet(fsys fs.FS, name string) DocumentOption {
	return later(func(d *Document) {
		s, warns, err := css.Load(fsys, name, d.parsing()...)
		if err != nil {
			d.warns = append(d.warns, err)
			return
		}
		for _, w := range warns {
			d.warns = append(d.warns, w)
		}
		d.addFontFaces(s.FontFaces(), fsys)
		text, _ := fs.ReadFile(fsys, name)
		d.authored = append(d.authored, &Sheet{doc: d, whole: s, src: string(text), name: name, fsys: fsys})
		d.resheet = true
		d.files = append(d.files, fsys)
	})
}

// later has load run once every option of the Document has been
// applied, in the order the options stand, so BrowserSheets reads the
// sheets given before it too.
func later(load func(d *Document)) DocumentOption {
	return func(d *Document) { d.loading = append(d.loading, func() { load(d) }) }
}

// SheetOption says more of a stylesheet given to CSS.
type SheetOption func(*sheetSource)

type sheetSource struct{ from, media string }

// From gives a sheet the address it was fetched from, as a browser
// knows a linked sheet's: a url() in it is then resolved from that
// address, where it otherwise names a picture as the Document's file
// systems or its fetcher know it. The sheet's From reads it back.
func From(address string) SheetOption { return func(s *sheetSource) { s.from = address } }

// Images adds fsys to where url() and an img's src look for pictures, as a
// stylesheet's own files are: the game's art passed in, read by name, the
// last added first. It adds no rules.
func Images(fsys fs.FS) DocumentOption {
	return later(func(d *Document) { d.files = append(d.files, fsys) })
}

// CSS adds a stylesheet written in the code, as a <style> element does,
// after the sheets before it. Its @font-face rules have no files to read:
// they are fetched with FetchFonts, and warnings without it. A sheet
// fetched from elsewhere says where with From.
func CSS(text string, opts ...SheetOption) DocumentOption {
	var src sheetSource
	for _, o := range opts {
		o(&src)
	}
	return later(func(d *Document) {
		var parse []css.ParseOption
		if src.from != "" {
			parse = append(parse, css.From(src.from))
		}
		s, warns := css.Parse("style", text, d.parsing(parse...)...)
		for _, w := range warns {
			d.warns = append(d.warns, w)
		}
		d.addFontFaces(s.FontFaces(), nil)
		d.authored = append(d.authored, &Sheet{doc: d, whole: s, src: text, name: "style", from: src.from,
			media: src.media})
		d.resheet = true
	})
}

// ByID is the element whose id is id, as getElementById: the first in tree
// order, or nil.
func (d *Document) ByID(id string) *Node {
	var found *Node
	d.html.Walk(func(n *Node) bool {
		if found == nil && n.ID() == id {
			found = n
		}
		return found == nil
	})
	return found
}

// Sheets are the document's stylesheets as the cascade reads them, in
// order, after css.Default: those not disabled, one changed since the
// Document was made as it now stands.
func (d *Document) Sheets() []*css.Sheet {
	d.take()
	return d.sheets
}

// Warnings is what the stylesheets and style attributes got wrong, and
// files that could not be read.
func (d *Document) Warnings() []error { return d.warns }

// HTML is the root element.
func (d *Document) HTML() *Node { return d.html }

// Body is where the game's elements go.
func (d *Document) Body() *Node { return d.body }

// Layout styles and lays the tree out in a viewport of size points, as
// each frame does before drawing, and updates every Node.Box. The
// sheets' @media rules are answered for that viewport, and the tree is
// styled again when a size crosses one's boundary. Pictures
// fetched since the last layout are taken first. When nothing
// changed that layout reads, the boxes are kept and only scroll offsets
// and sticky boxes are redone. Controls
// show their state before styling, so their pseudo-classes restyle them
// the same frame.
func (d *Document) Layout(size geom.Vec2) {
	d.landImages()
	d.landFonts()
	d.landSheets()
	if d.relayout && d.trans.FontsChanged() {
		dom.Data(d.html).Dirty |= dom.DirtyStyle
	}
	media := d.medium()
	media.Width, media.Height, media.Resolution = size.X, size.Y, d.fonts.Scale()
	if d.trans.SetMedia(media) {
		dom.Data(d.html).Dirty |= dom.DirtyStyle
	}
	if d.trans.SetTitleBarArea(d.titleBarArea(size)) {
		dom.Data(d.html).Dirty |= dom.DirtyStyle
	}
	d.style()
	if d.choosePictures(media, false) {
		d.style()
	}
	if d.relayout {
		d.kept, d.relayout = layout.Retained{}, false
	} else if d.tree != nil && size == d.laidAt && d.fonts.Scale() == d.laidScale &&
		dom.Data(d.html).Dirty&(dom.DirtyLayout|dom.DirtyDescendant) == 0 {
		moved := layout.Refresh(d.tree, d.html) || d.scrolled
		d.scrolled = false
		if !moved {
			return
		}
		d.replay.repaint = true
		return
	}
	d.replay.repaint = true
	d.scrolled = false
	d.lay(size)
	d.requery(size)
	if tips, lists, title := d.placeTips(), d.placeSuggestions(), d.placeTitle(); tips || lists || title {
		d.style()
		d.lay(size)
		d.requery(size)
	}
	if d.choosePictures(media, true) {
		d.style()
		d.lay(size)
	}
}

// lay lays the tree out at size, points each node at its box, a node the
// layout left out (under display: none) at none, and notes the tooltips
// showing. A node moved to another Document is that one's to place.
func (d *Document) lay(size geom.Vec2) {
	d.tree = layout.Layout(d.html, size, d.fonts, layout.Images(d), layout.Retain(&d.kept), layout.Translate(d.tr()))
	d.landFonts()
	for _, n := range d.tree.Dropped {
		if d.inDocument(n) {
			dom.Data(n).Placed = nil
		}
		d.tips, d.followers = dropNode(d.tips, n), dropNode(d.followers, n)
	}
	d.laidAt, d.laidScale = size, d.fonts.Scale()
	for _, n := range d.tree.Written {
		b := d.tree.Of[n]
		if b == nil {
			continue
		}
		dom.Data(n).Placed = b
		if tip := b.Parent != nil && b.Kind == box.Element && isTip(n); tip != slices.Contains(d.tips, n) {
			if tip {
				d.tips = append(d.tips, n)
			} else {
				d.tips = dropNode(d.tips, n)
			}
		}
		if dom.Data(n).Follow != nil && !slices.Contains(d.followers, n) {
			d.followers = append(d.followers, n)
		}
	}
	d.findFollowers()
	d.placed += len(d.tree.Of)
}

// maxRequeries bounds the layouts one Layout runs again for its @container
// rules: one a level of containers nested whose answers moved.
const maxRequeries = 8

// requery answers the @container rules and container units for the
// containers' boxes just laid out, and styles and lays out again while
// that changes an answer (decision 0120).
func (d *Document) requery(size geom.Vec2) {
	for range maxRequeries {
		if !d.trans.Requery(contentBox) {
			return
		}
		d.style()
		d.lay(size)
	}
}

// contentBox is the size of n's content box, as a container query reads
// it: none for an inline box, a table or a part of one, which size
// containment skips, nor for an element with no box.
func contentBox(n *Node) (float32, float32, bool) {
	b, placed := dom.Data(n).Placed.(*box.Box)
	if !placed || b == nil || b.Style == nil {
		return 0, 0, false
	}
	if st := b.Style; st.Display == style.Inline || st.Display == style.Contents || st.Display == style.DisplayNone ||
		st.Table || st.TablePart != style.NoTablePart {
		return 0, 0, false
	}
	w := b.Size.X - b.Border.Horizontal() - b.Padding.Horizontal() - b.Scrollbar.X
	h := b.Size.Y - b.Border.Vertical() - b.Padding.Vertical() - b.Scrollbar.Y
	return max(w, 0), max(h, 0), true
}

// dropNode is s less n.
func dropNode(s []*Node, n *Node) []*Node {
	if i := slices.Index(s, n); i >= 0 {
		return slices.Delete(s, i, i+1)
	}
	return s
}

// style shows the controls' state, then computes the styles of what
// changed, a Shadow text's from its element's.
func (d *Document) style() {
	d.take()
	d.pin()
	if dom.Data(d.html).Dirty == 0 {
		// Nothing was added or changed: only the controls' own state can
		// have moved, so sync those and leave the rest of the tree.
		d.syncControls()
	} else {
		d.controls, d.forms = d.controls[:0], d.forms[:0]
		d.choices.imgs, d.choices.stale = d.choices.imgs[:0], true
		d.html.Walk(func(n *Node) bool {
			if dom.Data(n).Dirty&dom.DirtyTree != 0 {
				d.restructure = true
			}
			if m := d.machine(n); m != nil {
				d.controls = append(d.controls, n)
				d.sync(n, m)
			}
			if n.Tag() == dom.Form || n.Tag() == dom.Fieldset {
				d.forms = append(d.forms, n)
			}
			d.noteImg(n)
			return !n.IsText()
		})
	}
	d.validate()
	if dom.Data(d.html).Dirty == 0 {
		return
	}
	d.replay.repaint = true
	for _, w := range d.trans.Compute(d.html, d.sheets) {
		d.warn(w)
	}
	for _, n := range d.controls {
		if sh := dom.Data(n).Shadow; sh != nil {
			dom.Data(sh).Style = dom.Data(n).Style
		}
	}
}

// noteImg keeps an img n among those that choose their picture, styled
// again when the source it chose by changed, or has it show its src.
func (d *Document) noteImg(n *Node) {
	if n.Tag() != dom.Img {
		return
	}
	if !choosesPicture(n) {
		dom.Unchoose(n)
		return
	}
	d.choices.imgs = append(d.choices.imgs, n)
	if s := dom.ChosenSource(n); s != nil && dom.Data(s).Dirty&dom.DirtyStyle != 0 {
		dom.Data(n).Dirty |= dom.DirtyStyle | dom.DirtyLayout
	}
}

// syncControls syncs the controls the last walk of the tree found in it.
func (d *Document) syncControls() {
	for _, n := range d.controls {
		d.sync(n, d.machines[n])
	}
}

// docOf is the Document the internal packages hand back through an any:
// they only ever hold a *Document there.
func docOf(d any) *Document {
	doc, ok := d.(*Document)
	if !ok {
		panic(fmt.Sprintf("ui: internal caller passed %T, not *Document", d))
	}
	return doc
}

//nolint:gochecknoinits // the internal packages reach the Document through these hooks, set once
func init() {
	assets.Of = func(d any, url string) ([]byte, bool) { return docOf(d).imageBytes(url) }
	assets.MediaType = mediaType
	fonts.Of = func(d any) *fonts.Set { return docOf(d).fonts }
	assets.Styled = func(d any) { docOf(d).style() }
	assets.Modal = func(doc any, n *dom.Node) (uint64, bool) {
		m, ok := docOf(doc).machine(n).(*control.Dialog)
		return m.Layer, ok && m.Open && m.Modal
	}
	assets.Tree = func(d any) *box.Tree { return docOf(d).tree }
	assets.Viewport = func(d any) geom.Vec2 { return docOf(d).laidAt }
	assets.Medium = func(d any) css.Media { return docOf(d).medium() }
	assets.Path = func(doc any, n *dom.Node) string {
		leaf := docOf(doc).leaf(n)
		if leaf == "" {
			return ""
		}
		return strings.Join(append(scope(n), automation.Slug(leaf)), "/")
	}
	assets.Pin = func(doc any, n *dom.Node, s dom.State) {
		d := docOf(doc)
		if d.pinned == nil {
			d.pinned = map[*Node]dom.State{}
		}
		d.pinned[n] |= s
	}
}

// Draw advances the transitions by the real time since the last frame,
// so they run in a paused game too, then lays the document out and paints
// it: at the window's size, or at c.Size() when an element hosting the
// document draws it in its own space.
func (d *Document) Draw(c *twod.Canvas) {
	if win := c.Window(); win != nil {
		d.fonts.SetScale(win.ContentScale() * d.Zoom() * d.outerZoom())
	}
	animating := d.tick(c)
	if (animating || d.replay.animating) && d.tree != nil {
		d.tree.Inked = false
	}
	size := d.drawSize(c)
	d.viewport = geom.Rect{W: size.X, H: size.Y}
	d.Layout(size)
	if d.follow(c) {
		d.Layout(size)
	}
	d.replay.quiet = d.quietNow(c, animating)
	if d.interact(c) {
		d.Layout(size)
	}
	d.followCaret(size)
	d.record(c)
	why := d.repaintReason(c, size, animating)
	d.replay.animating = animating
	live := &d.replay.live
	invalidated := d.replay.invalidated
	d.replay.invalidated = false
	switch {
	case why != "":
	case live.Any() && d.replay.inPlace && live.Timed():
		why = repaintLive
	case live.Any() && d.replay.inPlace && invalidated:
		why = repaintCanvas
	case c.Replay(&d.replay.last):
		if live.Any() && !d.replay.inPlace {
			d.zoomed(c, func() { paint.PaintLive(c, live, d.clock, d.options()...) })
			count(c, replayedLive)
		} else {
			count(c, replayed)
		}
		d.replay.replayedLast = true
		return
	default:
		why = repaintRefused
	}
	count(c, why)
	m := c.Mark()
	apart := !d.replay.inPlace
	d.painted++
	d.unsettled = false
	d.replay.frame = paint.Frame{Time: d.clock, Live: live, Apart: apart, Keep: d.tree.Of[d.focused()],
		Selection: d.selectionPaint()}
	d.zoomed(c, func() { paint.Paint(c, d.inked(), d.frameOptions()...) })
	d.replay.recorded = c.Keep(m, &d.replay.last)
	d.paintApart(c, apart)
	d.replay.inPlace = live.Covered()
	d.replay.replayedLast = false
	d.replay.repaint, d.replay.size, d.replay.scale = d.unsettled, size, d.fonts.Scale()
}

// tick moves the document's clock to the World's real time and steps its
// transitions, reporting whether one is still running.
func (d *Document) tick(c *twod.Canvas) bool {
	animating := false
	if w := c.World(); w != nil {
		now := w.Time().RealElapsed
		if d.clock >= 0 {
			animating = d.trans.Step(now - d.clock)
		}
		d.clock = now
		d.animate()
		d.drawnIn(w.Time().Frame)
	}
	return animating
}

// paintApart draws the live parts a paint kept out of its recording on
// top of it, and has the next frame paint again where the recording
// isn't one to replay.
func (d *Document) paintApart(c *twod.Canvas, apart bool) {
	live := &d.replay.live
	switch {
	case !live.Any():
	case apart:
		d.zoomed(c, func() { paint.PaintLive(c, live, d.clock, d.options()...) })
		if live.Covered() {
			// Kept apart, they are in no recording: drawn on top this once,
			// the next frame paints them in their place.
			d.replay.recorded = false
		}
	case !live.Covered():
		// Painted in their place and covered no more, they are in the
		// recording: the next frame paints them apart again, or a replay
		// would draw them twice.
		d.replay.recorded = false
	}
}

// drawSize is the size the document lays out at, in CSS pixels: c's own, or
// the window's when c has none.
func (d *Document) drawSize(c *twod.Canvas) geom.Vec2 {
	size := c.Size()
	if size.X <= 0 || size.Y <= 0 {
		s := c.Screen()
		size = geom.V2(s.W, s.H)
	}
	return size.Scale(1 / d.Zoom())
}

func (d *Document) face(f style.Font) *twod.Face {
	if ch := d.fonts.Chained(f); ch != nil {
		return d.chainFace(ch)
	}
	ff := d.fonts.Face(f)
	if tf, ok := d.faces[ff]; ok {
		return tf
	}
	var opts []twod.FaceOption
	if fb := d.fonts.Fallback(); fb != nil && fb != ff {
		opts = append(opts, twod.Fallback(d.face(style.Font{Family: []string{fb.Family}, Weight: float32(fb.Weight)})))
	}
	if d.emoji != nil {
		opts = append(opts, twod.Emoji(d.emoji))
	}
	tf, err := twod.NewFace(ff.TTF, opts...)
	if err != nil {
		tf = twod.Regular()
	}
	if d.faces == nil {
		d.faces = map[*fonts.Face]*twod.Face{}
	}
	d.faces[ff] = tf
	return tf
}

// warn keeps a style attribute's warning once, however often its element
// restyles.
func (d *Document) warn(err error) {
	if d.warned == nil {
		d.warned = map[string]bool{}
	}
	if !d.warned[err.Error()] {
		d.warned[err.Error()] = true
		d.warns = append(d.warns, err)
	}
}

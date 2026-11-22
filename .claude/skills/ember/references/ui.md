# ui: screens of HTML elements and CSS

A screen is a `*ui.Document`: HTML elements built from Go, styled by a CSS subset checked against Chrome, added to a window as an entity. `ui/kit` builds ember's components on it. Guide: `docs/05-ui`; examples `examples/11-ui-elements` to `examples/18-testing-ui`, every kit component alone in `examples/92-ui-kit -element <name>`.

## Make a screen

| Need | API |
|---|---|
| a Document | `ui.NewDocument(opts ...ui.DocumentOption) *ui.Document` |
| ember's look (kit components need it) | `kit.Theme() ui.DocumentOption` |
| a sheet written in the code | `ui.CSS(text string, opts ...ui.SheetOption) ui.DocumentOption`: an `@font-face` in it is a warning; `ui.From(address string) ui.SheetOption` for a sheet fetched from elsewhere, its `url()` fetched from that address |
| a CSS file, with its fonts | `ui.Stylesheet(fsys fs.FS, name string) ui.DocumentOption`: `@font-face` `src` files read from `fsys` |
| pictures for `url()` and `img` | `ui.Images(fsys fs.FS) ui.DocumentOption` |
| sheets an `@import` names | `ui.FetchSheets(fetch ui.SheetFetcher) ui.DocumentOption`: `func(url, sheet string) ([]byte, error)`, called off the frame, each sheet spliced where its import stands (media, `layer()`); `web.Sheets(base string)` fetches over HTTP; a `Stylesheet`'s imports are read from its `fsys` first; `doc.SheetsLoading()` |
| pictures no file system holds | `ui.FetchImages(fetch ui.ImageFetcher) ui.DocumentOption`: `func(url string) ([]byte, error)`, called off the frame; `web.Images(base string)` (package `ui/web`) fetches over HTTP and adds JPEG and GIF |
| SVG pictures in `img` and `url()` | `import _ "github.com/lantervale/ember/ui/svg"`: sized as Chrome sizes them (width and height, a view box alone, neither), rasterised at the size shown times the pixel ratio; text, clip-path, masks, filters and patterns are not drawn and go to `doc.Warnings()` naming the file |
| scrollbars over the content, taking no room | `ui.OverlayScrollbars() ui.DocumentOption`, `doc.OverlayScrollbars()`: every scroller of the Document; classic bars (a 15-point gutter) are the default |
| a page taller than its window that scrolls | `ui.ScrollingViewport() ui.DocumentOption`, `doc.ScrollingViewport()`: the viewport scrolls as a browser's (`html` as high as the page, `position: fixed` stays put, `html { overflow: hidden }` stops it); off by default, a game's screen never scrolls as a whole; `doc.ScrollIntoView(n)` shows an element by the least scroll, `doc.ScrollToStart(n)` puts it at the top of its scrollers, `doc.Scroll(n) geom.Vec2` and `doc.SetScroll(n, off geom.Vec2)` read and set a scroller's offset (the viewport's on `doc.HTML()`) |
| a page written for browsers | `ui.BrowserSheets() ui.DocumentOption`, `doc.BrowserSheets()`: sheets and style attributes read as Chrome reads them (`-webkit-` aliases, other engines' names dropped in silence, `@supports` answered, `@layer`, `box-sizing: content-box`); off by default, never for a game's own sheet, whose warnings it hides |
| everything bigger, as browser zoom | `ui.Zoom(f float32) ui.DocumentOption`, `doc.SetZoom(f)`, `doc.Zoom()` |
| show it | `doc.Body().Append(nodes...)`, then `win.Add(doc)`: it lays out at the window's size |

- Sheets apply in the order given: `kit.Theme()` first, the game's sheet after it.
- `html, body { height: 100%; }` lets the body centre or stretch what it holds.
- A sheet's mistakes go to `doc.Warnings()`, never a crash.

## Elements and arguments

Every element is `func(args ...ui.Arg) *ui.Node`: `ui.Div`, `Section`, `Header`, `Footer`, `Nav`, `Main`, `Aside`, `Article`, `P`, `H1`–`H4`, `Span`, `Strong`, `Em`, `Code`, `Kbd`, `Samp`, `Pre`, `Label`, `Button`, `Input`, `Select`, `Option`, `Textarea`, `Details`, `Summary`, `Dialog`, `Table`, `Thead`, `Tbody`, `Tfoot`, `Caption`, `Tr`, `Th`, `Td`, `Progress`, `Meter`, `Img`, `Hr`, and the text elements in Chrome's default look: `Ul`, `Ol`, `Li`, `Menu`, `Dl`, `Dt`, `Dd`, `H5`, `H6`, `Blockquote`, `Figure`, `Figcaption`, `Form`, `Fieldset`, `Legend`, `Address`, `Hgroup`, `Search`, `Center`, `Br`, `A` (with an `href` it takes the focus and its `OnClick`), `B`, `I`, `U`, `S`, `Small`, `Big`, `Sub`, `Sup`, `Abbr`, `Cite`, `Q`, `Mark`, `Time`, `Var`, `Del`, `Ins`, `Dfn`, `Bdi`, `Bdo`, `Data`, `Font`. A `*ui.Node` is itself an Arg: as an argument, it becomes a child.

| Arg | For |
|---|---|
| `ui.ID(id)`, `ui.Class(names...)` | selectors, `doc.ByID`, automation paths |
| `ui.Text(s)` | a text child |
| `ui.Style(css)` | the style attribute; adds to what earlier args (`kit.Padding`) set |
| `ui.Attr(name, value)`, `ui.Type(t)`, `ui.Value(v)`, `ui.Placeholder(s)`, `ui.For(id)` | attributes |
| `ui.Checked(on)`, `ui.Open()`, `ui.Disabled()`, `ui.Selected()`, `ui.Draggable()`, `ui.TabIndex(i)` | flags |
| `ui.Attr("autofocus", "")` | the first focus |

## The kit catalogue

Each returns `*ui.Node` unless said; `args ...ui.Arg` go on the element.

| Component | Signature | For |
|---|---|---|
| Panel | `kit.Panel(args...)`, `kit.Title(text) ui.Arg` | a framed box; Title puts a heading above |
| CollapsiblePanel | `kit.CollapsiblePanel(title, open, args...)` | a panel that folds to its title |
| Card | `kit.Card(args...)` | a pressable panel, clicked by pointer or keys |
| Row, Column | `kit.Row(args...)`, `kit.Column(args...)` | flex lines, 4 points apart by default |
| Grid | `kit.Grid(columns int, args...)`, `kit.Columns(tracks string)` | children in rows of columns |
| spacing | `kit.Gap(px)`, `kit.Padding(px)` | the gap and padding of any box |
| placement | `kit.Expand()`, `kit.Fill()`, `kit.AlignStart/Center/End()`, `kit.JustifyStart/Center/End/Between()` | flex children and their line |
| Label | `kit.Label(text, args...)`, `kit.Wrap()`, `kit.Truncate()`, `kit.Center()`, `kit.Right()`, `kit.Caps()` | one line of text unless Wrap |
| RichText | `kit.RichText(text, args...)`, `kit.SetRich(n, text)` | text with inline tags |
| Button | `kit.Button(text, args...)`, `kit.Primary()`, `kit.Destructive()` | a push button |
| Check, Switch | `kit.Check(text, args...)`, `kit.Switch(text, args...)` | on/off; read with `doc.Checked(n)` |
| Slider | `kit.Slider(value, args...)`, `kit.Range(lo, hi)`, `kit.Step(s)` | a value in a range; `doc.Value(n)` |
| NumberField | `kit.NumberField(value, args...)` | a number typed or stepped |
| Progress | `kit.Progress(value, args...)`, `kit.SetProgress(p, value)`, `kit.ShowValue(on)` | a bar, 0–100 unless `kit.Range` |
| TextInput | `kit.TextInput(text, args...)`, `kit.Password()`, `kit.MaxLength(n)`, `kit.ReadOnly()` | one line of text; `doc.Value(n)` |
| SearchField | `kit.SearchField(text, args...)` | a text input with a clear button |
| TextArea | `kit.TextArea(text, args...)`, `kit.Rows(n)`, `kit.NoWrap()` | several lines |
| Choice | `kit.Choice(selected, items, args...)`, `kit.ChoiceItem(content, args...)`, `kit.Chosen(c)`, `kit.SetChosen(c, i)` | a segmented pick of one |
| PopUp | `kit.PopUp(selected int, items []string, args...)` | a select of strings |
| Form | `kit.Form(rows...)`, `kit.Field(label, field)` | labels and fields lined up |
| Tabs | `kit.Tabs(kit.Tab(title, page, args...)...)`, `kit.SelectedTab(doc, tabs)`, `kit.SetSelectedTab(doc, tabs, i)` | one page at a time |
| Table | `kit.Table(columns, rows []*ui.Node, args...)`, `kit.TableColumn(title, args...)`, `kit.TableRow(cells...)` | data in sortable rows |
| Scroll | `kit.Scroll(args...)`, `kit.VerticalBar(m)`, `kit.HorizontalBar(m)` | a scrolled area |
| Separator | `kit.Separator(args...)`, `kit.Vertical()` | a line between groups |
| Image | `kit.Image(src, args...)`, `kit.TileImage(src, args...)`, `kit.NinePatch(src, l, t, r, b, args...)` | pictures from `ui.Images` |
| Tooltip | `kit.Tooltip(tip *ui.Node) ui.Arg` | help beside an element on hover or focus |
| Hint | `kit.Hint(kit.KeyboardHint(s), kit.GamepadHint(s), kit.MouseHint(s), kit.TouchHint(s))`, `kit.SetHint(h, device, s)` | help for the device used last |
| StatusBar | `kit.StatusBar(text, args...)`, `kit.SetStatus(bar, text)` | a line along a window's foot |
| Alert | `kit.Alert(title, message string, buttons ...*ui.Node)`, `kit.SetAlert(a, title, message)` | a question answered by its buttons |
| Window | `kit.Window(title, args...)`, `kit.Toolbar(args...)`, `kit.Footer(args...)`, `kit.Resizable()` | an in-page window, dragged by its bar |
| Menus | `kit.MenuBar(kit.MenuBarItem(title, kit.Menu(kit.MenuItem(text, kit.Shortcut("Cmd+S"))...))...)`, `kit.MenuButton(text, menu)` | app menus |
| Sidebar, Thumbnails | `kit.Sidebar(kit.SidebarSection(title, kit.SidebarItem(text)...)...)`, `kit.Thumbnails(kit.Thumbnail(name, src)...)` | tool screens |
| FileDialog | `kit.NewFileDialog(doc, fsys, opts...) *kit.FileDialog` | open and save, an ember dialog, never the OS's |

## The CSS subset

Full table: `docs/05-ui/03-css.md`. Lay screens out with flex or grid.

- **Works:** selectors `div .c #id [a="v"] a b a>b`, `:hover :focus :focus-visible :active :disabled :checked :open :first-child :last-child :nth-child() :not() :root`; `px % em rem fr`, `calc() var() minmax() repeat()`.
- **Layout:** `display` block, inline, inline-block, flex, inline-flex, grid, flow-root, none, contents; `position` up to sticky; `float` and `clear` for text round a picture (a container holds its floats with `display: flow-root`), `text-indent`; `column-count`, `column-width`, `columns`, `column-gap`, `column-rule`, `column-fill` for text in balanced columns, cut between lines (`break-inside: avoid` keeps a box whole, `break-before: column` starts a column, `orphans`, `widows`; a cut element's `Box()` is the box round its parts); `overflow`; `z-index`; flex and grid in full.
- **Look:** backgrounds of several layers, the first on top (`background-size` in lengths, percentages, `auto`, `cover`, `contain`; `background-repeat` with `space` and `round`; `background: linear-gradient(transparent, #0009), url(x.png) no-repeat center / cover`) and linear, radial and conic gradients, `border-radius`, `border-image`, `box-shadow`, `opacity`, `transform`, `filter` (blur, colour functions, drop-shadow; one picture through WGSL effects), `clip-path` (`circle()`, `ellipse()`, `inset()`, `polygon()`, `path()` in a reference box; cuts drawing and the pointer), `transition`, `cursor`, `pointer-events`, `app-region: drag` (a header that drags a `platform.OwnTitleBar()` window; its buttons still click), `image-rendering: pixelated`, `-webkit-font-smoothing: none` for pixel fonts, `antialiased` for glyphs at their exact coverage (no stem darkening).
- **Text extras:** the `font` shorthand; `text-decoration` (underline, overline, line-through; solid, double, dotted, dashed); `vertical-align` (every value; `top | middle | bottom` on a cell); `list-style-type` disc, circle, square, a string or any counter style (decimal, lower/upper-alpha, lower/upper-roman, lower-greek… or one of the sheet's `@counter-style` rules: cyclic, numeric, alphabetic, symbolic, additive, fixed, extends, prefix/suffix, range, pad, negative, fallback), `list-style-image: url()`, `ol`/`li` `type`; `:link` matches an element with an `href`; installed fonts by name only with `ui.SystemFonts()` on the Document.
- **Pseudo-elements:** `::before` and `::after` make a box inside their element once `content` is set (`content: ""` for a shape): any display, position, background or transition, `li:nth-child(2)::before`, `a:hover::after`. `content` takes strings, `attr(name)`, `url(pic.png)`, `counter(name[, style])`, `counters(name, ".")`, `open-quote`/`close-quote`; `counter-reset`, `counter-increment`, `counter-set` count, and a list's numbers are the `list-item` counter (`ol` `start`, `reversed`, `li` `value`). `li::marker` takes colour, font and `content` (`::before::marker` for a list-item `::before`); `content: url(pic.png)` alone on an element replaces it by the picture; checkboxes and radios take `::before`/`::after`; `q`'s marks follow `quotes`. Also `::placeholder` (color, opacity), `::selection` (background-color, color), `::backdrop`. A generated box is no element: no selector, automation or `Children()` sees it, and a click on it clicks its element. `::first-line`, `::first-letter` are refused.
- **Selecting text:** nothing of a screen is selectable by default (`html { user-select: none }`, kit included). `user-select: text` on an element lets the player select its text with the pointer (drag, double click a word, triple click a block, Shift+click) and copy it (Cmd/Ctrl+C; Cmd/Ctrl+A selects all that can be) while no field has the focus; `user-select: all` takes an element whole, `none` leaves it out. `::selection { background-color; color }` styles it, inherited by what the element holds. From code: `doc.Selection()` (`Anchor`, `Focus`, `Text`, `Empty()`), `doc.Select(ui.Position{Node: textNode, Offset: n}, …)`, `doc.SelectAll()`, `doc.SelectContents(n)`, `doc.ClearSelection()`, `doc.SelectionRects()`; `ui.OnSelectionChange(fn).Apply(doc.Body())` or the World event `ui.SelectionChanged`. Under `ui.BrowserSheets()` a page's text is selectable as in a browser.
- **Refused** (a warning): `vertical-align` but baseline, sub and super, `column-span: all`, `text-decoration: wavy`, `animation`, `@import` without `ui.FetchSheets` (a `Stylesheet`'s imports are read from its files).
- **`@media`:** answered for the Document's viewport in its own pixels (window points divided by `Zoom`) and followed on resize: `@media (max-width: 600px) { .side { width: auto } }`; `width`, `height`, `aspect-ratio`, `orientation`, `resolution`, ranges (`width >= 600px`), `and`, `,`, `not`; the scheme is `light` unless the program passes `ui.Preferences(prefs)` or `ui.ColorScheme(a)`, the pointer a mouse until a finger touches the screen; an unknown feature is false with a warning.
- **Colour schemes:** `color-scheme: light dark` on `:root` draws the page in the Document's preferred scheme (`dark` always dark, `normal` always light); system colours (`Canvas`, `CanvasText`, `Field`, `ButtonFace`, `LinkText`…) and `light-dark(#fff, #111)` resolve in it, and the browser sheet's controls go dark: `.card { background: light-dark(#fff, #1e1e2e); color: CanvasText }`.
- `px` is a point, not a device pixel; `box-sizing` is border-box by default (content-box with `ui.BrowserSheets()`).

## React to the player

Callbacks go on the element: `ui.OnClick(fn)`, `OnInput`, `OnChange`, `OnSubmit`, `OnToggle`, `OnClose`, `OnSort`, `OnKey`, `OnBack`, `OnContextMenu`, `OnDrop`, `OnDragEnd`, `OnPointerEnter`, `OnPointerLeave`, `OnSelectionChange` (on `body`). `fn` is any of `func()`, `func(ui.Event)` and `func(*world.World, ui.Event)`.

| Need | API |
|---|---|
| what happened | `ui.Event{Type, Node, Clicks, Value, Key, Text, …}`: `Clicks` 2 or 3 for double or triple; `Value` a closed dialog's |
| a callback on an existing node | `remove := ui.On(n, ui.EventClick, fn)` |
| stop a Back or key going up | `e.StopPropagation()`; Back and keys bubble from the focused element to its ancestors, clicks don't |
| from code that didn't build the element | World events `ui.Clicked`, `Changed`, `Edited`, `Submitted`, `Toggled`, `BackPressed`, `Closed` |

- A click reaches the innermost element with behaviour: a control, a canvas, a `label`, a `summary` or an element with a `tabindex`. A plain `div` needs `ui.TabIndex(0)` for its OnClick.
- What a callback changes shows the next frame.
- A menu of the screen's own on a right click: `ui.OnContextMenu` on the root (it bubbles), `win.Input().Mouse()` for where, `doc.ElementAt(at) *ui.Node` for what is under a point (window points, as `BoxOf`; nil where no box takes the pointer).
- A text field's selection: `doc.SelectionRange(n) (from, to int)`, `doc.SetSelectionRange(n, from, to)`, in characters of `doc.Value(n)`.
- Values: `doc.Value(n)`, `doc.SetValue(n, v)`, `doc.Checked(n)`, `doc.SetChecked(n, on)`, `doc.Selected(n)`, `doc.SetSelected(n, i)`. The setters emit no event.

## Change a screen at run time

| Need | API |
|---|---|
| text | `n.SetText(s)`, read with `n.Text()` |
| attributes, style | `n.SetAttr(name, v)`, `n.RemoveAttr(name)`, `n.SetStyle(css)` |
| classes | `n.AddClass(c)`, `n.RemoveClass(c)`, `n.HasClass(c)` |
| hide, show | `n.SetAttr("hidden", "")`, `n.RemoveAttr("hidden")` |
| add, remove | `n.Append(children...)`, `n.Remove(child)` |
| dialogs | `doc.ShowModal(d)`, `doc.Show(d)`, `doc.Close(d, value)`, `doc.Open(d)`: a `kit.Alert` or `ui.Dialog` appended to the body first; a modal ignores keys and pad buttons still held from before it opened, until they're released |
| redraw a canvas | `doc.Invalidate(canvas)` |
| a tag that follows a 2D entity | `ui.Follow(target twod.Entity, offset geom.Vec2) ui.Arg`, `ui.SetFollow(n, target, offset)` (`examples/16-hud-over-world`) |

## Draw your own: ui.Canvas

`ui.Canvas(draw func(c *twod.Canvas), args ...ui.Arg) *ui.Node` is an element the game draws (`examples/17-custom-elements`).

- For HUD widgets only (an orb, a minimap); a game's play area is twod entities (SKILL.md), so effects, particles and lights can go on it.

- `draw` runs in the content box: the origin is its top-left, and `c.Size()` is the size the layout gave it.
- `draw` runs when the canvas first shows, when its box moves or resizes, and after `doc.Invalidate(n)`. Otherwise the last drawing stays.
- Draw with `c.Rect(r, col)`, `c.RoundRect(r, radius, col)`, `c.Circle(center, radius, col)`, `c.Line(a, b, width, col)`, `c.Border(r, radius, thickness, col)`, `c.Text(at, text, opts...)`, `c.Image(r, tex, uv, tint)`.
- Size it with CSS (`#orb { width: 96px; height: 96px; }`); without it, a canvas is 300 by 150.
- `ui.OnClick` on a canvas runs on a click on it; `doc.PointerOn(n) (geom.Vec2, bool)` is then the pointer in the canvas's own points, false when it is off the canvas.
- Tell automation what a canvas shows with `ui.Attr("role", "meter")` and `ui.Attr("aria-valuenow", v)`.

## A Document in another: ui.Iframe

`ui.Iframe(child *ui.Document, args ...ui.Arg) *ui.Node` shows a Document the program built in an `iframe` element: 300 by 150 with an inset border unless CSS or its `width`, `height` and `frameborder` attributes say otherwise.

- The child lays out in the frame's content box (its `@media` follows it), is clipped to it, and scrolls there with `ui.ScrollingViewport()`.
- `doc.SetFrame(n, child)` swaps what it shows (nil for nothing), `doc.Frame(n)` reads it, `doc.FrameShown(n)` says whether it was drawn last frame.
- The pointer reaches the child only where nothing of the page lies over the frame; the wheel scrolls the child first, then the page; keys go to the child while the frame has the page's focus; Tab walks in and out.
- `doc.FramePoint(n, at) (geom.Vec2, bool)` turns a page point into the child's, for `child.ElementAt`.
- Taken out of its Document, the iframe lets go of the child at the next frame; put back later it shows nothing until `SetFrame`.
- A child the program does not trust (a page it loaded): `ui.Iframe(child, ui.Guarded())` stops a panic of its layout, input, callbacks or paint at the frame, which empties; `doc.FrameBroken(n) error` (a `*ui.FrameError`: `Value`, `Stack`) says so until `SetFrame`, and the stack is in `errors.log`. Without it a panic is the program's.
- In tests, drive it through the window's input as any screen; automation ids of the child are under the frame's id.

## Focus and keys

| Need | API |
|---|---|
| give or read the focus | `doc.Focus(n)`, `doc.Focused(n) bool`: an element shown this frame takes it at its first layout |
| one Back for a whole screen | `ui.OnBack(fn)` on its root; with nothing focused, Back reaches the body's: `ui.On(doc.Body(), ui.EventBack, fn)` |
| a key nothing took | `ui.OnKey(func(e ui.Event) { … e.Key … })`; a pad button navigation leaves (West, North, shoulders, triggers, Back, Start) comes as `e.Pad` with `e.Key == input.KeyUnknown` |
| keyboard order | Tab and Shift-Tab once something has the focus; the arrows move it spatially |
| keys left to the game | with nothing focused and no modal open; read `win.Input()` as usual |

Escape and the pad's East are Back. Enter, Space and the pad's South press the focused element. A modal dialog keeps the focus and its window's input until it closes. While a modal dialog of another Document is open in the window, a Document's focused element keeps the focus (`doc.Focused(n)` stays true, no event) but matches neither `:focus` nor `:focus-visible`; they return when the dialog closes or its Document is hidden (decision 0073).

## Test a screen

See [testing.md](testing.md): `doc.ByID`, `doc.BoxOf`, simulated clicks, `align.Check(doc)` and `doc.Warnings()`, which every screen's test runs. Automation names an element by the path of its ancestors' IDs, `settings/save` (`examples/18-testing-ui`).

## A HUD over a 2D board

The board is a twod entity, drawn every frame; the ui is the HUD beside it: the score, a bar, and a restart that asks first. Escape asks too. A click the HUD took is `PointerCaptured`, so it never reaches the board.

```go
package main

import (
	"fmt"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/lantervale/ember/world"
)

// The board's top-left in the window, and its cells.
const left, top, cell, size = 56, 60, 48, 5

// The HUD sits right; the board is drawn left of it, in the 2D world.
const page = `html, body { height: 100%; }
body { display: flex; align-items: center; justify-content: flex-end; padding-right: 48px; }
#hud { min-width: 200px; }`

// Lights is the rules: plain Go, tested without a window.
type Lights struct {
	Lit   [size][size]bool
	Score int
}

func (l *Lights) Toggle(x, y int) {
	if x < 0 || y < 0 || x >= size || y >= size {
		return
	}
	l.Lit[x][y] = !l.Lit[x][y]
	if l.Lit[x][y] {
		l.Score++
	}
}

// Board is the play area, a twod entity drawn every frame.
type Board struct {
	twod.Item
	game *Lights
}

func (b *Board) Draw(c *twod.Canvas) {
	c.RoundRect(geom.Rect{W: size * cell, H: size * cell}, 8, color.Hex("#2b2d42"))
	for x := range size {
		for y := range size {
			col := color.Hex("#3d405b")
			if b.game.Lit[x][y] {
				col = color.Hex("#f2cc8f")
			}
			c.RoundRect(geom.Rect{X: float32(x*cell) + 4, Y: float32(y*cell) + 4, W: cell - 8, H: cell - 8}, 6, col)
		}
	}
}

// HUD is the ui around the board: the score, a bar, and a restart that asks first.
type HUD struct {
	doc             *ui.Document
	score, bar, ask *ui.Node
}

func newHUD(game *Lights) *HUD {
	h := &HUD{doc: ui.NewDocument(kit.Theme(), ui.CSS(page))}
	h.score = kit.Label("0", ui.ID("score"))
	h.bar = kit.Progress(0, kit.Range(0, 25))
	h.ask = kit.Alert("Restart?", "The score goes back to zero.",
		kit.Button("Cancel"), kit.Button("Restart", kit.Primary(), ui.Value("restart")))
	ui.On(h.ask, ui.EventClose, func(e ui.Event) {
		if e.Value == "restart" {
			*game = Lights{}
		}
	})
	restart := kit.Button("Restart", ui.ID("restart"), ui.OnClick(func() { h.doc.ShowModal(h.ask) }))
	h.doc.Body().Append(kit.Panel(kit.Title("Score"), ui.ID("hud"), kit.Padding(16), kit.Column(kit.Gap(8),
		h.score, h.bar, restart, kit.Hint(kit.KeyboardHint("Click a cell to light it")))), h.ask)
	ui.On(h.doc.Body(), ui.EventBack, func() { h.doc.ShowModal(h.ask) })
	return h
}

func (h *HUD) show(game *Lights) {
	h.score.SetText(fmt.Sprint(game.Score))
	kit.SetProgress(h.bar, float64(min(game.Score, 25)))
}

// Play turns a click on the board into a toggle, unless the HUD took it.
type Play struct {
	game *Lights
	in   *input.State
	hud  *HUD
}

func (p *Play) Update(world.Frame) {
	if p.in.Pressed(input.MouseLeft) && !p.in.PointerCaptured() {
		at := p.in.Mouse().Sub(geom.V2(left, top))
		p.game.Toggle(int(at.X)/cell, int(at.Y)/cell)
	}
	p.hud.show(p.game)
}

func main() {
	win := platform.NewWindow(platform.Title("lights"), platform.Size(640, 360), platform.FixedSize())
	game := &Lights{}
	board := twod.Build(&Board{Item: twod.Base(), game: game}, "board", twod.At(left, top))
	hud := newHUD(game)
	win.Add(board, hud.doc)
	ember.New().AddEntities(win).AddSystems(&Play{game: game, in: win.Input(), hud: hud}).MustRun()
}
```

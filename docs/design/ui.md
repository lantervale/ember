# uiv2: modern HTML elements and a CSS subset, written from Go

The package this note calls `uiv2` is `ember/ui` since decision [0059](../99-decisions/0059-ember-ui-is-uiv2.md); its `ember/ui` is the old package, removed then.

Status: **design, approved by the user (2026-11-02), decision 0049.** It replaces `ember/ui`'s hand-sized layout once it covers the tileset tool, the dialogs and the demo's screens.

## Why

`ember/ui` sizes layout by hand: a column is "150 points", a field is "Fill+Expand", and nothing stops content from overflowing its parent. Each screen fixes its own widths, and each fix shows up as a new misalignment or a scrollbar where none is meant. CSS already solved this: `grid-template-columns: max-content 1fr`, `min-width: 0` and `text-overflow: ellipsis` rule out the whole class of bug. uiv2 takes that model, and its names, without becoming a browser.

## What it is, and what it is not

- **Written, never read.** The game builds the tree in Go. There is no HTML parser, no DOM query API and no scripting.
- **Modern elements only.** `div section header footer nav main aside article p h1…h4 span strong em code label button input(text number range checkbox search password) select option textarea details summary dialog table thead tbody tr th td progress meter img hr canvas` (a canvas is drawn by the game's `func(*twod.Canvas)`, again when `Invalidate` asks). Since decision 0082 (2026-11-17) also the text elements pages are written with, in Chrome's default look: `ul ol li menu dl dt dd h5 h6 blockquote figure figcaption form fieldset legend address hgroup search br a b i u s small big sub sup abbr cite q mark time var del ins dfn bdi bdo data center font`.
- **A modern CSS subset**, written as CSS text: stylesheets (files next to the game, like themes today) plus a `style` attribute.
  - Selectors: type, `.class`, `#id`, descendant, child `>`, attributes `[attr]` and `[attr="value"]` (exact match only), `:hover :focus :focus-visible :active :disabled :checked :open :first-child :last-child :nth-child() :not()`, specificity, inheritance.
  - Values: `px` (= points), `%`, `em`, `rem`, `fr`, `auto`, `min-content`, `max-content`, `fit-content`, `minmax()`, `repeat()`, `calc()`, `var()`, colours (hex, `rgb()`, `oklch()`, the CSS named colours, `transparent`, `currentColor`), `!important`.
  - Layout: `display: block | inline | inline-block | flex | inline-flex | grid | none | contents`, `box-sizing` (border-box by default), `width height min-* max-*`, `margin padding border gap`, `flex` (grow, shrink, basis, wrap, direction, align, justify), `grid-template-columns/rows`, `grid-column/row`, `place-*`, `align-self`, `aspect-ratio`, `overflow` (visible, hidden, clip, auto, scroll) and its axes, `position: static | relative | absolute | fixed | sticky`, `inset`, `z-index`.
  - Text: `font-family` (TrueType faces by file, as themes do), `font-size font-weight line-height letter-spacing word-spacing text-align text-transform white-space text-overflow overflow-wrap color`. `text-transform` cases what text shows, words split as Unicode splits them for `capitalize`, and keeps the source for automation and the node's text; edit fields (input, textarea) draw their text transformed too, while their value stays as typed (5178fa6e).
  - Look: `background-color` and images (nine-patch as `border-image`), `linear-gradient()` as a `background-image` or in `background` (an angle in deg/rad/grad/turn or `to` a side or corner, two colour stops or more at lengths or percentages, two positions per stop for hard edges; blended in encoded sRGB as Chrome does; no repeating or radial gradients, no transition hints, one background layer), `border` with `border-radius`, `box-shadow`, `outline`, `opacity`, `cursor`, `pointer-events` (`auto` or `none`, inherited: hit testing passes through), `app-region` (`drag`, `no-drag`, `none`, inherited: a press in a drag region off what clicks moves the system's window, decision 0091), `transform` (2D), `transition` on colours, opacity and transforms.
  - Added for M3 (user and architect, 2026-11-02): `text-shadow`, `image-rendering: pixelated`, `accent-color`, `caret-color`, `::selection { background-color }`, `::placeholder { color }`, `object-fit`, `object-position`, `background-repeat`, `background-position`, `background-size` (lengths or auto, a 2x picture drawn at its point size), `border-spacing`, `resize` (Chrome's grip; ember draws none), `scrollbar-width` (`none` scrolls without bars). A range's `background-color` paints its track and its `color` its idle thumb, a progress or meter's `background-color` its track, and an open select's highlighted option takes its `accent-color` (documented conventions, ember only; export has Chrome draw ranges and progress bars the same way).
  - Alignment (kit): controls sharing a flex row (buttons, selects, text fields, switches, search fields) are one height, a control's label sits in the middle of its padding box (across too for a centred button), and a row's single-line text shares a baseline, each within 0.5pt; kit's TestKitPagesLineUp checks every kit page (internal/align). A control, or a row of them inside a row, keeping a height of its own (a taller one on purpose, a window's lights) carries class `own-height`.
  - Tables: the `<table>` elements (`caption`, `thead`, `tbody`, `tfoot`, `tr`, `th`, `td`) with `border-spacing`, `caption-side` (top or bottom) and `table-layout` (auto or fixed); `colspan` and `rowspan`; the caption sits outside the table's border, background and padding, as in Chrome. Collapsed borders are refused.
  - Text (2026-11-17): the `font` shorthand, `text-decoration`, `vertical-align` (every value on inline boxes, inline-blocks and pictures, top, middle and bottom on cells; decision 0104), list markers (`list-style-type` disc, circle, square, decimal), `groove` and `ridge` borders, `background-repeat: repeat-x | repeat-y`, `:link` and `:visited`; installed fonts by name with `ui.SystemFonts()` (decision 0081).
  - SVG pictures (2026-11-17, decision 0087): `ui.RegisterVectorFormat(mime, is, parse)` registers a format of pictures drawn from shapes (`ui.VectorImage`: `Width`, `Height`, `Ratio`, `Rasterise(w, h, scale)`, `Unsupported`), and `ui/svg` registers the root `svg` reader on import; ui never imports `svg` (root `svg_test.go`). A file is recognised by its `.svg` name or its bytes (ui/svg `sniff`); `ImageFetcher` carries no content type, so there is no third route. Layout reads `box.Intrinsic` through `box.IntrinsicImages` (ui/layout/image.go: a view box alone `fills` the width it is given; no ratio without one), paint through `paint.Vectors` (`tileSize` and `shapesImg`), and asks `Vector.Texture(w, h, scale)` for the size it draws in device pixels. ui/vector.go keeps the rasters: one per size, drawn in the frame for a file and off it for a fetched picture, at most `vectorRasters` (32) and `vectorPixels` (1<<24) a picture, a side at most `vectorSide` (8192); a size is drawn once two paints running ask for it, the nearest raster stretched meanwhile. `scale` is what a unit of a file with no view box takes (`svg.RasteriseScaled`). An `img`'s `width` and `height` attributes are presentational hints (ui/css `hints`), and a block-level replaced box no longer stretches. Chrome: box cases in ui/svg/chrome_test.go, pixel page `svg-pictures`. Not matched: a view-box-only `img` as an unsized flex item or absolutely positioned (Chrome gives it the room, ui nothing), and replaced grid items, which stretch.
  - Background sizes (2026-11-17): `background-size` takes percentages of the padding box, `auto` on either side, `cover` and `contain`, and the `background` shorthand a size after its position and a slash; the shorthand now resets `background-size`. Paint works the tile's size out when it draws (`tileSize` in ui/paint/decorate.go), so a picture fetched late is sized when it lands; gradients take the size, position and repeat too, a gradient's own size being the padding box. As Chrome does, contain rounds the side it scales to a device pixel, and an image that doesn't repeat along an axis starts on a device pixel when it starts inside the box. Pixel page `background-size`.
  - Overlay scrollbars (decision 0083, 2026-11-17): `ui.OverlayScrollbars()` on a Document draws its bars over the content with no gutter, as macOS does; classic bars with a 15-point gutter stay the default, and no CSS chooses between them.
  - Scrolling viewport (decision 0084, 2026-11-17): `ui.ScrollingViewport()` on a Document has its viewport scroll as a browser's: css hands the root's overflow (or body's) to the viewport (`style.Box.Viewport`), layout puts a `box.Viewport` box above the root's, a scroll container as big as the window, and everything that reads a position goes through `Box.Rect`, so scrolling, hit testing, reveal, automation and `ui.Follow` see the scrolled page. Off by default: a game's screen never scrolls as a whole.
  - Media queries (decision 0086, 2026-11-17): `@media` is read by every sheet and answered for the Document's viewport in its own pixels (`css.Media`, `ui/css/media.go`); a rule keeps the indexes of the `@media` rules round it, `Transitions.SetMedia` reports whether which of them hold changed, and only then does `Document.Layout` restyle the tree.
  - Browser sheets (decision 0085, 2026-11-17): `ui.BrowserSheets()` on a Document reads its sheets and style attributes as Chrome does (`css.Browser()`): `@supports` answered against ui's own parser, `@layer` as real layers (`Rule.layer`, ranked across sheets by `layerRanks`), `box-sizing: content-box` from `css.BrowserDefault`, and vendor prefixes by one table per vendor (`ui/css/vendors.go`), `-webkit-`'s checked against Chrome. A sheet says where it came from with `ui.From(address)` (`css.From`), and its `url()` resolve there. Off by default: a game's sheet keeps its warnings.
  - Floats (decision 0080, 2026-11-17): `float`, `clear` and `display: flow-root`, Taffy's float layout with lines that go round them, and `text-indent`.
  - Columns (decision 0088, 2026-11-17): `column-count`, `column-width`, `columns`, `column-gap`, `column-rule`, `column-fill`, `break-inside`, `break-before`, `break-after`, `orphans`, `widows`. ui/layout/columns.go lays a container's content out once, as one column of the used width, lists what can't be cut (lines, whole boxes; empty room cuts anywhere), cuts that list into columns with Chrome's rules (`fillColumns`) at the balanced height (`balancedHeight`), and moves each node to its column when the tree is written (`keepFlow`, `writeColumns`). A cut box is one `box.Box`: `Broken`, its `Fragments` the parts, its rectangle the box round them; `box.Columns` on the container is what paint draws the rules from. Paint slices decorations by part (`pieces`) and hit testing takes the parts. Nothing runs for a Document without columns but one test in `dispatch`. Chrome: ui/layout/columns_chrome_test.go (`chrome.Match` also compares the parts with `getClientRects()`), pixel page `multicol`. Not matched: `column-span: all` (refused), flex containers and tables, which Chrome cuts and ui keeps whole.
  - Pseudo-elements (decision 0089, 2026-11-17): `::before`, `::after`, `::marker`, `::placeholder`, `::selection`, `::backdrop`; `content`, `counter-reset`, `counter-increment`, `counter-set`, `quotes`, and the numeric `list-style-type`s. ui/css/pseudo.go is the table of every pseudo-element Chrome reads and what ui does with it. A selector's pseudo-element puts its rule in that pseudo-element's own buckets of the sheet's index, so an element is asked for one only when a bucket it is in has a rule. ui/css/generated.go makes a `::before` or `::after` node (`dom.NewGenerated`, kept in the element's `Pipeline.Before`/`After`, its parent the element, never among its children) and fills it with text and `img` nodes; layout reads `Node.Flow()`, paint and retention see ordinary nodes, and `dom.Origin` takes a hit back to the element. ui/css/counters.go is one pass after the cascade, only in a tree that counts (`Pipeline.Counts`) and only after a change that can move a number: it walks the flow with a stack of counters, numbers list items through the implicit `list-item` counter (`Pipeline.Ordinal`), tracks the quote depth, and writes the text of generated boxes and markers. `q`'s marks are two rules of the user-agent sheet. chrome.Match compares generated boxes too, measured in Chrome through anchor positioning.
  - Selecting text (decision 0090, 2026-11-17): `user-select` (auto, text, none, all; inherited, `auto` on an element is text, as Chrome) and one selection per Document outside its fields. ember's sheet has `html { user-select: none }`, `css.BrowserDefault` `auto`. ui/selection.go holds it as two `Position{Node, Offset}`, character offsets in the text node's text as shown (`layout.Shown`), so it outlives layout; `box.Fragment.Start` says where each fragment's text starts in its node, and a `runMap`, made per text node when first asked and dropped at the next layout, places each fragment character in the node's text past the white space layout collapsed. From it come the hit test (`positionNear`: the nearest fragment that takes a selection, by line then across, inside the pressed element's box first), the copied text (`copier`, Chrome's `Selection.toString()`: `TestTheCopiedTextIsChromesSelectionToString`) and the drawn parts (`selectedParts`, cached by layout serial and ends). paint gets them through `paint.Frame.Selection`, nil while nothing is selected, so a Document without a selection pays nothing: a highlight rectangle per fragment before its text, the text drawn again clipped where `::selection` sets a `color`, a tint over a picture inside. ui/selecting.go is the pointer (press, double and triple click by `byWord`/`byBlock`, Shift, the drag's scroll) and the keys (`textedit.Resolve`, only while no field has the keyboard). `SelectionChanged` is emitted on `body`. export writes the ends as `data-ember-anchor`/`data-ember-focus` and a script that selects them, so the pixel page `selection` and `chrome.Eval` see Chrome's own selection.
  - `display: table` off a table (decision 0094, 2026-11-18): a flow root that shrinks to its content (`style.Box.Table`), which is what a clearfix and a centred block ask of it; rows and cells on other elements stay refused (tables as layout: use grid).
  - Not supported: old properties.
- **Pure Go, no cgo**, like `ui`, so servers and tests stay GPU-free. It draws through `twod.Canvas`.

## Shape of the API

```go
doc := uiv2.NewDocument(uiv2.Stylesheet(fsys, "tool.css"))
win.Add(doc)

doc.Body().Append(
	uiv2.Section(uiv2.Class("inspector"),
		uiv2.Details(uiv2.Open(),
			uiv2.Summary(uiv2.Text("tile 0")),
			uiv2.Div(uiv2.Class("props"),
				uiv2.Label(uiv2.For("solid"), uiv2.Text("solid")),
				uiv2.Input(uiv2.ID("solid"), uiv2.Type("checkbox"), uiv2.Checked(true)),
			),
		),
	),
)
```

```css
.props { display: grid; grid-template-columns: max-content 1fr; gap: 6px 12px; }
.props > * { min-width: 0; }
```

- Elements are plain values in a tree owned by one `Document` entity per window. That's cheaper than an entity per node, and the tree is the unit that's laid out.
- Constructors take attributes and children as options (decisions 0007, 0018). Setters change a live node (`SetText`, `SetAttr`, `AddClass`, `RemoveClass`, `Append`, `Remove`), and the next frame restyles and relays only what changed.
- Interaction is World events, as in `ui`: `uiv2.Clicked`, `Changed`, `Input`, `Submitted`, `Toggled` (details), `Closed` (dialog), `Sorted` (grid), each carrying the node, and `KeyPressed` for a key no control, menu, grid, listbox or focus move took, on the focused element, else the open modal dialog (none with neither: those keys are the game's). Focus, Tab and spatial navigation, pads, the clipboard, text editing (`internal/textedit`) and automation IDs (the path of names; an element's own part is its `id`, else its `data-name`, `aria-label`, text, label, placeholder, with `aria-hidden` text left out) are reused from `ui`. An element's `data-name` is an automation-only name, like `ui`'s `Name`: its leaf when it is recorded, and the scope of its descendants' IDs, without the uniqueness of an `id`. Automation's press on a grid row or listbox option selects and activates it; select only selects. A Document with an open modal dialog owns its window's input (decision 0053): from `ShowModal` to its close, other Documents, ember/ui and twod widgets in that window get none, and game systems read it as captured. While one of its elements (not its page, nor a `pointer-events: none` box) is under the pointer, a Document covers the layers drawn before it for the pointer, through the next frame (decision 0055): ember/ui and twod widgets under an open menu neither hover nor click; keys are untouched. `ShowModal` runs HTML's dialog focusing steps (its autofocus, else its first control, else the dialog itself, ringed); `Show` doesn't, so a tool window opened with it leaves the focus where it was. Menu items of role menuitemcheckbox turn when chosen and menuitemradio check among their group (Document.Checked reads them, automation records the state as their value); an item's aria-keyshortcuts ("Cmd+S", Cmd being Command on macOS and Control elsewhere; kit.Shortcut sets it) chooses it whenever its Document is drawn and not under another's modal, even with nothing focused, the one exception to unfocused keys being the game's: a focused text field keeps its editing keys, an open menu its moving keys, the keys are captured ahead while their modifiers are held, and of two Documents in a window the last drawn chooses, the frame after.
- A node's box is readable after layout (`n.Box()`), for tests and for games that place things by it.

### Stateful kit components

Most kit components are node builders: their behaviour is uiv2's (a grid, a listbox, a window) and the game reads the events. A component with logic of its own, such as `kit.FileDialog`, is a value made from the Document (`kit.NewFileDialog(doc, fsys, opts…)`, which appends its node to the body) that the game adds to the World as a system, as `collide.NewTracker`. Its `Update` reads uiv2's events and filters them by its subtree, so two such components, or one and the game's own UI, never cross. Two rules:

- **Lifetime follows the node:** while its node is out of the Document, or the Document out of the World, it reacts to nothing, emits nothing and drops what it holds (listings, fs handles); appending the node again brings it back.
- **Its events carry it:** `kit.FileChosen{Dialog}` and `kit.DialogCancelled{Dialog}` tell two dialogs on one page apart. They are emitted in `Update` and read the next frame, like uiv2's.

Its options are typed (`kit.FileDialogOption`), each with a getter of the same name, and setters only where a runtime change is meant (decision 0018). Table sorting stays root behaviour.

## Pipeline per frame

1. **Style:** match selectors and cascade into computed styles, only for nodes marked dirty (an attribute, class, state or sheet change).
2. **Layout:** block, inline, flex and grid formatting contexts with intrinsic sizing, porting Taffy's algorithms (MIT; the reference Bevy and Servo use) into `uiv2/layout`, a pure package with no twod. Text measures through the twod faces.
3. **Paint:** walk the box tree in stacking order into `twod.Canvas` draw lists: backgrounds, borders, shadows, text, clipping for `overflow`, and scroll containers with `ui`'s scrollbars.
4. **Hit testing and events:** from the box tree.

## The reference: Chrome

`uiv2/export` writes any tree as a real HTML page with its stylesheets. Under the `chrome` build tag, a test opens it in headless Chrome, reads every element's `getBoundingClientRect()` and compares it with `n.Box()`, within 0.5 px. Every layout feature lands with Chrome cases, and a screen counts as done only when Chrome and uiv2 agree. That's the end of "laid out the way the author thought". It's a test tool only; the engine never ships a browser.

Known differences, kept on purpose: a missing picture draws nothing (Chrome draws a broken-image icon with the alt text); automation reads alt.

## Packages

| Package | Holds |
|---|---|
| `ember/uiv2` | Document, elements, attributes, events, setters, controls' behaviour |
| `ember/uiv2/css` | tokenizer, parser of the subset, selectors, cascade, computed values |
| `ember/uiv2/layout` | box tree; block, inline, flex, grid; intrinsic sizes (pure) |
| `ember/uiv2/paint` | box tree to `twod.Canvas`, stacking, clipping, scroll |
| `ember/uiv2/export` | tree to HTML for the Chrome reference tests and for docs |

## Team and order

The ui lead owns `ember/uiv2` and this note's open questions, reviews every merge, and keeps the Chrome harness green.

| Milestone | Who | Done when |
|---|---|---|
| M0 skeleton: Document, five elements (div, span, p, button, input text), export, the Chrome harness | lead | a three-element page matches Chrome |
| M1 layout: block, inline text with wrapping and ellipsis, flex, grid, intrinsic sizes, overflow | ui developer 1 | Taffy's flex and grid fixtures ported and matching Chrome |
| M1 css: parser, selectors, cascade, var/calc, states | ui developer 2 | specificity and inheritance cases match Chrome's computed styles |
| M1 paint and controls: paint, scroll, focus, the form controls, details, dialog, table | ui developer 3 | every element's look and keyboard behaviour tested |
| M2 the tileset inspector rebuilt on uiv2 | lead + all | it matches the approved layout at 1100×700 and 1280×800, with Chrome agreeing |
| M3 FileDialog, settings and the demo's screens; `ui` layout retired | all | no screen left on `ember/ui` layout |

## Open questions (the lead's)

- Animation beyond `transition`: likely `@keyframes` later, for game menus.
- Whether `ember/ui`'s element types stay as thin wrappers over uiv2 during M3, or games move to uiv2 directly.
- Right-to-left text: out of scope until a game needs it.
- `placeholder` on `<select>` (lead, 2026-11-02): kept, as a uiv2 extension. A select with a placeholder starts with nothing chosen and shows the placeholder dimmed. Export writes it as the HTML pattern Chrome shows the same way, `<option value="" disabled selected hidden>…</option>`, so pages stay valid HTML and layouts compare.
- The default sheet (lead, 2026-11-03) follows Chrome's user-agent sheet, from Chrome's computed values on a bare page (hr, p, headings, monospace, disabled controls, dialogs and `:modal`, `::backdrop`, cursors, body's 8px margin). One exception: text inputs keep `overflow: hidden` rather than Chrome's `clip`, because uiv2 scrolls an input's own box to keep the caret in view, where Chrome scrolls an inner editor.
- Characters a face lacks (lead, 2026-11-02): they come from Go Mono, which has the arrows, triangles and ticks Go has not. Export gives Chrome the same face for exactly those characters through `unicode-range`. Lines stretch to fit the fallback's metrics, as Chrome's do (inline layout).

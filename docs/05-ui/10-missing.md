# What is missing

What ui does not do yet, where a page written for a browser would expect it. Each line is a known gap, not a choice: what ui refuses on purpose is in the [Refused table](03-css.md). A line leaves this page when its test against Chrome passes.

## Text

| Missing | Today |
|---|---|
| a selected empty line | not drawn, where Chrome draws a space-wide mark at its start |
| a double click on a word cut by an element (`wo<b>rd</b>`) | takes the part in the text node pressed |
| the selection of an unfocused window | drawn as a focused one's, where Chrome greys it; `::selection:window-inactive` is refused |
| `::selection` with `text-decoration`, `text-shadow`, `-webkit-text-stroke`, `-webkit-text-fill-color` | refused with a warning; Chrome reads them |
| `::selection`'s `color` in a text field | the field's text keeps its colour; only `background-color` applies |
| the default selection colour on Windows and Linux | macOS's blue everywhere |
| selecting by touch, dragging selected text, a rich (HTML) copy | none: the pointer and the keys select, and the clipboard gets plain text |
| text of a size between two whole ones (13.44px) | measured and placed at its size, drawn from the glyphs of the nearest whole size scaled to it: a little softer than Chrome's |
| a tab or a carriage return in `white-space: pre` text | no tab stops: measured with no width and drawn as the face has it (often `?`), so a selection stands left of its text; replace tabs by spaces before showing it |
| a drag's scroll sideways | a drag past an edge scrolls up and down only |
| `::first-line`, `::first-letter` | refused with a warning, the rule dropped |
| `text-decoration-skip-ink`, a decoration's shadow | a line is drawn through descenders, and `text-shadow` leaves it out |
| text on a display of one pixel to the point | a glyph's darkened edge shows on its left and top and, as rounding has it, not always on its right and bottom: the last half pixel of its quad has its centre on the quad's edge. Drawn every time, 27 of the 47 pixel pages with text measured further from Chrome at that scale and none nearer, so it stays; `-webkit-font-smoothing: antialiased` has no such edge |
| `-webkit-font-smoothing` on the options of an open `select` and on a picture's alt text | not read: drawn with the default smoothing |
| a `top` or `bottom` aligned inline box holding a shifted one (`sub`, a length…) | the line is sized from the boxes as the outer one aligns them; Chrome also counts the inner box where it stood before, a few points more |
| fonts a script loads (Typekit…) | ui runs no script: a sheet a browser's script adds gives its `@font-face` rules as any other, but a `FontFace` a script makes itself is not known, and the next family of the list is used |
| some installed fonts | `ui.SystemFonts()` skips a file it cannot read; nothing waits for the scan in the public API |
| a span ending a centred line | up to 3 points narrower than Chrome's |
| a control's own font, colour and line height under `ui.BrowserSheets()` | a `button`, `input`, `select` or `textarea` takes its parent's `font`, `color` and `line-height`, where Chrome's sheet starts each from its own (13.33px Arial, a textarea's monospace, `FieldText`, `normal`); so does `option` its `white-space` and `optgroup` its weight |
| a `word-spacing` or `letter-spacing` so far below zero that a run of text ends left of its start | the run's box is empty at its start, as Chrome's, but what follows on the line starts where the run's glyphs end, left of it, where Chrome starts it at the run's start; a block sized by such text can differ from Chrome's by a glyph |
| `letter-spacing` as a percentage | refused with a warning; Chrome reads it as that much of the font size, as `word-spacing`'s |
| `word-spacing` and `letter-spacing` in a select's list and a broken picture's alt text | the list and the alt text are measured without them |
| `word-spacing` on the first kept space of a block that collapses its own (`<div><span style="white-space: pre"> a</span></div>`) | widened as every other kept space; Chrome leaves that one space as the font has it |
| an em space or a thin space (U+2003, U+2009) | a few points wider than Chrome's in the Go faces |

## Generated content

| Missing | Today |
|---|---|
| `::before`, `::after` on `progress`, `meter`, and checkbox, radio and range inputs | no box, where Chrome makes one |
| `::marker` with `line-height`, `text-shadow`, `white-space`, `overflow-wrap`, `cursor`, `quotes`, transitions | read, and they change nothing |
| `::before::marker`, `::after::marker` | refused, where Chrome reads them |
| `content: url()` on an element itself | read, and the element is not replaced |
| `counter-reset: reversed(name)` | refused |
| `ol type` | not read: `list-style-type` says the numbering |

## Layout

| Missing | Today |
|---|---|
| `column-span: all` | refused: put the element before the columns |
| a flex container or a table cut between rows by a column | kept whole |
| a positioned box cut by a column | kept whole |
| columns right to left | left to right only |
| `border-style: double`, and a collapsed border's `inset` and `outset` as Chrome shades them | `double` is refused; the two are drawn as a groove and a ridge |
| `visibility: collapse` on a row or a column, `empty-cells: hide` | the row or the column keeps its room; an empty cell keeps its borders and background |
| a table's collapsed borders where they meet at a corner, borders of different widths | the wider way goes through and the other stops beside it; Chrome's joins differ by a few pixels there |
| a float and text beside it in a box that shrinks to fit (a float, an inline-block, a `display: table`) | as wide as the wider of the two, where Chrome adds them |
| `border-inline`, `border-block` and their `-width`, `-style`, `-color`, `overflow-inline`, `overflow-block` | refused; the logical names of one side are [read](03-css.md#logical-properties) |
| a selector naming an element ui has none of inside `:not()` under `ui.BrowserSheets()` | its rule is dropped with a warning, where Chrome's matches every element |
| the diagonal where two solid border sides of other colours meet | not smoothed, where Chrome blends the two colours along it |
| a picture with a ratio and no size (an SVG with only a `viewBox`) as an unsized flex item or absolutely positioned | no width, where Chrome gives it one: give it a width |
| a replaced grid item | stretched to its cell |
| the `width` and `height` attributes of an `img` as its proportions, when a rule takes one of the two back (`height: auto`, as responsive sheets do) | the box has its broken picture's proportions until the picture lands, then is laid out again, where Chrome keeps the room the attributes give (`aspect-ratio: auto 250 / 167`) |
| a picture made a block (`display: block`) with a percentage `width` or `max-width`, in a box as wide as its content (a float, a flex item, a grid track) | counts for its own width at the box's narrowest, where Chrome counts it for none, so the box is wider, and a float round one is 3 points taller; one left inline counts for none |
| a grid line named by no area (`grid-column-end: b` with no area `b`), or `grid-area: nowhere` | auto, where Chrome makes a line of that name after the grid |
| names of lines in a track list (`[full-start] 1fr [full-end]`) | refused |
| a `position: fixed` box in an element that scrolls on its own | moves with that element |
| an absolutely positioned box placed by a block above an element that scrolls | not cut by that element, as in Chrome, but moves with its scroll |
| a part with no text of an inline element cut by a block inside it, when it has a border or a padding | no line and no box, where Chrome keeps a line for it |
| `position: relative`, `opacity` on an inline element holding blocks | its lines move and fade, the blocks inside it do not |
| an empty inline element after a `br` that ends its block | its box of no size stands on the `br`'s line, where Chrome puts it a line lower |
| a `summary`'s marker | about 3 points narrower than Chrome's, so what follows sits left |
| `disabled` on a `form` or a `fieldset`, `bdo`, a `font` element's attributes | read, and they do nothing |
| the presentational attributes but those [read](03-css.md): `border`, `rules` and `frame` on a `table`, `align="left"` and `"right"` on a `table` or an `img`, `clear` on a `br`, `hspace`, `vspace`, `background`, `link`, `vlink`, `size` and `noshade` on an `hr`, `align` and `valign` on a `col` or a `colgroup` | read, and they do nothing |
| a percentage width on a cell spanning columns | not counted: the cell is as wide as its content asks |

## Backgrounds and borders

| Missing | Today |
|---|---|
| several background layers | one: in the `background` shorthand the first with a picture, over the last one's colour; in a longhand the list is refused |
| `background-clip: text`, `background-repeat: space` and `round` | refused |
| `background-attachment: fixed` on an element under a transform | placed in the viewport, where Chrome places it in the element |
| a scaled picture's tile seams | each tile keeps its own edge, where Chrome blends across: about a pixel row |
| a `border-image` from an SVG on a dense screen | rasterised at one pixel a point, so blurred |
| a canvas picture in a Document without `ui.ScrollingViewport()` | placed in the window, where Chrome places it in `html`'s box |

## Sheets

| Missing | Today |
|---|---|
| `@import` | not followed |
| `@keyframes`, `animation` | refused: `transition` |
| `@font-face` | fetched with `ui.FetchFonts` ([decision 0099](../99-decisions/0099-a-pages-fonts-are-fetched-and-woff2-is-read-in-pure-go.md)); what is left: `font-display` is always `swap`; `font-stretch`, `size-adjust`, `ascent-override`, `descent-override`, `line-gap-override`, `font-feature-settings` and `font-variation-settings` are read and do nothing; a variable font draws its default instance; no bold or italic is made up from a regular face; colour fonts, collections and `CFF2` are not read; a rule inserted alone (`InsertRule`) is not read, and a face stays when its sheet is removed |
| `prefers-color-scheme`, `hover`, `pointer`, `prefers-reduced-motion` | fixed answers: light, a mouse, no preference |
| viewport units in a media query's length | refused |
| `-webkit-linear-gradient()` and the other prefixed gradients, `display: -webkit-box`, `-webkit-fill-available` | keep their warning under `ui.BrowserSheets()`: Chrome reads them and ui cannot |
| `filter`, `clip-path` | refused |
| a `mask-image` over anything but the background colour: a background image, a border, the content | drawn whole; only the colour takes the picture's shape (decision [0093](../99-decisions/0093-a-mask-image-is-the-shape-of-the-background-colour.md)) |
| the `mask` shorthand, a gradient mask, several layers, `mask-origin`, `mask-clip`, `mask-mode`, `mask-composite` | refused; `-webkit-mask-position` keeps its warning, write `mask-position` beside it |
| `:has()`, `:focus-within`, `:lang()`, `:empty`, `:target`, `:read-only`, `:read-write`, the `s` flag of an attribute selector | refused: the rule is dropped |
| `min()`, `max()`, `clamp()` in a length, `vh` and `vw` | refused |
| `list-style-image`, `direction`, `word-break`, `mix-blend-mode`, `font-variant`, `@counter-style`, `revert` | refused |

## Windows

| Missing | Today |
|---|---|
| `env(titlebar-area-x)` and the other window-controls-overlay values | not read: a header pads itself in Go from `win.TitleBarArea()` |
| `app-region` as Chrome computes it | inherited here, where Chrome's is not: a child of a drag box computes `drag`, not `none`; the regions are the same |
| a header merged with the title bar on Windows and Linux | `platform.OwnTitleBar()` keeps the system's title bar there, and `app-region: drag` does nothing |
| leaving a text field from code | no `doc.Blur()`: the web explorer's address field keeps the focus after Enter |
| a key pressed in a focused text field, told to a callback | `OnKey` hears none, where a page's `keydown` hears every key and may prevent what it does: light's console walks its history by Alt+Up and Alt+Down (menu shortcuts), not the arrows alone, and has no Shift+Enter for a new line |
| setting a focused text field's value while it is typed in | `doc.SetValue` waits for a change or a submit: light's console moves the focus away and back to show a line of its history |
| a context click by Control and the main button on macOS | only the secondary button raises `OnContextMenu` |
| the place of a context click | the event carries none: read `win.Input().Mouse()` in the callback |
| a press on an element and the pointer's moves while it is held (`pointerdown`, `pointermove`, pointer capture) | no callback hears them: light's developer tools read `win.Input().Mouse()` and `Down` each frame to drag their edge, and find what was pressed with `doc.ElementAt` |
| the tooltip of a `title` attribute | not shown: light's Network rows name their address in its status bubble instead |
| a `role="menu"` shown without an opener closing on a choice | stays open: its item's callback removes `open` |

## Frames

| Missing | Today |
|---|---|
| a select's list, a menu, a tooltip or a dialog of a Document in an `iframe`, where it passes the frame's box | cut at the box: the top layer is the framed Document's own |
| a modal dialog in a framed Document | holds that Document's input, not the page's round it |
| the arrows, a pad's stick and Back (Escape) across a frame | stay in the Document that has the focus: only Tab walks in and out |
| an `OnKey` or `OnBack` of the page round a frame, for a key pressed inside it | not heard: keys bubble inside one Document |
| a selection running from a page into its frame, and select all over both | each Document selects its own text, one at a time |
| `opacity` below 1 on an ancestor of an `iframe` | doesn't fade it, as a canvas |
| an element lying over an `iframe` | the page is painted each frame while it does, where it replays with nothing over the frame |
| `aspect-ratio` on an `iframe` with neither a width nor a height | 300 wide and the ratio's height, where Chrome fills the line |
| a percentage `height` on an inline `iframe`, `canvas` or `img` | read as `auto`; made a block, it is the share of its container |
| `scrolling="no"`, `allowtransparency`, `loading`, `sandbox`, `allow`, `name` | not read: what a frame may do is the program's |
| a black `border: 2px inset` on an `iframe` | drawn as on any box, where Chrome keeps the greys of its default border |
| an `iframe` moved to another Document | lets go of what it showed, as one taken out: `SetFrame` there shows it again |
| a panic of a framed Document outside its own frame: a call the program makes on it, a fetch it started | the program's own, with or without `ui.Guarded()` |
| a control taken out of its Document (a field, a canvas, an `iframe`) | its node and state stay held by the Document until the Document goes; an `iframe` lets go of what it showed |

## Pictures

| Missing | Today |
|---|---|
| an animated WebP | no picture, and a warning (`webp: invalid format`); an animated GIF shows its first frame |
| AVIF, BMP, ICO | no picture, and a warning; `web.Images` asks for none of them |
| `srcset` and `sizes` on an `img`, `picture` with its `source`s | not read: the `img`'s `src` is the picture |
| the shapes inside an `svg` element | not elements of the tree: no rule of the Document reaches a `path` or a `g` (`svg path { fill: … }` matches nothing), and `fill` or `stroke` set on the `svg` by a rule is not read; presentation attributes and the element's own `style` are |
| an `svg` element with neither a size nor a `viewBox` as a flex item | keeps its 300 points where Chrome shrinks it |
| an `svg` element whose markup changes every frame | each state is one more picture the Document keeps |

## SVG pictures

| Missing | Today |
|---|---|
| `text`, `clip-path`, `mask`, `filter`, `pattern`, `image`, `marker`, `symbol`, a nested `svg`, animation | not drawn, and named in a warning; `clip-path` is the commonest in real logos |
| a `transform` on the root, `!important` in a style, a UTF-16 file | ignored, ignored, rejected |
| hairline strokes, dashes along curves | differ a little from Chrome's |
| `currentColor` in an `img` or a `url()`, `image-rendering: pixelated` | black, as in Chrome, where only an `svg` element takes the page's colour; ignored |
| an SVG as a twod texture | by hand: `svg.LoadPicture`, `Rasterise`, `twod.NewTexture` |

## Not checked

Things built and tested, with a part no test against Chrome covers yet.

- `background-size: cover` and `contain` under a Document `Zoom`.
- Selecting with the pointer and the keys: Chrome cannot be driven headless, so the places a press, a double click and Shift with a key reach are tested in ember alone; the copied text, the rectangles and the picture are compared with Chrome's.
- `user-select: auto` set inside `user-select: all`, read as `text`.
- `app-region`: headless Chrome has no window to drag, so drag regions are tested in ember alone.
- A selection inside a transformed box other than a translation, and under a Document `Zoom`.
- A float low in a column, at narrow widths.
- A percentage `top` on a relative child inside columns.
- The largest raster of an SVG (8192 pixels a side) against the device's own limit.
- The Linux and Windows thresholds of the pixel pages added since `viewport-top`.

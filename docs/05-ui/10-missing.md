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
| a first letter's text | not selected or copied with the rest of its text node, as `::before`'s text is not |
| `initial-letter` | refused as an unknown property: float a `::first-letter` for a drop cap |
| `::first-line`'s `opacity`, `box-shadow`, `visibility`, `vertical-align` | read, and they change nothing |
| a first letter in text a translation changes; one after spaces `white-space` keeps | none is taken; the spaces are dropped |
| a block's min-content width under a `::first-line`, and the padding of inline elements in `em` on it | measured in the elements' own styles |
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
| `visibility: collapse` on a row or a column, `empty-cells: hide` | the row or the column keeps its room; an empty cell keeps its borders and background |
| a table's collapsed borders where they meet at a corner, borders of different widths, or as wide in a style of two shades (`inset`, `outset`, `groove`, `ridge`) | the wider way goes through and the other stops beside it, as wide the one across; Chrome's joins differ by a few pixels there, an edge down going through at the table's right corners and between cells |
| a selector naming an element ui has none of inside `:not()` under `ui.BrowserSheets()` | its rule is dropped with a warning, where Chrome's matches every element |
| `align-self: baseline` in a subgrid | its items align on their own baseline, not shared with the grid round it; aligned at the start where they share the grid's rows |
| a subgrid's own `repeat(auto-fill, …)` of names beside a grid's names on its lines | the grid's names stay on its lines, where Chrome drops some of them |
| the height of a subgrid's item sizing the grid's columns, in a subgrid of columns alone | measured without the subgrid's rows: a percentage height or an aspect ratio there counts as on no row |
| `opacity` on an inline element holding blocks | its lines fade, the blocks inside it do not |
| an absolutely positioned box inside an inline element `position: relative` moves | placed where the element stood before it moved |
| an `img` whose picture is broken, as tall as its attributes' proportions make it (`width: 100%; height: auto`) | stands on its line by its bottom, where Chrome's stands by the line of its icon and alt text inside it: its line is 3 points taller than Chrome's |
| `bdo`, a `font` element's attributes | read, and they do nothing |
| the presentational attributes but those [read](03-css.md): `border`, `rules` and `frame` on a `table`, `align="left"` and `"right"` on a `table` or an `img`, `clear` on a `br`, `hspace`, `vspace`, `background`, `link`, `vlink`, `size` and `noshade` on an `hr`, `align` and `valign` on a `col` or a `colgroup` | read, and they do nothing |
| a percentage width on a cell spanning columns | not counted: the cell is as wide as its content asks |

## Backgrounds and borders

| Missing | Today |
|---|---|
| a rounded border whose sides differ in width or in style, or that is `inset`, `outset`, `groove` or `ridge`; a rounded `dashed` or `dotted` one | drawn square; the dashed or dotted one drawn solid |
| a rounded border of three or four colours | each meets the next on Chrome's line, with a trace of the first colour through the blend where two later ones meet |
| `outline-style` but `solid` and `double` (`dashed`, `dotted`, `inset`, `outset`, `groove`, `ridge`, `auto`) | drawn solid |
| `background-clip: text` | refused |
| `background-attachment: fixed` on an element under a transform | placed in the viewport, where Chrome places it in the element |
| a scaled picture's tile seams, the gaps of `background-repeat: space` | each tile keeps its own edge, where Chrome blends across: about a pixel row; a gap of a fraction of a pixel is a whole pixel or none, where Chrome blends it |
| a `border-image` from an SVG on a dense screen | rasterised at one pixel a point, so blurred |
| a canvas picture in a Document without `ui.ScrollingViewport()` | placed in the window, where Chrome places it in `html`'s box |

## Sheets

| Missing | Today |
|---|---|
| an `@import` inserted with `InsertRule` after other rules | followed, where Chrome refuses the insertion |
| an animated or transitioned `font-weight` | moves in whole numbers, where Chrome keeps quarters |
| an animated border, outline or column-rule width, `outline-offset`, `border-spacing` | snapped to whole points, where Chrome snaps to device pixels: steps of half a point on a dense screen |
| an animated `border-image-slice` from a percentage to a number, `border-image-width` from a length to a percentage | interpolates the numbers and flips, where Chrome flips and makes a `calc()` |
| an animated `background-image`, `mask-image` or `border-image-source` between two pictures | flips halfway, where Chrome cross-fades them |
| `display` in a keyframe, `transition-behavior: allow-discrete` | dropped with a warning, and not read |
| an element's own transition of a property it inherits, while an ancestor's transition moves it | follows the ancestor's moving value, where Chrome runs it on to the end value |
| `animation-composition`, `animation-timeline`, `animation-range`, animations of `::marker` and `::placeholder`, the animation objects of a page's script | not read |
| `@font-face` | fetched with `ui.FetchFonts` ([decision 0099](../99-decisions/0099-a-pages-fonts-are-fetched-and-woff2-is-read-in-pure-go.md)); what is left: `font-display` is always `swap`; `font-stretch`, `size-adjust`, `ascent-override`, `descent-override`, `line-gap-override`, `font-feature-settings` and `font-variation-settings` are read and do nothing; a variable font draws its default instance; no bold or italic is made up from a regular face; colour fonts, collections and `CFF2` are not read; a rule inserted alone (`InsertRule`) is not read, and a face stays when its sheet is removed |
| `prefers-color-scheme` and `prefers-reduced-motion` on Linux and the web | no preference: neither says its scheme or its reduced motion yet, so a Document given `ui.Preferences` answers light and no preference there ([decision 0121](../99-decisions/0121-a-document-answers-media-features-from-the-preferences-it-is-given.md)) |
| `<meta name="color-scheme">` | not read: ui reads no page's `head`; a program reading a page puts `:where(:root) { color-scheme: … }` in a sheet of its own ([decision 0126](../99-decisions/0126-an-element-is-drawn-in-the-scheme-its-color-scheme-allows.md)) |
| the system's own system colours | Chrome's on macOS on every system, where Chrome on Windows and Linux takes `Highlight`, `SelectedItem` and their texts, and on Windows its light buttons, links and grays, from the system |
| a dark control's hover and press, a dark select's chevron, meter's dark fills, the scrollbars, an iframe whose scheme isn't its host's | a hovered or pressed button keeps its dark face, the chevron stays dark, a meter fills in its light colours, the bars keep ember's own look in either scheme, and an iframe's canvas is never made opaque, where Chrome draws its own dark ones and backs an iframe of the other scheme |
| `prefers-contrast`, `forced-colors`, `any-hover: none` on a touch-only device | fixed: no preference, none, and a mouse that hovers is always there |
| `ex`, `ch` and the font's other units in a media query's length | refused: Chrome measures them in the initial font, ui would take CSS's fallbacks |
| `-webkit-linear-gradient()` and the other prefixed gradients, `display: -webkit-box`, `-webkit-fill-available` | keep their warning under `ui.BrowserSheets()`: Chrome reads them and ui cannot |
| a `url()` filter, in `filter` or `backdrop-filter`; `mix-blend-mode: plus-lighter`, `plus-darker` | refused |
| a backdrop filter inside an isolated group that is no backdrop root (`isolation: isolate`, a `z-index` holding a mixed element) | reads the group only, where Chrome reads on to the backdrop root ([decision 0130](../99-decisions/0130-a-mixed-element-reads-its-group-so-far-as-a-picture.md)) |
| a mixed element or a backdrop filter reaching the page | mixes with the Document's canvas, not with what the game drew under the Document |
| `opacity` below 1 on an ancestor of an element drawn as a picture, itself not one | each of its pictures and boxes faded apart, where Chrome fades the ancestor as one |
| a `clip-path` inside another's (one on an element held by one) | the inner shape cuts, the outer one only by its bounds ([decision 0119](../99-decisions/0119-a-clip-path-is-a-mask-the-gpu-cuts-by-and-an-outline-the-pointer-is-tested-against.md)) |
| `clip-path` on an inline element over several lines | placed in the box round all its lines; not compared with Chrome |
| `path()` transitioning to another `path()` with the same commands | flips half way; Chrome moves each point |
| a pointer in the corner a rounded border cuts off | hits the box, where Chrome lets it through to what is under it |
| a filtered element's live parts (a caret, a canvas element) when its Document draws them apart, and a filtered inline element holding absolute boxes | drawn unfiltered over it, placed by the nearest positioned block ([decision 0118](../99-decisions/0118-a-filtered-element-is-a-picture-drawn-through-wgsl-effects.md)) |
| a `transform` holding fixed descendants | placed in the viewport, where Chrome places them in the transformed element, as it does under a `filter` |
| a blur reaching past the window | what lies past the window's edge is not blurred in |
| a `mask-image` over anything but the background colour: a background image, a border, the content | drawn whole; only the colour takes the picture's shape (decision [0093](../99-decisions/0093-a-mask-image-is-the-shape-of-the-background-colour.md)) |
| the `mask` shorthand, a gradient mask, several layers, `mask-origin`, `mask-clip`, `mask-mode`, `mask-composite` | refused; `-webkit-mask-position` keeps its warning, write `mask-position` beside it |
| a `:has()` reading, through an `:is()` or a `:not()` in it, an element above the one it is on (`.card:has(:is(.dark img))`, `.dark` on `body`) | matched, but a change of that element above does not restyle the card |
| `:dir()` on an element of `dir="auto"` whose text changes below it (a field typed in excepted) | answered from the text as it stood when the element was last restyled |
| a `pattern` Go's `regexp` cannot read (a lookahead, a backreference) | constrains nothing, where Chrome reads it as JavaScript does |
| an email address whose domain is not in ASCII letters | invalid, where Chrome turns the domain to punycode first |
| `:user-invalid` after a form is submitted | ui submits no form: only the fields the player changed or submitted match |
| a radio group by its form | grouped by name across the Document, where Chrome groups by name within each form |
| `ex` in a bold or italic the engine makes from another face | the x-height of the face it is made from, where Chrome measures the thickened `x` |
| `scroll-state()` container queries and `container-type: scroll-state`; `style()` on a standard property | refused, with a warning: the condition never holds |
| a `style()` query compared with a custom property whose value differs only in its white space (`--d: 1px  2px`) | equal, where Chrome compares the text as written |
| `@keyframes` inside `@container` | applies whatever the container |
| container queries nested more than eight containers deep, each answer moving the next | answered as far as the eighth in one layout, the rest the next frame |
| math functions in a colour's channels and in times (`calc(1s + 100ms)`) | refused |
| a percentage inside `sign()` in a `line-height` | of the font size, where Chrome measures it against something else |
| `list-style-image`, `direction`, `word-break`, `font-variant`, `@counter-style`, `revert` | refused |

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
| AVIF | no picture, and a warning; `web.Images` does not ask for it |
| a run-length encoded BMP (or such a bitmap in an ICO), a `.cur` cursor file | no picture, and a warning |
| the picture of an ICO nearest the size it shows at | the largest of the file is shown, scaled, then the one of the most bits a pixel |
| a `source` of AVIF (`type="image/avif"`) | passed over, where Chrome takes it: the next source, or the `img`'s own pictures |
| `loading="lazy"` | read only to let `sizes` say `auto`: every picture is asked for when it is laid out, where Chrome waits for it to near the viewport |
| the `width` and `height` attributes of a `source` | not read, where Chrome sizes the `img` by those of the source it chose |
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
- `:user-valid` and `:user-invalid`: no player changes a field in headless Chrome, so they are tested in ember alone; Chrome is asked only that nothing matches them before.
- The picture `sizes="auto"` chooses: Chrome chooses it once the page is laid out, and takes a denser one already in its cache, so what it shows changes from run to run; the box is compared with Chrome's, the choice is tested in ember alone.
- Choosing a picture again as the viewport or the pixels a point change, the one shown kept until the one chosen lands: headless Chrome is neither resized nor moved to another screen, so it is tested in ember alone.
- A selection inside a transformed box other than a translation, and under a Document `Zoom`.
- A float low in a column, at narrow widths.
- A percentage `top` on a relative child inside columns.
- The largest raster of an SVG (8192 pixels a side) against the device's own limit.
- The Linux and Windows thresholds of the pixel pages added since `word-spacing`: those of `iframe`, `table-model`, `background-boxes`, `word-spacing`, `vertical-align` and `font-smoothing` are what the runners measured once, their text drawn differently there.
- The `mitres` page on the macOS and Linux runners, where Chrome's screenshot has the page's background and none of its boxes: it is compared on a desktop Mac and on the Windows runner only.

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
| monospace text at a fractional size (a `code` in 13px text is 10.5625px) | drawn a little wider than it is measured, so a selection after it covers the edge of its last letter |
| a tab or a carriage return in `white-space: pre` text | no tab stops: measured with no width and drawn as the face has it (often `?`), so a selection stands left of its text; replace tabs by spaces before showing it |
| a drag's scroll sideways | a drag past an edge scrolls up and down only |
| `::first-line`, `::first-letter` | refused with a warning, the rule dropped |
| `text-decoration-skip-ink`, a decoration's shadow | a line is drawn through descenders, and `text-shadow` leaves it out |
| `vertical-align` but `baseline`, `sub`, `super` | refused |
| fonts a script loads (Typekit…) | ui runs no script: the next family of the list is used, so lines wrap elsewhere |
| some installed fonts | `ui.SystemFonts()` skips a file it cannot read; nothing waits for the scan in the public API |
| a span ending a centred line | up to 3 points narrower than Chrome's |

## Generated content

| Missing | Today |
|---|---|
| `::before`, `::after` on `progress`, `meter`, `table`, row groups, rows, and checkbox, radio and range inputs | no box, where Chrome makes one |
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
| `display: table` for layout, `border-collapse` | refused: grid, `border-spacing` |
| a picture with a ratio and no size (an SVG with only a `viewBox`) as an unsized flex item or absolutely positioned | no width, where Chrome gives it one: give it a width |
| a replaced grid item | stretched to its cell |
| a `position: fixed` box in an element that scrolls on its own | moves with that element |
| a `summary`'s marker | about 3 points narrower than Chrome's, so what follows sits left |
| `disabled` on a `form` or a `fieldset`, `bdo`, a `font` element's attributes | read, and they do nothing |

## Backgrounds and borders

| Missing | Today |
|---|---|
| several background layers | one |
| `background-origin`, `background-clip`, `background-attachment` | not read: a background is placed in the padding box and cut to it, so it does not show under a translucent or dashed border |
| a scaled picture's tile seams | each tile keeps its own edge, where Chrome blends across: about a pixel row |
| a `border-image` from an SVG on a dense screen | rasterised at one pixel a point, so blurred |
| a canvas picture in a Document without `ui.ScrollingViewport()` | placed in the window, where Chrome places it in `html`'s box |

## Sheets

| Missing | Today |
|---|---|
| `@import` | not followed |
| `@keyframes`, `animation` | refused: `transition` |
| `@font-face` in a `ui.CSS` sheet | warns: fonts are read from a `Stylesheet`'s files |
| `prefers-color-scheme`, `hover`, `pointer`, `prefers-reduced-motion` | fixed answers: light, a mouse, no preference |
| viewport units in a media query's length | refused |
| `-webkit-linear-gradient()` and the other prefixed gradients, `display: -webkit-box`, `-webkit-fill-available` | keep their warning under `ui.BrowserSheets()`: Chrome reads them and ui cannot |
| `filter`, `clip-path`, `mask` | refused |

## Windows

| Missing | Today |
|---|---|
| `env(titlebar-area-x)` and the other window-controls-overlay values | not read: a header pads itself in Go from `win.TitleBarArea()` |
| `app-region` as Chrome computes it | inherited here, where Chrome's is not: a child of a drag box computes `drag`, not `none`; the regions are the same |
| a header merged with the title bar on Windows and Linux | `platform.OwnTitleBar()` keeps the system's title bar there, and `app-region: drag` does nothing |
| leaving a text field from code | no `doc.Blur()`: the web explorer's address field keeps the focus after Enter |
| a context click by Control and the main button on macOS | only the secondary button raises `OnContextMenu` |
| the place of a context click | the event carries none: read `win.Input().Mouse()` in the callback |
| a `role="menu"` shown without an opener closing on a choice | stays open: its item's callback removes `open` |
| a Document drawn in another's canvas, under an element of the outer one | still takes the pointer there: the outer stops drawing it (`doc.Invalidate`) while its own menu is open over it |

## SVG pictures

| Missing | Today |
|---|---|
| `text`, `clip-path`, `mask`, `filter`, `pattern`, `image`, `marker`, `symbol`, a nested `svg`, animation | not drawn, and named in a warning; `clip-path` is the commonest in real logos |
| a `transform` on the root, `!important` in a style, a UTF-16 file | ignored, ignored, rejected |
| hairline strokes, dashes along curves | differ a little from Chrome's |
| `currentColor` from the page, `image-rendering: pixelated` | black, ignored |
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

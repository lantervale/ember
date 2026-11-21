# CSS

![A card styled by its style attribute: a grid of an orange disc beside a title and a muted line](styled.png "ember · styled")

```go
ui.Div(ui.Style("display: grid; grid-template-columns: 48px 1fr; gap: 4px 12px; border-radius: 8px"),
	ui.Div(ui.Style("grid-row: span 2; width: 48px; height: 48px; border-radius: 24px; background-color: #e0a040")),
	ui.Strong(ui.Text("Ember Quest")),
	ui.Span(ui.Text("Area 2 · 1.5k to home")),
)
```

Stylesheets come from `ui.Stylesheet(fsys, name)`, after the user-agent sheet; `ui.Style(css)` is an element's style attribute, adding to what earlier arguments, such as `kit.Padding`, set. What a sheet gets wrong is in `doc.Warnings()`, never a crash.

The presentational attributes of old pages style their element as Chrome reads them, under every rule of the sheets (decision 0092): `width` and `height` on an `img`, an `iframe`, a `table`, a `td` or a `th` (pixels or a percentage), `bgcolor` on `body` and a table's parts, `text` on `body`, `align` on a `div`, a `p`, a heading and a table's parts, `valign` on a cell, its row or its row group, `nowrap` on a cell, `cellspacing` and `cellpadding` on a `table`. The others are in [What is missing](10-missing.md).

| Selectors | |
|---|---|
| `div`, `.class`, `#id`, `[attr]`, `[attr="value"]` | simple |
| `[attr~="word"]`, `[attr\|="en"]`, `[attr^="start"]`, `[attr$="end"]`, `[attr*="part"]`, each with the `i` flag | an attribute's value by a word, a prefix before a hyphen, its start, its end, a part; `i` compares whatever the case. The `s` flag drops the rule, as Chrome drops it |
| `a b`, `a > b`, `a + b`, `a ~ b` | descendant, child, the next sibling, a later sibling |
| `:hover`, `:focus`, `:focus-visible`, `:active`, `:disabled`, `:enabled`, `:checked`, `:open` | states; `:enabled` and `:disabled` match controls only |
| `:focus-within` | the focused element and its ancestors; an iframe holding its Document's focus holds the page's, so the elements round it match too |
| `:target` | the element `doc.SetTarget(n)` made the Document's target, as a browser does the one a fragment names; `ScrollToStart` scrolls to it |
| `:read-write`, `:read-only` | a text field (an `input` typed in, a `textarea`) neither `readonly` nor disabled, or an element a `contenteditable` on it or above it makes editable; `:read-only` is every other element |
| `:lang(en)` | the `lang` attribute of the element or of the nearest one above it, else the Document's language (`ui.Language(tag)`, `doc.SetLanguage`, a page's Content-Language): that range or it followed by a hyphen, whatever the case. As in Chrome, one identifier only, no list and no string, both tags well formed, and a `*` matches nothing |
| `:first-child`, `:last-child`, `:only-child`, `:nth-child()`, `:nth-last-child()`, `:first-of-type`, `:last-of-type`, `:only-of-type`, `:nth-of-type()`, `:nth-last-of-type()`, `:root`, `:empty` | structure; `:empty` is an element with no element in it and no text but empty text (a space is text, as in Chrome) |
| `:not()`, `:is()`, `:where()` | a list of selectors: `:is()` and `:not()` weigh as their most specific one, `:where()` nothing; a selector `:is()` or `:where()` cannot read is left out of its list, as in Chrome |
| `:has()` | relative selectors from the element: `.card:has(img)`, `li:has(> a)`, `h2:has(+ p)`, `:has(~ .x)`, weighing as their most specific one. As in Chrome, a `:has()` in another, a pseudo-element or an empty one drops the rule; a change below or after an element restyles it where its `:has()` reads what changed (decision [0113](../99-decisions/0113-has-flags-its-anchors-before-the-cascade.md)) |
| `:link`, `:any-link`, `:visited` | an element with an `href`; nothing is visited, ui keeping no history, so a `:visited` rule is kept and matches nothing |
| `::before`, `::after`, `::marker`, `::placeholder`, `::selection`, `::backdrop` | pseudo-elements, last in their selector: `li:nth-child(2)::before`, `a:hover::after` ([below](#pseudo-elements)) |

| Values | |
|---|---|
| `px` (a point), `%`, `em`, `rem`, `fr`, and the absolute `pt`, `pc`, `in`, `cm`, `mm`, `Q` at 96 `px` an inch | units |
| `ex`, `ch`, `cap`, `ic`, `lh`, and the root's `rex`, `rch`, `rcap`, `ric`, `rlh` | the element's font: its x-height, the advance of its `0`, the height of its capitals, the advance of its `水`, its line height; half an em, half an em, the ascent and an em where the face says nothing, as Chrome. In `font-size` they read the parent's font, and `lh` in `line-height` the parent's line. A font that lands styles again the lengths written in them |
| `vw`, `vh`, `vmin`, `vmax`, `vi`, `vb`, and each with `s`, `l` or `d` before it (`dvh`, `svw`…) | hundredths of the Document's viewport, an iframe's being its own box; the small, large and dynamic viewports are the one there is. A viewport that changes styles again the lengths written in them, and nothing where none is |
| `auto`, `min-content`, `max-content`, `fit-content`, `minmax()`, `repeat()`, `calc()`, `var()` | sizes and functions |
| `min()`, `max()`, `clamp()` | the least, the greatest, a value between two, of lengths and percentages or of numbers, in each other and in `calc()`; over a percentage they are worked out by layout against what the percentage is of, as `width: min(100%, 500px)` |
| hex, `rgb()`, `oklch()`, named colours, `transparent`, `currentColor` | colours |
| `linear-gradient(<angle> or to <side or corner>, <colour> [<length or %> [<length or %>]], …)` | in `background-image` or `background`: two stops or more, two positions on one stop a hard edge; `deg`, `rad`, `grad` or `turn`; `to` a corner follows the box's shape; colours blend in encoded sRGB, as Chrome's |
| `radial-gradient([circle or ellipse] [closest-side, closest-corner, farthest-side or farthest-corner] [at <position>], <colour> [<length or %>], …)` | an ellipse to the farthest corner from the middle by default; length stops measure along the horizontal radius |
| `conic-gradient([from <angle>] [at <position>], <colour> [<angle or %> [<angle or %>]], …)` | stops at angles or percentages of a turn, two positions a hard stop: a cooldown sweep is `conic-gradient(transparent 0 70%, rgba(0,0,0,.72) 70% 100%)` |
| | all three blend in encoded sRGB as Chrome's and are cut to the box, rounded or not |

| Properties | |
|---|---|
| layout | `display` (block, inline, inline-block, flex, inline-flex, grid, flow-root, table, inline-table, table-row-group, table-header-group, table-footer-group, table-row, table-cell, table-column, table-column-group, table-caption, none, contents), `position` (static, relative, absolute, fixed, sticky), `float` (left, right, none), `clear` (left, right, both, none), `top` `right` `bottom` `left` `inset`, the logical names of the sides and sizes ([below](#logical-properties)), `z-index`, `box-sizing` (border-box by default; content-box under [browser sheets](#browser-sheets)), `width` `height` and their `min-` `max-`, `aspect-ratio`, `margin`, `padding`, `overflow` (visible, hidden, clip, auto, scroll) and its axes (clipping on both with a `border-radius` clips to the rounded inner border, the padding box's curve, and a pointer in a cut corner reaches what's under it; nested rounded clips round only the innermost; under rotation or skew the clip is its bounding rectangle, [decision 0058](../99-decisions/0058-rounded-clips-through-a-per-batch-rounded-rect.md)), `visibility`, `order` |
| columns | `column-count`, `column-width`, `columns`, `column-gap` (normal is 1em), `column-rule` and its `-width` `-style` `-color`, `column-fill` (balance, auto), `break-inside` (auto, avoid; `page-break-inside` is read as it), `break-before` `break-after` (auto, avoid, column), `orphans`, `widows` |
| flex and grid | `flex` and its parts, `gap` `row-gap` `column-gap`, `align-*`, `justify-*`, `place-*`, `grid-template-columns` `-rows` `-areas` and the `grid-template` shorthand, with names of lines in brackets (`[full-start] 1fr [content-start]`, in `repeat()` too), `grid-column` `grid-row` and their `-start` `-end`, `grid-area` (an area's name, or its four lines), a line by number, by name (an area's `-start` or `-end` line, or the first of that name), by `name n` and by `span name n`, a name no line has standing for the lines after the grid, as in Chrome, `grid-auto-*` |
| text | `font-family` `font-size` `font-weight` `font-style` and the `font` shorthand, `line-height`, `letter-spacing`, `word-spacing` (a length or a percentage of the font size, after each space and no-break space, as Chrome), `text-align`, `text-indent`, `text-transform` (in text fields too: drawn cased, the value kept as typed, the caret and selection following what's drawn), `white-space`, `text-overflow`, `overflow-wrap`, `text-shadow`, `text-decoration` and its `-line` `-style` `-color` `-thickness`, `vertical-align` (every value, below), `color`, `list-style-type` (disc, circle, square, decimal, none), `-webkit-font-smoothing: none` (glyphs unsmoothed on the pixel grid, for pixel fonts) and `antialiased` (glyphs at their exact coverage, without the stem darkening of the default: thinner, and a square glyph on whole pixels is those pixels alone, as in Chrome on macOS) |
| look | `background-color` `-image` `-repeat` (repeat, no-repeat, repeat-x, repeat-y, or one of the first two for each axis) `-attachment` (scroll, fixed, local) `-origin` `-clip` (border-box, padding-box, content-box) `-size` (lengths, percentages, auto, cover, contain) `-position`, `border` and its parts (solid, dashed, dotted, inset, outset, groove, ridge), `border-radius`, `border-image` (nine-patches), `border-image-repeat` and `border-image-width`, `box-shadow`, `outline`, `opacity`, `transform` `transform-origin`, `transition` (colours, opacity, transforms), `cursor`, `pointer-events` (auto, none: the pointer passes through, inherited), `app-region` (drag, no-drag, none; inherited: a left press in a drag region moves the system's window and a double press is a double click on its title bar, for a header in a window with `platform.OwnTitleBar()`; what clicks is never part of the region), `image-rendering` (pixelated), `object-fit` `object-position`, `accent-color`, `caret-color`, `scrollbar-width`, `appearance`, `resize` |
| tables | `border-spacing`, `border-collapse` (separate, collapse; inherited), `caption-side` (top, bottom; inherited), `table-layout` (auto, fixed) |
| selecting | `user-select` (auto, text, none, all; inherited), `::selection` ([below](#selecting-text)) |

A table is what its `display` says, not its element (decision 0105): `table`, `tr`, `td` and the others only start with theirs, from ui's own sheet, and any element is a table, a row, a cell, a column or a caption by saying so. What misses round a part is made as CSS 2.1 §17.2.1 says and Chrome does: cells with no row get an anonymous one, rows with no table an anonymous table, and what stands in a row without being a cell goes in an anonymous cell; white space alone between parts makes nothing. A `td` shown as a `block` is a block. `col` and `colgroup` (`span`, `width`) size their columns and paint behind their cells: a table's backgrounds are layers, the column groups under the columns, under the row groups, under the rows, under the cells, each behind the cells alone, never in the spacing (§17.5.1). Cells whose `vertical-align` is `baseline` have their first lines on one baseline in their row, and an `inline-table` stands on its first row's.

With `border-collapse: collapse` a table has no spacing and no padding, and each edge of its grid has one border: the one that wins among those of the cells beside it, their rows, row groups, columns, column groups and the table (§17.6.2.1): a `hidden` one first, which leaves the edge bare, then the widest, then by style (solid, dashed, dotted, ridge, outset, groove, inset), then the nearest the cell, the left and the top first. A cell and the table get half of each of their edges for borders, and the table draws them all over its cells' backgrounds.

Lay a screen out with flex or grid; tables are for tabular data, floats for text that goes round a picture or a note, and columns for text that reads down one column and on at the top of the next.

![A quest card: its text goes round an orange portrait floated left and a tip floated right, each paragraph's first line indented, the reward below both](floated.png "ember · floated")

```css
.quest    { display: flow-root; text-indent: 16px; }
.portrait { float: left; margin: 2px 12px 6px 0; }
.tip      { float: right; width: 96px; text-indent: 0; }
.reward   { clear: both; }
```

```
go run ./examples/92-ui-kit -element floated
```

A float leaves the flow for the left or right edge of its container, and the lines after it take the room beside it; a line whose first word is wider than that room goes below the float. `clear` puts a block below the floats of a side. A container is only as tall as its flow: give it `display: flow-root` (or an `overflow` other than visible) to hold its floats; such a block, like a picture, a control or a table, also stands beside the floats before it and never under them. `display: table` on an element that is no `table` makes such a block too, as narrow as its content unless it has a width ([decision 0094](../99-decisions/0094-display-table-off-a-table-is-a-block-that-shrinks-and-holds-its-floats.md)): the empty `::after { content: ""; display: table; clear: both }` that old sheets end a container of floats with holds them, and `display: table; margin: 0 auto` centres a block of unknown width; `inline-table` is an `inline-block`. Its rows and cells are not read: `display: table-row`, `table-cell` and the others are refused. Floats paint over the backgrounds of the blocks after them and under their text. A float is a block whatever its `display`, and one in a flex or grid container is an item like the others. Two limits: a float written inside a paragraph's text stands at the top of that paragraph (in Chrome, at the line it was written in when that line has room), and there is no `shape-outside`.

![A journal of three entries in two columns with a thin rule between them: Tuesday's entry starts at the foot of the first column and goes on at the top of the second](columns.png "ember · columns")

```css
.journal { column-count: 2; column-gap: 24px; column-rule: 1px solid #4a4f5c; }
.journal .last { break-inside: avoid; }
```

```
go run ./examples/92-ui-kit -element columns
```

A block with a `column-count` or a `column-width` lays its content out in columns ([decision 0088](../99-decisions/0088-columns-cut-one-flow-and-a-cut-box-stays-one-box.md)), as Chrome does. `column-count` is the most columns, `column-width` the least width of one: the container takes as many columns of at least that width as fit, up to the count, and shares its width out between them, `column-gap` apart. `columns` sets both, in either order. Without a height the columns are balanced: as short as they can be made, all the same height. With a `height` or a `max-height` the columns are still balanced when the content fits (`column-fill: balance`), or filled one after the other (`column-fill: auto`); what doesn't fit goes on in more columns of the same width, past the container's end. `column-rule` draws a line in the middle of each gap between two columns that have content, as high as the columns; it takes no room.

Content is cut between blocks and between lines. A margin at a cut is dropped, at the foot of a column and at the top of the next. A paragraph keeps at least `orphans` lines at the foot of a column and `widows` at the top of the next (2 each, inherited). A picture, a control, a table, a flex or grid container, a float, a box that scrolls and a multi-column container inside another are never cut, nor is a box with `break-inside: avoid`; one taller than the columns makes them as tall as it is. `break-before: column` and `break-after: column` start a new column; `avoid` keeps two blocks together where it can. A cut box is still one element: its background and border are drawn on each part, open where it is cut, the pointer reaches it on its parts only, and its `Box()` and `BoxOf` are the box round its parts, as `getBoundingClientRect()`'s. A container is a formatting context: it holds its floats, and its margins don't collapse with its content's. An absolutely positioned child with no inset stays where its column shows it.

Limits: `column-span: all` is refused (put the heading before the container); Chrome also cuts flex containers and tables between their rows, and ui keeps them whole; a positioned box is never cut; columns go left to right only.

A box whose `overflow` is `auto` or `scroll` shows a scrollbar in a gutter of 15 points beside its content, kept always for `scroll` and once the content overflows for `auto`; `scrollbar-width: none` scrolls with no bar and no gutter, and `thin` is as `auto`. No CSS draws the bar over the content: that is the Document's [`ui.OverlayScrollbars()`](01-document.md), for all its boxes.

A program scrolls by an element ([decision 0111](../99-decisions/0111-a-program-scrolls-a-document-by-an-element.md)). `doc.ScrollIntoView(n)` scrolls every box round `n` by the least that shows it, as `scrollIntoView({block: "nearest"})`; `doc.ScrollToStart(n)` puts its top at the top of each, as far as each scrolls, as `scrollIntoView()` and a browser going to a fragment. Both wait for an element not laid out yet. `doc.Scroll(n)` is how far a box's content is scrolled and `doc.SetScroll(n, off)` scrolls it there, in the Document's own points before its zoom, kept within what the box scrolls; with a scrolling viewport, below, the viewport's scroll is `doc.HTML()`'s. An offset set before the first layout holds from it, so a page can be shown where it was left.

The window itself does not scroll: `html` is the outermost box, and a screen taller than its window is cut. A Document made with [`ui.ScrollingViewport()`](01-document.md) scrolls its viewport as a browser does. `html` then has its own height, the page's, and percentages on `html` and `body` are still of the window. `overflow` on `html`, or on `body` while `html`'s is visible, is the viewport's: `hidden` stops the wheel and the keys and shows no bar, `scroll` keeps the bar's room. A `position: fixed` box stays where it is in the window, and a sticky one sticks to the window's edges. The canvas background, `html`'s or else `body`'s, covers the window; its picture is placed in `html`'s padding box, so `bottom right` is the page's end, and it scrolls with the page. Without the option that picture is placed in the window. Two limits: there is no `background-attachment`, and a fixed box inside a box that scrolls on its own moves with that box.

`text-indent` starts a block's first line in by a length or a percentage of the block's width, negative to hang it out (`-9999px` still hides a replaced heading's text). It is inherited, counts in shrink-to-fit widths, and form controls reset it to 0, as in Chrome. `each-line` and `hanging` are refused.

The `font` shorthand is `[style] [weight] size[/line-height] family, …`, as CSS writes it: `font: italic bold 12px/1.5 Georgia, serif`. `small-caps`, the stretch words and the system fonts (`caption`, `menu`…) are refused. A family is one of the Document's own (`@font-face`, from a file or [fetched](01-document.md#fonts-that-arent-files), ember's faces) unless the Document has [`ui.SystemFonts()`](01-document.md), which adds the machine's.

`text-decoration` draws `underline`, `overline` and `line-through`, `solid`, `double`, `dotted` or `dashed`, in the text's colour or `text-decoration-color`, a tenth of the font size thick or `text-decoration-thickness`. A box's lines go on all the text inside it, in its own thickness and colour, whatever its descendants set; floats, positioned boxes and inline blocks leave them behind. Places and thicknesses are Chrome's, measured. Three limits: lines cross descenders where Chrome leaves gaps (`text-decoration-skip-ink`), they cast no `text-shadow`, and `wavy` is refused.

`text-align` also takes Chrome's `-webkit-left`, `-webkit-right` and `-webkit-center`, which HTML's `align` attribute and the `center` element stand for: lines as the plain word, and a block narrower than its container, a table or a sized box with no `auto` margin, placed the same way.

`vertical-align` places an inline box, an inline-block or a picture on its line (decision 0104): `baseline`, `sub` (a fifth of its parent's font size and a pixel down), `super` (a third and a pixel up), a length or a percentage of its own line height (up), `middle` (its middle half an x-height above the baseline), `text-top` and `text-bottom` (on its parent's text), `top` and `bottom` (on the line, sized by the rest first), the line growing to hold each, as in Chrome. An inline-block stands on the baseline of its last line, or on its bottom margin edge when it has none or scrolls. On a table cell it places the content: `top`, `middle` (the default) or `bottom`, the other values as `top`.

A list item (`display: list-item`, an `li`) has its marker outside its box, beside its first line: `disc`, `circle`, `square`, or its number in its list as `decimal`, `decimal-leading-zero`, `lower-alpha` (`lower-latin`), `upper-alpha` (`upper-latin`), `lower-roman` or `upper-roman` writes it. The number is the `list-item` counter ([counters](#counters)), which `ol`'s `start` and `reversed` and `li`'s `value` set; `::marker` styles the marker. `list-style-position` and `list-style-image` are not in the subset.

Solid sides of more than one colour, or seen through, meet on the diagonal of their corner, as Chrome draws them: a box of no size with one coloured side and its neighbours transparent is a triangle, the arrow of a menu or a tooltip. Dashed and dotted sides run corner to corner.

A `border-image` draws its corners and sides `border-image-width` thick and lays its edges and centre along their length, as `border-image-repeat` says, one value for both axes or two (across, then down): `stretch` (the default) stretches the piece, `repeat` lays copies at their size, centred, the ends cut, `round` fits a whole number of copies, scaled, and `space` fits a whole number at their size with the leftover shared out as equal gaps. A patterned frame (ivy, cloth, iron) keeps its pattern with `round`: `border-image: url(frame.png) 8 fill round`.

`border-image-width` takes one to four values, as margins do: a number times the border's width, a length, a percentage of the box's side, or `auto` for the slice's own size. Its initial 1 is the border's widths, so `border: 8px solid` with an 8-texel slice draws the slice as it is, a thinner border draws it thinner, and `border-width: 0 16px` draws no top or bottom row, stretching the slice's middle band over the box. The shorthand takes it after the slice: `border-image: url(frame.png) 8 / 16px`. When opposite sides would overlap, all four shrink by the same factor, and repeated copies scale with their side: an 8-texel slice drawn 16 thick tiles 16-long copies.

`background-size` takes `cover` or `contain`, or a width then a height, each a length, a percentage of the padding box or `auto`; one value is the width. `auto` keeps the picture's proportions beside the other side, and its own size when both are auto. `background-position` places the picture at that size and `background-repeat` tiles it at that size. The `background` shorthand takes the size after the position and a slash: `background: url(beach.png) no-repeat center / cover`. An SVG picture may have no size of its own either ([SVG pictures](01-document.md#svg-pictures)): with a `viewBox` alone `auto` is `contain`, with nothing it is the whole padding box, and it is drawn at the tile's size whatever that is. A gradient has no size of its own: `auto`, `cover` and `contain` are the whole padding box, and a length or a percentage sizes the box the gradient is drawn in, which then repeats, so `background-size: 20px 20px` over a radial gradient makes dots. Where two tiles of a scaled picture meet, each keeps its own edge, where Chrome blends the two over a pixel.

A background shows in the box `background-clip` names (the border box, so it shows between the dashes of a border), and its picture is placed in the one `background-origin` names (the padding box). `background-attachment: fixed` places the picture in the Document's viewport, an iframe's own, and shows the part of it under the element; `local` places it in a scroller's content, with which it scrolls; `scroll` keeps it in the box. The `background` shorthand takes them in any order with its other parts: one box is the origin and the clip, two the origin then the clip. An inline element over several lines is one box sliced by its lines (`box-decoration-break: slice`): its picture is placed in the lines laid end to end, so one at the right shows after the last word.

`image-rendering: pixelated` draws an `<img>` or a background from its nearest texels. A pixel whose centre falls exactly on the line between two texels takes the upper left one, as Chrome does; that happens when a texel is a whole number and a half of device pixels wide, such as 16 texels over 24 points under a `Zoom` of 1.5 on a 2× screen. Smooth images are not concerned.

`mask-image` paints an element's background colour through a picture's alpha, which is how a page draws an icon in one colour (decision [0093](../99-decisions/0093-a-mask-image-is-the-shape-of-the-background-colour.md)): `mask-image` takes `none` or one `url()`, and `mask-size`, `mask-repeat` and `mask-position` size, tile and place it in the border box as their `background-` twins do a background. Nothing else of the element is masked: its background image, border and content draw whole.

A `url()` whose address is a `data:` one, base64 or percent-encoded, is read from the address itself, in a sheet as in an `img`'s `src`.

### Logical properties

ui writes text left to right in horizontal lines only, so a logical property is the physical one of that writing, and takes its place in the cascade: the last declared wins, logical or not. `margin-`, `padding-` and `border-` with `block-start`, `block-end`, `inline-start`, `inline-end` (a border's also with `-width`, `-style`, `-color`) are top, bottom, left and right; `inset-block-start` and the three others are `top`, `bottom`, `left`, `right`; `inline-size` and `block-size`, with their `min-` and `max-`, are `width` and `height`; `border-start-start-radius` and the three others are the corners from the top left. `margin-inline`, `margin-block`, `padding-inline`, `padding-block`, `inset-inline` and `inset-block` take one value for both sides or two, start then end.

What ui refuses on purpose is below; what it does not do yet is in [What is missing](10-missing.md).

| Refused | Use |
|---|---|
| `column-span: all` | the element before the columns |
| `text-decoration: wavy`, `blink` | solid, double, dotted or dashed |
| `word-wrap`, `grid-gap`, `grid-row-gap`, `grid-column-gap` | `overflow-wrap`, `gap`, `row-gap`, `column-gap` |
| `zoom` | `transform: scale()` |
| `clip` | `overflow: clip` |
| `animation` | `transition` |
| the `mask` shorthand, a gradient as a `mask-image`, several mask layers | `mask-image: url()` and its longhands |
| repeating gradients, colour hints, several background layers | one gradient |
| `@import` and other at-rules but `@font-face` and `@media` | not supported |

## Pseudo-elements

A pseudo-element ends its selector, after any compound: `.card:hover::before`, `nav ul li:nth-child(2)::before`, `ol > li::marker`. It counts as a type selector in the specificity. `:before`, `:after` are read as the two-colon forms, as Chrome reads them ([decision 0089](../99-decisions/0089-a-generated-box-is-a-node-of-its-element.md)).

`::before` and `::after` make a box inside their element, before its first child and after its last, once their `content` is something other than `none` or `normal`. The box takes any property an element takes: a display, a position, a float, a place in a flex or a grid, backgrounds, transforms, transitions. It inherits from its element. It is no element: no selector matches it, `ByID`, `Children()` and automation never see it, its text is not selected or translated, and a click on it is a click on its element. The elements Chrome gives none have none (`img`, `br`, `input`, `textarea`, `select`, `canvas`), and neither have `progress`, `meter`, `table` and a table's row groups and rows, which ui lays out whole. It is sized as the Document's elements are: by its border box, or by its content under `ui.BrowserSheets()`.

![A journal's contents: chapters numbered I and II, their parts 1.1 to 2.2, a page count after the first title, an orange bar down the right side, a note with a plus for its marker and a quotation in guillemets](generated.png "ember · generated")

```css
#contents { position: relative; counter-reset: chapter; }
#contents::before { content: ""; position: absolute; top: 0; right: 0; width: 6px; height: 100%; background-color: #e0a040; }
#contents h4 { counter-increment: chapter; }
#contents h4::before { content: counter(chapter, upper-roman) ". "; }
#contents h4[data-pages]::after { content: " · " attr(data-pages) " pages"; }
#contents ul li::marker { content: "+ "; }
```

```
go run ./examples/92-ui-kit -element generated
```

| `content` | |
|---|---|
| `"text"` | a string; several values side by side are joined |
| `attr(name)` | the element's attribute, empty without it; it follows the attribute |
| `url(picture.png)` | the picture at its own size, an inline box |
| `counter(name)`, `counter(name, style)` | the counter's value, in a list style (`decimal` by default) |
| `counters(name, "separator")`, `counters(name, "separator", style)` | every counter of that name from the outermost in, joined |
| `open-quote`, `close-quote`, `no-open-quote`, `no-close-quote` | a quotation mark of `quotes` at the depth, and one level deeper or back |
| `<values> / "alternative text"` | the values; the alternative text is read and not used |
| `none`, `normal` | no box |

On an element itself `content` is read and does nothing; Chrome replaces the element by a `url()` there, ui does not.

### Counters

`counter-reset: name [value]` makes a counter on an element (0 by default), for its siblings after it and their descendants, `counter-increment: name [value]` adds to the nearest one (1 by default) and `counter-set: name [value]` sets it; each takes several, or `none`. A counter of the same name made deeper nests inside the outer one, which `counters()` writes whole. A counter used where none was made is made there, at 0. `counter-reset: reversed(name)` is refused.

```css
body { counter-reset: chapter; }
h2 { counter-increment: chapter; counter-reset: section; }
h2::before { content: "Chapter " counter(chapter, upper-roman) ": "; }
h3 { counter-increment: section; }
h3::before { content: counter(chapter) "." counter(section) " "; }
ol.toc { counter-reset: item; list-style: none; }
ol.toc li { counter-increment: item; }
ol.toc li::before { content: counters(item, ".") " "; }
```

A list numbers its items with the same counters: `ol`, `ul` and `menu` reset `list-item`, each list item adds one (takes one off in a `reversed` list), `start` and `value` set it, and `counter(list-item)` reads it. A sheet may set or reset `list-item` as any other counter.

### Quotes

`quotes` (inherited) lists pairs of strings, an opening and a closing mark for each depth, the last pair serving any deeper; `auto` (the default) is `“ ” ‘ ’`, and `none` writes no mark. `q` has its marks from the user-agent sheet, `q::before { content: open-quote }` and `q::after { content: close-quote }`, so a sheet restyles or removes them as any generated box.

### `::marker`

`::marker` styles a list item's marker: `color`, the font properties, `letter-spacing`, `word-spacing`, `text-transform`, and `content`, a string, a counter or an `attr()` that replaces the bullet or the number. The other properties Chrome's `::marker` takes that ui has (`cursor`, `line-height`, `overflow-wrap`, `quotes`, `text-shadow`, `white-space`, the transition properties) are read without a warning and change nothing yet. Any other property is refused with a warning, as Chrome ignores it.

### Every pseudo-element

Every pseudo-element Chrome reads is here (`TestChromeReadsThePseudoElementsOfTheTable` asks Chrome). A refused one drops its rule with a warning in `doc.Warnings()`. One that matches nothing is read, so its rule is valid and says nothing, and styles nothing: ui has no such part.

| Pseudo-element | In ui |
|---|---|
| `::before`, `::after` | a generated box, every property |
| `::marker` | the list item's marker (above) |
| `::placeholder` | an empty field's placeholder: `color`, `opacity` |
| `::selection` | selected text, in a page and in a field: `background-color`, and in a page `color` ([Selecting text](#selecting-text)) |
| `::backdrop` | behind a modal dialog: `background-color`, `opacity` |
| `::first-line`, `::first-letter` (and `:first-line`, `:first-letter`) | refused: ui styles no line or letter of a block apart |
| `::file-selector-button` | refused: ui has no file input |
| `::details-content` | refused: a details' content has no box of its own |
| `::part()`, `::slotted()` | matches nothing: no shadow trees |
| `::highlight()`, `::spelling-error`, `::grammar-error`, `::target-text`, `::search-text` | matches nothing: no highlights, no spelling, no find in page |
| `::cue`, `::cue()` | matches nothing: no video captions |
| `::view-transition`, `::view-transition-group()`, `::view-transition-image-pair()`, `::view-transition-old()`, `::view-transition-new()`, `::view-transition-group-children()` | matches nothing: no view transitions |
| `::scroll-marker`, `::scroll-marker-group`, `::scroll-button()`, `::column` | matches nothing: no scroll markers or buttons, a column is no box |
| `::checkmark`, `::picker()`, `::picker-icon` | matches nothing: a select is drawn whole |
| `::permission-icon` | matches nothing: no permission element |
| a vendor's (`::-webkit-scrollbar`, `::-moz-focus-inner`…) | refused; under `ui.BrowserSheets()` as the [vendor tables](#vendor-prefixes) say |
| any other name | refused: unknown, as Chrome drops the rule |

A pseudo-element's rule that sets a property it does not take keeps its other declarations and warns of that one. Nothing may follow a pseudo-element in a selector: `::before:hover` and `li::marker::before` are refused as Chrome refuses them; `::before::marker` and `::selection:window-inactive`, which Chrome reads, are refused too.

## Selecting text

![A letter on a dark panel, selected in orange from the middle of its first paragraph to the middle of its second](selection.png "ember · selection")

```css
#letter { user-select: text; }
#letter ::selection { background-color: #e0a040; color: #1d2129; }
#letter code { user-select: all; }
#letter .seal { user-select: none; }
```

```
go run ./examples/92-ui-kit -element selection
```

`user-select` says whether a selection takes an element's text ([decision 0090](../99-decisions/0090-a-screens-text-takes-no-selection-until-its-sheet-says-so.md)). It takes Chrome's values: `text`, `none`, `all` (the element whole or not at all: a press on it selects it, a drag into it takes all of it) and `auto`. It is inherited, as in Chrome: an element with no rule takes its parent's, and `auto` set on an element is `text`. ember's sheet has `html { user-select: none }`, so nothing of a game screen is selected by accident, kit's components included; `user-select: text` on the element whose text the player may copy turns it on from there down. Under `ui.BrowserSheets()` the root is `auto`, as a browser's page.

What is selected is copied as Chrome's `Selection.toString()` writes it: the text as drawn, its white space collapsed and its `text-transform` applied, a line break for a `br` and round each block and table row, two after a `p`, a tab between a row's cells; nothing of `user-select: none`, `display: none` or `visibility: hidden` text, of generated content, of a picture or of a field. The clipboard gets plain text, its no-break spaces as plain ones.

The selection is drawn behind the text, on each line from its first selected character to its last, as high as the line box and no less than the glyphs, and a space wider where it goes on past the line's end, as Chrome draws a selected line break. A picture wholly inside it is tinted. `::selection` sets its `background-color` and the text's `color`; it is inherited from the parent's `::selection`, so one rule on a container styles what it holds. A rule that sets only one of the two leaves the other as CSS has it: no background, or the text's own colour. With no rule the background is Chrome's on macOS, `#b3d7fe` over white, and the text keeps its colour. A field's own selection takes `::selection`'s `background-color` only.

## Media queries

Decision [0086](../99-decisions/0086-media-queries-are-answered-for-the-documents-viewport.md).

```css
.side { width: 240px; }
@media (max-width: 600px) { .side { width: auto; } }
```

An `@media` rule applies while its query holds for the Document's viewport: the size the Document is laid out in, in its own pixels, so a window 1000 points wide under a `Zoom` of 2 is 500 wide. The rules follow a resize; a resize that crosses no query's boundary restyles nothing.

| Media queries | |
|---|---|
| `all`, `screen` | hold; `print` and any other type never do |
| `width`, `height`, `aspect-ratio`, `resolution`, each with `min-` and `max-` | the viewport and the window's pixel ratio; lengths in `px`, `em` and `rem` (16px), the other absolute units too, and a `calc()` of them, as `(max-width: calc(1120px - 1px))`; ranges as `(width >= 600px)` and `(400px <= width < 900px)` |
| `orientation` | `portrait` when the viewport is as high as wide or higher |
| `device-width`, `device-height`, `device-aspect-ratio` | the viewport's: ui knows no screen |
| `hover`, `any-hover`, `pointer`, `any-pointer` | `hover` and `fine`: a mouse |
| `prefers-color-scheme`, `prefers-reduced-motion`, `prefers-contrast`, `forced-colors` | `light`, `no-preference`, `no-preference`, `none` |
| `color`, `monochrome`, `grid` | 8, 0, 0 |
| `and`, `,`, `not`, `only`, `or` between conditions in parentheses, `@media` inside `@media` | as in Chrome |

A feature ui doesn't know makes its query false, as in Chrome, with a warning; the other queries of the list still count. Every answer is Chrome's for the same viewport (`TestChromeAnswersMediaQueriesTheSame`).

## Where a sheet came from

A `url()` in a sheet is read beside the sheet: `ui.Stylesheet(fsys, "ui/tool.css")` reads `url(img/panel.png)` as `ui/img/panel.png` of its files. A sheet fetched from elsewhere and given as text says where it came from, and its `url()` are fetched from there by the Document's `ui.FetchImages`:

```go
ui.NewDocument(
	ui.FetchImages(web.Images(page)),
	ui.CSS(text, ui.From("https://cdn.example/site/css/main.css")), // url(../img/wall.png) is https://cdn.example/site/img/wall.png
)
```

`sheet.From()` reads it back. A `url()` in a style attribute is the page's, resolved by the fetcher. `@font-face` files are read from a `Stylesheet`'s files, and fetched from the sheet's address with [`ui.FetchFonts`](01-document.md#fonts-that-arent-files).

## Sheets changed while a Document shows

A Document's stylesheets are `*ui.Sheet`s, in the order the cascade takes them: those its options gave, and those added since. A tool that edits a screen's look, or a page whose scripts write its styles, changes them there ([decision 0097](../99-decisions/0097-a-documents-sheets-are-changed-through-handles-once-a-frame.md)):

```go
theme := doc.AddSheet(`.card { width: 260px; }`)                  // after the others: wins at equal specificity
reset := doc.InsertSheet(theme, `* { margin: 0; }`)                // before theme: loses to it
theme.InsertRule(len(theme.Rules()), `.card:hover { color: #e0a040; }`)
theme.SetMedia("(min-width: 600px)")
reset.Remove()
```

| | |
|---|---|
| `doc.AddSheet(text, opts…)` | a sheet after the Document's others, read as `ui.CSS` reads one: `ui.From(address)`, `ui.Media(query)` |
| `doc.InsertSheet(before, text, opts…)` | the same before the sheet `before`, which then wins where both say the same; after them all for nil |
| `doc.StyleSheets()` | the Document's sheets in cascade order, those of `ui.Stylesheet`, `ui.CSS` and `kit.Theme()` first |
| `sheet.Replace(text)`, `sheet.Text()` | another text for the sheet, read as the first was; its text |
| `sheet.Remove()` | out of the Document, which looks as it did without it |
| `sheet.Media()`, `sheet.SetMedia(query)` | the media query list its rules apply under, as a `style` element's `media`; `""` for always |
| `sheet.Disabled()`, `sheet.SetDisabled(off)` | its rules left out of the cascade, its place kept |
| `sheet.Rules()` | its top-level rules as CSS: each style rule, each at-rule with what it holds |
| `sheet.InsertRule(i, text)`, `sheet.DeleteRule(i)` | one rule put before rule `i`, or taken out. Only the new rule is read. A rule ui reads part of is taken, what it drops a warning; the error is for a text that is no one rule, `ui.ErrNoRule` for an `i` out of range |

- However many sheets change between two frames, and however often, the Document works its cascade out once, at the next frame, and lays out only what the new styles move. A thousand rules inserted one by one cost about 5 ms to read.
- A run-time sheet is read as the Document's others: under `ui.BrowserSheets()` as a browser's, its `@media` answered for the viewport as it is resized, its `@layer` names shared with the other sheets, its warnings in `doc.Warnings()`. An `@font-face` in it is read as in `ui.CSS` when the sheet is added or replaced whole (not one inserted as a rule, and a face stays when its sheet goes), and `@import` is not read.
- `doc.MatchedRules` and `doc.ComputedStyle` read the sheets as they stand, without waiting for the frame. A rule of a run-time sheet is named by its sheet (`style`, with its `From`), and one inserted alone stands at line 1 of its own text.
- Each step of `TestSheetsChangedWhileADocumentShowsLayOutAsChromes` lays out as Chrome given the same sheets: a rule added later wins or loses by order, specificity and importance, and a removed sheet leaves the look before it.

## Browser sheets

Decision [0085](../99-decisions/0085-browser-sheets-are-an-option-of-a-document.md).

A sheet written for browsers holds what a game's sheet never does: prefixes of four engines, `@supports`, `@layer`, widths that are the content's. ui refuses each with a warning, since in a game's sheet each is a mistake. `ui.BrowserSheets()` is the Document option for showing such a page (`doc.BrowserSheets()` reads it back): every sheet and style attribute of the Document is read as Chrome reads it. Off, nothing below applies and every warning stays.

| With `ui.BrowserSheets()` | |
|---|---|
| `@supports` | answered: a declaration is supported when ui reads it without a warning, `selector()` when ui reads the selector; `not`, `and`, `or` as in CSS. The rules of a condition that fails go without a word |
| `@layer` | real cascade layers: statements, blocks, nested and dotted names, anonymous layers; a later layer wins, unlayered rules win over every layer, and `!important` turns the order round. Layers are the Document's, shared by its sheets in order |
| `user-select` | `auto` on the root, so the page's text is selected and copied as in a browser ([Selecting text](#selecting-text)) |
| `box-sizing` | `content-box` unless the sheet says otherwise, as in a browser's own sheet. Sized by their border as in Chrome: `button`, `select`, `table`, `progress`, `meter`, and `input` of type checkbox, radio, button, submit, reset, search and color (`TestChromeSizesTheSameBoxesByTheirBorder`) |
| `text-align` of a control | `start` on `input`, `select` and `textarea`, `center` on an `input` of type button, submit and reset, whatever their parent's, as in Chrome (`TestChromeAlignsTheTextOfControlsAsABrowsersSheetDoes`); without the option a control takes its parent's |
| `letter-spacing`, `word-spacing`, `text-transform`, `text-shadow` of a control | `normal` and `none` on `button`, `input`, `select` and `textarea`, whatever their parent's, as in Chrome (`TestChromeSpacesTheTextOfControlsAsABrowsersSheetDoes`); without the option a control takes its parent's |
| an element ui has none of in a selector (`applet`, `object`, a custom element) | the selector matches nothing and goes without a word, its rule keeping the others: a reset that lists every element of HTML still applies. Inside `:not()` it keeps its warning and drops the rule |
| vendor prefixes | read as the plain name where Chrome does, dropped without a word where Chrome drops them: the tables below |

A prefixed name read as a plain one keeps its place in the cascade, so `-webkit-transform: rotate(3deg); transform: none` ends as `none` and the other order as the rotation. A warning about one names it as it was written.

What ui can't do stays a warning with the option too: a property or value Chrome reads and ui doesn't (`columns`, `animation`, `-webkit-line-clamp`, `display: -webkit-box`, a `-webkit-linear-gradient()`), `@import`, `@font-face` in a sheet given as text without `ui.FetchFonts`. A declaration ui refuses whose value holds another engine's word (`transition: opacity 1s, -moz-transform 1s`) goes whole without a word.

### Vendor prefixes

One table for each vendor, as `ui/css/vendors.go` has them. `-webkit-`'s lists are Chrome's own: a test asks the Chrome the suite runs which plain property each `-webkit-` name sets (`TestChromeReadsWebkitPropertiesAsTheTable`), and another that it knows no name of the other four.

#### `-webkit-`

| | |
|---|---|
| properties read as the plain one of the same name (73) | `-webkit-align-content` `-webkit-align-items` `-webkit-align-self` `-webkit-animation` `-webkit-animation-delay` `-webkit-animation-direction` `-webkit-animation-duration` `-webkit-animation-fill-mode` `-webkit-animation-iteration-count` `-webkit-animation-name` `-webkit-animation-play-state` `-webkit-animation-timing-function` `-webkit-app-region` `-webkit-appearance` `-webkit-backface-visibility` `-webkit-background-clip` `-webkit-background-origin` `-webkit-background-size` `-webkit-border-bottom-left-radius` `-webkit-border-bottom-right-radius` `-webkit-border-radius` `-webkit-border-top-left-radius` `-webkit-border-top-right-radius` `-webkit-box-shadow` `-webkit-box-sizing` `-webkit-clip-path` `-webkit-column-count` `-webkit-column-gap` `-webkit-column-rule` `-webkit-column-rule-color` `-webkit-column-rule-style` `-webkit-column-rule-width` `-webkit-column-span` `-webkit-column-width` `-webkit-columns` `-webkit-filter` `-webkit-flex` `-webkit-flex-basis` `-webkit-flex-direction` `-webkit-flex-flow` `-webkit-flex-grow` `-webkit-flex-shrink` `-webkit-flex-wrap` `-webkit-font-feature-settings` `-webkit-hyphenate-character` `-webkit-justify-content` `-webkit-mask-clip` `-webkit-mask-image` `-webkit-mask-origin` `-webkit-mask-repeat` `-webkit-mask-size` `-webkit-opacity` `-webkit-order` `-webkit-perspective` `-webkit-perspective-origin` `-webkit-print-color-adjust` `-webkit-shape-image-threshold` `-webkit-shape-margin` `-webkit-shape-outside` `-webkit-text-emphasis` `-webkit-text-emphasis-color` `-webkit-text-emphasis-position` `-webkit-text-emphasis-style` `-webkit-text-size-adjust` `-webkit-transform` `-webkit-transform-origin` `-webkit-transform-style` `-webkit-transition` `-webkit-transition-delay` `-webkit-transition-duration` `-webkit-transition-property` `-webkit-transition-timing-function` `-webkit-user-select` |
| properties read as a plain one of another name (30) | `-webkit-border-after` as `border-block-end`, `-webkit-border-after-color` as `border-block-end-color`, `-webkit-border-after-style` as `border-block-end-style`, `-webkit-border-after-width` as `border-block-end-width`, `-webkit-border-before` as `border-block-start`, `-webkit-border-before-color` as `border-block-start-color`, `-webkit-border-before-style` as `border-block-start-style`, `-webkit-border-before-width` as `border-block-start-width`, `-webkit-border-end` as `border-inline-end`, `-webkit-border-end-color` as `border-inline-end-color`, `-webkit-border-end-style` as `border-inline-end-style`, `-webkit-border-end-width` as `border-inline-end-width`, `-webkit-border-start` as `border-inline-start`, `-webkit-border-start-color` as `border-inline-start-color`, `-webkit-border-start-style` as `border-inline-start-style`, `-webkit-border-start-width` as `border-inline-start-width`, `-webkit-logical-height` as `block-size`, `-webkit-logical-width` as `inline-size`, `-webkit-margin-after` as `margin-block-end`, `-webkit-margin-before` as `margin-block-start`, `-webkit-margin-end` as `margin-inline-end`, `-webkit-margin-start` as `margin-inline-start`, `-webkit-max-logical-height` as `max-block-size`, `-webkit-max-logical-width` as `max-inline-size`, `-webkit-min-logical-height` as `min-block-size`, `-webkit-min-logical-width` as `min-inline-size`, `-webkit-padding-after` as `padding-block-end`, `-webkit-padding-before` as `padding-block-start`, `-webkit-padding-end` as `padding-inline-end`, `-webkit-padding-start` as `padding-inline-start` |
| properties left as written (48), so each keeps its warning unless ui reads it | `-webkit-mask` `-webkit-mask-box-image` `-webkit-mask-box-image-outset` `-webkit-mask-box-image-repeat` `-webkit-mask-box-image-slice` `-webkit-mask-box-image-source` `-webkit-mask-box-image-width` `-webkit-mask-position` `-webkit-mask-position-x` `-webkit-mask-position-y`: Chrome's own masks, kept apart from mask and its longhands; `-webkit-border-horizontal-spacing` `-webkit-border-vertical-spacing` `-webkit-box-decoration-break` `-webkit-box-reflect` `-webkit-line-break` `-webkit-line-clamp` `-webkit-locale` `-webkit-rtl-ordering` `-webkit-ruby-position` `-webkit-tap-highlight-color` `-webkit-text-combine` `-webkit-text-decorations-in-effect` `-webkit-text-fill-color` `-webkit-text-orientation` `-webkit-text-security` `-webkit-text-stroke` `-webkit-text-stroke-color` `-webkit-text-stroke-width` `-webkit-user-drag` `-webkit-user-modify` `-webkit-writing-mode`: Chrome's own, with no plain property; `-webkit-font-smoothing`: a property of ui under this very name; `-webkit-border-image`: fills the middle without the fill word, where border-image leaves it empty; `-webkit-perspective-origin-x` `-webkit-perspective-origin-y` `-webkit-transform-origin-x` `-webkit-transform-origin-y` `-webkit-transform-origin-z`: one axis of the plain property, which has no such longhand; `-webkit-column-break-after` `-webkit-column-break-before` `-webkit-column-break-inside`: takes always, which the break-* property it sets refuses; `-webkit-mask-composite`: takes source-over and the other compositing words, which mask-composite refuses; `-webkit-box-align` `-webkit-box-direction` `-webkit-box-flex` `-webkit-box-ordinal-group` `-webkit-box-orient` `-webkit-box-pack`: the old flexbox of display: -webkit-box, with no plain property |
| any other property | goes without a word, as Chrome drops it |
| values read as the plain one | `-webkit-calc` as `calc`, `-webkit-fit-content` as `fit-content`, `-webkit-flex` as `flex`, `-webkit-grab` as `grab`, `-webkit-grabbing` as `grabbing`, `-webkit-inline-flex` as `inline-flex`, `-webkit-max-content` as `max-content`, `-webkit-min-content` as `min-content`, and the properties above in `transition` |
| any other value | keeps its warning, the declaration dropped |
| pseudo-elements read as the plain one | `::-webkit-input-placeholder` as `::placeholder` |
| any other pseudo-element | matches nothing: its selector alone goes, without a word |
| pseudo-classes read as the plain one | `:-webkit-any-link` as `:any-link` |
| pseudo-classes nothing in ui is ever in | `:-webkit-autofill` `:-webkit-drag` `:-webkit-full-page-media` `:-webkit-full-screen` `:-webkit-full-screen-ancestor`: the selector alone goes, without a word |
| pseudo-classes Chrome reads and ui can't | `:-webkit-any`: the rule keeps its warning |
| any other pseudo-class | its whole rule goes without a word, as Chrome drops it |
| at-rules read as the plain one, with the warning ui has for it | `@-webkit-keyframes` as `@keyframes` |
| any other at-rule | goes without a word, as Chrome drops it |
| media features read as the plain one, a bare number as dppx | `-webkit-device-pixel-ratio` as `resolution`, `-webkit-max-device-pixel-ratio` as `max-resolution`, `-webkit-min-device-pixel-ratio` as `min-resolution` |
| media features Chrome answers and ui can't | `-webkit-transform-3d`: the query is false, with a warning |
| any other media feature | its query is false without a word, as in Chrome |

#### `-moz-`

| | |
|---|---|
| any property | goes without a word, as Chrome drops it |
| a value ui refuses that holds one of its words or functions | its declaration goes without a word, as Chrome drops it |
| any pseudo-element or pseudo-class | its whole rule goes without a word, as Chrome drops it |
| any at-rule | goes without a word, as Chrome drops it |
| any media feature | its query is false without a word, as in Chrome |

#### `-ms-`

| | |
|---|---|
| any property | goes without a word, as Chrome drops it |
| a value ui refuses that holds one of its words or functions | its declaration goes without a word, as Chrome drops it |
| any pseudo-element or pseudo-class | its whole rule goes without a word, as Chrome drops it |
| any at-rule | goes without a word, as Chrome drops it |
| any media feature | its query is false without a word, as in Chrome |

#### `-o-`

| | |
|---|---|
| any property | goes without a word, as Chrome drops it |
| a value ui refuses that holds one of its words or functions | its declaration goes without a word, as Chrome drops it |
| any pseudo-element or pseudo-class | its whole rule goes without a word, as Chrome drops it |
| any at-rule | goes without a word, as Chrome drops it |
| any media feature | its query is false without a word, as in Chrome |

#### `-khtml-`

| | |
|---|---|
| any property | goes without a word, as Chrome drops it |
| a value ui refuses that holds one of its words or functions | its declaration goes without a word, as Chrome drops it |
| any pseudo-element or pseudo-class | its whole rule goes without a word, as Chrome drops it |
| any at-rule | goes without a word, as Chrome drops it |
| any media feature | its query is false without a word, as in Chrome |

```
go run ./examples/92-ui-kit -element styled
```

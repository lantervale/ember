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

The presentational attributes of old pages style their element as Chrome reads them, under every rule of the sheets (decision 0092): `width` and `height` on an `img`, an `iframe`, a `table`, a `td` or a `th` (pixels or a percentage), `bgcolor` on `body` and a table's parts, `text` on `body`, `align` on a `div`, a `p`, a heading and a table's parts, `valign` on a cell, its row or its row group, `nowrap` on a cell, `cellspacing` and `cellpadding` on a `table`. So are, since [decision 0152](../99-decisions/0152-the-old-attributes-of-tables-rules-and-fonts-are-hints-as-chromes.md): `border` and `bordercolor` on a `table` (an outset border, a solid one in its colour, and a pixel's inset or solid border on its cells), `frame` (thin solid sides, the others hidden; of `border` and `frame` the later sets the width, as in Chrome) and `rules` (`all`, `rows`, `cols` lines on its cells, `groups` on its row and column groups, collapsing its borders); `align="left"` and `"right"` floating a `table`, an `img` or an `iframe`, and an `img`'s other aligns (`top`, `texttop`, `absmiddle`, `absbottom`, `bottom`, `middle` as `vertical-align: -webkit-baseline-middle`, its middle on the baseline); an `img`'s `border`, `hspace` and `vspace`; `background` on `body` and a table and its parts; `align` and `valign` on a `col` or a `colgroup` (their own style: no cell takes it, as in Chrome); `clear` on a `br`; `size`, `width`, `align`, `color` and `noshade` on an `hr`; `size` (1 to 7, or a step from 3), `face` and `color` on a `font`. A colour attribute is read by HTML's legacy rules, as Chrome: a name, `#rgb`, else any text as three runs of hexadecimal digits (`bgcolor="fff"` is `#0f0f0f`). The others are in [What is missing](10-missing.md).

| Selectors | |
|---|---|
| `div`, `.class`, `#id`, `[attr]`, `[attr="value"]` | simple |
| `[attr~="word"]`, `[attr\|="en"]`, `[attr^="start"]`, `[attr$="end"]`, `[attr*="part"]`, each with the `i` flag | an attribute's value by a word, a prefix before a hyphen, its start, its end, a part; `i` compares whatever the case, and so do the values of `type`, `lang`, `rel`, `dir`, `align` and HTML's other enumerated attributes on an HTML element (`[type=checkbox]` matches `type="CHECKBOX"`), as in Chrome. The `s` flag drops the rule, as Chrome drops it |
| `a b`, `a > b`, `a + b`, `a ~ b` | descendant, child, the next sibling, a later sibling |
| `:hover`, `:focus`, `:focus-visible`, `:active`, `:disabled`, `:enabled`, `:checked`, `:open` | states; `:enabled` and `:disabled` match controls only, a control in a disabled `fieldset` (but in its first `legend`) disabled as in Chrome; `:checked` is also a select's chosen `option` |
| `:valid`, `:invalid`, `:user-valid`, `:user-invalid` | a field (an `input`, a `textarea`, a `select`, a submit button) by constraint validation as Chrome runs it: `required`, `pattern`, `min`, `max`, `step`, `minlength` and `maxlength` once the player edited the field, `type` `email` (and `multiple`), `url`, `number`, a required `select` left on its empty first option, a required radio group none of which is checked. A disabled or read-only field, a hidden input, a plain or reset button is neither; a `form` is `:invalid` when a field it owns is (one inside it, or one whose `form` attribute names it), a `fieldset` when a field in it is. `pattern` is read as JavaScript's v flag reads it, lookarounds, backreferences, `\p{…}` and class set operations with it, and one Chrome refuses constrains nothing; an email typed or given in the `value` attribute has its domain turned to punycode first ([decision 0159](../99-decisions/0159-a-pattern-is-read-as-javascript-and-a-form-is-submitted-by-its-button.md)). `:user-valid` and `:user-invalid` wait for the player to change the field (its `Changed`, `Submitted`) or to submit its form. `doc.Valid(n)` answers the same in Go (decision [0124](../99-decisions/0124-a-fields-validity-is-a-state-ui-keeps.md)) |
| `:in-range`, `:out-of-range` | a number input against its `min` and `max`: an empty or unreadable one is in range, one with a value and neither limit neither; a range input is always in range |
| `:required`, `:optional` | the `required` attribute on a `select`, a `textarea` or an input asked for a value (not `hidden`, `range`, `color` or a button); `:optional` is every other `input`, `select`, `textarea` and `button` |
| `:placeholder-shown` | an empty text, search, email, url, tel, password or number input, or `textarea`, with a `placeholder` attribute, even an empty one |
| `:indeterminate`, `:default` | a checkbox `doc.SetIndeterminate(n, true)` set so until it is clicked or checked, a radio of a group none of which is checked, a `progress` without a value; `:default` a checkbox or radio with the `checked` attribute, an `option` with `selected`, its form's first submit button |
| `:dir(ltr)`, `:dir(rtl)` | the `dir` attribute of the element or of the nearest one above it with `ltr`, `rtl` or `auto`; `auto` (and a `bdi` without one) takes the direction of the first letter of its text that has one, an input's of its value, leaving out a `bdi` and the elements with their own `dir`; ltr without one |
| `:autofill`, `:-webkit-autofill` | nothing: ui fills no field itself |
| `:focus-within` | the focused element and its ancestors; an iframe holding its Document's focus holds the page's, so the elements round it match too |
| `:target` | the element `doc.SetTarget(n)` made the Document's target, as a browser does the one a fragment names; `ScrollToStart` scrolls to it |
| `:read-write`, `:read-only` | a text field (an `input` typed in, a `textarea`) neither `readonly` nor disabled, or an element a `contenteditable` on it or above it makes editable; `:read-only` is every other element |
| `:lang(en)` | the `lang` attribute of the element or of the nearest one above it, else the Document's language (`ui.Language(tag)`, `doc.SetLanguage`, a page's Content-Language): that range or it followed by a hyphen, whatever the case. As in Chrome, one identifier only, no list and no string, both tags well formed, and a `*` matches nothing |
| `:first-child`, `:last-child`, `:only-child`, `:nth-child()`, `:nth-last-child()`, `:first-of-type`, `:last-of-type`, `:only-of-type`, `:nth-of-type()`, `:nth-last-of-type()`, `:root`, `:empty` | structure; `:empty` is an element with no element in it and no text but empty text (a space is text, as in Chrome) |
| `:not()`, `:is()`, `:where()` | a list of selectors: `:is()` and `:not()` weigh as their most specific one, `:where()` nothing; a selector `:is()` or `:where()` cannot read is left out of its list, as in Chrome |
| `:has()` | relative selectors from the element: `.card:has(img)`, `li:has(> a)`, `h2:has(+ p)`, `:has(~ .x)`, weighing as their most specific one. As in Chrome, a `:has()` in another, a pseudo-element or an empty one drops the rule; a change below or after an element restyles it where its `:has()` reads what changed (decision [0113](../99-decisions/0113-has-flags-its-anchors-before-the-cascade.md)) |
| `:link`, `:any-link`, `:visited` | an element with an `href`; nothing is visited, ui keeping no history, so a `:visited` rule is kept and matches nothing |
| `::before`, `::after`, `::marker`, `::first-line`, `::first-letter`, `::placeholder`, `::selection`, `::backdrop` | pseudo-elements, last in their selector: `li:nth-child(2)::before`, `a:hover::after` ([below](#pseudo-elements)) |

| Values | |
|---|---|
| `px` (a point), `%`, `em`, `rem`, `fr`, and the absolute `pt`, `pc`, `in`, `cm`, `mm`, `Q` at 96 `px` an inch | units |
| `ex`, `ch`, `cap`, `ic`, `lh`, and the root's `rex`, `rch`, `rcap`, `ric`, `rlh` | the element's font: its x-height, the advance of its `0`, the height of its capitals, the advance of its `水`, its line height; half an em, half an em, the ascent and an em where the face says nothing, as Chrome. In `font-size` they read the parent's font, and `lh` in `line-height` the parent's line. A font that lands styles again the lengths written in them |
| `vw`, `vh`, `vmin`, `vmax`, `vi`, `vb`, and each with `s`, `l` or `d` before it (`dvh`, `svw`…) | hundredths of the Document's viewport, an iframe's being its own box; the small, large and dynamic viewports are the one there is. A viewport that changes styles again the lengths written in them, and nothing where none is |
| `cqw`, `cqh`, `cqi`, `cqb`, `cqmin`, `cqmax` | hundredths of the nearest [container](#container-queries)'s content box on their axis, the small viewport's where there is none |
| `auto`, `min-content`, `max-content`, `fit-content`, `minmax()`, `repeat()`, `calc()`, `var()` | sizes and functions |
| `inherit`, `initial`, `unset`, `revert`, `revert-layer` | on any property: `revert` rolls an author's declaration back to the user-agent sheet's, the property unset where that sheet has none; `revert-layer` back to the cascade of the layers under its own, as `revert` from the first; left in a keyframe or a `var()` they are `unset` |
| `env()` | `titlebar-area-x`, `-y`, `-width`, `-height`: the part of the title bar a window with `platform.OwnTitleBar()` leaves its Document between the system's buttons (`win.TitleBarArea()`), undefined elsewhere, as in Chrome without the window controls overlay; `safe-area-inset-*` and `keyboard-inset-*` are `0px`; an undefined name takes its fallback, else the declaration is invalid at computed-value time, as `var()` ([decision 0142](../99-decisions/0142-the-pointer-and-the-keys-reach-callbacks-as-a-pages-dom-events.md)) |
| `min()`, `max()`, `clamp()` | the least, the greatest, a value between two (`none` for no bound), of lengths and percentages, of angles or of numbers, in each other and in `calc()`; over a percentage they are worked out by layout against what the percentage is of, as `width: min(100%, 500px)`, in a `line-height`, a `word-spacing`, a `vertical-align` and a gradient's stop too ([decision 0123](../99-decisions/0123-math-functions-over-percentages-are-a-program-layout-runs.md)) |
| `round()` (`nearest`, `up`, `down`, `to-zero`), `mod()`, `rem()`, `abs()`, `sign()`, `hypot()` | stepped and signed values, over percentages too: `width: round(50%, 8px)`, `calc(10px * sign(50% - 100px))` |
| `sin()`, `cos()`, `tan()`, `asin()`, `acos()`, `atan()`, `atan2()`, `pow()`, `sqrt()`, `exp()`, `log()`, and `e`, `pi`, `infinity`, `-infinity`, `NaN` | numbers and angles, as `rotate(atan2(1em, 20px))`; units multiply and divide as Chrome's (`calc(1em / 10px)` is a number), a declaration whose math ends in a type its property doesn't take is dropped with a warning; NaN at the top is 0, an infinity the longest length |
| times in math (`transition: opacity calc(1s + 100ms)`, `animation-delay: max(2s, 300ms * 10)`) | worked out in seconds; a negative duration is 0, as Chrome clamps it. In a `line-height`, a percentage inside `sign()` is of 100 pixels, as Chrome reads it, where the rest are of the font size |
| hex, named colours, `transparent`, `currentColor`, `rgb()`, `hsl()`, `hwb()`, `lab()`, `lch()`, `oklab()`, `oklch()`, `color()` (`srgb`, `srgb-linear`, `display-p3`, `a98-rgb`, `prophoto-rgb`, `rec2020`, `xyz`, `xyz-d50`, `xyz-d65`) | colours, their channels numbers, percentages, `none` or math (`rgb(calc(255 / 2) 0 0)`); `rgb()` and `hsl()` take the legacy commas too. Whatever the space, a colour is drawn in sRGB, clipped to it as Chrome clips ([decision 0139](../99-decisions/0139-colours-in-any-space-are-worked-out-in-srgb-as-the-cascade-reads-them.md)) |
| `rgb(from <colour> r g b / alpha)` and the other functions' (`hsl(from teal calc(h + 120) s l)`, `color(from red srgb r g b)`) | a relative colour: each channel's name is the origin's channel in the function's space, a number, in math too |
| `color-mix(in <space> [shorter, longer, increasing or decreasing hue], <colour> [<%>], <colour> [<%>])` | the two mixed in any space above, premultiplied, a missing channel taking the other's; percentages under 100% in all fade the mix. `currentColor` in a mix or an origin is the element's colour |
| `linear-gradient(<angle> or to <side or corner>, <colour> [<length or %> [<length or %>]], …)` | in `background-image` or `background`: two stops or more, two positions on one stop a hard edge; `deg`, `rad`, `grad` or `turn`; `to` a corner follows the box's shape; colours blend in encoded sRGB, as Chrome's |
| `radial-gradient([circle or ellipse] [closest-side, closest-corner, farthest-side or farthest-corner] [at <position>], <colour> [<length or %>], …)` | an ellipse to the farthest corner from the middle by default; length stops measure along the horizontal radius |
| `conic-gradient([from <angle>] [at <position>], <colour> [<angle or %> [<angle or %>]], …)` | stops at angles or percentages of a turn, two positions a hard stop: a cooldown sweep is `conic-gradient(transparent 0 70%, rgba(0,0,0,.72) 70% 100%)` |
| | all three blend in encoded sRGB as Chrome's and are cut to the box, rounded or not |

| Properties | |
|---|---|
| layout | `display` (block, inline, inline-block, flex, inline-flex, grid, flow-root, table, inline-table, table-row-group, table-header-group, table-footer-group, table-row, table-cell, table-column, table-column-group, table-caption, none, contents, and `-webkit-box` and `-webkit-inline-box`, [below](#clamping-lines)), `position` (static, relative, absolute, fixed, sticky), `float` (left, right, inline-start, inline-end, none), `clear` (left, right, both, inline-start, inline-end, none), `top` `right` `bottom` `left` `inset`, the logical names of the sides and sizes ([below](#logical-properties)), `z-index`, `box-sizing` (border-box by default; content-box under [browser sheets](#browser-sheets)), `container-type`, `container-name` and `container` ([below](#container-queries)), `width` `height` and their `min-` `max-` (lengths, percentages, `auto`, `none`, `min-content`, `max-content`, `fit-content`, and `stretch` for a width or a height, [below](#clamping-lines)), `aspect-ratio`, `margin`, `padding`, `overflow` (visible, hidden, clip, auto, scroll) and its axes (clipping on both with a `border-radius` clips to the rounded inner border, the padding box's curve, and a pointer in a cut corner reaches what's under it; nested rounded clips round only the innermost; under rotation or skew the clip is its bounding rectangle, [decision 0058](../99-decisions/0058-rounded-clips-through-a-per-batch-rounded-rect.md)), `visibility`, `order` |
| columns | `column-count`, `column-width`, `columns`, `column-gap` (normal is 1em), `column-rule` and its `-width` `-style` `-color`, `column-fill` (balance, auto), `column-span` (none, all), `break-inside` (auto, avoid; `page-break-inside` is read as it), `break-before` `break-after` (auto, avoid, column), `orphans`, `widows` |
| flex and grid | `flex` and its parts, `gap` `row-gap` `column-gap`, `align-*`, `justify-*`, `place-*`, `grid-template-columns` `-rows` `-areas`, the `grid-template` shorthand and `grid` (a template, `rows / auto-flow [dense] columns-sizes` or `auto-flow [dense] row-sizes / columns`, resetting the implicit tracks and the flow but not the gaps, as Chrome), `subgrid` for either template with the names of its lines (`subgrid [a] repeat(2, [b]) repeat(auto-fill, [c])`: its tracks are those it spans of its grid, its items size them with its margin, border and padding at its edges and half what its own gap differs from the grid's beside a gutter, it inherits the grid's names of those lines, and it is stretched over them, [decision 0125](../99-decisions/0125-a-subgrids-items-size-its-grids-tracks-hoisted-to-its-lines.md)), with names of lines in brackets (`[full-start] 1fr [content-start]`, in `repeat()` too), `grid-column` `grid-row` and their `-start` `-end`, `grid-area` (an area's name, or its four lines), a line by number, by name (an area's `-start` or `-end` line, or the first of that name), by `name n` and by `span name n`, a name no line has standing for the lines after the grid, as in Chrome, `grid-auto-*` |
| text | `font-family` `font-size` `font-weight` `font-style` `font-stretch` `font-synthesis` `font-variation-settings` and the `font` shorthand, `line-height`, `letter-spacing` (a length or a percentage of the font size, after each character), `font-variant` and its longhands `-caps`, `-ligatures`, `-numeric`, `-east-asian`, `-alternates`, `-position`, `font-feature-settings` and `font-kerning` (each kept as Chrome writes it back, and the text shaped with the face's OpenType features they ask for, [below](#opentype-features)), `word-spacing` (a length or a percentage of the font size, after each space and no-break space but a space an element keeps first in a block that collapses its own, as Chrome), `text-align`, `text-indent`, `direction` and `unicode-bidi` ([below](#right-to-left)), `text-transform` (in text fields too: drawn cased, the value kept as typed, the caret and selection following what's drawn), `white-space` (a tab it keeps goes to the next stop, `tab-size` spaces of the font apart or a length, measured from the line's start before its indent, and a carriage return takes no room), `tab-size`, `text-overflow`, `line-clamp` and `-webkit-line-clamp` ([below](#clamping-lines)), `overflow-wrap`, `word-break` (`break-all` breaks between letters to fill each line, `keep-all` keeps words of ideographs whole, `break-word` as `overflow-wrap: anywhere`), `line-break` (`anywhere` breaks between any two characters), `text-shadow`, `text-decoration` and its `-line` `-style` `-color` `-thickness`, `text-decoration-skip-ink`, `vertical-align` (every value, below), `color`, `list-style-type`, `list-style-image` and `list-style-position` ([lists](#lists-and-counter-styles)), `-webkit-text-fill-color` (the glyphs' fill, and a `currentColor` decoration's colour, as Chrome; `transparent` with a stroke for outlined text), `-webkit-text-stroke` and its `-width` and `-color` (each glyph's outline over its fill, half outside and half inside, mitred, shaped glyphs and variable instances alike; `thin`, `medium` and `thick` 1, 3 and 5 pixels wide as Chrome draws them, decision [0151](../99-decisions/0151-stroked-text-is-a-font-of-its-own-glyphs-outlines.md)), `-webkit-font-smoothing: none` (glyphs unsmoothed on the pixel grid, for pixel fonts) and `antialiased` (glyphs at their exact coverage, without the stem darkening of the default: thinner, and a square glyph on whole pixels is those pixels alone, as in Chrome on macOS) |
| look | `background-color` `-image` `-repeat` (repeat, no-repeat, repeat-x, repeat-y, space, round, or one of repeat, no-repeat, space and round for each axis) `-attachment` (scroll, fixed, local) `-origin` `-clip` (border-box, padding-box, content-box, text) `-size` (lengths, percentages, auto, cover, contain) `-position`, `border` and its parts (solid, dashed, dotted, double, inset, outset, groove, ridge; sides of other colours meet on a smoothed diagonal, decision [0132](../99-decisions/0132-border-sides-are-mitered-pieces-smoothed-by-their-own-geometry.md); round corners in every style, width and colour, dashes and dots laid round them, decision [0149](../99-decisions/0149-rounded-sides-are-bands-of-the-ring-cut-to-their-side-as-chrome-paints-them.md)), `border-radius`, `border-image` (nine-patches), `border-image-repeat` and `border-image-width`, `box-shadow`, `outline` (every border style, round corners too, and `auto`, Chrome's focus ring: a stroke two thirds of its width over a white one a third wide), `opacity` (on an inline element, the blocks it holds fade with it, as Chrome's), `transform` `transform-origin`, `filter` (`blur()`, `brightness()`, `contrast()`, `grayscale()`, `hue-rotate()`, `invert()`, `opacity()`, `saturate()`, `sepia()`, `drop-shadow()`, in any list: the element and its content filtered as one picture, a stacking context holding its absolute and fixed descendants; `url()` filters are refused, decision 0118), `backdrop-filter` (the same functions on what lies behind the element in its rounded border box, up to its backdrop root), `mix-blend-mode` and `background-blend-mode` (`normal`, `multiply`, `screen`, `overlay`, `darken`, `lighten`, `color-dodge`, `color-burn`, `hard-light`, `soft-light`, `difference`, `exclusion`, `hue`, `saturation`, `color`, `luminosity`; `mix-blend-mode` also `plus-lighter`, the element's colours and alpha added to its backdrop's up to white, as Chrome adds them), `isolation` (decision 0130), `clip-path` (basic shapes and reference boxes, below), `transition` (every property that interpolates, below), `animation` and `@keyframes` (below), `cursor`, `pointer-events` (auto, none: the pointer passes through, inherited), `app-region` (drag, no-drag, none, which set computes no-drag as in Chrome; inherited: a left press in a drag region moves the system's window and a double press is a double click on its title bar, for a header in a window with `platform.OwnTitleBar()`; what clicks is never part of the region), `image-rendering` (pixelated), `object-fit` `object-position`, `accent-color`, `caret-color`, `scrollbar-width`, `appearance`, `resize` |
| tables | `border-spacing`, `border-collapse` (separate, collapse; inherited), `caption-side` (top, bottom; inherited), `table-layout` (auto, fixed), `empty-cells` (show, hide; inherited) |
| selecting | `user-select` (auto, text, none, all; inherited), `::selection` ([below](#selecting-text)) |

A table is what its `display` says, not its element (decision 0105): `table`, `tr`, `td` and the others only start with theirs, from ui's own sheet, and any element is a table, a row, a cell, a column or a caption by saying so. What misses round a part is made as CSS 2.1 §17.2.1 says and Chrome does: cells with no row get an anonymous one, rows with no table an anonymous table, and what stands in a row without being a cell goes in an anonymous cell; white space alone between parts makes nothing. A `td` shown as a `block` is a block. `col` and `colgroup` (`span`, `width`) size their columns and paint behind their cells: a table's backgrounds are layers, the column groups under the columns, under the row groups, under the rows, under the cells, each behind the cells alone, never in the spacing (§17.5.1). A cell spanning columns with a percentage `width` shares it out to its columns that have none, in proportion to their content. `visibility: collapse` on a row, a row group, a column or a column group takes it out with its room and a spacing, the table narrower or shorter, its own `width` or `height` too, as Chrome does: the other rows and columns keep the sizes they had, and a cell it narrows keeps its content laid out as before, clipped to its box. A flex item of an automatic width keeps the width it had before, its rows too, as Chrome. With `empty-cells: hide` a cell with nothing in it, white space aside, draws no border nor background, while the borders are apart; the layers behind it still show. Cells whose `vertical-align` is `baseline` have their first lines on one baseline in their row, and an `inline-table` stands on its first row's.

With `border-collapse: collapse` a table has no spacing and no padding, and each edge of its grid has one border: the one that wins among those of the cells beside it, their rows, row groups, columns, column groups and the table (§17.6.2.1): a `hidden` one first, which leaves the edge bare, then the widest, then by style (double, solid, dashed, dotted, ridge, outset, groove, inset), then the nearest the cell, the left and the top first. A cell and the table get half of each of their edges for borders, and the table draws them all over its cells' backgrounds: an inset as a ridge and an outset as a groove, as Chrome shades them, and a double's two lines crossing those of a double as wide. Where edges meet, the one Chrome picks goes through the corner: the widest, then by style, then the one from the earlier box (the cells in the document's order, then the rows, the row groups, the columns, the column groups, the table); the others stop beside it.

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

A block with a `column-count` or a `column-width` lays its content out in columns ([decision 0088](../99-decisions/0088-columns-cut-one-flow-and-a-cut-box-stays-one-box.md)), as Chrome does. `column-count` is the most columns, `column-width` the least width of one: the container takes as many columns of at least that width as fit, up to the count, and shares its width out between them, `column-gap` apart. `columns` sets both, in either order. Without a height the columns are balanced: as short as they can be made, all the same height. With a `height` or a `max-height` the columns are still balanced when the content fits (`column-fill: balance`), or filled one after the other (`column-fill: auto`); what doesn't fit goes on in more columns of the same width, past the container's end. `column-rule` draws a line in the middle of each gap between two columns that have content, as high as the columns; it takes no room. A child with `column-span: all` spans the columns ([decision 0129](../99-decisions/0129-spanners-cut-the-flow-in-parts-each-laid-out-alone.md)): it is as wide as the container, and cuts the columns in rows, each balanced on its own and ruled only as high as it goes; with `column-fill: auto` and a height only the last row fills in turn. Its margins don't collapse with the columns', only with an adjacent spanner's.

Content is cut between blocks and between lines. A margin at a cut is dropped, at the foot of a column and at the top of the next. A paragraph keeps at least `orphans` lines at the foot of a column and `widows` at the top of the next (2 each, inherited). A table is cut between its rows, its header not repeated, a flex container between its items and its lines and inside its items, each item of a row cut where the column ends, and an absolutely positioned box whose containing block is in the columns is cut with them, as Chrome cuts them. A picture, a control, a table's row, a grid container, a float, a box that scrolls and a multi-column container inside another are never cut, nor is a box with `break-inside: avoid` but where a block in it forces a break; one taller than the columns makes them as tall as it is. `break-before: column` and `break-after: column` start a new column; `avoid` keeps two blocks together where it can. A cut box is still one element: its background and border are drawn on each part, open where it is cut, the pointer reaches it on its parts only, and its `Box()` and `BoxOf` are the box round its parts, as `getBoundingClientRect()`'s. A container is a formatting context: it holds its floats, and its margins don't collapse with its content's. An absolutely positioned child with no inset stays where its column shows it. How columns cut what they hold is decision [0156](../99-decisions/0156-columns-cut-what-they-hold-through-one-flow-of-units.md).

Limits: only a child of the container spans, where Chrome also spans a block deeper in; Chrome also cuts flex containers and tables between their rows, and ui keeps them whole; a positioned box is never cut; columns go left to right only.

A box whose `overflow` is `auto` or `scroll` shows a scrollbar in a gutter of 15 points beside its content, kept always for `scroll` and once the content overflows for `auto`; `scrollbar-width: none` scrolls with no bar and no gutter, and `thin` is as `auto`. `scrollbar-gutter: stable` keeps the vertical bar's gutter whenever the box clips (`hidden` too), whether its content overflows or not. `scrollbar-color: thumb track` draws ui's thumbs in the first colour on gutters filled with the second. No CSS draws the bar over the content: that is the Document's [`ui.OverlayScrollbars()`](01-document.md), for all its boxes.

A sheet styles a box's bars as real pages do, through Chrome's `::-webkit-scrollbar` and its parts ([decision 0158](../99-decisions/0158-styled-scrollbars-are-boxes-of-their-parts-as-chromes.md)). Once a rule styles the box's `::-webkit-scrollbar`, and its `scrollbar-color` and `scrollbar-width` are `auto`, its bars are drawn from its sheets alone, each part a box of its own with backgrounds, borders, rounded corners and shadows, as in Chrome:

```css
.list::-webkit-scrollbar { width: 10px; height: 10px; background-color: #2b2722; }
.list::-webkit-scrollbar-track { margin: 4px 0; border-radius: 5px; background-color: #3a332c; }
.list::-webkit-scrollbar-thumb { border: 2px solid transparent; border-radius: 5px; background: #ad9773 padding-box; }
.list::-webkit-scrollbar-thumb:horizontal { background-color: #6b8f5a; }
.list::-webkit-scrollbar-corner { background-color: transparent; }
```

| Part | |
|---|---|
| `::-webkit-scrollbar` | each bar: its `width` is the vertical bar's gutter, its `height` the horizontal one's (15 points when `auto`); `display: none` leaves the box no bar and no gutter |
| `::-webkit-scrollbar-track` | along the bar, less its `margin` at the bar's two ends |
| `::-webkit-scrollbar-track-piece` | the track before the thumb (`:start`) and after it (`:end`) |
| `::-webkit-scrollbar-thumb` | as thick as the bar, as long as the share of the content in view, no shorter than its `min-height` (`min-width` across), 26 points when that is `auto`; none where it doesn't fit the track |
| `::-webkit-scrollbar-corner` | where the two bars meet: white unless a rule styles it |
| `::-webkit-scrollbar-button`, `::-webkit-resizer` | read, match nothing: ui's bars have no buttons and boxes no resizer |

A part's rule may end with the pseudo-classes Chrome gives them: `:vertical` and `:horizontal`, `:start` and `:end`, `:enabled`, `:no-button` and `:corner-present` (the box scrolls both ways) hold where Chrome's do; `:hover`, `:active`, `:window-inactive`, `:disabled` and the buttons' `:decrement`, `:increment`, `:single-button`, `:double-button` never hold. The parts are read in every sheet, with or without `ui.BrowserSheets()`.

A program scrolls by an element ([decision 0111](../99-decisions/0111-a-program-scrolls-a-document-by-an-element.md)); a box of direction `rtl` scrolls from zero at its right end to below zero, as Chrome's `scrollLeft` ([decision 0146](../99-decisions/0146-a-right-to-left-scroller-scrolls-from-zero-to-below-it.md)). `doc.ScrollIntoView(n)` scrolls every box round `n` by the least that shows it, as `scrollIntoView({block: "nearest"})`; `doc.ScrollToStart(n)` puts its top at the top of each, as far as each scrolls, as `scrollIntoView()` and a browser going to a fragment. Both wait for an element not laid out yet. `doc.Scroll(n)` is how far a box's content is scrolled and `doc.SetScroll(n, off)` scrolls it there, in the Document's own points before its zoom, kept within what the box scrolls; with a scrolling viewport, below, the viewport's scroll is `doc.HTML()`'s. An offset set before the first layout holds from it, so a page can be shown where it was left. `doc.ScrollBy(n, by)` scrolls by an offset, as `scrollBy()`.

A scroll container snaps its scroll to its descendants as Chrome does ([decision 0163](../99-decisions/0163-scrolls-snap-and-glide-as-chromes-strategies-on-the-documents-clock.md)): `scroll-snap-type` (`none`, `x`, `y`, `block`, `inline` or `both`, then `mandatory` or `proximity`, the default) names the axes, and each element whose nearest scroll container it is and whose `scroll-snap-align` (`none`, `start`, `end`, `center`; block then inline) isn't `none` is a snap area, lined up with the container's snapport, its scrollport less `scroll-padding`, its own box grown by `scroll-margin`. `SetScroll`, a thumb let go, `ScrollIntoView` and `ScrollToStart` land on the snap position nearest where they were sent; `ScrollBy`, the page keys, a click on the track and the wheel, once two frames pass without it, on the nearest of those ahead, and never past an area of `scroll-snap-stop: always`. A `proximity` container snaps only within a third of its snapport, and an area bigger than the snapport lets the scroll rest anywhere it covers it. The first layout snaps a container, and a layout that moves its areas snaps it again to the ones it was snapped to. `scroll-padding` also keeps `ScrollIntoView` and `ScrollToStart` clear of a sticky header, and `scroll-margin` keeps room round the element they show. On a scrolling viewport these are `html`'s.

`scroll-behavior: smooth` has `SetScroll`, `ScrollBy`, `ScrollIntoView` and `ScrollToStart` glide on the Document's clock, as Chrome glides `scrollTo()` on the desktop: along `cubic-bezier(0.4, 0, 0, 1)`, in as many sixtieths of a second as the square root of the distance in points, at most a second and a half. The wheel's snap, and a thumb's once let go, glide that way whatever the property; the wheel, the keys and the thumb themselves move at once. `doc.Scroll(n)` reads where the glide is.

`overscroll-behavior` (`auto`, `contain`, `none`; x then y, and its `-x`, `-y`, `-inline` and `-block` longhands) keeps a wheel or a page key that reaches a box's end on an axis from scrolling the boxes round it, and an iframe's page from scrolling its host.

The window itself does not scroll: `html` is the outermost box, and a screen taller than its window is cut. A Document made with [`ui.ScrollingViewport()`](01-document.md) scrolls its viewport as a browser does. `html` then has its own height, the page's, and percentages on `html` and `body` are still of the window. `overflow` on `html`, or on `body` while `html`'s is visible, is the viewport's: `hidden` stops the wheel and the keys and shows no bar, `scroll` keeps the bar's room. A `position: fixed` box stays where it is in the window, and a sticky one sticks to the window's edges. The canvas background, `html`'s or else `body`'s, covers the window; its picture is placed in `html`'s padding box (the box its `background-origin` names), so `bottom right` is the page's end, and it scrolls with the page. Without the option it is placed in `html`'s box too, which then stays put. A fixed box inside a box that scrolls on its own stays put as that box scrolls, and so does an absolutely positioned one whose containing block is above it, as in Chrome ([decision 0137](../99-decisions/0137-a-box-placed-past-a-scroller-is-moved-back-as-it-scrolls.md)).

`text-indent` starts a block's first line in by a length or a percentage of the block's width, negative to hang it out (`-9999px` still hides a replaced heading's text). It is inherited, counts in shrink-to-fit widths, and form controls reset it to 0, as in Chrome. `each-line` and `hanging` are refused.

The `font` shorthand is `[style] [small-caps] [weight] [stretch] size[/line-height] family, …`, as CSS writes it: `font: italic bold condensed 12px/1.5 Georgia, serif`; it resets `font-variant`, `font-feature-settings` and `font-kerning`. The system fonts (`caption`, `menu`…) are refused.

An `@font-face` says more of its face than its family, weight, style and `unicode-range`, as in Chrome ([decision 0143](../99-decisions/0143-a-faces-look-and-the-faces-a-family-has-follow-its-sheets.md)): `font-stretch` (one width or a variable font's range) chooses among a family's faces by `font-stretch`, before style and weight; `size-adjust` scales its glyphs, advances and metrics; `ascent-override`, `descent-override` and `line-gap-override` set its metrics as a percentage of the font size; `font-feature-settings` shapes its text with those features ([below](#opentype-features)). Where a family has no bold (600 and up) or no italic, its face is made bold or slanted as Chrome makes it up, its advances kept, unless `font-synthesis` leaves out `weight` or `style`; without `small-caps` the small capitals `font-variant-caps` asks for are drawn as the letters are. A variable font is drawn at the instance its text asks for ([decision 0148](../99-decisions/0148-variable-fonts-are-read-in-pure-go-and-font-display-is-timed-on-the-documents-clock.md)): `font-weight` on its `wght` axis and `font-stretch` on `wdth`, each kept within the face's `font-weight` and `font-stretch` (the font's own axes when the face gives none, or `auto`), then `font-variation-settings` (`"wght" 650, "wdth" 75`) on any axis it has; a face fixed at one weight is made bold past 600 as a static one. `font-display` sets how its text shows while its file is on its way ([fonts that aren't files](01-document.md#fonts-that-arent-files)). A family is one of the Document's own (`@font-face`, from a file or [fetched](01-document.md#fonts-that-arent-files), ember's faces) unless the Document has [`ui.SystemFonts()`](01-document.md), which adds the machine's.

`text-decoration` draws `underline`, `overline` and `line-through`, `solid`, `double`, `dotted` or `dashed`, in the text's colour or `text-decoration-color`, a tenth of the font size thick or `text-decoration-thickness`. A box's lines go on all the text inside it, in its own thickness and colour, whatever its descendants set; floats, positioned boxes and inline blocks leave them behind. Places and thicknesses are Chrome's, measured. An underline or an overline leaves a gap round each glyph whose ink crosses it, as wide as the line is thick on either side, unless `text-decoration-skip-ink: none`, and every line casts the text's `text-shadow` under it, as Chrome draws them ([decision 0135](../99-decisions/0135-small-capitals-and-skipped-ink-are-synthesised-from-the-faces-outlines.md)); `wavy` is refused.

`text-align` also takes Chrome's `-webkit-left`, `-webkit-right` and `-webkit-center`, which HTML's `align` attribute and the `center` element stand for: lines as the plain word, and a block narrower than its container, a table or a sized box with no `auto` margin, placed the same way.

`vertical-align` places an inline box, an inline-block or a picture on its line (decision 0104): `baseline`, `sub` (a fifth of its parent's font size and a pixel down), `super` (a third and a pixel up), a length or a percentage of its own line height (up), `middle` (its middle half an x-height above the baseline), `-webkit-baseline-middle` (its middle on the baseline, as Chrome), `text-top` and `text-bottom` (on its parent's text), `top` and `bottom` (on the line, sized by the rest first), the line growing to hold each, as in Chrome. An inline-block stands on the baseline of its last line, or on its bottom margin edge when it has none or scrolls. On a table cell it places the content: `top`, `middle` (the default) or `bottom`, the other values as `top`.

A list item (`display: list-item`, an `li`, a `::before` or an `::after`) has its marker outside its box, beside its first line, or inside it with `list-style-position: inside`: a `disc`, `circle` or `square` bullet, its number in its list written in a counter style, a string, or a picture ([below](#lists-and-counter-styles)). The number is the `list-item` counter ([counters](#counters)), which `ol`'s `start` and `reversed` and `li`'s `value` set, and `ol`'s and `li`'s `type` attribute (`1`, `a`, `A`, `i`, `I`) and `ul`'s (`disc`, `circle`, `square`, `none`) pick the style, as in Chrome; `::marker` styles the marker, its `cursor` shown over it and its transitions run as an element's.

Solid sides of more than one colour, or seen through, meet on the diagonal of their corner, as Chrome draws them: a box of no size with one coloured side and its neighbours transparent is a triangle, the arrow of a menu or a tooltip. Dashed and dotted sides run corner to corner, their dashes counted and placed in device pixels as Chrome strokes them, at fractions of a pixel and smoothed there; dots up to three pixels wide are squares, wider ones round ([decision 0153](../99-decisions/0153-text-is-rasterized-at-its-own-size-and-dashes-are-laid-in-device-pixels.md)).

A `border-image` draws its corners and sides `border-image-width` thick and lays its edges and centre along their length, as `border-image-repeat` says, one value for both axes or two (across, then down): `stretch` (the default) stretches the piece, `repeat` lays copies at their size, centred, the ends cut, `round` fits a whole number of copies, scaled, and `space` fits a whole number at their size with the leftover shared out as equal gaps. A patterned frame (ivy, cloth, iron) keeps its pattern with `round`: `border-image: url(frame.png) 8 fill round`. Without `fill` the middle is left empty. An SVG picture is drawn at the screen's density, its slices counted in its own units.

`border-image-width` takes one to four values, as margins do: a number times the border's width, a length, a percentage of the box's side, or `auto` for the slice's own size. Its initial 1 is the border's widths, so `border: 8px solid` with an 8-texel slice draws the slice as it is, a thinner border draws it thinner, and `border-width: 0 16px` draws no top or bottom row, stretching the slice's middle band over the box. The shorthand takes it after the slice: `border-image: url(frame.png) 8 / 16px`. When opposite sides would overlap, all four shrink by the same factor, and repeated copies scale with their side: an 8-texel slice drawn 16 thick tiles 16-long copies.

`background-size` takes `cover` or `contain`, or a width then a height, each a length, a percentage of the padding box or `auto`; one value is the width. `auto` keeps the picture's proportions beside the other side, and its own size when both are auto. `background-position` places the picture at that size and `background-repeat` tiles it at that size. The `background` shorthand takes the size after the position and a slash: `background: url(beach.png) no-repeat center / cover`. An SVG picture may have no size of its own either ([SVG pictures](01-document.md#svg-pictures)): with a `viewBox` alone `auto` is `contain`, with nothing it is the whole padding box, and it is drawn at the tile's size whatever that is. A gradient has no size of its own: `auto`, `cover` and `contain` are the whole padding box, and a length or a percentage sizes the box the gradient is drawn in, which then repeats, so `background-size: 20px 20px` over a radial gradient makes dots. Where two tiles of a scaled picture meet, each keeps its own edge, where Chrome blends the two over a pixel.

A background shows in the box `background-clip` names (the border box, so it shows between the dashes of a border), and its picture is placed in the one `background-origin` names (the padding box). `background-attachment: fixed` places the picture in the Document's viewport, an iframe's own, and shows the part of it under the element; `local` places it in a scroller's content, with which it scrolls; `scroll` keeps it in the box. The `background` shorthand takes them in any order with its other parts: one box is the origin and the clip, two the origin then the clip, and `text` the clip alone. `background-clip: text` shows the layer through the glyphs and decorations of the element's text and of all it holds ([decision 0144](../99-decisions/0144-a-background-clipped-to-text-shows-through-a-picture-of-its-glyphs.md)); the text draws over it in its own colour, so a gradient heading is `background: linear-gradient(90deg, #c33, #36c) text; color: transparent`, or, as pages write it under `ui.BrowserSheets()`, `-webkit-background-clip: text; -webkit-text-fill-color: transparent`. Under a `transform`, on the element or round it, a `fixed` picture is placed in the element's box, as Chrome places it. An inline element over several lines is one box sliced by its lines (`box-decoration-break: slice`): its picture is placed in the lines laid end to end, so one at the right shows after the last word.

A background has as many layers as `background-image` has pictures, comma-separated, the first on top: `background: linear-gradient(transparent, rgba(0, 0, 0, 0.6)), url(beach.png) center / cover` darkens a picture's bottom. Each other longhand takes a list too, a value a layer: one shorter than the pictures starts again from its first value, one longer is cut, so `background-repeat: no-repeat` holds for every layer. In the shorthand a layer's parts are written as one background's, and the colour only in the last layer; it is painted under all of them, in the box the last layer's `background-clip` names. `none` is a layer that draws nothing. `background-repeat: space` repeats a picture as often as it fits whole, the first and the last at the edges of the area and the room left shared out between them, the position then not used; where it fits less than twice it is placed alone, as by `no-repeat`. `round` scales it so a whole number of copies, the nearest, fills the area; when one axis alone is round, the other side follows the picture's proportions unless `background-size` gives it. The layers of a box showing in all of it are drawn under one clip, so the colour and pictures over it batch together.

`image-rendering: pixelated` draws an `<img>` or a background from its nearest texels. A pixel whose centre falls exactly on the line between two texels takes the upper left one, as Chrome does; that happens when a texel is a whole number and a half of device pixels wide, such as 16 texels over 24 points under a `Zoom` of 1.5 on a 2× screen. Smooth images are not concerned.

`mask-image` shows an element and everything it holds through pictures: a page fades a list out with `mask-image: linear-gradient(#000 70%, transparent)` and draws an icon in one colour as an empty element with a `background-color` and a `mask` (decision [0133](../99-decisions/0133-a-masked-element-is-a-picture-cut-by-the-picture-of-its-mask.md)). Its layers, the first on top, take `url()` and gradients, and `mask-size`, `mask-repeat`, `mask-position`, `mask-origin` and `mask-clip` size, tile, place and show each as their `background-` twins do, in the border box by default (`fill-box` is the content box, `stroke-box` and `view-box` the border box; `mask-clip: no-clip` shows the mask wherever the element draws). `mask-mode: luminance` masks by a layer's colours, white showing and black hiding; `alpha` and `match-source` by its alpha. `mask-composite` joins a layer to those under it: `add` over them, `subtract` where they are clear, `intersect` where they are opaque, `exclude` where exactly one is. The `mask` shorthand sets them all, layer by layer. Under `ui.BrowserSheets()`, Chrome's `-webkit-mask` and `-webkit-mask-position` are read as the plain ones, a position of three values included (`right 4px top`), and `-webkit-mask-composite` takes the compositing operators (`source-over`, `source-in`, `source-out`, `source-atop`, `destination-over`, `destination-in`, `destination-out`, `destination-atop`, `xor`, `copy`, `clear`, `plus-lighter`), kept as written and joined as Chrome joins them, so the gradient border of the web (`-webkit-mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0); -webkit-mask-composite: xor`) draws. A masked element starts a stacking context, its clip-path and opacity apply to it whole, and its content is hit in its border box alone, wherever its mask is clear, as Chrome hits it. Each masked element is two offscreen pictures, more for each layer joined by another word than `add` or masking by luminance; an element that paints only its background colour through one picture's alpha, an icon, costs none.

`clip-path` cuts an element and everything it holds to a shape: what falls outside is neither drawn nor hit, and the element starts a stacking context (decision [0119](../99-decisions/0119-a-clip-path-is-a-mask-the-gpu-cuts-by-and-an-outline-the-pointer-is-tested-against.md)). It takes `inset()` (with `round` and its corners' radii, a slash before the vertical ones), `xywh()` and `rect()` (computed as the `inset()` they leave), `circle()` and `ellipse()` (lengths, percentages, `closest-side`, `farthest-side`, and `at` a position of one, two or four values), `polygon()` and `path()` (each after an optional `nonzero` or `evenodd`), placed in `border-box` (the default), `padding-box`, `content-box` or `margin-box` (`fill-box` is the content box, `stroke-box` and `view-box` the border box). A box alone cuts to that box with its rounded corners. Its shape is drawn once into a mask the GPU cuts by, and the pointer is tested against the outline itself, as Chrome does: on a rectangle's left and top edges and not its right and bottom ones, on any other outline. Two shapes of a kind in the same box transition and animate between each other (a polygon with as many points and the same fill rule; a `path()` only flips); any other pair flips, as Chrome does.

A `url()` whose address is a `data:` one, base64 or percent-encoded, is read from the address itself, in a sheet as in an `img`'s `src`.

### Clamping lines

A card's text cut after a few lines, its last ending in an ellipsis, is the web's `display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 3; overflow: hidden`, and ui lays it out as Chrome does (decision [0147](../99-decisions/0147-lines-are-clamped-as-chrome-clamps-them.md)): the box is a `flow-root`, its height ends below its third line, counted through the blocks in its flow (not those that start a formatting context of their own), and that line makes room for an ellipsis when anything follows it. The lines after it are laid out and drawn, as Chrome draws them, so `overflow: hidden` is what hides them. `line-clamp: 3` does the same on any block container, starting a formatting context, and draws nothing after the clamp: the standard property, which Chrome reads behind its `CSSLineClamp` flag; `line-clamp: 3 no-ellipsis` leaves the ellipsis out. Its longhands `max-lines`, `block-ellipsis` and `continue` are refused, as Chrome has none. Any other `-webkit-box` is laid out as Chrome lays out its old flexbox: as `flex` on one line, in a row, or in a column with `-webkit-box-orient: vertical`, reversed by `-webkit-box-direction: reverse`, packed by `-webkit-box-pack` (start, end, center, justify as space-between) and aligned by `-webkit-box-align` (stretch, start, end, center, baseline), its items growing and shrinking both by `-webkit-box-flex` and ordered by `-webkit-box-ordinal-group`; `justify-content`, `align-items`, `align-self`, `flex-grow`, `flex-shrink` and `order` are not read there, `flex-basis` is. An item shrinks to nothing at least, and one of a row is no wider than the box unless it says so. The box's inline children stay inline: the text and inline elements between two blocks are one item, the white space between them dropped.

```css
.blurb { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 3; overflow: hidden; }
.title { line-clamp: 1; }
```

`width: stretch` and `height: stretch` fill the containing block less the box's margins, where `auto` would shrink a table, a float, an inline block or an absolutely positioned box to its content; a height stretches only in a containing block whose height is known. So do `stretch` in `min-width`, `max-width`, `min-height`, `max-height` and `flex-basis`, a flex item's stretch filling its container (then growing and shrinking as any item), a grid item's its area; on a table cell or a column, `stretch` is `auto`, as in Chrome. Under [browser sheets](#browser-sheets) Chrome's `-webkit-fill-available` is read as `stretch`.

### OpenType features

Text is shaped by its face's OpenType tables as Chrome shapes it with HarfBuzz ([decision 0145](../99-decisions/0145-text-is-shaped-by-a-pure-go-opentype-layout-reader.md)): GSUB's ligatures and alternates and GPOS's kerning, by pairs and by classes, a space's with the words beside it included. Kerning (`kern`), the common and contextual ligatures (`liga`, `clig`, `calt`) and the face's required forms are on unless turned off; `font-variant-ligatures` (`dlig`, `hlig` and the rest; `none` turns them all off), `font-variant-numeric` (`lnum`, `onum`, `pnum`, `tnum`, `frac`, `afrc`, `ordn`, `zero`), `font-variant-east-asian`, `font-variant-alternates: historical-forms` (`hist`) and `font-kerning` ask for others, a non-zero `letter-spacing` turns the optional ligatures off, an `@font-face`'s `font-feature-settings` comes after them, and an element's `font-feature-settings` last, as in Chrome: `font-feature-settings: "ss01", "liga" 0`. `font-variant-caps` takes the face's own capitals where it has them (`smcp`, `c2sc`, `pcap`, `c2pc`, `unic`, `titl`; petite capitals drawn by small ones where the face has only those, a unicase's capitals by `smcp` and `c2sc`), else synthesises them as Chrome does, the letters it makes small uppercased at 70% of the size; `font-synthesis` without `small-caps` stops only the synthesis, as in Chrome. A feature the face lacks does nothing. A variable font is shaped at its instance: its glyphs' advances are the instance's, its kerning the default instance's. `font-variant-position` takes the face's superiors or inferiors (`sups`, `subs`) after `font-feature-settings`, as Chrome, and leaves the characters the face has none for as they are. Arabic and the other scripts whose letters join are shaped as Chrome's HarfBuzz shapes them ([decision 0157](../99-decisions/0157-arabic-is-joined-and-marks-attached-by-harfbuzzs-stages-and-masks.md)): each letter in the form its neighbours give it (`isol`, `init`, `medi`, `fina`), then `rlig`'s lam-alef and `calt`, marks sorted, placed on their bases, ligatures' components and other marks (`mark`, `mkmk`) and letters attached cursively (`curs`); text right to left is shaped in the order it is read and drawn from its end. Left out: Apple's `morx` and `kerx` tables, and the reordering of Indic scripts.

### Logical properties

ui writes text in horizontal lines only, so a logical property is a physical one, chosen by the element's `direction`, and takes its place in the cascade: the last declared wins, logical or not. `margin-`, `padding-` and `border-` with `block-start`, `block-end`, `inline-start`, `inline-end` (a border's also with `-width`, `-style`, `-color`) are top, bottom, left and right (the inline ones right and left under `direction: rtl`), and `border-inline` and `border-block`, with their `-width`, `-style`, `-color`, the left and right sides and the top and bottom ones; `inset-block-start` and the three others are `top`, `bottom`, `left`, `right`; `inline-size` and `block-size`, with their `min-` and `max-`, are `width` and `height`, `overflow-inline` and `overflow-block` are `overflow-x` and `overflow-y`, as `overscroll-behavior-inline` and `-block` are its `-x` and `-y`; `scroll-padding-` and `scroll-margin-` with a side's logical name are those of `margin-`, with `scroll-padding-inline`, `-block` and `scroll-margin-inline`, `-block` for two sides; `border-start-start-radius` and the three others are the corners from the top left (from the top right under rtl). `margin-inline`, `margin-block`, `padding-inline`, `padding-block`, `inset-inline` and `inset-block` take one value for both sides or two, start then end.

### Right to left

`direction: rtl`, or the `dir` attribute (`ltr`, `rtl`, `auto` read from the text, a `bdi` as `auto`), turns what starts an element's inline axis to its right, as in Chrome ([decision 0128](../99-decisions/0128-right-to-left-is-a-mirror-of-the-layout-and-bidi-orders-each-line.md)): lines start on the right (`text-align: start` and `end` follow it, `justify` ends there too; `text-indent` takes room on the right), a block narrower than its container stands on its right, the logical properties name the other side, a list item's marker stands on its right, a flex row's main start and a column's cross start are on the right, a grid's, a table's and a multi-column container's columns run from the right, a collapsed table border is resolved from the start side of each cell, an absolutely positioned box's static place is on the right, and a scroller's vertical bar stands on its left. A box too wide for its container and a line too long for its box overflow the left, and a scroller starts at its right end: `doc.Scroll(n)` is then zero there and below zero once scrolled left, as Chrome's `scrollLeft` ([decision 0146](../99-decisions/0146-a-right-to-left-scroller-scrolls-from-zero-to-below-it.md)). `text-overflow: ellipsis` cuts a line at its left, a select holds its arrow on its left, and a field's text and caret stand from its right. Logical properties in `@keyframes` and `transition-property` move the side the element's direction gives them. ui's sheet gives `dir` its direction and a `bdo` and the block elements their `unicode-bidi`, as Chrome's.

Each line is ordered by the Unicode Bidirectional Algorithm: Hebrew and Arabic letters are shown right to left among others, numbers and brackets take their place as UAX #9 says, every character with a mirrored glyph (brackets, `<`, `≤`, `«`…) drawn as it, and an inline element's `unicode-bidi` (`embed`, `isolate`, `bidi-override`, `isolate-override`, `plaintext`) opens and closes its part of the text as the bidi controls would; a block of `unicode-bidi: plaintext` takes its direction from its text. An inline element's margin, border and padding stay on the side its own direction gives them. Text is not shaped: Arabic shows each letter in its own form.

```css
.chat { direction: rtl; }
.chat .name { unicode-bidi: isolate; }
```

What ui refuses on purpose is below; what it does not do yet is in [What is missing](10-missing.md).

| Refused | Use |
|---|---|
| `text-decoration: wavy`, `blink` | solid, double, dotted or dashed |
| `word-wrap`, `grid-gap`, `grid-row-gap`, `grid-column-gap` | `overflow-wrap`, `gap`, `row-gap`, `column-gap` |
| `zoom` | `transform: scale()` |
| `clip` | `overflow: clip` |
| `clip-path: url()` | a basic shape: `polygon()` or `path()` for an SVG `clipPath`'s outline |
| `filter: url()`, `backdrop-filter: url()` | the filter functions |
| `mix-blend-mode: plus-darker`, as Chrome refuses it | the other blend modes |
| repeating gradients, colour hints | a gradient's plain stops |
| at-rules but `@import`, `@font-face`, `@media`, `@container`, `@keyframes` and `@starting-style` | not supported |

## Animations

Decision [0114](../99-decisions/0114-animations-run-on-transitions-clock-over-the-cascade.md).

```css
@keyframes in { from { opacity: 0; transform: translateY(8px) } }
@keyframes spin { to { transform: rotate(360deg) } }
.card { animation: in 250ms ease-out both; }
.spinner::before { content: ""; animation: spin 1s linear infinite; }
```

An `@keyframes` rule (`@-webkit-keyframes` under `ui.BrowserSheets()`) names keyframes at `from`, `to` and percentages, a list of them sharing a block; the last rule of a name wins, one inside an `@media` while its query holds. An element runs it once its name is in `animation-name`, with every longhand and the `animation` shorthand: `animation-duration`, `-timing-function` (any easing `transition` takes, `steps()` included, and a keyframe's own for the interval it begins), `-delay` (a negative one starts partway), `-iteration-count` (a number or `infinite`), `-direction`, `-fill-mode` and `-play-state`. A later name wins a property over an earlier one, unless `animation-composition` (not part of the shorthand; a keyframe may set its own) has it `add` its values to what is under it, the element's own or an earlier animation's, or `accumulate` them: lengths, numbers and colours sum either way; `transform`, `box-shadow` and `filter` lists are put after the one under with `add`, and summed function by function with `accumulate` when the two lists hold the same functions ([decision 0131](../99-decisions/0131-discrete-values-and-composition-move-as-chromes.md)).

Each property runs on the keyframes that set it; a missing 0% or 100% is the element's own value. A keyframe is computed on its element as Chrome does, so `var()`, `em` and `currentColor` in it are the element's, and follow it when its style changes. Each property interpolates as in Chrome, by its type ([decision 0122](../99-decisions/0122-each-longhand-declares-how-it-animates-in-the-properties-table.md)): lengths and percentages by computed value, a mix of the two as `calc()` (`width`, `height`, their `min-` and `max-`, `top`, `right`, `bottom`, `left`, margins, paddings, `flex-basis`, the gaps, `text-indent`, `vertical-align`, `column-width`, `text-decoration-thickness`), numbers (`flex-grow`, `flex-shrink`, `font-size`, `letter-spacing`, `line-height` with one of its kind), integers rounded halves up (`z-index`, `order`, `column-count`, `orphans`, `widows`), `font-weight` in quarters, widths snapped to device pixels as every border width is (`border-*-width`, `outline-width`, `outline-offset`, `column-rule-width`, `border-spacing`: whole points on a display of one pixel a point, halves on one of two), pairs (`border-*-radius`, `transform-origin`), positions and sizes, an offset from the end edge as `calc(100% - x)` (`background-position` and `background-size` item by item, their lists repeated to a common length, `mask-position`, `mask-size`, `object-position`), shadows (`box-shadow`, `text-shadow`, the shorter list padded with transparent ones), `aspect-ratio` by its logarithm, grid tracks of the same shape (`grid-template-columns`, `-rows`, their line names flipping halfway, a `subgrid` flipping), two basic shapes alike of `clip-path`, `filter` lists function by function, `border-image-slice` and `-width`, and every colour. A keyword (`auto`, `normal`, `none`, `cover`) only interpolates with itself; against anything else the property flips halfway through the interval, as every other property does, and `visibility` shows all along while one end is visible. `display` animates too: it flips halfway, but from or to `none` the other value shows all along, so the element keeps its box through the interval. Colours, `opacity`, `transform`, `box-shadow`, `clip-path` and `filter` only repaint; the rest restyle the element, and lay it out again where layout reads them, an `em` of the element following its moving `font-size`. An `!important` declaration of the element wins over its animations, and one in a keyframe is dropped with a warning, as Chrome ignores it; custom properties and the animation and transition properties don't animate, with a warning.

A transition runs on any of those properties `transition-property` names, a shorthand standing for its longhands, between two values that interpolate; between two that don't, the new one shows at once, as in Chrome, unless the entry has `transition-behavior: allow-discrete` (also a word of the `transition` shorthand): then it flips halfway, and `display` keeps the box all along, so `transition: opacity 0.3s, display 0.3s allow-discrete` fades an element out before it goes to `none`. An element that showed no box (its first style, or one out of `display: none`) transitions nothing, unless an `@starting-style` rule matches it: its transitions then start from the style those rules give it, which fades an element in as it appears. A child's own transition of a value it inherits runs to its parent's end value, whatever the parent's transition shows meanwhile. `TestInterpolationMatchesChrome` moves fifty properties by keyframes and by transitions and compares them with Chrome's at four times.

An animation starts when its name enters the element's list and keeps its time while the other longhands change; it stops when the name leaves, when the element or one above it is `display: none` or taken out, and starts over when the name comes back. It runs on the World's real time, as transitions do, so `RunFrames` steps it the same in a test and a capture, in a paused game too. Each one tells what it did as World events, `ui.AnimationStarted`, `AnimationIterated`, `AnimationEnded` and `AnimationCanceled`, and to the element's callbacks `ui.OnAnimationStart`, `OnAnimationIteration`, `OnAnimationEnd`, `OnAnimationCancel`, which don't bubble. A Document with `ui.ReducedMotion()`, or given the system's [preferences](#the-users-preferences) where the user asked to reduce motion, answers `prefers-reduced-motion: reduce`, where a page stops its motion itself, as in a browser. `TestAnimationsMatchChrome` compares ten animations with Chrome's paused at five times.

## Pseudo-elements

A pseudo-element ends its selector, after any compound: `.card:hover::before`, `nav ul li:nth-child(2)::before`, `ol > li::marker`. It counts as a type selector in the specificity. `:before`, `:after` are read as the two-colon forms, as Chrome reads them ([decision 0089](../99-decisions/0089-a-generated-box-is-a-node-of-its-element.md)).

`::before` and `::after` make a box inside their element, before its first child and after its last, once their `content` is something other than `none` or `normal`. The box takes any property an element takes: a display, a position, a float, a place in a flex or a grid, backgrounds, transforms, transitions, animations. It inherits from its element. It is no element: no selector matches it, `ByID`, `Children()` and automation never see it, its text is not selected or translated, and a click on it is a click on its element. The elements Chrome gives none have none (`img`, `br`, `input`, `textarea`, `select`, `canvas`), and neither have `progress`, `meter`, `table` and a table's row groups and rows, which ui lays out whole. It is sized as the Document's elements are: by its border box, or by its content under `ui.BrowserSheets()`.

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

On an element itself, `content: url(picture.png)` alone replaces the element by the picture, sized as an `img` (its children are not shown), as Chrome does; any other `content` there is read and does nothing. A form control, a frame and a canvas keep their own look.

A checkbox and a radio take `::before` and `::after`, laid out in their box and past it as in Chrome; a range, a progress bar and a meter take none yet, nor do text fields, as in Chrome.

### Lists and counter styles

`list-style-type` is `none`, `disc`, `circle`, `square`, a string (`"→ "`, written as it is), a counter style's name or `symbols()`; `list-style-image: url(dot.png)` shows the picture at its own size in place of the type, its bottom on the first line's baseline and its right edge 7 points before the item's content, unless `::marker`'s `content` says otherwise. A gradient there is a square half the font's ascent wide, as Chrome sizes a picture of no size of its own. `list-style-position: inside` makes the marker the first inline box of the item's first line, as Chrome lays it: its text as wide as it is written, a picture with 7 points after it, a bullet with the font's size after it; an item that starts with a block, or holds nothing, has the marker on a line of its own, and a marker in a bigger font grows its line. The `list-style` shorthand takes a position, a type and a picture, in any order.

A counter style writes a number for markers and for `counter()` and `counters()`. Every style CSS Counter Styles predefines is there: `decimal`, `decimal-leading-zero`, `lower-roman`, `upper-roman`, `lower-alpha` and `lower-latin`, `upper-alpha` and `upper-latin`, `lower-greek`, `armenian` and its two cases, `georgian`, `hebrew`, the digits of `arabic-indic`, `bengali`, `cambodian`, `khmer`, `devanagari`, `gujarati`, `gurmukhi`, `kannada`, `lao`, `malayalam`, `mongolian`, `myanmar`, `oriya`, `persian`, `tamil`, `telugu`, `thai` and `tibetan`, `cjk-decimal`, `cjk-earthly-branch`, `cjk-heavenly-stem`, `hiragana`, `hiragana-iroha`, `katakana`, `katakana-iroha`, the bullets, and the complex ones written by their algorithm (decision [0154](../99-decisions/0154-complex-counter-styles-inside-markers-and-scroll-state-follow-chromes-algorithms.md)): `japanese-informal` and `japanese-formal` from -9999 to 9999 as the spec writes them, `simp-chinese-informal`, `simp-chinese-formal`, `trad-chinese-informal`, `trad-chinese-formal`, `cjk-ideographic`, `korean-hangul-formal`, `korean-hanja-informal` and `korean-hanja-formal` for any number, in groups of four digits (万, 億…) as Chrome writes them, and `ethiopic-numeric`. A name nothing defines writes as `decimal`. `symbols(cyclic "*" "†")` is a style of its own made in place, for `list-style-type` and `counter()`: a system (`symbolic` when none is given, `cyclic`, `numeric`, `alphabetic` or `fixed`) and strings, no prefix and a space after; a picture as a symbol is refused, as Chrome refuses it. A list item's marker is the style's prefix, the number written, then its suffix (`". "` for decimal); `counter()` writes the number alone.

`@counter-style name { … }` defines one, or redefines a predefined one but `decimal`, `disc`, `circle`, `square` and the disclosure triangles, which are dropped with a warning, as in Chrome:

| Descriptor | |
|---|---|
| `system` | `cyclic`, `numeric`, `alphabetic`, `symbolic` (the default), `additive`, `fixed [first]`, or `extends name`, every descriptor not given taken from that style |
| `symbols` | strings or names; `alphabetic` and `numeric` take two at least. No pictures |
| `additive-symbols` | `weight symbol` pairs, the weights going down |
| `prefix`, `suffix` | around a marker's number (`""` and `". "`) |
| `negative` | the sign before and after a number below 0 (`"-"`) |
| `range` | `auto`, or pairs of bounds, `infinite` for none; out of it the fallback writes the number |
| `pad` | `3 "0"`: as many symbols before a shorter number |
| `fallback` | the style a number it can't write goes to (`decimal`) |
| `speak-as` | read, and nothing is spoken |

```css
@counter-style paren { system: extends lower-alpha; prefix: "("; suffix: ") "; }
@counter-style tally { system: additive; additive-symbols: 5 "V", 1 "I"; suffix: ": "; }
ol.steps { list-style: paren; }
ul.todo { list-style: url(box.png); }
```

### Counters

`counter-reset: name [value]` makes a counter on an element (0 by default), for its siblings after it and their descendants, `counter-increment: name [value]` adds to the nearest one (1 by default) and `counter-set: name [value]` sets it; each takes several, or `none`. A counter of the same name made deeper nests inside the outer one, which `counters()` writes whole. A counter used where none was made is made there, at 0. `counter-reset: reversed(name)` is refused, as Chrome 157 refuses it; a reversed `ol` counts down all the same.

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

`::marker` styles a list item's marker, and `::before::marker` and `::after::marker` those of a generated box that is a list item: `color`, the font properties, `letter-spacing`, `word-spacing`, `text-transform`, `text-shadow`, `quotes` (for an `open-quote` or a `close-quote` in its content), and `content`, a string, a counter or an `attr()` that replaces the bullet or the number. Its `line-height` pushes the item's first line down to hold the marker's top, its bottom left to overflow, as Chrome places an outside marker. `white-space` and `overflow-wrap` are read and, as in Chrome, change nothing: a marker keeps its spaces and stays on one line. `cursor` and the transition properties are read without a warning and change nothing yet. Any other property is refused with a warning, as Chrome ignores it.

### `::first-line` and `::first-letter`

`::first-line` styles what a block container (a block, a list item, an inline block, a table cell or caption) shows on its first line: `color`, the font properties and `line-height`, `letter-spacing`, `word-spacing`, `text-transform`, `text-decoration`, `text-shadow`, `-webkit-font-smoothing` and the backgrounds, drawn behind the line's text. The line is broken in those styles, so a larger font puts fewer words on it. The inline elements on the line inherit from it, so `span { font-size: 0.75em }` there is three quarters of the first line's size, and a block's first child block, when nothing comes before it, has its first line styled too. `opacity`, `box-shadow`, `visibility` and `vertical-align` are read and change nothing.

`::first-letter` is a box round the first letter of a block's first line, with the punctuation before and after it (`"W` of `"Well,"`), wherever that letter is: in an inline element, in `::before`'s text, in the first child block. It takes what `::first-line` takes, and margins, padding, borders, their radii and `float`: a floated letter is a drop cap, the lines going round it. It inherits from the element its letter is in. A picture, a control, an inline block or a `br` before any text leaves the block none, as in Chrome.

```css
article p:first-of-type::first-letter { float: left; font-size: 3.2em; line-height: 1; margin: 2px 6px 0 0; }
article p:first-of-type::first-line { font-weight: 700; letter-spacing: 1px; }
```

### Every pseudo-element

Every pseudo-element Chrome reads is here (`TestChromeReadsThePseudoElementsOfTheTable` asks Chrome). A refused one drops its rule with a warning in `doc.Warnings()`. One that matches nothing is read, so its rule is valid and says nothing, and styles nothing: ui has no such part.

| Pseudo-element | In ui |
|---|---|
| `::before`, `::after` | a generated box, every property |
| `::marker` | the list item's marker (above) |
| `::first-line`, `::first-letter` (and `:first-line`, `:first-letter`) | a block's first line and letter (above) |
| `::placeholder` | an empty field's placeholder: `color`, `opacity` |
| `::selection` | selected text, in a page and in a field: `background-color`, `color`, `text-shadow`, `text-decoration` and `-webkit-text-stroke` (the selected glyphs filled in its `color`: its `-webkit-text-fill-color` is read and, as in Chrome, unused); `::selection:window-inactive` while the window is inactive ([Selecting text](#selecting-text)) |
| `::backdrop` | behind a modal dialog: `background-color`, `opacity` |
| `::file-selector-button` | refused: ui has no file input |
| `::details-content` | refused: a details' content has no box of its own |
| `::part()`, `::slotted()` | matches nothing: no shadow trees |
| `::highlight()`, `::spelling-error`, `::grammar-error`, `::target-text`, `::search-text` | matches nothing: no highlights, no spelling, no find in page |
| `::cue`, `::cue()` | matches nothing: no video captions |
| `::view-transition`, `::view-transition-group()`, `::view-transition-image-pair()`, `::view-transition-old()`, `::view-transition-new()`, `::view-transition-group-children()` | matches nothing: no view transitions |
| `::scroll-marker`, `::scroll-marker-group`, `::scroll-button()`, `::column` | matches nothing: no scroll markers or buttons, a column is no box |
| `::checkmark`, `::picker()`, `::picker-icon` | matches nothing: a select is drawn whole |
| `::permission-icon` | matches nothing: no permission element |
| `::-webkit-scrollbar`, `::-webkit-scrollbar-track`, `::-webkit-scrollbar-track-piece`, `::-webkit-scrollbar-thumb`, `::-webkit-scrollbar-corner` | a scroll container's styled bars (above, beside `overflow`); `::-webkit-scrollbar-button` and `::-webkit-resizer` match nothing |
| any other vendor's (`::-webkit-slider-thumb`, `::-moz-focus-inner`…) | refused; under `ui.BrowserSheets()` as the [vendor tables](#vendor-prefixes) say |
| any other name | refused: unknown, as Chrome drops the rule |

A pseudo-element's rule that sets a property it does not take keeps its other declarations and warns of that one. Nothing may follow a pseudo-element in a selector: `::before:hover` and `li::marker::before` are refused as Chrome refuses them; `::before::marker` and `::after::marker` are read as Chrome reads them, and so is `::selection:window-inactive`, the one pseudo-class after a pseudo-element; `:window-inactive` elsewhere matches nothing, as in Chrome.

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

The selection is drawn behind the text, on each line from its first selected character to its last, as high as the line box and no less than the glyphs, and a space wider where it goes on past the line's end, as Chrome draws a selected line break; an empty line it goes past, a `br` alone on its line or one of preformatted text, is marked a space wide from its start. A picture wholly inside it is tinted. `::selection` sets its `background-color` and the text's `color`, `text-shadow` (the selected text's shadows in place of its own) and `text-decoration` (drawn on the selected text over its own); it is inherited from the parent's `::selection`, so one rule on a container styles what it holds. A rule that sets only one of the two colours leaves the other as CSS has it: no background, or the text's own colour. With no rule the background is Chrome's on macOS, `#b3d7fe` over white, and the text keeps its colour; a Document given the system's [preferences](#the-users-preferences) takes the colours the system selects in instead. A field's own selection takes `::selection`'s `background-color` and `color` too.

While the window the Document shows in is not the focused one (`win.Focused()`, which a test sets with `win.SetFocused`), a selection with no `::selection` colours is greyed as Chrome greys it on macOS, `#dcdcdc` over white (or in the system's own inactive colour), and `::selection:window-inactive` rules hold: the cascade is answered again when the window gains or loses the focus ([decision 0140](../99-decisions/0140-a-documents-window-focus-is-a-state-the-cascade-answers.md)). A double click takes the word under the pointer across the inline elements that cut it, `wo<b>rd</b>` whole, and on Windows the spaces after it too, as Chrome there (the text of fields keeps to the word); a drag past an edge of a scroller scrolls it that way, sideways too.

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
| `width`, `height`, `aspect-ratio`, `resolution`, each with `min-` and `max-` | the viewport and the window's pixel ratio; lengths in `px`, `em` and `rem` (16px), the other absolute units too, the viewport's (`vw`, `vh`, `vmin`, `vmax`, and their `s`, `l` and `d` forms), the initial font's (`ex`, `ch`, `cap`, `ic`, `lh` and their root ones, measured in the Document's `sans-serif` at 16px), and a `calc()`, `min()`, `max()` or `clamp()` of them, as `(max-width: calc(1120px - 1px))` or `(min-width: calc(50vh + 100px))`; ranges as `(width >= 600px)` and `(400px <= width < 900px)` |
| `orientation` | `portrait` when the viewport is as high as wide or higher |
| `device-width`, `device-height`, `device-aspect-ratio` | the viewport's: ui knows no screen |
| `hover`, `pointer` | `hover` and `fine`: a mouse; `none` and `coarse` while the pointer last used is a finger on a touchscreen, until the mouse moves (a trackpad's fingers are the mouse's) |
| `any-hover`, `any-pointer` | `hover` and `fine`, and `coarse` too after a finger on a touchscreen; `none` and `coarse` alone on a Document with `ui.TouchOnly()` |
| `prefers-color-scheme` | `light`; [the program's or the system's](#the-users-preferences) |
| `prefers-reduced-motion` | `no-preference`; [the program's or the system's](#the-users-preferences) |
| `prefers-contrast` | `no-preference`; [the program's or the system's](#the-users-preferences): `ui.PrefersContrast(ui.ContrastMore)` (`ContrastLess`, `ContrastCustom`, `doc.SetContrast`) |
| `forced-colors` | `none`; [the program's or the system's](#the-users-preferences), `active` with `ui.ForcedColors()` (`doc.SetForcedColors`): the query only, ui forces no colour |
| `color`, `monochrome`, `grid` | 8, 0, 0 |
| `and`, `,`, `not`, `only`, `or` between conditions in parentheses, `@media` inside `@media` | as in Chrome |

A feature ui doesn't know makes its query false, as in Chrome, with a warning; the other queries of the list still count. Every answer is Chrome's for the same viewport and the same medium (`TestChromeAnswersMediaQueriesTheSame`, `TestChromeAnswersTheMachinesQueriesTheSame`).

### The user's preferences

Decision [0121](../99-decisions/0121-a-document-answers-media-features-from-the-preferences-it-is-given.md).

A Document answers light and no preference, the same on every machine, until the program says otherwise. Given the system's [preferences](../03-windows/README.md#the-systems-appearance-and-languages), it answers as the user set their system, and restyles where a rule asks when they change it:

```go
prefs := platform.NewPreferences()
doc := ui.NewDocument(ui.Stylesheet(files, "ui/menu.css"), ui.Preferences(prefs))
win.Add(doc)
w := ember.New().AddEntities(win, prefs)
```

```css
@media (prefers-color-scheme: dark) { :root { --page: #1e1f24; --ink: #e8e8ea; } }
@media (prefers-reduced-motion: reduce) { .card { animation: none; } }
```

It answers `prefers-contrast` and `forced-colors` as the system too ([decision 0155](../99-decisions/0155-the-system-says-its-contrast-forced-colours-and-selection-colours-read-only.md)): more contrast where the user asked macOS, GNOME or the browser for it, and on Windows active forced colours under a contrast theme, with more, less or custom contrast by how far its text stands from its background. A selection no `::selection` rule colours is drawn in the colours the system selects in where it says them, blended with white as Chrome blends macOS's, and in its text colour where it gives one (Windows, KDE).

The program's own answer wins: `ui.ColorScheme(platform.AppearanceDark)` (`doc.SetColorScheme`, `platform.AppearanceUnknown` to follow the system again) for a game with a theme switch of its own, `ui.ReducedMotion()` (`doc.SetReducedMotion`), `ui.PrefersContrast(c)` for a game's high-contrast option (`doc.SetContrast`) and `ui.ForcedColors()` (`doc.SetForcedColors`). `doc.ColorScheme()`, `doc.ReducedMotion()`, `doc.Contrast()` and `doc.ForcedColors()` read the answer; a Document in an iframe answers as its host unless it has its own. A test sets the system with `prefs.SyncAppearance`, `prefs.SyncMotion`, `prefs.SyncContrast`, `prefs.SyncForcedColors` and `prefs.SyncHighlight`.

No system says whether it has a mouse, so `ui.TouchOnly()` is the program's alone ([decision 0139](../99-decisions/0139-colours-in-any-space-are-worked-out-in-srgb-as-the-cascade-reads-them.md)), for a device with a touchscreen and no mouse (`any-hover: none`), with `doc.SetTouchOnly` and its getter, and an iframe's Document answering as its host where it says nothing.

### Colour schemes

Decision [0126](../99-decisions/0126-an-element-is-drawn-in-the-scheme-its-color-scheme-allows.md).

`color-scheme` (`normal`, `light`, `dark`, `light dark`, names of the page's own, `only`) says which schemes an element can be drawn in, and its children inherit it. An element is drawn in the scheme the Document prefers where it names both, in the one it names otherwise, and light under `normal`, so a page that names no scheme stays light whatever the system says:

```css
:root { color-scheme: light dark; }
.card { background: light-dark(#ffffff, #1e1e2e); border: 1px solid light-dark(#d0d0d0, #3a3a4a); color: CanvasText; }
```

In that scheme the system colours (`Canvas`, `CanvasText`, `Field`, `FieldText`, `ButtonFace`, `ButtonText`, `ButtonBorder`, `LinkText`, `GrayText`, `Highlight`, `Mark`, `AccentColor`… and the old names that stand for them) take Chrome's values, and `light-dark(a, b)` its first colour or its second, in any property that takes colours: gradients, shadows and a variable's value where it is used included. Both are resolved where they are declared, as Chrome does, so a child of another scheme inherits the colour its parent shows. When the scheme an element follows flips, its colours are styled again and run their transitions. The root's text is `CanvasText`, and a dark root with no background has the canvas Chrome's dark `Canvas` (`#121212`) where a light one leaves the window's colour showing. The user agent sheet draws its controls, links, marks, dialogs and fieldsets in Chrome's dark colours under a dark scheme; under `ui.BrowserSheets()` a control's text is `FieldText` or `ButtonText`, where a game's takes its parent's colour. kit's theme sets its own colours and doesn't change.

## Container queries

Decision [0120](../99-decisions/0120-container-queries-restyle-after-layout-until-their-answers-hold.md).

```css
.panel { container: panel / inline-size; }
.stats { display: flex; flex-direction: column; }
@container panel (width > 420px) { .stats { flex-direction: row; } .stat { width: 30cqw; } }
```

An element with `container-type: inline-size` or `size` is a container: an `@container` rule applies to an element while its condition holds for the nearest container above it, so a component follows the room its panel leaves it, not the window's. `container-name` gives a container names a rule can ask for (`@container panel (…)`), and `container` sets both, the type after a slash. The query is answered for the container's content box, the same frame it is laid out: `Document.Layout` lays the tree out, answers the containers, and styles and lays out again what an answer moved, until none moves. A resize that crosses no container's boundary restyles nothing.

A container's size doesn't come from what it holds, as in Chrome: an `inline-size` container shrinking to fit (an inline block, a float, a flex or grid item, an absolutely placed box) is as wide as its padding and border, within its `min-width`, its text wrapping inside; a `size` container is as high as nothing too, unless its `height` says. Every container holds its floats and its children's margins, as a `flow-root` does. An inline box, a table or a part of one is no size container: a size query never holds there.

| Container queries | |
|---|---|
| `@container [name] condition`, a list of them | a condition asks the nearest container above the element (the element itself for its `::before` and `::after`) with that name whose type answers every size feature it reads: an `inline-size` container answers the width only. A name alone holds wherever an element above has it |
| `width`, `height`, `inline-size`, `block-size`, `aspect-ratio`, `orientation`, each size with `min-` and `max-` | the container's content box; ranges as `(400px <= width < 900px)`; lengths in any unit, a font's as the container's font measures it, `calc()` too |
| `and`, `or`, `not`, `@container` inside `@container` or `@media` | as in Chrome; a container nobody has, or a size not laid out yet, is unknown, false under `not` too |
| `style(--name: value)`, `style(--name)`, and `and`, `or`, `not` inside `style()` | the container's custom property, its text compared; every element is a container for these |
| `style()` on a standard property (`style(display: flex)`) | never holds, as Chrome reads it: the whole condition fails, whatever else it asks, with a warning |
| `@keyframes` inside `@container` | apply whatever the container, as in Chrome, which never asks their condition |
| `scroll-state(stuck: top)`, `scroll-state(scrollable: y)`, `scroll-state(scrolled: bottom)`, `scroll-state(snapped: x)`, alone or joined by `and`, `or`, `not` inside | a `container-type: scroll-state` container's scroll state (with `size` or `inline-size` too): the sides a sticky container is stuck to, the sides a scroll container can still scroll toward and those its last scroll by the user or `ScrollBy` went toward on each axis (`top`… `left`, `block-start`… `inline-end`, `x`, `y`, `block`, `inline`; `none` before any), the axes a snap area is snapped on by its container (`x`, `y`, `block`, `inline`, `both`, `none`), answered after each layout and each scroll |

A malformed condition (`@container none (…)`, a name after the query) drops the rule, as Chrome drops it. `doc.MatchedRules(n)` names the `@container` rules a match stands in.

## Where a sheet came from

A `url()` in a sheet is read beside the sheet: `ui.Stylesheet(fsys, "ui/tool.css")` reads `url(img/panel.png)` as `ui/img/panel.png` of its files. A sheet fetched from elsewhere and given as text says where it came from, and its `url()` are fetched from there by the Document's `ui.FetchImages`:

```go
ui.NewDocument(
	ui.FetchImages(web.Images(page)),
	ui.CSS(text, ui.From("https://cdn.example/site/css/main.css")), // url(../img/wall.png) is https://cdn.example/site/img/wall.png
)
```

`sheet.From()` reads it back. A `url()` in a style attribute is the page's, resolved by the fetcher. `@font-face` files are read from a `Stylesheet`'s files, and fetched from the sheet's address with [`ui.FetchFonts`](01-document.md#fonts-that-arent-files), and so are the sheets it imports, with `ui.FetchSheets` ([below](#imported-sheets)).

## Imported sheets

Decision [0112](../99-decisions/0112-imported-sheets-are-fetched-and-spliced-where-the-import-stands.md).

An `@import` brings another sheet's rules in where it stands, as in a browser. A `Stylesheet` reads the sheets it imports beside it in its file system; any other sheet, or one its files lack, has them fetched by the Document's `ui.FetchSheets`, from the importing sheet's address:

```go
doc := ui.NewDocument(
	ui.BrowserSheets(),
	ui.CSS(`@import url(theme.css) layer(theme) (min-width: 600px); body { margin: 0; }`,
		ui.From("https://site.example/css/site.css")), // theme.css is https://site.example/css/theme.css
	ui.FetchSheets(web.Sheets("https://site.example/")),
)
```

| | |
|---|---|
| `ui.SheetFetcher` | `func(url, sheet string) ([]byte, error)`: the sheet's text for the address its `@import` names, in full when the importing sheet says where it came from (`ui.From`), beside its name otherwise; `sheet` is the importing sheet's address, what a request gives as its referrer |
| `web.Sheets(base)` (package `ui/web`) | a fetcher over HTTP, as `web.Fonts`: a sheet served as a type other than `text/css` (none is taken, as Chrome takes it), over `web.MaxSheetBytes` (8 MB) or slower than `web.Timeout` is an error |
| `doc.SheetsLoading()` | how many imported sheets are on their way |

- `@import url(x.css)`, `@import "x.css"`, then `layer` or `layer(name)`, `supports(condition)` and a media query list, as CSS has them. `layer()` and `supports()` need `ui.BrowserSheets()`; without it they are warnings and the import goes.
- The imported rules stand where the `@import` is: before the importing sheet's own, in order, so a later rule wins at equal specificity; under the import's media query list, answered for the viewport as it is resized; in its layer, the imported sheet's layers inside it, or named where the import stands when it has none. An import whose `supports()` fails is not fetched.
- An imported sheet's own imports are followed from its address. An import of a sheet it is imported through (`a.css` importing `b.css` importing `a.css`) stops there with a warning, as Chrome stops it.
- An `@import` after any rule but `@charset` and `@layer` statements, or inside a block, is ignored with a warning, as browsers ignore it.
- A sheet is fetched once per Document, however many imports name it, and stands in each of their places. Until it lands the Document shows without it, then styles again, once a frame for all that landed. Its warnings, its `@font-face` rules and a failed fetch are the Document's, each with the sheet's address, and `doc.MatchedRules` names an imported rule by the sheet it came from.
- Without `ui.FetchSheets`, an `@import` no file system holds is a warning, and the sheet shows without it.
- `TestImportsCascadeAsChromes` (package `ui/web`) has Chrome and ui follow the same imports from a local server: order, a layer named before the import, media at two widths, `supports()`, a nested import, a cycle, a sheet imported twice, a sheet of the wrong type, one missing, one after a rule.

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
- A run-time sheet is read as the Document's others: under `ui.BrowserSheets()` as a browser's, its `@media` answered for the viewport as it is resized, its `@layer` names shared with the other sheets, its warnings in `doc.Warnings()`. An `@font-face` in it is read as in `ui.CSS`, one inserted as a rule too, and its face goes with the sheet removed or disabled, or the rule deleted, each file read or fetched once however often its face comes back; its `@import` rules are followed as `ui.CSS`'s.
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
| `font`, `line-height` and `white-space` of a control | Chrome's small control font on `button`, `input`, `select` and `textarea`, `13.333333px Arial` (a `textarea`'s `monospace`) at a `normal` line height; a text field takes its parent's `white-space`, a button input and a `select` `pre`, a `select` with `multiple` and an `option` `nowrap` (`TestChromeFontsTheControlsAsABrowsersSheetDoes`); without the option a control takes its parent's font and keeps its text on one line. Arial is drawn by the machine's own once `ui.SystemFonts()` is on, else by Go |
| an element ui has none of in a selector (`applet`, `object`, a custom element) | the selector matches nothing and goes without a word, its rule keeping the others: a reset that lists every element of HTML still applies. Inside `:not()` it keeps its warning and drops the rule |
| vendor prefixes | read as the plain name where Chrome does, dropped without a word where Chrome drops them: the tables below |

Chrome's legacy gradients are read as the plain ones it draws: `-webkit-linear-gradient()` gives where it starts, not where it goes, so `top` is `to bottom`, and its angle turns the other way (`30deg` is `60deg`); `-webkit-radial-gradient(<position>, <shape> <size>, …)` is `radial-gradient(<shape> <size> at <position>, …)`, `contain` and `cover` being `closest-side` and `farthest-corner`. A corner or a centre in a linear one keeps its warning, as Chrome draws it corner to corner of the box (`TestChromeDrawsTheLegacyGradientsAsTheirPlainOnes`).

A prefixed name read as a plain one keeps its place in the cascade, so `-webkit-transform: rotate(3deg); transform: none` ends as `none` and the other order as the rotation. A warning about one names it as it was written.

What ui can't do stays a warning with the option too: a property or value Chrome reads and ui doesn't (`columns`, `-webkit-line-clamp`, `display: -webkit-box`, `-webkit-fill-available`, a `-webkit-gradient()`), `@font-face` in a sheet given as text without `ui.FetchFonts`, `@import` without `ui.FetchSheets`. A declaration ui refuses whose value holds another engine's word (`transition: opacity 1s, -moz-transform 1s`) goes whole without a word.

### Vendor prefixes

One table for each vendor, as `ui/css/vendors.go` has them. `-webkit-`'s lists are Chrome's own: a test asks the Chrome the suite runs which plain property each `-webkit-` name sets (`TestChromeReadsWebkitPropertiesAsTheTable`), and another that it knows no name of the other four.

#### `-webkit-`

| | |
|---|---|
| properties read as the plain one of the same name (73) | `-webkit-align-content` `-webkit-align-items` `-webkit-align-self` `-webkit-animation` `-webkit-animation-delay` `-webkit-animation-direction` `-webkit-animation-duration` `-webkit-animation-fill-mode` `-webkit-animation-iteration-count` `-webkit-animation-name` `-webkit-animation-play-state` `-webkit-animation-timing-function` `-webkit-app-region` `-webkit-appearance` `-webkit-backface-visibility` `-webkit-background-clip` `-webkit-background-origin` `-webkit-background-size` `-webkit-border-bottom-left-radius` `-webkit-border-bottom-right-radius` `-webkit-border-radius` `-webkit-border-top-left-radius` `-webkit-border-top-right-radius` `-webkit-box-shadow` `-webkit-box-sizing` `-webkit-clip-path` `-webkit-column-count` `-webkit-column-gap` `-webkit-column-rule` `-webkit-column-rule-color` `-webkit-column-rule-style` `-webkit-column-rule-width` `-webkit-column-span` `-webkit-column-width` `-webkit-columns` `-webkit-filter` `-webkit-flex` `-webkit-flex-basis` `-webkit-flex-direction` `-webkit-flex-flow` `-webkit-flex-grow` `-webkit-flex-shrink` `-webkit-flex-wrap` `-webkit-font-feature-settings` `-webkit-hyphenate-character` `-webkit-justify-content` `-webkit-mask-clip` `-webkit-mask-image` `-webkit-mask-origin` `-webkit-mask-repeat` `-webkit-mask-size` `-webkit-opacity` `-webkit-order` `-webkit-perspective` `-webkit-perspective-origin` `-webkit-print-color-adjust` `-webkit-shape-image-threshold` `-webkit-shape-margin` `-webkit-shape-outside` `-webkit-text-emphasis` `-webkit-text-emphasis-color` `-webkit-text-emphasis-position` `-webkit-text-emphasis-style` `-webkit-text-size-adjust` `-webkit-transform` `-webkit-transform-origin` `-webkit-transform-style` `-webkit-transition` `-webkit-transition-delay` `-webkit-transition-duration` `-webkit-transition-property` `-webkit-transition-timing-function` `-webkit-user-select` |
| properties read as a plain one of another name (30) | `-webkit-border-after` as `border-block-end`, `-webkit-border-after-color` as `border-block-end-color`, `-webkit-border-after-style` as `border-block-end-style`, `-webkit-border-after-width` as `border-block-end-width`, `-webkit-border-before` as `border-block-start`, `-webkit-border-before-color` as `border-block-start-color`, `-webkit-border-before-style` as `border-block-start-style`, `-webkit-border-before-width` as `border-block-start-width`, `-webkit-border-end` as `border-inline-end`, `-webkit-border-end-color` as `border-inline-end-color`, `-webkit-border-end-style` as `border-inline-end-style`, `-webkit-border-end-width` as `border-inline-end-width`, `-webkit-border-start` as `border-inline-start`, `-webkit-border-start-color` as `border-inline-start-color`, `-webkit-border-start-style` as `border-inline-start-style`, `-webkit-border-start-width` as `border-inline-start-width`, `-webkit-logical-height` as `block-size`, `-webkit-logical-width` as `inline-size`, `-webkit-margin-after` as `margin-block-end`, `-webkit-margin-before` as `margin-block-start`, `-webkit-margin-end` as `margin-inline-end`, `-webkit-margin-start` as `margin-inline-start`, `-webkit-max-logical-height` as `max-block-size`, `-webkit-max-logical-width` as `max-inline-size`, `-webkit-min-logical-height` as `min-block-size`, `-webkit-min-logical-width` as `min-inline-size`, `-webkit-padding-after` as `padding-block-end`, `-webkit-padding-before` as `padding-block-start`, `-webkit-padding-end` as `padding-inline-end`, `-webkit-padding-start` as `padding-inline-start` |
| properties left as written (45), so each keeps its warning unless ui reads it | `-webkit-mask-box-image` `-webkit-mask-box-image-outset` `-webkit-mask-box-image-repeat` `-webkit-mask-box-image-slice` `-webkit-mask-box-image-source` `-webkit-mask-box-image-width` `-webkit-mask-position-x` `-webkit-mask-position-y`: Chrome's own masks, kept apart from mask and its longhands; `-webkit-border-horizontal-spacing` `-webkit-border-vertical-spacing` `-webkit-box-decoration-break` `-webkit-box-reflect` `-webkit-line-break` `-webkit-locale` `-webkit-rtl-ordering` `-webkit-ruby-position` `-webkit-tap-highlight-color` `-webkit-text-combine` `-webkit-text-decorations-in-effect` `-webkit-text-orientation` `-webkit-text-security` `-webkit-user-drag` `-webkit-user-modify` `-webkit-writing-mode`: Chrome's own, with no plain property; `-webkit-box-align` `-webkit-box-direction` `-webkit-box-flex` `-webkit-box-ordinal-group` `-webkit-box-orient` `-webkit-box-pack` `-webkit-font-smoothing` `-webkit-line-clamp` `-webkit-text-fill-color` `-webkit-text-stroke` `-webkit-text-stroke-color` `-webkit-text-stroke-width`: a property of ui under this very name; `-webkit-border-image`: fills the middle without the fill word, where border-image leaves it empty; `-webkit-perspective-origin-x` `-webkit-perspective-origin-y` `-webkit-transform-origin-x` `-webkit-transform-origin-y` `-webkit-transform-origin-z`: one axis of the plain property, which has no such longhand; `-webkit-column-break-after` `-webkit-column-break-before` `-webkit-column-break-inside`: takes always, which the break-* property it sets refuses |
| Chrome's own properties read as the plain one (3) | `-webkit-mask` as `mask`, a shorthand of Chrome's own longhands, which take three-value positions and refuse no-clip; `-webkit-mask-composite` as `mask-composite`, takes the compositing operators (copy, destination-in…), which mask-composite keeps as written; `-webkit-mask-position` as `mask-position`, a shorthand of -webkit-mask-position-x and -y, which take three-value positions |
| any other property | goes without a word, as Chrome drops it |
| values read as the plain one | `-webkit-calc` as `calc`, `-webkit-fill-available` as `stretch`, `-webkit-fit-content` as `fit-content`, `-webkit-flex` as `flex`, `-webkit-grab` as `grab`, `-webkit-grabbing` as `grabbing`, `-webkit-inline-flex` as `inline-flex`, `-webkit-max-content` as `max-content`, `-webkit-min-content` as `min-content`, and the properties above in `transition` |
| any other value | keeps its warning, the declaration dropped |
| pseudo-elements read as the plain one | `::-webkit-input-placeholder` as `::placeholder` |
| any other pseudo-element | matches nothing: its selector alone goes, without a word |
| pseudo-classes read as the plain one | `:-webkit-any-link` as `:any-link` |
| pseudo-classes nothing in ui is ever in | `:-webkit-autofill` `:-webkit-drag` `:-webkit-full-page-media` `:-webkit-full-screen` `:-webkit-full-screen-ancestor`: the selector alone goes, without a word |
| pseudo-classes Chrome reads and ui can't | `:-webkit-any`: the rule keeps its warning |
| any other pseudo-class | its whole rule goes without a word, as Chrome drops it |
| at-rules read as the plain one | `@-webkit-keyframes` as `@keyframes` |
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

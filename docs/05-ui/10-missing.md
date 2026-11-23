# What is missing

What ui does not do yet, where a page written for a browser would expect it. Each line is a known gap, not a choice: what ui refuses on purpose is in the [Refused table](03-css.md). A line leaves this page when its test against Chrome passes.

## Text

| Missing | Today |
|---|---|
| a `text-shadow` on stroked text whose fill is transparent | the shadow of the glyphs filled, where Chrome's is the shadow of their stroke |
| a stroke's mitre at a sharp corner between two short curves of a glyph | the curves are stroked as straight pieces, and the mitre can reach a little further than Chrome's |
| the default selection colour on Windows and Linux, focused or not | the system's colours under `ui.Preferences` (Windows' `Highlight` and `HighlightText`, KDE's selection colours), blended with white as Chrome on macOS blends its own: the machines here run Chrome on macOS only, which can't say what the others draw; GNOME's accent colour is not read, an inactive window's selection there is Chrome's macOS grey, and without `ui.Preferences` macOS's blue everywhere ([decision 0155](../99-decisions/0155-the-system-says-its-contrast-forced-colours-and-selection-colours-read-only.md)) |
| the spaces after a word on Windows, in a field's double click, Control+Shift+Right and a drag by words | the word alone, where Chrome on Windows takes the spaces after it as a page's double click does |
| selecting by touch, dragging selected text | none: the pointer and the keys select, and a press inside the selection starts a new one where Chrome drags the text |
| a rich copy's look, and one on a Linux desktop or the web | the HTML holds the elements and their attributes, where Chrome writes each element's computed style inline; a Linux desktop and the web get the plain text alone, GLFW offering no `text/html` target ([decision 0162](../99-decisions/0162-a-copy-puts-the-selections-markup-beside-its-text.md)) |
| `word-break: break-all` after a letter carrying two combining marks | a break after the marked letter, where Chrome keeps it with the next ([decision 0178](../99-decisions/0178-spaces-break-spaces-keeps-are-units-of-a-line-and-break-all-reads-letters-as-ideographs.md)) |
| `hyphens: auto` | read, and words break at their soft hyphens alone, as `manual`: Chrome on macOS hyphenates through the system's dictionaries, which no pattern file ui could embed reproduces ([decision 0177](../99-decisions/0177-lines-are-chosen-again-after-the-greedy-ones-and-calc-size-is-a-keyword-scaled.md)) |
| a space after a tab under `white-space: break-spaces` that ends past the line by less than a layout unit | goes to the next line, where Chrome keeps it at the end of its own |
| `text-wrap: balance` and `pretty` in a block whose lines go round floats, under a `::first-line` or a `line-clamp` | broken greedily, as `auto`, where Chrome scores them too (round floats it scores lines whose room is a float's alone) |
| `text-spacing-trim` and `text-autospace` in vertical text, text fields and a select's list | the full-width punctuation and the room between ideographs and letters left as the face draws them, where Chrome trims by `vhal` in vertical text and spaces both everywhere ([decision 0187](../99-decisions/0187-han-punctuation-is-trimmed-by-halt-and-autospace-is-room-after-a-character.md)) |
| `text-spacing-trim: trim-start` on an opening bracket that starts a block or follows a forced break | trimmed, as the specification has it, where Chrome leaves it full width at the start of a text node in Hiragino Sans and trims it inside an element or in Hiragino Sans GB |
| a face with `chws` | its pairs of punctuation trimmed by Chrome's rules and `halt`, where Chrome hands the middle of a run to the face's own `chws` |
| `line-break: loose` in Chinese, Japanese and Korean, round prefixes and postfixes beyond those measured | `＄` `￥` `€` and `％` `°` `℃` `‰` `′` `″` `￠` break as Chrome breaks them; ICU's other full-width prefixes and postfixes (`￡`, `￦`…) break as under `normal` |
| `text-box-trim` at the end of a column whose last line a block's bottom padding or border follows, and in the boxes side by side a column cuts (a flex or grid row's items, a table row's cells) | not compared with Chrome: the padding keeps the line whole, and the lines of boxes side by side are never trimmed at a column's start or end ([decision 0192](../99-decisions/0192-text-box-trim-follows-chromes-lines-into-columns-and-floats.md)) |
| a drag's scroll sideways | a drag past an edge scrolls up and down only |
| `initial-letter` on an element (the first inline child of a block), `initial-letter-align`, and a drop cap in vertical text | not read: only a `::first-letter` is made a drop cap, its letter's ink measured in its first face |
| `text-underline-position` and `from-font` on a variable font | the default instance's post and OS/2 values, not the instance's MVAR deltas ([decision 0189](../99-decisions/0189-an-underline-is-placed-by-its-decorating-box-as-chrome-places-it.md)) |
| the characters whose ink `text-decoration-skip-ink: auto` leaves whole, outside the blocks measured (CJK, symbols, pictographs, Hangul, Linear B) | skipped as a Latin letter's, where Chrome's table may name more ([decision 0182](../99-decisions/0182-skipped-ink-follows-the-glyphs-drawn-and-chromes-table-of-characters.md)) |
| text on a display of one pixel to the point | a glyph's darkened edge shows on its left and top and, as rounding has it, not always on its right and bottom: the last half pixel of its quad has its centre on the quad's edge. Drawn every time (each quad a texel longer right and below), 27 of the 44 pixel pages with text measured further from Chrome at that scale and 2 nearer, none changed at two pixels a point, so it stays; `-webkit-font-smoothing: antialiased` has no such edge |
| fonts a script loads (Typekit…) | ui runs no script: a sheet a browser's script adds gives its `@font-face` rules as any other, but a `FontFace` a script makes itself is not known, and the next family of the list is used |
| some installed fonts | `ui.SystemFonts()` skips a file it cannot read; nothing waits for the scan in the public API |
| typing several letters quickly to find a listbox's option | each letter finds the next option starting with it alone, where Chrome matches what was typed within a second ([decision 0202](../99-decisions/0202-a-select-with-multiple-or-a-size-is-a-listbox-of-rows-in-the-page.md)) |
| an empty listbox's rows | as tall as ui's line of the font and a point, a point more than Chrome's for Arial at 13.33px |
| a drag past a listbox's edge, and scrolling it as a run is extended | the run reaches only the options under the pointer, where Chrome scrolls the listbox on and selects as it goes |
| a listbox's keys scrolling it | at once, by the least that shows the option reached, where Chrome scrolls smoothly to it |
| a listbox's selected rows under forced colours | Chrome's grey, light blue and dark scheme colours of the default sheet, never the system's `SelectedItem` |
| a bare `select` dropdown's height under `ui.BrowserSheets()` | a point taller than Chrome's |
| an open select's `optgroup` labels and indented options | drawn bold and behind four spaces in ui's own list; not compared: Chrome on macOS shows the list as a native menu no screenshot takes |
| a `word-spacing` or `letter-spacing` so far below zero that a run of text ends left of its start | the run's box is empty at its start, as Chrome's, but what follows on the line starts where the run's glyphs end, left of it, where Chrome starts it at the run's start; a block sized by such text can differ from Chrome's by a glyph |
| `word-spacing`, `letter-spacing` and `-webkit-font-smoothing` in an open select's list | drawn and measured with them, as the select's own text; not compared: Chrome on macOS shows the list as a native menu no screenshot takes |
| a space Go and Go Mono lack that Chrome draws in a face of the system (U+3000, U+1680) | a missing glyph's room, where Chrome takes the system face's, and its line grows to that face's height |
| Arabic in a face without `init`, `medi` and `fina` (only the presentation forms of its cmap) | each letter in its own form, where Chrome's HarfBuzz joins it with those forms; `stch` too |
| a space between two Hangul syllables in Noto Sans KR | the face's own space, as `hb-shape` draws it, where Chrome's is 56 units of 1000 wider (the face's proportional space for Korean, which it takes even under `lang="en"`) |
| a mark a face without GPOS places by its combining class | placed by its glyph's box from `xMin`, where HarfBuzz takes the left side bearing of `hmtx`: the same in a face whose two agree, as faces' do |
| an emoji followed by VS15 or VS16 | VS16 draws the emoji set's picture and VS15 the face's glyph ([decision 0171](../99-decisions/0171-thai-hangul-and-hebrew-are-shaped-as-harfbuzz-and-a-run-between-the-text-round-it.md)); not compared, as Chrome takes its emoji from the system's colour font |
| a face whose only layout tables are Apple's `morx` and `kerx` | drawn without `morx`'s substitutions (ligatures, contextual forms, the joining of Geeza Pro and the Mac's other Arabic faces) and `kerx`'s kerning, where Chrome's HarfBuzz reads them. About 150 of the Mac's faces have no GSUB or GPOS, among them Helvetica, Helvetica Neue, Times, Courier, Menlo, Geneva, Palatino, Optima, Baskerville, Didot and Zapfino; two thirds of them have the old `kern` too, which ember reads, so their kerning is Chrome's but for the few that have `kerx` (Helvetica Neue's bolder faces, the Indic MN faces). Reading `morx`'s four state machines and `kerx`'s formats is a shaper of its own, and only the machine's faces (`ui.SystemFonts()`) have these tables |
| a pre-base matra after a letter of another element | takes `init` as at a word's start |
| a line that fits only broken inside a ligature of an element's text that doesn't start it (`o<span>ffi</span>` or `بلا<span>لا</span>` under `word-break: break-all` or `overflow-wrap: anywhere`) | the ligature kept whole and the line overflowing, where Chrome breaks inside it and reshapes the line's end ([decision 0197](../99-decisions/0197-text-right-to-left-and-sideways-is-shaped-as-one-and-a-reshaped-end-keeps-its-joins.md)) |
| an inline element split by bidi reordering, one part of it empty (its text inside a ligature, `ab ل<span>ا cd</span>`) | its box spans from the empty part, across the text between, where Chrome's leaves the empty part out |
| `letter-spacing` on Arabic text | spaced, where Chrome leaves its letters unspaced |
| a line whose ends, shaped again ([decision 0190](../99-decisions/0190-line-ends-are-shaped-again-and-carets-share-a-ligature.md)), fit differently | broken where the text shaped whole fits, then its ends reshaped, where Chrome fits the line again with them |
| a ligature that draws a space (a face with `liga` across one) | a shrink-to-fit box measures it whole at a line's end where Chrome measures its characters' shares; a line may break at the space before it where Chrome takes a later break; a chain of such ligatures is shaped with five characters on either side and may pair them otherwise |
| a selection of vertical lines going past an empty line of preformatted text, a br alone on its line or a tab kept between fragments ([decision 0193](../99-decisions/0193-vertical-lines-are-selected-in-the-flow-of-their-lines.md)) | the empty line and the br unmarked and the tab's room left out, where Chrome marks them as on horizontal lines |
| a double click on ideographs | takes one ideograph, where Chrome takes the word ICU's dictionary finds (一丁) |
| a character standing in a vertical line that a fallback face draws (an emoji of the system's colour font, a symbol of Go Mono) | it takes the first face's ascent and descent, or its vertical advance, where Chrome takes the fallback face's |

## Generated content

| Missing | Today |
|---|---|
| a range of `appearance: slider-vertical` whose height is `auto`, and a vertical range with a `::before` or an `::after` | 16 tall, and its generated content in a row across it, where Chrome makes it 129 by 129 and lays the row down the range ([decision 0199](../99-decisions/0199-a-controls-parts-are-hovered-and-pressed-as-chromes-shadow-elements.md)) |
| a part's `:focus`, `:focus-visible` and `:focus-within`, an author's `!important` `width` on a progress's value, a thumb hovered where it overhangs its range's box | the rule dropped with a warning, the value's width its fraction, and the thumb hovered only within the range's box, where Chrome reads the pseudo-classes and matches nothing, takes the width, and hovers the thumb wherever it is drawn |
| a meter of its own look whose parts a sheet styles | ui's flat bar in the parts' `background-color`s, where Chrome draws its rounded native bar in them |
| a viewport's styled bars, the scrollbar parts' `:hover`, `:active` and `:window-inactive`, `::-webkit-scrollbar-button`, `::-webkit-resizer`, styled parts under `scrollbar-width: thin` | ui's own bars on a scrolling viewport, the parts drawn as at rest, no buttons and no resizer, and the parts drawn under `thin`, where Chrome draws the root's styled bars on the viewport, its hovered and pressed parts, buttons sized by their rules, and its system bars once `scrollbar-width` isn't `auto` |
| `scrollbar-gutter: both-edges` on the viewport, a textarea or in a vertical writing mode, and the sideways scroll of a box with both gutters | the viewport, a textarea and a vertical box keep their end gutter alone, and a box's scrollport takes in its start gutter, so it scrolls sideways one gutter further, where Chrome keeps a gutter at each edge of them all and scrolls within the two |
| a right-to-left box with a stable gutter that also overflows sideways | shown from its start edge, where Chrome starts it scrolled by its gutters |
| `scrollbar-color`'s bars at rest, their track, and Chrome's classic system bars | ui draws its overlay thumbs in the first colour whether or not the box scrolls, and no track, where Chrome on macOS shows its overlay bars only while the box scrolls, its track on hover, and Chrome on Linux and Windows (whose overlay bars the comparisons turn on) its classic bars in a gutter |
| a block `::before` or `::after` in a checkbox or a radio | centred in the box with its other content, where Chrome starts it at the top |
| `counter(list-item)` in a reversed `ol` with no `start`, or on an item with a `value`, once Chrome counts again | as Chrome first counts: from 1 down, the value numbering the marker alone; Chrome counting again (its comparison harness naming each `::before`, a `@counter-style` in use, at times a font landing) resets the counter to one over the number of items and an item's value as a counter of the item's own, so a page shows either |
| a list item's inside marker at its min-content width | kept on one line with what follows it, where Chrome breaks after the marker's trailing space (a shrunk flex item of a list) |
| a counter style's text in a font of its own, and its glyphs a face lacks | the complex predefined styles write their CJK, Korean and Ethiopic characters through the face's fallbacks, which ui has none of unless `ui.SystemFonts()` finds them |

## Layout

| Missing | Today |
|---|---|
| a positioned box beside the lines of its containing block, cut by a column, their lines not lining up | cut only where none of them has a line across the column's end, where Chrome cuts each where its own content allows |
| a forced break (`break-before: column`) inside one of boxes side by side cut by columns | breaks that box alone, the others filled on as they would be; not compared with Chrome |
| a selector naming an element ui has none of inside `:not()` under `ui.BrowserSheets()` | its rule is dropped with a warning, where Chrome's matches every element |
| `align-self: baseline` in a subgrid | its items align on their own baseline, not shared with the grid round it; aligned at the start where they share the grid's rows |
| a subgrid's own `repeat(auto-fill, …)` of names beside a grid's names on its lines | the grid's names stay on its lines, where Chrome drops some of them |
| the height of a subgrid's item sizing the grid's columns, in a subgrid of columns alone | measured without the subgrid's rows: a percentage height or an aspect ratio there counts as on no row |
| an `img` whose picture is broken, as tall as its attributes' proportions make it (`width: 100%; height: auto`) | stands on its line by its bottom, where Chrome's stands by the line of its icon and alt text inside it: its line is 3 points taller than Chrome's; one with alt text stands by its bottom too, where Chrome's stands on that text's line, which ui cannot tell from a picture still loading, whose landing lays nothing out |
| a `summary`'s marker in a font without ▸ | measured with ui's fallback face for the triangle, where Chrome on macOS takes 1.06em whatever the font: about 3 points narrower at 16px |
| the presentational attributes but those [read](03-css.md): `link`, `vlink` and `alink` on `body` | read, and they do nothing |
| a `br` ending the last line a `-webkit-line-clamp` or `line-clamp` shows | no wider than before, where Chrome's is as wide as the ellipsis drawn in its place |
| `line-clamp: auto` whose height holds no line | shows the first, where Chrome shows none and is no taller than its padding |
| `line-clamp: auto` whose last line fits but the padding below it in its block doesn't | the blocks after the clamp laid out whole, where Chrome's keep the clamp it first tried, one line lower |
| `-webkit-line-clamp` and `line-clamp` on one element | `line-clamp` wins whatever their order, where Chrome reads them as one property, the last declared winning |
| past a `line-clamp`'s last line, a float inside its lines, an inline block | still drawn, where Chrome hides them |
| a viewport that scrolls under a root of vertical writing mode | the viewport doesn't scroll: the root is laid out in the viewport as one that doesn't |
| an absolutely positioned box of a vertical box whose containing block is outside it, in another frame (a box of `vertical-rl` in a page of horizontal lines) | placed as its containing block's frame had it, sideways, where Chrome places it in that block's writing mode |
| a percentage padding of an orthogonal flow | of its container's block size, where Chrome takes its inline size |
| `position: sticky` in vertical writing modes | not checked against Chrome |
| a size container query on a box of vertical lines: `width`, `height`, `inline-size` and `block-size` | read as in horizontal lines, the inline size the width |
| the rules between columns of a multi-column box of vertical lines | drawn as if its columns stood side by side |

## Backgrounds and borders

| Missing | Today |
|---|---|
| `background-clip: text` on the root or `body` painting the canvas | the canvas is painted whole, not clipped to the text |
| `-webkit-text-stroke` | refused |
| a scaled picture's tile seams, the gaps of `background-repeat: space` | each tile keeps its own edge, where Chrome blends across: about a pixel row; a gap of a fraction of a pixel is a whole pixel or none, where Chrome blends it |
| a `border-image-slice` of `100%` without `fill` | the picture drawn whole over the border box, where Chrome draws its corners alone |
| the round dots of a dotted side over three device pixels wide; the squares of one one or three pixels wide | the dots a shade darker at their smoothed edge than Chrome's; the squares laid out as measured on sides two pixels wide, unchecked at the other widths |

## Sheets

| Missing | Today |
|---|---|
| an `@import` inserted with `InsertRule` after other rules | followed, where Chrome refuses the insertion |
| an animated `border-image-slice` from a percentage to a number, `border-image-width` from a length to a percentage | interpolates the numbers and flips, where Chrome flips and makes a `calc()` |
| an animated `background-image`, `mask-image` or `border-image-source` between two pictures | flips halfway, where Chrome cross-fades them |
| `content-visibility` and `overlay`, in a transition or anywhere | not read |
| `@starting-style` of an element whose ancestor leaves `display: none` | not read: only the element's own leaving |
| the rules nested in a style rule as objects of their own (a style rule's `cssRules`, `CSSNestedDeclarations`), an `@scope`'s | read and applied, but `Sheet.Rules`, `InsertRule` and `DeleteRule` reach the top-level rules only, each with all it holds |
| an `@scope` of no start, in a `style` element's sheet | rooted at the root element, where Chrome roots it at the `style` element's parent; ui's sheets have no element |
| an `@scope` of no start nested in a style rule (`.a { @scope to (.b) { … } }`) | rooted at the root element, where Chrome roots it at the style rule's elements |
| `:scope` in an `:is()`, `:not()` or `:where()` inside a `:has()` | the element the `:has()` is on, where Chrome reads the `@scope`'s root or the root element |
| a rule nested in a rule of pseudo-elements (`.a::before { &:hover { … } }`) | matches nothing, as `:is()` of a pseudo-element; its declarations after a nested rule apply, as in Chrome |
| an element's own transition of a property it inherits, while a transition above its parent moves it | follows the moving value, where Chrome runs it on to the end value |
| `animation-composition: add` or `accumulate` on a property other than lengths, numbers, colours, `transform`, `box-shadow`, `filter` and `font-weight` (positions, sizes, grid tracks, `clip-path`…) | replaces, where Chrome adds |
| `animation-timeline`, `animation-range`, animations of `::marker` and `::placeholder`, the animation objects of a page's script | not read |
| `@font-face` | fetched with `ui.FetchFonts` ([decision 0099](../99-decisions/0099-a-pages-fonts-are-fetched-and-woff2-is-read-in-pure-go.md)), its widths, size, metrics and kerning read ([decision 0143](../99-decisions/0143-a-faces-look-and-the-faces-a-family-has-follow-its-sheets.md)); variable fonts drawn at their instance and `font-display` timed ([decision 0148](../99-decisions/0148-variable-fonts-are-read-in-pure-go-and-font-display-is-timed-on-the-documents-clock.md)); what is left: a variable font's `MVAR` (its metrics stay the default instance's), `font-named-instance`, `font-optical-sizing` (`opsz`), and `ital`/`slnt` from `font-style` are not read, nor variations of `CFF2` outlines or of the machine's fonts (`ui.SystemFonts()`); `font-variation-settings` jumps where Chrome interpolates it in transitions and animations; a page's text in its block period is checked against ember's own clock, Chrome's timing of it only at its end (which font a late file leaves); `CFF2` outlines are not read, nor a WOFF or WOFF2 packing a collection |
| a text shadow of a colour glyph | the glyph's picture cast in its own colours, where Chrome casts its shape in the shadow's colour ([decision 0172](../99-decisions/0172-colour-fonts-are-drawn-in-pure-go-as-pictures-of-their-glyphs.md)) |
| the machine's emoji font as a fallback | `ui.SystemFonts()` draws Apple Color Emoji, Segoe UI Emoji or Noto Color Emoji where a page names them, but an emoji in text of another family is not looked for there as Chrome does; Apple's pictures are not checked against Chrome, which draws them about a tenth of an em lower than their strike says |
| `COLR` variations, `palette-mix()` | a variable colour font's paints are drawn at its default instance, and `font-palette` jumps in transitions and animations where Chrome mixes palettes |
| made-up faces | a bold or italic is made up from an `@font-face` family's face as Chrome does, but not from ember's faces nor the machine's (`ui.SystemFonts()`), which draw their closest face as it is; `font-synthesis: position` is read and does nothing (`small-caps` left out draws the letters as they are, [decision 0148](../99-decisions/0148-variable-fonts-are-read-in-pure-go-and-font-display-is-timed-on-the-documents-clock.md)) |
| `font-stretch` | chooses among a family's `@font-face` faces; the machine's families are not chosen among by width, and no face is stretched |
| Windows' and a Linux desktop's preferences told as they change | asked again only when one of the program's windows gains the focus, so a change made while it keeps the focus (a hotkey, a script) waits for the next: ember speaks no D-Bus to hear the portal's `SettingChanged` and does not listen to `WM_SETTINGCHANGE`; a desktop with neither a portal, GNOME's settings, KDE's nor GTK's files says nothing ([decision 0155](../99-decisions/0155-the-system-says-its-contrast-forced-colours-and-selection-colours-read-only.md)) |
| `<meta name="color-scheme">` | not read: ui reads no page's `head`; a program reading a page puts `:where(:root) { color-scheme: … }` in a sheet of its own ([decision 0126](../99-decisions/0126-an-element-is-drawn-in-the-scheme-its-color-scheme-allows.md)) |
| the system's own system colours | Chrome's on macOS on every system, where Chrome on Windows and Linux takes `Highlight`, `SelectedItem` and their texts, and on Windows its light buttons, links and grays, from the system |
| a select's line at `line-height: normal` on Linux | its font's normal line, as Chrome on macOS makes it, where Chrome on Linux makes a select's a point shorter than the font's other lines (Go Regular at 16px: 17 points, not 18); ui lays out alike everywhere |
| a dark control's hover and press, a dark select's chevron, meter's dark fills, the scrollbars, an iframe whose scheme isn't its host's | a hovered or pressed button keeps its dark face, the chevron stays dark, a meter fills in its light colours, the bars keep ember's own look in either scheme, and an iframe's canvas is never made opaque, where Chrome draws its own dark ones and backs an iframe of the other scheme |
| a touch-only device from the system | the program's answer alone (`ui.TouchOnly`): no system is read for it ([decision 0139](../99-decisions/0139-colours-in-any-space-are-worked-out-in-srgb-as-the-cascade-reads-them.md)) |
| `forced-colors: active` | answers the query, where Chrome also forces the system's colours on the page |
| `ex`, `ch` and the font's other units in a media query's length | measured in ui's initial font (Go, 16px), where Chrome measures in its default font (Times on macOS): a bound between the two is answered otherwise |
| a corner or a centre in `-webkit-linear-gradient()`, `-webkit-gradient()`, the prefixed repeating gradients | keep their warning under `ui.BrowserSheets()`: Chrome reads them and ui cannot |
| a colour out of sRGB (`color(display-p3 1 0 0)`, a far `lab()`) | clipped to sRGB, as Chrome draws it on an sRGB display; ui draws no wider gamut |
| a `line-height` adding a percentage to a `sign()` of one (`calc(20px * sign(50% - 49px) + 150%)`) | the sum, the percentage outside `sign()` of the font size, where Chrome computes `0px` |
| `currentColor` in `color-mix()` or a relative colour, inherited | worked out on the element that declares it, where Chrome keeps the function and works it out again on each child that inherits it |
| in a `filter` element named by `url(#id)` on an element: `feTile`, the `k1` term of `feComposite`'s arithmetic, and the primitives the svg package does not draw | a clear result, `k1` left out ([0165](../99-decisions/0165-a-filter-element-runs-on-the-gpu-over-an-elements-picture-and-a-drop-cap-is-a-trimmed-float.md)) |
| a `filter` element named by `url(#id)` whose region reaches past a tenth of the element's box round it | drawn, but the element is culled by its box grown by a tenth, as the default region: scrolled almost out of view, what lies past that is not drawn |
| a `url(#id)` filter's steps in linear light | each picture between them is 8 bits a channel, so the darks band a little where Chrome's stay smooth |
| `mix-blend-mode: plus-lighter` over a backdrop seen through | added in encoded colours and drawn over the backdrop as the colour and alpha that make the sum: exact on an opaque backdrop, near it where the backdrop is translucent |
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
| a mask inside an element's `clip-path` | the mask cuts, the clip-path only by its bounds ([decision 0133](../99-decisions/0133-a-masked-element-is-a-picture-cut-by-the-picture-of-its-mask.md)) |
| `mask-image: url(#id)`, an SVG `mask` element | drawn as a picture that is not there: nothing of the element shows |
| a gradient as `-webkit-mask-box-image`'s source | refused with a warning, as `border-image-source` refuses one: only a picture's `url()` is cut in nine |
| a masked element under a rotation or a skew | hit in its border box, where Chrome reaches about half a pixel further |
| a masked element on a surface other than `twod.Canvas` | drawn unmasked, but for the one-colour icon |
| a `pattern` of a property of strings (`\p{RGI_Emoji}`), an emoji or case-mapping property (`\p{Emoji}`, `\p{Cased}`), a script's short name (`\p{sc=Grek}`) or `Script_Extensions`, or a group name used twice | constrains nothing, where Chrome reads it ([decision 0159](../99-decisions/0159-a-pattern-is-read-as-javascript-and-a-form-is-submitted-by-its-button.md)) |
| a typed email's `Value` | the address as typed, where Chrome's value holds its domain in punycode |
| a form's submission | Chrome's validation bubble is not shown, and nothing is sent: `action`, `method`, `formaction`, `target` are the program's to read |
| `ex` in a bold or italic the engine makes from another face | the x-height of the face it is made from, where Chrome measures the thickened `x` |
| `container-type: anchored` | refused with a warning |
| `@position-try` inside `@media`, `@container` or `@supports` | applies whatever the condition says |
| `anchor()`'s `start` and `end`, and `position-area`'s logical keywords, in a box of another writing mode or direction than its containing block | read the positioned box's own writing mode and direction, where Chrome reads its containing block's |
| `position-visibility: anchors-valid` | read, and hides nothing |
| an anchored box placed in a grid area of its containing block | its anchors are measured from the containing block's padding box, not the grid area |
| a box anchored to a scrolled anchor | laid out again on the frame the scroll moves its anchor: right, but a layout where Chrome moves it by the scroll |
| `interestfor`, popover `beforetoggle` that a callback cancels, `showPopover()` and `hidePopover()` as DOM events | not read: no interest invokers, and `Toggled` is told after the change |
| a popover shown under a transformed or filtered element | laid out in that element as its containing block, where Chrome's top layer lays it out in the viewport |
| a trackpad's or a smooth wheel's scroll on X11 | counted as a mouse's notches, as GLFW reads X11's core buttons only, where Chrome reads XInput2's smooth scrolling ([decision 0175](../99-decisions/0175-relative-snaps-keep-their-unsnapped-axes-and-a-held-track-pages-on.md)) |
| a high-resolution wheel's turn on Windows, of parts of a notch | moved at once by its points as a trackpad's, where Chrome glides it as a part of a notch; a precision touchpad's that Windows sends as a whole notch glides as a notch |
| a container snapping on both axes whose areas are as big as its snapport or bigger | an area bigger than the snapport lets the scroll rest anywhere it covers it, and one as big is aligned as any, where Chrome dodges the areas that intrude on it and sometimes stays put where none meets the scroll ([decision 0175](../99-decisions/0175-relative-snaps-keep-their-unsnapped-axes-and-a-held-track-pages-on.md)) |
| the arrows and Space with nothing focused on a Document without `ScrollingViewport()` | left to the game, where Chrome scrolls ([decision 0169](../99-decisions/0169-the-users-scrolls-glide-on-chromes-curves-and-two-axis-snaps-land-where-both-areas-show.md)) |
| `overscroll-behavior` on a touch's scroll, `overscroll-behavior: none`'s bounce | ui scrolls by no touch and bounces nothing: `contain` and `none` stop the chain alike |
| `scroll-state(scrollable: …)` asked of the root, for the viewport's scroll | answers none: ui answers it for scroll containers only |
| a `style()` query compared with a custom property whose value differs only in its white space (`--d: 1px  2px`) | equal, where Chrome compares the text as written |
| `calc-size()` in `width`, the `min-` and `max-` sizes, `block-size`, `inline-size` and `flex-basis`, and a `calc-size()` height of a flex or grid item, a table part or an absolutely positioned box | refused with a warning outside `height`, and a size keyword flips halfway there; such a height is laid out as its keyword ([decision 0177](../99-decisions/0177-lines-are-chosen-again-after-the-greedy-ones-and-calc-size-is-a-keyword-scaled.md)) |
| a `calc-size()` whose calculation isn't a multiple of `size` plus a length-percentage (`min(size, 100px)`) | read as the line through its values where `size` is 0 and 1 |
| a registered `<transform-list>` interpolated between lists of other functions, a `<length-percentage>` or `<transform-function>` holding `min()` or another function over a percentage | flips halfway, and the value is unset, where Chrome interpolates the matrices and keeps the function |
| a registered `<color>` in a space other than sRGB (`oklch(…)`), `<image>` and `<url>` values | computed as `rgb()`, and written as given, where Chrome keeps the colour's space and makes the url absolute |
| the white space of a custom property's value, unregistered or of syntax `*` (`--a:  a   b`) | one space where the value had several, where Chrome keeps it as written |
| an `@property` rule inside `@media` or `@supports`, and `@property` in a style attribute's sheet | registered whatever the condition says |
| animation-composition `add` and `accumulate` on a custom property | replace |
| `@container` inside `@function` | dropped with a warning |
| container queries nested more than eight containers deep, each answer moving the next | answered as far as the eighth in one layout, the rest the next frame |

## Windows

| Missing | Today |
|---|---|
| a header merged with the title bar on Windows and Linux | `platform.OwnTitleBar()` keeps the system's title bar there, and `app-region: drag` does nothing |
| `env(titlebar-area-*)` in an `iframe`, and the overlay's `geometrychange` | undefined in a framed Document; a change restyles, with no event |
| `env(viewport-segment-*)` and its indices, `env(preferred-text-scale)`, `safe-area-max-inset-*` | not defined: a fallback is taken, none with indices |
| the pointer's events for a pen or a finger, `pointerId`, `pointercancel`, `pointerover`/`out`, and `preventDefault` on a press | the mouse's buttons only, and a press always goes on to focus, click and select |
| a key's repeats told apart, and `keyup` | `OnKey` hears repeats as presses in a text field, and no release |
| the title tooltip's own delay and look per system | Chrome's on Windows and Linux everywhere (half a second, 10 and 15 points from the pointer); macOS's native tip shows later |
| a `role="menu"` shown without an opener closing on a choice | stays open: its item's callback removes `open` |

## Frames

| Missing | Today |
|---|---|
| a press on the page while a framed select's list is open | closes the list and goes on to the page, where Chrome's list, a window of the system's, takes the press on macOS ([decision 0166](../99-decisions/0166-a-page-replays-round-its-covered-live-parts-and-a-framed-list-shows-over-the-window.md)) |
| a framed select's list where the outermost Document is not a window's (drawn by a canvas element or a 2D entity) | kept inside that Document's canvas, not the window |
| a selection running from a page into its frame, and select all over both | each Document selects its own text, one at a time |
| a live part (a frame, a canvas, a caret) in an `opacity`'s, a filter's or a mask's picture | its stacking context is drawn again each frame, the page replayed round it; in a picture of the whole page (a blend mode on the root) the page is painted each frame |
| `scrolling="no"`, `allowtransparency`, `loading`, `sandbox`, `allow`, `name` | not read: what a frame may do is the program's |
| a panic of a framed Document outside its own frame: a call the program makes on it, a fetch it started | the program's own, with or without `ui.Guarded()` |
| a control taken out of its Document (a field, a canvas, an `iframe`) | its node and state stay held by the Document until the Document goes; an `iframe` lets go of what it showed |

## Pictures

| Missing | Today |
|---|---|
| an animated picture out of view under a transform, or in a box paint can't bound (`Box.Loose`), or with `EMBER_UI_CULL=off` | plays on, as paint draws it, where Chrome stands it still when none of it shows; one scrolled out of the viewport or of a scroll container stands still as in Chrome ([decision 0167](../99-decisions/0167-pictures-load-play-and-choose-again-when-a-desktop-chrome-does.md)) |
| an animation out of view for more than one play of a picture played a number of times | counts the plays it missed as one, as Chrome is assumed to; only one played for ever is compared |
| AVIF | no picture, and a warning; `web.Images` does not ask for it: no reader in pure Go ([decision 0138](../99-decisions/0138-animated-pictures-play-as-one-texture-per-address.md)) |
| a `source` of AVIF (`type="image/avif"`) | passed over, where Chrome takes it: the next source, or the `img`'s own pictures |
| `loading="lazy"` on a slow network | asked for within Chrome's margins on a fast one, 1250 points along the page and 625 across, where Chrome widens them on 3G and slower |
| `loading="lazy"` in an `iframe` | near its own Document's viewport, where Chrome measures it against the page's; an `iframe`'s own `loading` is not read |
| the order Chrome asks for pictures in | by `fetchpriority` alone, six at a time; Chrome also lets a few low ones through while the page is laid out, raises those in view and starts what its parser found first, so that only a high picture going before the rest is compared |
| the shapes inside an `svg` element | not elements of the tree: no rule of the Document reaches a `path` or a `g` (`svg path { fill: … }` matches nothing), and `fill` or `stroke` set on the `svg` by a rule is not read; presentation attributes and the element's own `style` are |
| an `svg` element with neither a size nor a `viewBox` as a flex item | keeps its 300 points where Chrome shrinks it |
| an `svg` element whose markup changes every frame | each state is one more picture the Document keeps |

## SVG pictures

| Missing | Today |
|---|---|
| `foreignObject`, animation, `vector-effect` other than `non-scaling-stroke` | not drawn, and named in a warning |
| `feConvolveMatrix` and the lighting primitives in a small picture (an icon of 48 pixels) | several levels from Chrome's on average, both working on whole pixels; a convolution's `edgeMode` where its region's edge falls between pixels reads the partly covered pixels there, a fainter edge than Chrome's |
| `kernelUnitLength` of `feConvolveMatrix` and the lighting primitives | ignored: their kernels are over the pixels the picture shows at |
| an `feConvolveMatrix` of more than 256 weights | draws nothing |
| a filter over more than 4 million pixels (2048 a side) | not run: the element is drawn unfiltered |
| a `mask` of another file, a conic gradient or another CSS image, a `clip-path` of another file, `clip-path` on the shapes inside a `clipPath` | the first two named in a warning and not applied, the last ignored |
| masks of more than one layer, `mask-size`, `mask-position`, `mask-composite`; `space` and `round` of `mask-repeat` | the first layer alone, the image the size of its box at its corner, named in a warning; repeated as `repeat` |
| a gradient mask's colours between stops of different alpha, under `mask-mode: luminance` | mixed with alpha not premultiplied, where CSS mixes them premultiplied |
| the letters of `text` | Go's faces, the monospace one for a family naming one, whatever font the file asks for (Chrome's sans-serif differs by a few levels on average); no `textLength`, `dominant-baseline`, vertical text, ligatures or shaping of complex scripts; `text-decoration` at the Go face's own place and thickness, thinner than Chrome's |
| text after a `textPath` in the same `text` | set from the path's end point |
| an `image` or an `feImage` of a file | not fetched, and named in a warning: only `data:` addresses are drawn |
| an SVG `image` sized by its own `width` and `height` with no `viewBox`, under a `preserveAspectRatio` other than the default | fitted from its own size as if it had a view box of it |
| hairline strokes, dashes along curves | differ a little from Chrome's |
| `currentColor` in an `img` or a `url()`, `image-rendering: pixelated` | black, as in Chrome, where only an `svg` element takes the page's colour; ignored |
| an SVG as a twod texture | by hand: `svg.LoadPicture`, `Rasterise`, `twod.NewTexture` |

## Not checked

Things built and tested, with a part no test against Chrome covers yet.

- `background-size: cover` and `contain` under a Document `Zoom`.
- Selecting with the pointer and the keys: Chrome cannot be driven headless, so the places a press and Shift with a key reach are tested in ember alone, a double click's word against the one Chrome's `Selection.modify` reaches; the copied text, the rectangles and the picture are compared with Chrome's.
- A selection in an inactive window: headless Chrome's page is neither focused nor not, its `::selection:window-inactive` holding in one run and not in the next, so the grey and the rules are tested in ember alone, with Chrome asked only which selectors it reads.
- The Linux and Windows thresholds of the `selection-marks` and `field-selection` pages: measured on macOS only.
- `user-select: auto` set inside `user-select: all`, read as `text`.
- `app-region`: headless Chrome has no window to drag, so drag regions are tested in ember alone; the computed values are compared with Chrome's.
- Pointer and key callbacks, `doc.Blur()`, the title tooltip and `env(titlebar-area-*)` with a title bar: headless Chrome is not driven by input and has no window controls overlay, so they are tested in ember alone; `env()` without an overlay is compared with Chrome.
- `:user-valid` and `:user-invalid`: no player changes a field in headless Chrome, so a change is tested in ember alone; Chrome is asked that nothing matches them before, and what they match once a script submits the forms. Enter's implicit submission is tested in ember alone.
- The picture `sizes="auto"` chooses: Chrome chooses it once the page is laid out, and takes a denser one already in its cache, so what it shows changes from run to run; the box is compared with Chrome's, the choice is tested in ember alone.
- Choosing a picture again as the viewport or the pixels a point change, the one shown kept until the one chosen lands: headless Chrome is neither resized nor moved to another screen, so it is tested in ember alone.
- A selection inside a transformed box other than a translation, and under a Document `Zoom`.
- A float low in a column, at narrow widths.
- A percentage `top` on a relative child inside columns.
- The largest raster of an SVG (8192 pixels a side) against the device's own limit.
- The Linux and Windows thresholds of the pixel pages added since `word-spacing`: those of `iframe`, `table-model`, `background-boxes`, `word-spacing`, `vertical-align` and `font-smoothing` are what the runners measured once, their text drawn differently there.

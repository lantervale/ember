# What is missing

What ui does not do yet, where a page written for a browser would expect it. Each line is a known gap, not a choice: what ui refuses on purpose is in the [Refused table](03-css.md). A line leaves this page when its test against Chrome passes.

## Text

| Missing | Today |
|---|---|
| a `text-shadow` on stroked text whose fill is transparent | the shadow of the glyphs filled, where Chrome's is the shadow of their stroke |
| `-webkit-text-stroke` on a colour emoji | drawn as the picture alone |
| a stroke's mitre at a sharp corner between two short curves of a glyph | the curves are stroked as straight pieces, and the mitre can reach a little further than Chrome's |
| the default selection colour on Windows and Linux, focused or not | macOS's blue everywhere, and its grey in an inactive window: the machines here run Chrome on macOS only, which can't say what the others draw |
| a focused field's caret beside its selection | drawn at the selection's moving end, where Chrome draws none while characters are selected |
| selecting by touch, dragging selected text, a rich (HTML) copy | none: the pointer and the keys select, and the clipboard gets plain text |
| text of a size between two whole ones (13.44px) | measured and placed at its size, drawn from the glyphs of the nearest whole size scaled to it: a little softer than Chrome's |
| a selected tab or carriage return in text that keeps them | the selection is drawn on the text around it, not over the tab's room |
| `word-break: break-all` between two elements (`ab<b>cd</b>`), and between a letter and punctuation other than a hyphen | no break where the element changes; elsewhere ui breaks between two letters or digits and before a hyphen, which Chrome's table may differ from round brackets and marks |
| `line-break: loose`, `normal` and `strict` | read, and lines break as `auto`: they differ only on the small kana and the punctuation of Chinese and Japanese |
| a drag's scroll sideways | a drag past an edge scrolls up and down only |
| a first letter's text | not selected or copied with the rest of its text node, as `::before`'s text is not |
| `initial-letter` | refused as an unknown property: float a `::first-letter` for a drop cap |
| `::first-line`'s `opacity`, `box-shadow`, `visibility`, `vertical-align` | read, and they change nothing |
| a first letter in text a translation changes; one after spaces `white-space` keeps | none is taken; the spaces are dropped |
| a block's min-content width under a `::first-line`, and the padding of inline elements in `em` on it | measured in the elements' own styles |
| `text-decoration-skip-ink: all` | as `auto`: line-throughs go through the ink, where `all` skips it there too |
| small capitals in a text field, a select's list and a broken picture's alt text | drawn as `normal` there |
| a face with its own small capitals (`smcp`) | synthesised from its capitals all the same, where Chrome draws the face's own ([decision 0135](../99-decisions/0135-small-capitals-and-skipped-ink-are-synthesised-from-the-faces-outlines.md)) |
| text on a display of one pixel to the point | a glyph's darkened edge shows on its left and top and, as rounding has it, not always on its right and bottom: the last half pixel of its quad has its centre on the quad's edge. Drawn every time, 27 of the 47 pixel pages with text measured further from Chrome at that scale and none nearer, so it stays; `-webkit-font-smoothing: antialiased` has no such edge |
| `-webkit-font-smoothing` on the options of an open `select` and on a picture's alt text | not read: drawn with the default smoothing |
| a `top` or `bottom` aligned inline box holding a shifted one (`sub`, a length…) | the line is sized from the boxes as the outer one aligns them; Chrome also counts the inner box where it stood before, a few points more |
| fonts a script loads (Typekit…) | ui runs no script: a sheet a browser's script adds gives its `@font-face` rules as any other, but a `FontFace` a script makes itself is not known, and the next family of the list is used |
| some installed fonts | `ui.SystemFonts()` skips a file it cannot read; nothing waits for the scan in the public API |
| a span ending a centred line | up to 3 points narrower than Chrome's |
| a control's own font, colour and line height under `ui.BrowserSheets()` | a `button`, `input`, `select` or `textarea` takes its parent's `font`, `color` and `line-height`, where Chrome's sheet starts each from its own (13.33px Arial, a textarea's monospace, `FieldText`, `normal`); so does `option` its `white-space` and `optgroup` its weight |
| a `word-spacing` or `letter-spacing` so far below zero that a run of text ends left of its start | the run's box is empty at its start, as Chrome's, but what follows on the line starts where the run's glyphs end, left of it, where Chrome starts it at the run's start; a block sized by such text can differ from Chrome's by a glyph |
| `word-spacing` and `letter-spacing` in a select's list and a broken picture's alt text | the list and the alt text are measured without them |
| `word-spacing` on the first kept space of a block that collapses its own (`<div><span style="white-space: pre"> a</span></div>`) | widened as every other kept space; Chrome leaves that one space as the font has it |
| `font-variant-ligatures`, `-numeric`, `-east-asian`, `-alternates`, `-position` on a face that has the features (`tnum`, `liga`, `sups`…) | read and kept, drawn as `normal`: ui applies no OpenType feature, where Chrome does; with the Go faces, which have none, both draw alike |
| an em space or a thin space (U+2003, U+2009) | a few points wider than Chrome's in the Go faces |
| shaping right-to-left scripts: Arabic's joined forms, ligatures | each letter drawn in its own form, where Chrome joins them |
| selecting, the caret and `text-overflow` in text shown right to left | a selection and the caret measure a right-to-left run from its left, and the ellipsis cuts a line's right end |
| mirroring the characters of right-to-left text but brackets (`<`, `>`, `«`…) | drawn as they are, where Chrome mirrors every Bidi_Mirrored character |
| the text and the arrow of a text field, a `select` or a button of direction `rtl` | laid out as left-to-right controls |

## Generated content

| Missing | Today |
|---|---|
| `::before`, `::after` on `progress`, `meter` and range inputs | no box, where Chrome lays one out among the control's own parts |
| a block `::before` or `::after` in a checkbox or a radio | centred in the box with its other content, where Chrome starts it at the top |
| `::marker` with `line-height`, `text-shadow`, `white-space`, `overflow-wrap`, `cursor`, `quotes`, transitions | read, and they change nothing |
| `counter(list-item)` in a reversed `ol` | the item's number, where Chrome's counter counts down from 0 and its marker alone shows the number |
| the complex predefined counter styles (`japanese-formal`, `simp-chinese-informal`, `korean-hangul-formal`, `ethiopic-numeric`…), `symbols()`, a picture as a counter symbol | written as `decimal`; `symbols()` and pictures refused |
| a `list-style-image` gradient, `list-style-position: inside` | refused |
| a `list-style-image` fetched off the frame (`ui.FetchImages`) | the marker is laid out once the picture is found by another layout, not when it lands |

## Layout

| Missing | Today |
|---|---|
| `column-span: all` on a block deeper in the container than its children | laid out in its column, where Chrome spans it when the blocks round it start no formatting context |
| a flex container or a table cut between rows by a column | kept whole |
| a positioned box cut by a column | kept whole |
| columns right to left | left to right only, their content laid out right to left in each |
| a table's background round a collapsed column | drawn over the table's box, where Chrome draws it as wide as before the column was taken out |
| a scroller of direction `rtl` | scrolls from its left, where Chrome starts it at its right end; its bar is on its left, as Chrome's |
| collapsed borders of a table of direction `rtl` | each cell's left border still wins its left edge, where Chrome resolves its start edge first |
| logical properties in `@keyframes` and `transition-property` under `direction: rtl` | the left side's |
| a float of direction `rtl` beside a block that avoids floats, a block's own `text-align` of `-webkit-left` under rtl | not compared with Chrome |
| `visibility: collapse` on a row or a column, `empty-cells: hide` | the row or the column keeps its room; an empty cell keeps its borders and background |
| a table's collapsed borders where they meet at a corner, borders of different widths, or as wide in a style of two shades (`inset`, `outset`, `groove`, `ridge`) | the wider way goes through and the other stops beside it, as wide the one across; Chrome's joins differ by a few pixels there, an edge down going through at the table's right corners and between cells |
| a selector naming an element ui has none of inside `:not()` under `ui.BrowserSheets()` | its rule is dropped with a warning, where Chrome's matches every element |
| `align-self: baseline` in a subgrid | its items align on their own baseline, not shared with the grid round it; aligned at the start where they share the grid's rows |
| a subgrid's own `repeat(auto-fill, …)` of names beside a grid's names on its lines | the grid's names stay on its lines, where Chrome drops some of them |
| the height of a subgrid's item sizing the grid's columns, in a subgrid of columns alone | measured without the subgrid's rows: a percentage height or an aspect ratio there counts as on no row |
| `opacity` on an inline element holding blocks | its lines fade, the blocks inside it do not |
| an absolutely positioned box inside an inline element `position: relative` moves | placed where the element stood before it moved |
| an `img` whose picture is broken, as tall as its attributes' proportions make it (`width: 100%; height: auto`) | stands on its line by its bottom, where Chrome's stands by the line of its icon and alt text inside it: its line is 3 points taller than Chrome's |
| a `summary`'s marker in a font without ▸ | measured with ui's fallback face for the triangle, where Chrome on macOS takes 1.06em whatever the font: about 3 points narrower at 16px |
| a `font` element's attributes | read, and they do nothing |
| the presentational attributes but those [read](03-css.md): `border`, `rules` and `frame` on a `table`, `align="left"` and `"right"` on a `table` or an `img`, `clear` on a `br`, `hspace`, `vspace`, `background`, `link`, `vlink`, `size` and `noshade` on an `hr`, `align` and `valign` on a `col` or a `colgroup` | read, and they do nothing |
| an absolutely positioned element right in a block after its text (`<div>text<span style="position: absolute">`) | at the start of the line, where Chrome puts it after the text; inside an inline element it stands after the text |

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
| an animated `border-image-slice` from a percentage to a number, `border-image-width` from a length to a percentage | interpolates the numbers and flips, where Chrome flips and makes a `calc()` |
| an animated `background-image`, `mask-image` or `border-image-source` between two pictures | flips halfway, where Chrome cross-fades them |
| `content-visibility` and `overlay`, in a transition or anywhere | not read |
| `@starting-style` of an element whose ancestor leaves `display: none`, or nested in a style rule | not read: only the element's own leaving, and top-level rules |
| an element's own transition of a property it inherits, while a transition above its parent moves it | follows the moving value, where Chrome runs it on to the end value |
| `animation-composition: add` or `accumulate` on a property other than lengths, numbers, colours, `transform`, `box-shadow`, `filter` and `font-weight` (positions, sizes, grid tracks, `clip-path`…) | replaces, where Chrome adds |
| `animation-timeline`, `animation-range`, animations of `::marker` and `::placeholder`, the animation objects of a page's script | not read |
| `@font-face` | fetched with `ui.FetchFonts` ([decision 0099](../99-decisions/0099-a-pages-fonts-are-fetched-and-woff2-is-read-in-pure-go.md)), its widths, size, metrics and kerning read ([decision 0143](../99-decisions/0143-a-faces-look-and-the-faces-a-family-has-follow-its-sheets.md)); what is left: `font-display` is always `swap` (no block period, a font landing late still swaps in); of `font-feature-settings` only `"kern" 0` does something (`liga`, `tnum`… need a GSUB reader); `font-variation-settings`, `font-named-instance` and a variable font's instances are read and do nothing (x/image's sfnt reads no `fvar`/`gvar`), a variable font drawing its default instance; colour fonts, collections and `CFF2` are not read |
| made-up faces | a bold or italic is made up from an `@font-face` family's face as Chrome does, but not from ember's faces nor the machine's (`ui.SystemFonts()`), which draw their closest face as it is; `font-synthesis: small-caps` and `position` are read and do nothing |
| `font-stretch` | chooses among a family's `@font-face` faces; the machine's families are not chosen among by width, and no face is stretched |
| `prefers-color-scheme` and `prefers-reduced-motion` on Linux and the web | no preference: neither says its scheme or its reduced motion yet, so a Document given `ui.Preferences` answers light and no preference there ([decision 0121](../99-decisions/0121-a-document-answers-media-features-from-the-preferences-it-is-given.md)) |
| `<meta name="color-scheme">` | not read: ui reads no page's `head`; a program reading a page puts `:where(:root) { color-scheme: … }` in a sheet of its own ([decision 0126](../99-decisions/0126-an-element-is-drawn-in-the-scheme-its-color-scheme-allows.md)) |
| the system's own system colours | Chrome's on macOS on every system, where Chrome on Windows and Linux takes `Highlight`, `SelectedItem` and their texts, and on Windows its light buttons, links and grays, from the system |
| a dark control's hover and press, a dark select's chevron, meter's dark fills, the scrollbars, an iframe whose scheme isn't its host's | a hovered or pressed button keeps its dark face, the chevron stays dark, a meter fills in its light colours, the bars keep ember's own look in either scheme, and an iframe's canvas is never made opaque, where Chrome draws its own dark ones and backs an iframe of the other scheme |
| `prefers-contrast`, `forced-colors`, a touch-only device from the system | the program's answers alone (`ui.PrefersContrast`, `ui.ForcedColors`, `ui.TouchOnly`): no system is read for them ([decision 0139](../99-decisions/0139-colours-in-any-space-are-worked-out-in-srgb-as-the-cascade-reads-them.md)) |
| `forced-colors: active` | answers the query, where Chrome also forces the system's colours on the page |
| `ex`, `ch` and the font's other units in a media query's length | measured in ui's initial font (Go, 16px), where Chrome measures in its default font (Times on macOS): a bound between the two is answered otherwise |
| a corner or a centre in `-webkit-linear-gradient()`, `-webkit-gradient()`, the prefixed repeating gradients, `display: -webkit-box` and `-webkit-line-clamp`, `-webkit-fill-available` | keep their warning under `ui.BrowserSheets()`: Chrome reads them and ui cannot |
| a colour out of sRGB (`color(display-p3 1 0 0)`, a far `lab()`) | clipped to sRGB, as Chrome draws it on an sRGB display; ui draws no wider gamut |
| a `line-height` adding a percentage to a `sign()` of one (`calc(20px * sign(50% - 49px) + 150%)`) | the sum, the percentage outside `sign()` of the font size, where Chrome computes `0px` |
| `currentColor` in `color-mix()` or a relative colour, inherited | worked out on the element that declares it, where Chrome keeps the function and works it out again on each child that inherits it |
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
| a mask inside an element's `clip-path` | the mask cuts, the clip-path only by its bounds ([decision 0133](../99-decisions/0133-a-masked-element-is-a-picture-cut-by-the-picture-of-its-mask.md)) |
| `mask-image: url(#id)`, an SVG `mask` element | drawn as a picture that is not there: nothing of the element shows |
| `-webkit-mask-position` and `-webkit-mask` with three-value positions, `-webkit-mask-position-x` and `-y`, `-webkit-mask-composite`'s other words (`destination-in`, `copy`, `plus-lighter`…), `-webkit-mask-box-image` | refused with a warning; `-webkit-mask: … no-clip`, which Chrome refuses, is read |
| `mask-composite` serialised after `-webkit-mask-composite` | as the plain word (`exclude`), where Chrome keeps the legacy one (`xor`) |
| a masked element under a rotation or a skew | hit in its border box, where Chrome reaches about half a pixel further |
| a masked element on a surface other than `twod.Canvas` | drawn unmasked, but for the one-colour icon |
| a `:has()` reading, through an `:is()` or a `:not()` in it, an element above the one it is on (`.card:has(:is(.dark img))`, `.dark` on `body`) | matched, but a change of that element above does not restyle the card |
| `:dir()` on an element of `dir="auto"` whose text changes below it (a field typed in excepted) | answered from the text as it stood when the element was last restyled |
| a `pattern` Go's `regexp` cannot read (a lookahead, a backreference) | constrains nothing, where Chrome reads it as JavaScript does |
| an email address whose domain is not in ASCII letters | invalid, where Chrome turns the domain to punycode first |
| `:user-invalid` after a form is submitted | ui submits no form: only the fields the player changed or submitted match |
| a radio group by its form | grouped by name across the Document, where Chrome groups by name within each form |
| `ex` in a bold or italic the engine makes from another face | the x-height of the face it is made from, where Chrome measures the thickened `x` |
| `scroll-state()` container queries and `container-type: scroll-state` | refused, with a warning: the condition never holds |
| a `style()` query compared with a custom property whose value differs only in its white space (`--d: 1px  2px`) | equal, where Chrome compares the text as written |
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
| a select's list of a Document in an `iframe`, where it passes the frame's box | cut at the box, where Chrome shows its list whole over the page; a menu, a tooltip or a dialog are cut there as in Chrome, the top layer being each Document's own |
| a selection running from a page into its frame, and select all over both | each Document selects its own text, one at a time |
| an element lying over an `iframe`, or an `opacity` below 1 on it or round it | the page is painted each frame while it does, where it replays with nothing over the frame: the framed Document is drawn in the page's paint, which a replay skips |
| `aspect-ratio` on a flex item `iframe` with neither a width nor a height | no width, where Chrome gives it its container's |
| `scrolling="no"`, `allowtransparency`, `loading`, `sandbox`, `allow`, `name` | not read: what a frame may do is the program's |
| a panic of a framed Document outside its own frame: a call the program makes on it, a fetch it started | the program's own, with or without `ui.Guarded()` |
| a control taken out of its Document (a field, a canvas, an `iframe`) | its node and state stay held by the Document until the Document goes; an `iframe` lets go of what it showed |

## Pictures

| Missing | Today |
|---|---|
| an animated picture off-screen, or in no element shown | plays on, where Chrome pauses it; it begins when it lands, where Chrome begins it when first painted ([decision 0138](../99-decisions/0138-animated-pictures-play-as-one-texture-per-address.md)) |
| an animation of more than 2^25 pixels in all its frames | plays the frames before that, where Chrome decodes them as they show |
| a GIF frame reaching past the picture's size | no picture, and a warning (`gif: frame bounds larger than image bounds`), where Chrome cuts the frame |
| AVIF | no picture, and a warning; `web.Images` does not ask for it: no reader in pure Go ([decision 0138](../99-decisions/0138-animated-pictures-play-as-one-texture-per-address.md)) |
| a `source` of AVIF (`type="image/avif"`) | passed over, where Chrome takes it: the next source, or the `img`'s own pictures |
| `loading="lazy"` | read only to let `sizes` say `auto`: every picture is asked for when it is laid out, where a desktop Chrome waits for it to near the viewport; headless Chrome, the reference, asks for them all before the load event, so no margin can be checked ([decision 0138](../99-decisions/0138-animated-pictures-play-as-one-texture-per-address.md)) |
| the shapes inside an `svg` element | not elements of the tree: no rule of the Document reaches a `path` or a `g` (`svg path { fill: … }` matches nothing), and `fill` or `stroke` set on the `svg` by a rule is not read; presentation attributes and the element's own `style` are |
| an `svg` element with neither a size nor a `viewBox` as a flex item | keeps its 300 points where Chrome shrinks it |
| an `svg` element whose markup changes every frame | each state is one more picture the Document keeps |

## SVG pictures

| Missing | Today |
|---|---|
| `filter`, `foreignObject`, `textPath`, `mix-blend-mode`, `vector-effect`, animation | not drawn, and named in a warning |
| a `clip-path` or a `mask` of a basic shape or another file, `clip-path` on the shapes inside a `clipPath` | the first two named in a warning and not applied, the last ignored |
| the letters of `text` | Go's faces, the monospace one for a family naming one, whatever font the file asks for (Chrome's sans-serif differs by a few levels on average); no `letter-spacing`, `word-spacing`, `textLength`, `text-decoration`, `dominant-baseline`, vertical text, ligatures or shaping of complex scripts |
| an `image` of a file | not fetched, and named in a warning: only `data:` addresses are drawn |
| an SVG `image` sized by its own `width` and `height` with no `viewBox`, under a `preserveAspectRatio` other than the default | fitted from its own size as if it had a view box of it |
| a UTF-16 file | rejected |
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
- `:user-valid` and `:user-invalid`: no player changes a field in headless Chrome, so they are tested in ember alone; Chrome is asked only that nothing matches them before.
- The picture `sizes="auto"` chooses: Chrome chooses it once the page is laid out, and takes a denser one already in its cache, so what it shows changes from run to run; the box is compared with Chrome's, the choice is tested in ember alone.
- Choosing a picture again as the viewport or the pixels a point change, the one shown kept until the one chosen lands: headless Chrome is neither resized nor moved to another screen, so it is tested in ember alone.
- A selection inside a transformed box other than a translation, and under a Document `Zoom`.
- A float low in a column, at narrow widths.
- A percentage `top` on a relative child inside columns.
- The largest raster of an SVG (8192 pixels a side) against the device's own limit.
- The Linux and Windows thresholds of the pixel pages added since `word-spacing`: those of `iframe`, `table-model`, `background-boxes`, `word-spacing`, `vertical-align` and `font-smoothing` are what the runners measured once, their text drawn differently there.
- The `mitres` page on the macOS and Linux runners, where Chrome's screenshot has the page's background and none of its boxes: it is compared on a desktop Mac and on the Windows runner only.

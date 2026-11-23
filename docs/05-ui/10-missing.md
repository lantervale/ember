# What is missing

What ui does not do yet, where a page written for a browser would expect it. Each line is a known gap, not a choice: what ui refuses on purpose is in the [Refused table](03-css.md). A line leaves this page when its test against Chrome passes.

## Text

| Missing | Today |
|---|---|
| a `text-shadow` on stroked text whose fill is transparent | the shadow of the glyphs filled, where Chrome's is the shadow of their stroke |
| `-webkit-text-stroke` on a colour emoji | drawn as the picture alone |
| a stroke's mitre at a sharp corner between two short curves of a glyph | the curves are stroked as straight pieces, and the mitre can reach a little further than Chrome's |
| the default selection colour on Windows and Linux, focused or not | the system's colours under `ui.Preferences` (Windows' `Highlight` and `HighlightText`, KDE's selection colours), blended with white as Chrome on macOS blends its own: the machines here run Chrome on macOS only, which can't say what the others draw; GNOME's accent colour is not read, an inactive window's selection there is Chrome's macOS grey, and without `ui.Preferences` macOS's blue everywhere ([decision 0155](../99-decisions/0155-the-system-says-its-contrast-forced-colours-and-selection-colours-read-only.md)) |
| selecting by touch, dragging selected text | none: the pointer and the keys select, and a press inside the selection starts a new one where Chrome drags the text |
| a rich copy's look, and one on a Linux desktop or the web | the HTML holds the elements and their attributes, where Chrome writes each element's computed style inline; a Linux desktop and the web get the plain text alone, GLFW offering no `text/html` target ([decision 0162](../99-decisions/0162-a-copy-puts-the-selections-markup-beside-its-text.md)) |
| text of a size between two whole ones (13.44px) | measured and placed at its size, drawn from the glyphs of the nearest whole size scaled to it: a little softer than Chrome's |
| a carriage return in text that keeps it (`white-space: pre`) | laid out as a character of its line, the text after it on the same line, where Chrome breaks the line there |
| `word-break: break-all` between two elements (`ab<b>cd</b>`), and between a letter and punctuation other than a hyphen | no break where the element changes; elsewhere ui breaks between two letters or digits and before a hyphen, which Chrome's table may differ from round brackets and marks |
| `line-break: loose`, `normal` and `strict` | read, and lines break as `auto`: they differ only on the small kana and the punctuation of Chinese and Japanese |
| a drag's scroll sideways | a drag past an edge scrolls up and down only |
| a press or a drag on a `::first-letter`, and a text node that is its first letter alone | the letter is no place the pointer reaches: a press there takes the nearest character of the text beside it; a node of one letter is neither selected nor copied |
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
| `optgroup` | ui has no such element: an option's group label, which Chrome draws in bold, is the page's own element |
| a `word-spacing` or `letter-spacing` so far below zero that a run of text ends left of its start | the run's box is empty at its start, as Chrome's, but what follows on the line starts where the run's glyphs end, left of it, where Chrome starts it at the run's start; a block sized by such text can differ from Chrome's by a glyph |
| `word-spacing` and `letter-spacing` in a select's list and a broken picture's alt text | the list and the alt text are measured without them |
| `word-spacing` on the first kept space of a block that collapses its own (`<div><span style="white-space: pre"> a</span></div>`) | widened as every other kept space; Chrome leaves that one space as the font has it |
| an em space or a thin space (U+2003, U+2009) | a few points wider than Chrome's in the Go faces |
| Arabic in a face without `init`, `medi` and `fina` (only the presentation forms of its cmap) | each letter in its own form, where Chrome's HarfBuzz joins it with those forms; Mongolian's variation selectors and `stch` too |
| Indic, Khmer, Myanmar and the USE scripts' reordering, a mark placed by its combining class in a face whose GPOS does not place it, a precomposed character a face lacks drawn from its decomposition | left out ([decision 0157](../99-decisions/0157-arabic-is-joined-and-marks-attached-by-harfbuzzs-stages-and-masks.md) shapes Arabic, Hebrew and the joining scripts); a character the face lacks breaks the run, so letters on either side of it are not joined |
| a face whose only layout tables are Apple's `morx` and `kerx` | drawn without its ligatures and kerning, where Chrome's HarfBuzz reads them; most of the Mac's faces have GSUB and GPOS too |
| a variable font's GPOS values at an instance (their device and variation deltas) and its GSUB feature variations (`rvrn` by axis) | the default instance's kerning and substitutions, the glyphs' advances the instance's |
| ligatures and kerning across the edge of two inline elements of one font (`A<span>V</span>`), and a ligature or a contextual substitution across a space | shaped apart, where Chrome shapes the run whole; a space's pair kerning with the words beside it is Chrome's |

## Generated content

| Missing | Today |
|---|---|
| `::before`, `::after` on `progress`, `meter` and range inputs | no box, where Chrome lays one out among the control's own parts: in a progress or a meter on lines above and below its bar, overflowing the box, the last line giving the inline-block its baseline; in a range input at the start and the end of a row with the track between them |
| `::-webkit-slider-runnable-track` and `::-webkit-slider-thumb`, `::-webkit-progress-bar` and `::-webkit-progress-value`, `::-webkit-meter-bar` and its values (`-optimum-value`, `-suboptimum-value`, `-even-less-good-value`) | refused, or under `ui.BrowserSheets()` matching nothing: ui draws a range, a progress and a meter its own way. Chrome lays each part out as a box: a range of `appearance: none` centres its track across the input, the thumb a box in it at the value along the track less its own width, placed down by its `margin-top`, the input as tall as the thumb when its height is auto; a progress of `appearance: none` shows its bar part (grey) and its value part (green) unless styled; a meter draws its styled parts whatever its `appearance` |
| `appearance: none` on a control | read, changes nothing: ui's controls draw their marks always. Chrome then draws a checkbox or a radio as an empty box, 0 by 0 unless sized and with no tick; a range without its track, its thumb kept; a progress with its default parts |
| `accent-color` on a checkbox, a radio and a range's thumb | read for a range's fill and a progress's value only. Chrome fills a checked checkbox, a radio's ring and dot and a range's thumb in it too, the tick white or dark as the accent is dark or light |
| a viewport's styled bars, the scrollbar parts' `:hover`, `:active` and `:window-inactive`, `::-webkit-scrollbar-button`, `::-webkit-resizer`, styled parts under `scrollbar-width: thin` | ui's own bars on a scrolling viewport, the parts drawn as at rest, no buttons and no resizer, and the parts drawn under `thin`, where Chrome draws the root's styled bars on the viewport, its hovered and pressed parts, buttons sized by their rules, and its system bars once `scrollbar-width` isn't `auto` |
| `scrollbar-gutter: both-edges`, the content under an empty stable gutter | read: the gutter is kept at the end only, and nothing is drawn in it, where Chrome keeps one at each edge and draws the content under the gutter while no bar shows |
| `scrollbar-color` | ui's own thumbs in its first colour on gutters of its second, where Chrome draws its system bars in them, overlay ones on macOS |
| a block `::before` or `::after` in a checkbox or a radio | centred in the box with its other content, where Chrome starts it at the top |
| `::marker` with `cursor`, transitions | read, and they change nothing |
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
| a table with a collapsed column as a grid item, or as a flex item with a width of its own | laid out at the width its container gives it, where Chrome narrows the grid item's by the column and sizes the flex item's apart; a flex item of an automatic width keeps it, as Chrome |
| a multi-column scroller of direction `rtl` whose columns overflow | its columns past its left are not reached by scrolling |
| `visibility: collapse` on a row or a column, `empty-cells: hide` | the row or the column keeps its room; an empty cell keeps its borders and background |
| a selector naming an element ui has none of inside `:not()` under `ui.BrowserSheets()` | its rule is dropped with a warning, where Chrome's matches every element |
| `align-self: baseline` in a subgrid | its items align on their own baseline, not shared with the grid round it; aligned at the start where they share the grid's rows |
| a subgrid's own `repeat(auto-fill, …)` of names beside a grid's names on its lines | the grid's names stay on its lines, where Chrome drops some of them |
| the height of a subgrid's item sizing the grid's columns, in a subgrid of columns alone | measured without the subgrid's rows: a percentage height or an aspect ratio there counts as on no row |
| `opacity` on an inline element holding blocks | its lines fade, the blocks inside it do not |
| an `img` whose picture is broken, as tall as its attributes' proportions make it (`width: 100%; height: auto`) | stands on its line by its bottom, where Chrome's stands by the line of its icon and alt text inside it: its line is 3 points taller than Chrome's; one with alt text stands by its bottom too, where Chrome's stands on that text's line, which ui cannot tell from a picture still loading, whose landing lays nothing out |
| a `summary`'s marker in a font without ▸ | measured with ui's fallback face for the triangle, where Chrome on macOS takes 1.06em whatever the font: about 3 points narrower at 16px |
| the presentational attributes but those [read](03-css.md): `link`, `vlink` and `alink` on `body` | read, and they do nothing |
| a `br` ending the last line a `-webkit-line-clamp` or `line-clamp` shows | no wider than before, where Chrome's is as wide as the ellipsis drawn in its place |
| `line-clamp`'s longhands (`max-lines`, `block-ellipsis`, `continue`) and its values but a number and `none` | refused; past its last line the inline elements' backgrounds and borders, the floats and the positioned boxes are still drawn, and what it doesn't draw still takes the pointer |
| `stretch` in `min-width`, `max-width`, `min-height`, `max-height`, on a flex or a grid item, a table cell or a column | read, and taken as `auto` (`none` for a maximum) |

## Backgrounds and borders

| Missing | Today |
|---|---|
| `background-clip: text` on the root or `body` painting the canvas | the canvas is painted whole, not clipped to the text |
| `-webkit-text-stroke` | refused |
| a scaled picture's tile seams, the gaps of `background-repeat: space` | each tile keeps its own edge, where Chrome blends across: about a pixel row; a gap of a fraction of a pixel is a whole pixel or none, where Chrome blends it |
| a `border-image-slice` of `100%` without `fill` | the picture drawn whole over the border box, where Chrome draws its corners alone |

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
| `@font-face` | fetched with `ui.FetchFonts` ([decision 0099](../99-decisions/0099-a-pages-fonts-are-fetched-and-woff2-is-read-in-pure-go.md)), its widths, size, metrics and kerning read ([decision 0143](../99-decisions/0143-a-faces-look-and-the-faces-a-family-has-follow-its-sheets.md)); variable fonts drawn at their instance and `font-display` timed ([decision 0148](../99-decisions/0148-variable-fonts-are-read-in-pure-go-and-font-display-is-timed-on-the-documents-clock.md)); what is left: a variable font's `MVAR` (its metrics stay the default instance's), `font-named-instance`, `font-optical-sizing` (`opsz`), and `ital`/`slnt` from `font-style` are not read, nor variations of `CFF2` outlines or of the machine's fonts (`ui.SystemFonts()`); `font-variation-settings` jumps where Chrome interpolates it in transitions and animations; a page's text in its block period is checked against ember's own clock, Chrome's timing of it only at its end (which font a late file leaves); colour fonts, collections and `CFF2` are not read |
| made-up faces | a bold or italic is made up from an `@font-face` family's face as Chrome does, but not from ember's faces nor the machine's (`ui.SystemFonts()`), which draw their closest face as it is; `font-synthesis: position` is read and does nothing (`small-caps` left out draws the letters as they are, [decision 0148](../99-decisions/0148-variable-fonts-are-read-in-pure-go-and-font-display-is-timed-on-the-documents-clock.md)) |
| `font-stretch` | chooses among a family's `@font-face` faces; the machine's families are not chosen among by width, and no face is stretched |
| Windows' and a Linux desktop's preferences told as they change | asked again only when one of the program's windows gains the focus, so a change made while it keeps the focus (a hotkey, a script) waits for the next: ember speaks no D-Bus to hear the portal's `SettingChanged` and does not listen to `WM_SETTINGCHANGE`; a desktop with neither a portal, GNOME's settings, KDE's nor GTK's files says nothing ([decision 0155](../99-decisions/0155-the-system-says-its-contrast-forced-colours-and-selection-colours-read-only.md)) |
| `<meta name="color-scheme">` | not read: ui reads no page's `head`; a program reading a page puts `:where(:root) { color-scheme: … }` in a sheet of its own ([decision 0126](../99-decisions/0126-an-element-is-drawn-in-the-scheme-its-color-scheme-allows.md)) |
| the system's own system colours | Chrome's on macOS on every system, where Chrome on Windows and Linux takes `Highlight`, `SelectedItem` and their texts, and on Windows its light buttons, links and grays, from the system |
| a dark control's hover and press, a dark select's chevron, meter's dark fills, the scrollbars, an iframe whose scheme isn't its host's | a hovered or pressed button keeps its dark face, the chevron stays dark, a meter fills in its light colours, the bars keep ember's own look in either scheme, and an iframe's canvas is never made opaque, where Chrome draws its own dark ones and backs an iframe of the other scheme |
| a touch-only device from the system | the program's answer alone (`ui.TouchOnly`): no system is read for it ([decision 0139](../99-decisions/0139-colours-in-any-space-are-worked-out-in-srgb-as-the-cascade-reads-them.md)) |
| `forced-colors: active` | answers the query, where Chrome also forces the system's colours on the page |
| `ex`, `ch` and the font's other units in a media query's length | measured in ui's initial font (Go, 16px), where Chrome measures in its default font (Times on macOS): a bound between the two is answered otherwise |
| a corner or a centre in `-webkit-linear-gradient()`, `-webkit-gradient()`, the prefixed repeating gradients | keep their warning under `ui.BrowserSheets()`: Chrome reads them and ui cannot |
| `-webkit-box-pack`, `-webkit-box-align`, `-webkit-box-flex`, `-webkit-box-direction`, `-webkit-box-ordinal-group` | keep their warning: a `-webkit-box` that clamps no lines is laid out as `flex` along its orient, its inline children each an item, where Chrome's old flexbox keeps them on one line |
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
| `foreignObject`, animation, `vector-effect` other than `non-scaling-stroke` | not drawn, and named in a warning |
| `feTurbulence`, `feImage`, `feConvolveMatrix`, `feDisplacementMap`, `feDiffuseLighting`, `feSpecularLighting` | a clear result, and named in a warning |
| `BackgroundImage`, `BackgroundAlpha`, `FillPaint`, `StrokePaint` as a primitive's input; `edgeMode` of `feGaussianBlur` | read as clear and named in a warning; ignored, the edges clear |
| a filter over more than 4 million pixels (2048 a side) | not run: the element is drawn unfiltered |
| a filter of an element under a rotation or a skew | blurred, eroded and dilated along the pixels' axes, where Chrome filters in the element's own space |
| a filter list with a CSS function after a `url()` (`filter: url(#soft) drop-shadow(…)`) | the function reads the filter element's result cut to its region, where Chrome's shadow is fainter |
| a `mask` of a basic shape, a gradient or another file, a `clip-path` of another file, `clip-path` on the shapes inside a `clipPath` | the first two named in a warning and not applied, the last ignored |
| corners of their own in a basic shape's `round` (`inset(0 round 2px 8px)`) | every corner the first radius |
| the letters of `text` | Go's faces, the monospace one for a family naming one, whatever font the file asks for (Chrome's sans-serif differs by a few levels on average); no `textLength`, `dominant-baseline`, vertical text, ligatures or shaping of complex scripts; `text-decoration` at the Go face's own place and thickness, thinner than Chrome's |
| `textPath`'s `method`, `spacing`, `side` and the path's `pathLength`; text after a `textPath` in the same `text` | ignored; set from the path's end point |
| an `image` of a file | not fetched, and named in a warning: only `data:` addresses are drawn |
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
- `:user-valid` and `:user-invalid`: no player changes a field in headless Chrome, so they are tested in ember alone; Chrome is asked only that nothing matches them before.
- The picture `sizes="auto"` chooses: Chrome chooses it once the page is laid out, and takes a denser one already in its cache, so what it shows changes from run to run; the box is compared with Chrome's, the choice is tested in ember alone.
- Choosing a picture again as the viewport or the pixels a point change, the one shown kept until the one chosen lands: headless Chrome is neither resized nor moved to another screen, so it is tested in ember alone.
- A selection inside a transformed box other than a translation, and under a Document `Zoom`.
- A float low in a column, at narrow widths.
- A percentage `top` on a relative child inside columns.
- The largest raster of an SVG (8192 pixels a side) against the device's own limit.
- The Linux and Windows thresholds of the pixel pages added since `word-spacing`: those of `iframe`, `table-model`, `background-boxes`, `word-spacing`, `vertical-align` and `font-smoothing` are what the runners measured once, their text drawn differently there.
- The `mitres` page on the macOS and Linux runners, where Chrome's screenshot has the page's background and none of its boxes: it is compared on a desktop Mac and on the Windows runner only.

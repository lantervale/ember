package css

import _ "embed"

//go:embed images/select-arrow.png
var selectArrow []byte

// DefaultImages are the images Default names, by url: "ember:select-arrow", the select's
// chevron (a 10×6 PNG).
func DefaultImages() map[string][]byte {
	return map[string][]byte{"ember:select-arrow": selectArrow}
}

// BrowserDefault follows Default in the user-agent sheet of a Document
// whose sheets are read as a browser's: a width is the content's, as in
// a browser's own sheet, but for the controls Chrome sizes by their
// border box (TestChromeSizesTheSameBoxesByTheirBorder), and the page's
// text is selected as a browser's is, where a game's screen takes no
// selection but where it asks for one (decision 0090). A field's text
// starts at its start whatever its parent's text-align, as Chrome's
// (TestChromeAlignsTheTextOfControlsAsABrowsersSheetDoes), where a
// game's takes its parent's; so do a control's letter and word spacing,
// its case and its text shadow start from none
// (TestChromeSpacesTheTextOfControlsAsABrowsersSheetDoes). A control's
// text is its scheme's, as Chrome's, where a game's takes its parent's
// colour, and its font Chrome's small control font, 13.33px Arial (a
// textarea's monospace) at a normal line height, where a game's takes
// its parent's (TestChromeFontsTheControlsAsABrowsersSheetDoes); a
// text field takes its parent's white-space, a button's or a select's
// keeps it as written, and an option's on one line.
const BrowserDefault = `* { box-sizing: content-box; }
button, select, table, progress, meter, input[type="checkbox"], input[type="radio"], input[type="button"],
input[type="submit"], input[type="reset"], input[type="search"], input[type="color"] { box-sizing: border-box; }
input, select, textarea { color: FieldText; }
button, input[type="button"], input[type="submit"], input[type="reset"], input[type="color"] { color: ButtonText; }
html { user-select: auto; }
input, select, textarea { text-align: start; }
input[type="button"], input[type="submit"], input[type="reset"] { text-align: center; }
button, input, select, textarea {
  letter-spacing: normal; word-spacing: normal; text-transform: none; text-shadow: none;
}
button, input, select, textarea { font: 13.333333px Arial; }
textarea { font-family: monospace; }
input { white-space: inherit; }
input[type="button"], input[type="submit"], input[type="reset"], select { white-space: pre; }
select[multiple], option { white-space: nowrap; }
`

// Default is the user-agent sheet: what every element looks like before
// the game's sheets. Export writes the same text into every page, so
// Chrome starts from it too.
const Default = `* { box-sizing: border-box; }
html { font-family: "Go", sans-serif; font-size: 16px; line-height: normal; color: CanvasText; }
html, body, div, section, header, footer, nav, main, aside, article, p, h1, h2, h3, h4,
details, summary, dialog, hr, pre, ul, ol, menu, dl, dt, dd, h5, h6, blockquote, figure, figcaption, form, address,
hgroup, search, center, fieldset, legend { display: block; }
li { display: list-item; }
[hidden] { display: none; }
[dir]:dir(ltr), bdi:dir(ltr) { direction: ltr; }
[dir]:dir(rtl), bdi:dir(rtl) { direction: rtl; }
address, blockquote, center, div, figure, figcaption, footer, form, header, hr, legend, main, p, pre, search,
article, aside, nav, section, hgroup, h1, h2, h3, h4, h5, h6, summary, ul, ol, menu, li, dl, dt, dd, table, caption,
colgroup, col, thead, tbody, tfoot, tr, td, th, [dir], bdi { unicode-bidi: isolate; }
bdo, bdo[dir] { unicode-bidi: isolate-override; }
textarea[dir="auto" i], pre[dir="auto" i] { unicode-bidi: plaintext; }
input[type="hidden" i] { display: none !important; }
table { display: table; border-spacing: 2px; }
thead { display: table-header-group; }
tbody { display: table-row-group; }
tfoot { display: table-footer-group; }
tr { display: table-row; }
td, th { display: table-cell; }
caption { display: table-caption; }
col { display: table-column; }
colgroup { display: table-column-group; }
caption { text-align: center; }
html { margin: 0; user-select: none; }
body { margin: 8px; }
p { margin: 1em 0; }
h1 { margin: 0.67em 0; }
h2 { margin: 0.83em 0; }
h3 { margin: 1em 0; }
h4 { margin: 1.33em 0; }
h5 { margin: 1.67em 0; }
h6 { margin: 2.33em 0; }
ul, ol, menu { margin: 1em 0; padding-inline-start: 40px; }
ul, menu { list-style-type: disc; }
ol { list-style-type: decimal; }
ul ul, ul menu, ol ul, ol menu, menu ul, menu menu { list-style-type: circle; }
ul ul ul, ul ul menu, ul ol ul, ul ol menu, ul menu ul, ul menu menu, ol ul ul, ol ul menu, ol ol ul, ol ol menu, ol
menu ul, ol menu menu, menu ul ul, menu ul menu, menu ol ul, menu ol menu, menu menu ul, menu menu menu {
list-style-type: square; }
ul ul, ul ol, ul menu, ol ul, ol ol, ol menu, menu ul, menu ol, menu menu { margin-top: 0; margin-bottom: 0; }
dl { margin: 1em 0; }
dd { margin-inline-start: 40px; }
blockquote, figure { margin: 1em 40px; }
hr { margin: 0.5em auto; border: 1px inset gray; color: gray; overflow: hidden; }
h1 { font-size: 2em; }
h2 { font-size: 1.5em; }
h3 { font-size: 1.17em; }
h4 { font-size: 1em; }
h5 { font-size: 0.83em; }
h6 { font-size: 0.67em; }
h1, h2, h3, h4, h5, h6, th, strong, b { font-weight: bold; }
td, th { padding: 1px; }
th { text-align: center; }
em, i, cite, var, dfn, address { font-style: italic; }
small, sub, sup { font-size: smaller; }
thead, tbody, tfoot, table > tr { vertical-align: middle; }
thead, tbody, tfoot, tr {
  border-top-color: inherit; border-right-color: inherit; border-bottom-color: inherit; border-left-color: inherit;
}
tr, td, th { vertical-align: inherit; }
sub { vertical-align: sub; }
sup { vertical-align: super; }
q::before { content: open-quote; }
q::after { content: close-quote; }
big { font-size: larger; }
center { text-align: -webkit-center; }
fieldset { margin: 0 2px; padding: 0.35em 0.75em 0.625em; border: 2px groove ThreeDFace; min-width: min-content; }
fieldset { display: flow-root; }
fieldset > legend { width: fit-content; }
legend { padding: 0 2px; }
mark { background-color: Mark; color: MarkText; }
a:link { color: LinkText; cursor: pointer; text-decoration: underline; }
u, ins { text-decoration: underline; }
s, del { text-decoration: line-through; }
abbr[title] { text-decoration: underline dotted; }
a:link:active { color: ActiveText; }
code, kbd, samp, pre { font-family: "Go Mono", monospace; font-size: 0.8125em; }
pre { margin: 1em 0; white-space: pre; }
button, input, select, textarea {
  appearance: none; margin: 0; padding: 0; border: 0; background: none;
  font-family: inherit; font-size: inherit; font-weight: inherit; font-style: inherit;
  color: inherit; line-height: inherit; letter-spacing: inherit; word-spacing: inherit; text-align: inherit;
  text-indent: 0;
}
button, input, select, textarea, progress, meter, img, canvas, svg, iframe { display: inline-block; }
iframe { border: 2px inset #eeeeee; }
iframe[frameborder="0"], iframe[frameborder="no" i] { border-width: 0; }
button, select, input[type="checkbox"], input[type="range"] { cursor: default; }
input, textarea { cursor: text; }
button {
  padding: 1px 6px; border: 1px solid light-dark(#767676, #6b6b6b); border-radius: 3px; background-color: ButtonFace;
  text-align: center;
}
button:hover { background-color: light-dark(#e5e5e5, ButtonFace); }
button:active { background-color: light-dark(#f5f5f5, ButtonFace); }
input, select, textarea {
  padding: 1px 2px; border: 1px solid light-dark(#767676, #858585); border-radius: 2px; background-color: Field;
}
input[type="button"], input[type="submit"], input[type="reset"] {
  padding: 1px 6px; border-color: light-dark(#767676, #6b6b6b); border-radius: 3px; background-color: ButtonFace;
  text-align: center; cursor: default;
}
input[type="button"]:hover, input[type="submit"]:hover, input[type="reset"]:hover {
  background-color: light-dark(#e5e5e5, ButtonFace);
}
input[type="button"]:active, input[type="submit"]:active, input[type="reset"]:active {
  background-color: light-dark(#f5f5f5, ButtonFace);
}
::placeholder { color: #757575; }
button:disabled, input[type="button"]:disabled, input[type="submit"]:disabled, input[type="reset"]:disabled {
  color: light-dark(rgba(16, 16, 16, 0.3), rgba(255, 255, 255, 0.3));
  background-color: light-dark(rgba(239, 239, 239, 0.3), rgba(19, 1, 1, 0.3));
  border-color: light-dark(rgba(118, 118, 118, 0.3), rgba(195, 195, 195, 0.3));
}
input:disabled, textarea:disabled {
  color: light-dark(#545454, #aaa); background-color: light-dark(rgba(239, 239, 239, 0.3), rgba(59, 59, 59, 0.3));
  border-color: rgba(118, 118, 118, 0.3);
}
select:disabled { color: light-dark(gray, #aaa); border-color: rgba(118, 118, 118, 0.3); opacity: 0.7; }
option { display: block; padding: 0 2px 1px; }
option:disabled { color: light-dark(gray, #aaa); }
input, select { white-space: nowrap; }
input { overflow: hidden; }
select { overflow: clip; }
textarea { white-space: pre-wrap; overflow: auto; overflow-wrap: break-word; }
select {
  padding-right: 20px; background-image: url(ember:select-arrow); background-repeat: no-repeat;
  background-position: right 6px center;
}
input[type="checkbox"] { width: 13px; height: 13px; padding: 0; border-radius: 2px; }
input[type="checkbox"]:checked {
  background-color: light-dark(#0075ff, #99c8ff); border-color: light-dark(#0075ff, #99c8ff);
  color: light-dark(#fff, #3b3b3b);
}
input[type="radio"] { width: 13px; height: 13px; padding: 0; border-radius: 50%; }
input[type="radio"]:checked { border-color: light-dark(#0075ff, #99c8ff); color: light-dark(#0075ff, #99c8ff); }
/* Disabled controls and the switch in native Chrome's colours, measured from its own controls at 2x (headless Chrome
   through chrome.ScreenshotScaled, 2026-11-03, the dark ones 2026-11-22); its switch, unrendered there, follows the
   checkbox's. */
input[type="checkbox"]:disabled, input[type="radio"]:disabled {
  background-color: light-dark(#f8f8f8, #3b3b3b); border-color: light-dark(rgba(118, 118, 118, 0.3), #626262);
}
input[type="checkbox"]:checked:disabled {
  background-color: light-dark(#d1d1d1, #757575); border-color: light-dark(#d1d1d1, #757575);
  color: light-dark(#ededed, #3b3b3b);
}
input[type="range"]:disabled { color: light-dark(#cbcbcb, #757575); }
input[type="radio"]:checked:disabled {
  border-color: light-dark(#d1d1d1, #757575); color: light-dark(#d1d1d1, #757575);
}
input[type="checkbox"][switch] {
  width: 32px; height: 18px; border: 1px solid light-dark(#767676, #858585); border-radius: 9px;
  background-color: light-dark(#efefef, #3b3b3b); color: light-dark(#767676, #858585);
}
input[type="checkbox"][switch]:checked {
  background-color: light-dark(#0075ff, #99c8ff); border-color: light-dark(#0075ff, #99c8ff);
  color: light-dark(#fff, #3b3b3b);
}
input[type="checkbox"][switch]:disabled {
  background-color: light-dark(#f8f8f8, #3b3b3b); border-color: light-dark(rgba(118, 118, 118, 0.3), #626262);
  color: light-dark(#d1d1d1, #626262);
}
input[type="checkbox"][switch]:checked:disabled {
  background-color: light-dark(#d1d1d1, #757575); border-color: light-dark(#d1d1d1, #757575);
  color: light-dark(#f8f8f8, #3b3b3b);
}
input[type="range"] {
  width: 129px; height: 16px; padding: 0; border: 0; background: none; color: light-dark(#0075ff, #99c8ff);
}
progress { width: 160px; height: 16px; color: light-dark(#0075ff, #99c8ff); }
meter { width: 80px; height: 16px; }
summary { display: list-item; }
details > summary { list-style-type: disclosure-closed; }
details[open] > summary { list-style-type: disclosure-open; }
details:open > summary { list-style-type: disclosure-open; }
dialog { position: absolute; left: 0; right: 0; margin: auto; width: fit-content; height: fit-content; padding: 1em;
  border: solid; background-color: Canvas; color: CanvasText; }
dialog:modal {
  position: fixed; overflow: auto; top: 0; bottom: 0; max-width: calc(100% - 6px - 2em);
  max-height: calc(100% - 6px - 2em);
}
dialog::backdrop { background-color: rgba(0, 0, 0, 0.1); }
dialog:not([open]) { display: none; }
[role="menu"]:not([open]) { display: none; }
:focus-visible { outline: 2px solid #0075ff; outline-offset: 1px; }
.ember-title-tip {
  position: fixed; z-index: 2147483647; pointer-events: none; width: max-content; max-width: 400px; padding: 3px 6px;
  border: 1px solid #c8c8c8; border-radius: 4px; background-color: #fff; color: #222; font: 12px sans-serif;
  white-space: pre-line; overflow-wrap: break-word; box-shadow: 0 2px 6px rgba(0, 0, 0, 0.2); user-select: none;
}
`

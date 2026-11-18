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
// (TestChromeSpacesTheTextOfControlsAsABrowsersSheetDoes).
const BrowserDefault = `* { box-sizing: content-box; }
button, select, table, progress, meter, input[type="checkbox"], input[type="radio"], input[type="button"],
input[type="submit"], input[type="reset"], input[type="search"], input[type="color"] { box-sizing: border-box; }
html { user-select: auto; }
input, select, textarea { text-align: start; }
input[type="button"], input[type="submit"], input[type="reset"] { text-align: center; }
button, input, select, textarea {
  letter-spacing: normal; word-spacing: normal; text-transform: none; text-shadow: none;
}
`

// Default is the user-agent sheet: what every element looks like before
// the game's sheets. Export writes the same text into every page, so
// Chrome starts from it too.
const Default = `* { box-sizing: border-box; }
html { font-family: "Go", sans-serif; font-size: 16px; line-height: normal; color: #000; }
html, body, div, section, header, footer, nav, main, aside, article, p, h1, h2, h3, h4,
details, summary, dialog, hr, pre, ul, ol, menu, dl, dt, dd, h5, h6, blockquote, figure, figcaption, form, address,
hgroup, search, center, fieldset, legend { display: block; }
li { display: list-item; }
[hidden] { display: none; }
input[type="hidden" i] { display: none !important; }
table { border-spacing: 2px; }
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
ul, ol, menu { margin: 1em 0; padding-left: 40px; }
ul, menu { list-style-type: disc; }
ol { list-style-type: decimal; }
ul ul, ul menu, ol ul, ol menu, menu ul, menu menu { list-style-type: circle; }
ul ul ul, ul ul menu, ul ol ul, ul ol menu, ul menu ul, ul menu menu, ol ul ul, ol ul menu, ol ol ul, ol ol menu, ol
menu ul, ol menu menu, menu ul ul, menu ul menu, menu ol ul, menu ol menu, menu menu ul, menu menu menu {
list-style-type: square; }
ul ul, ul ol, ul menu, ol ul, ol ol, ol menu, menu ul, menu ol, menu menu { margin-top: 0; margin-bottom: 0; }
dl { margin: 1em 0; }
dd { margin-left: 40px; }
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
tr, td, th { vertical-align: inherit; }
sub { vertical-align: sub; }
sup { vertical-align: super; }
q::before { content: open-quote; }
q::after { content: close-quote; }
big { font-size: larger; }
center { text-align: -webkit-center; }
fieldset { margin: 0 2px; padding: 0.35em 0.75em 0.625em; border: 2px groove #efefef; min-width: min-content; }
fieldset { display: flow-root; }
fieldset > legend { width: fit-content; }
legend { padding: 0 2px; }
mark { background-color: #ff0; color: #000; }
a:link { color: #0000ee; cursor: pointer; text-decoration: underline; }
u, ins { text-decoration: underline; }
s, del { text-decoration: line-through; }
abbr[title] { text-decoration: underline dotted; }
a:link:active { color: #ff0000; }
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
  padding: 1px 6px; border: 1px solid #767676; border-radius: 3px; background-color: #efefef; text-align: center;
}
button:hover { background-color: #e5e5e5; }
button:active { background-color: #f5f5f5; }
input, select, textarea { padding: 1px 2px; border: 1px solid #767676; border-radius: 2px; background-color: #fff; }
input[type="button"], input[type="submit"], input[type="reset"] {
  padding: 1px 6px; border-radius: 3px; background-color: #efefef; text-align: center; cursor: default;
}
input[type="button"]:hover, input[type="submit"]:hover, input[type="reset"]:hover { background-color: #e5e5e5; }
input[type="button"]:active, input[type="submit"]:active, input[type="reset"]:active { background-color: #f5f5f5; }
::placeholder { color: #757575; }
button:disabled, input[type="button"]:disabled, input[type="submit"]:disabled, input[type="reset"]:disabled {
  color: rgba(16, 16, 16, 0.3); background-color: rgba(239, 239, 239, 0.3); border-color: rgba(118, 118, 118, 0.3);
}
input:disabled, textarea:disabled {
  color: #545454; background-color: rgba(239, 239, 239, 0.3); border-color: rgba(118, 118, 118, 0.3);
}
select:disabled { color: gray; border-color: rgba(118, 118, 118, 0.3); opacity: 0.7; }
option { display: block; padding: 0 2px 1px; }
option:disabled { color: gray; }
input, select { white-space: nowrap; }
input { overflow: hidden; }
select { overflow: clip; }
textarea { white-space: pre-wrap; overflow: auto; overflow-wrap: break-word; }
select {
  padding-right: 20px; background-image: url(ember:select-arrow); background-repeat: no-repeat;
  background-position: right 6px center;
}
input[type="checkbox"] { width: 13px; height: 13px; padding: 0; border-radius: 2px; }
input[type="checkbox"]:checked { background-color: #0075ff; border-color: #0075ff; color: #fff; }
input[type="radio"] { width: 13px; height: 13px; padding: 0; border-radius: 50%; }
input[type="radio"]:checked { border-color: #0075ff; color: #0075ff; }
/* Disabled controls and the switch in native Chrome's colours, measured from its own controls at 2x (headless Chrome
   through chrome.ScreenshotScaled, 2026-11-03); its switch, unrendered there, follows the checkbox's. */
input[type="checkbox"]:disabled { background-color: #f8f8f8; }
input[type="checkbox"]:checked:disabled { background-color: #d1d1d1; border-color: #d1d1d1; color: #ededed; }
input[type="range"]:disabled { color: #cbcbcb; }
input[type="radio"]:disabled { background-color: #f8f8f8; }
input[type="radio"]:checked:disabled { border-color: #d1d1d1; color: #d1d1d1; }
input[type="checkbox"][switch] {
  width: 32px; height: 18px; border: 1px solid #767676; border-radius: 9px; background-color: #efefef; color: #767676;
}
input[type="checkbox"][switch]:checked { background-color: #0075ff; border-color: #0075ff; color: #fff; }
input[type="checkbox"][switch]:disabled {
  background-color: #f8f8f8; border-color: rgba(118, 118, 118, 0.3); color: #d1d1d1;
}
input[type="checkbox"][switch]:checked:disabled { background-color: #d1d1d1; border-color: #d1d1d1; color: #f8f8f8; }
input[type="range"] { width: 129px; height: 16px; padding: 0; border: 0; background: none; color: #0075ff; }
progress { width: 160px; height: 16px; color: #0075ff; }
meter { width: 80px; height: 16px; }
summary { display: list-item; }
details > summary { list-style-type: disclosure-closed; }
details[open] > summary { list-style-type: disclosure-open; }
details:open > summary { list-style-type: disclosure-open; }
dialog { position: absolute; left: 0; right: 0; margin: auto; width: fit-content; height: fit-content; padding: 1em;
  border: solid; background-color: #fff; color: #000; }
dialog:modal {
  position: fixed; overflow: auto; top: 0; bottom: 0; max-width: calc(100% - 6px - 2em);
  max-height: calc(100% - 6px - 2em);
}
dialog::backdrop { background-color: rgba(0, 0, 0, 0.1); }
dialog:not([open]) { display: none; }
[role="menu"]:not([open]) { display: none; }
:focus-visible { outline: 2px solid #0075ff; outline-offset: 1px; }
`

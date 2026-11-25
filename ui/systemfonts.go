package ui

import (
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/fonts"
	"github.com/lantervale/ember/ui/internal/style"
)

// SystemFonts lets font-family name the fonts installed on the machine,
// as a web page's does: a family that is neither one of the engine's
// faces nor an @font-face of a Stylesheet is looked for among them, in
// any case, and the file of the asked weight and style is read when it
// is first drawn with; a family the machine lacks falls to the next name
// of the list, as without the option. The generic families stay the
// engine's faces. A character that no face of the list has, nor the
// engine's fallback, is drawn with an installed face that has it, as
// Chrome asks the platform for one (decision 0227): the platform's
// families for its script, in the text's language for Han and the
// punctuation Chinese, Japanese and Korean share (the lang attribute,
// else the Document's Language), serif ones under serif, the bold face
// under a bold font, then any installed face whose cmap has it; a run of
// one script goes to one face, so Arabic joins and CJK punctuation is
// trimmed by that face. The machine's font folders are scanned once, off
// the frame, and their cmaps read there too: text laid out before then
// shows in the next name's face, or as missing glyphs, then the Document
// lays out again. A font that can't be read is a warning, one only
// fallen back to is passed over. Without the option a Document looks the
// same on every machine; with it, it shows what each one has installed.
// Where there are no files to scan, in a browser, it finds none.
func SystemFonts() DocumentOption {
	return func(d *Document) {
		d.systemFonts = true
		d.fonts.SetSystemFallback(true)
		d.fonts.Scan(fonts.ScanSystem())
	}
}

// SystemFonts reports whether font-family names the machine's fonts.
func (d *Document) SystemFonts() bool { return d.systemFonts }

// landFonts takes the scan of the machine's fonts once it is done, which
// lays the Document out again with them, the fonts that could not be
// read since the last layout, and the page's fonts fetched since.
func (d *Document) landFonts() {
	if d.fonts.Land() {
		d.relayout = true
	}
	for _, err := range d.fonts.Warnings() {
		d.warn(err)
	}
	d.landPageFonts()
}

// fontUnits are what ex, ch, cap, ic and lh stand for in f, as Chrome
// has them: the face's x-height and capital height, half an em and the
// ascent where it says none, the advance of its zero and of its 水, half
// an em and an em where it has no such glyph, and its normal line.
func (d *Document) fontUnits(f style.Font) css.FontUnits {
	m := d.fonts.Metrics(f)
	u := css.FontUnits{Ex: m.XHeight, Ch: f.Size / 2, Cap: m.CapHeight, Ic: f.Size, Line: m.Ascent + m.Descent + m.LineGap,
		Ascent: m.Ascent, Descent: m.Descent}
	if u.Ex <= 0 {
		u.Ex = f.Size / 2
	}
	if u.Cap <= 0 {
		u.Cap = m.Ascent
	}
	if d.fonts.Has(f, '0') {
		u.Ch = d.fonts.Advance(f, "0")
	}
	if d.fonts.Has(f, '水') {
		u.Ic = d.fonts.Advance(f, "水")
	}
	return u
}

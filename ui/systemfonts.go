package ui

import "github.com/lantervale/ember/ui/internal/fonts"

// SystemFonts lets font-family name the fonts installed on the machine,
// as a web page's does: a family that is neither one of the engine's
// faces nor an @font-face of a Stylesheet is looked for among them, in
// any case, and the file of the asked weight and style is read when it
// is first drawn with; a family the machine lacks falls to the next name
// of the list, as without the option. The generic families stay the
// engine's faces. The machine's font folders are scanned once, off the
// frame: text laid out before the scan is done shows in the next name's
// face, then the Document lays out again. A font that can't be read is a
// warning. Without the option a Document looks the same on every
// machine; with it, it shows what each one has installed. Where there
// are no files to scan, in a browser, it finds none.
func SystemFonts() DocumentOption {
	return func(d *Document) {
		d.systemFonts = true
		d.fonts.Scan(fonts.ScanSystem())
	}
}

// SystemFonts reports whether font-family names the machine's fonts.
func (d *Document) SystemFonts() bool { return d.systemFonts }

// landFonts takes the scan of the machine's fonts once it is done, which
// lays the Document out again with them, and the fonts that could not be
// read since the last layout.
func (d *Document) landFonts() {
	if d.fonts.Land() {
		d.relayout = true
	}
	for _, err := range d.fonts.Warnings() {
		d.warn(err)
	}
}

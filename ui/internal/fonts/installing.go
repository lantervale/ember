package fonts

import (
	"fmt"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// installing is a Set's use of the machine's fonts: none until
// SetInstalled or Land gives it an index.
type installing struct {
	scanning  *Scanning
	installed *Installed
	// named is the family each name asked for draws with, as the index
	// spells it; "" when the machine has none.
	named map[string]string
	warns []error
}

// SetInstalled has Face look the families the set lacks up in idx, in
// any case: a family's faces are listed when it is first asked for, and
// each one's file read when it is first drawn with.
func (s *Set) SetInstalled(idx *Installed) {
	s.installed, s.named = idx, map[string]string{}
}

// Scan has the set take the index scanning makes, when Land finds it
// done.
func (s *Set) Scan(scanning *Scanning) { s.scanning = scanning }

// Land takes the index of Scan once its scan is done, and reports
// whether it just did: text measured before then was measured without
// the machine's fonts.
func (s *Set) Land() bool {
	if s.scanning == nil {
		return false
	}
	idx, ok := s.scanning.Ready()
	if !ok {
		return false
	}
	s.scanning = nil
	s.SetInstalled(idx)
	return true
}

// Await waits for the scan Land takes, however long it has left.
func (s *Set) Await() {
	if s.scanning != nil {
		s.scanning.Wait()
	}
}

// Warnings takes the installed fonts that could not be read since it
// was last asked.
func (s *Set) Warnings() []error {
	w := s.warns
	s.warns = nil
	return w
}

// installedFamily is the family name draws with, its faces added the
// first time: a family the set already has in another case, else the
// machine's.
func (s *Set) installedFamily(name string) string {
	if family, ok := s.named[name]; ok {
		return family
	}
	family := ""
	if in := s.installed.family(name); in != nil {
		family = in.name
		if !s.has(family) {
			for _, f := range in.faces {
				s.faces = append(s.faces, &Face{Family: family, Weight: f.weight, Italic: f.italic, file: f})
			}
		}
	}
	s.named[name] = family
	return family
}

// read reads c's file, and reports whether c draws: a font that can't be
// read or drawn with is a warning and leaves the set, and its family with
// it when it was the last.
func (s *Set) read(c *Face) bool {
	err := c.parse()
	if err == nil {
		return true
	}
	s.warns = append(s.warns, fmt.Errorf("ui: font %s (%s): %w", c.Family, c.file.path, err))
	for i, o := range s.faces {
		if o == c {
			s.faces = append(s.faces[:i], s.faces[i+1:]...)
			break
		}
	}
	if !s.has(c.Family) {
		for name, family := range s.named {
			if family == c.Family {
				s.named[name] = ""
			}
		}
	}
	return false
}

// parse reads the face's file and checks it draws Latin text. A font
// whose kern table can't be read draws without it.
func (c *Face) parse() error {
	ttf, err := c.file.read()
	if err != nil {
		return err
	}
	f, err := sfnt.Parse(ttf)
	if err != nil {
		bare, bareErr := c.file.read(tagKern)
		if bareErr != nil {
			return err
		}
		if f, bareErr = sfnt.Parse(bare); bareErr != nil {
			return err
		}
		ttf = bare
	}
	var buf sfnt.Buffer
	ppem := fixed.I(int(f.UnitsPerEm()))
	for _, r := range "Aag0" {
		var g sfnt.GlyphIndex
		if g, err = f.GlyphIndex(&buf, r); err != nil {
			return err
		}
		if g == 0 {
			continue
		}
		if _, err = f.LoadGlyph(&buf, g, ppem, nil); err != nil {
			return err
		}
		if _, err = f.GlyphAdvance(&buf, g, ppem, font.HintingNone); err != nil {
			return err
		}
	}
	c.TTF, c.font, c.upem = ttf, f, float32(f.UnitsPerEm())
	return nil
}

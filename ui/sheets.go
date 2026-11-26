package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/dom"
)

// Sheet is one stylesheet of a Document, there to be changed while the
// Document shows: AddSheet and InsertSheet give one, and StyleSheets
// those the Document has. However many are changed between two frames,
// and however often, the Document works its cascade out again once, at
// the next.
type Sheet struct {
	doc *Document
	// whole is the sheet read in one piece, src its text, and rules the
	// same rule by rule, read the first time one is asked for.
	whole *css.Sheet
	src   string
	rules []*css.Sheet
	split bool
	// read is what the cascade reads of it, nil until made again.
	read *css.Sheet
	// fsys is the file system a Stylesheet was read from, where its
	// imports are read first; nil for a sheet given as text.
	fsys     fs.FS
	name     string
	from     string
	media    string
	encoding string
	disabled bool
}

// AddSheet adds a stylesheet to the Document after those it has, as
// CSS does when the Document is made, and gives the Sheet to change it
// by. Its options are CSS's.
func (d *Document) AddSheet(text string, opts ...SheetOption) *Sheet {
	return d.InsertSheet(nil, text, opts...)
}

// InsertSheet adds a stylesheet before the Document's sheet before, so
// before wins where both say the same, as a style element put before
// another; after them all for nil or a sheet the Document has not.
func (d *Document) InsertSheet(before *Sheet, text string, opts ...SheetOption) *Sheet {
	var src sheetSource
	for _, o := range opts {
		o(&src)
	}
	s := &Sheet{doc: d, name: "style", from: src.from, media: src.media, encoding: src.encoding}
	s.Replace(text)
	at := len(d.authored)
	if i := slices.Index(d.authored, before); before != nil && i >= 0 {
		at = i
	}
	d.authored = slices.Insert(d.authored, at, s)
	return s
}

// StyleSheets are the Document's stylesheets in the order the cascade
// takes them, those its options gave first.
func (d *Document) StyleSheets() []*Sheet { return slices.Clone(d.authored) }

// Media has the sheet's rules apply only while the media query list
// holds, as the media attribute of a style element.
func Media(query string) SheetOption { return func(s *sheetSource) { s.media = query } }

// From is the address the sheet's url() are resolved from, "" for none.
func (s *Sheet) From() string { return s.from }

// Media is the media query list the sheet's rules apply under, "" for
// none.
func (s *Sheet) Media() string { return s.media }

// SetMedia changes the media query list the sheet's rules apply under;
// "" has them apply always.
func (s *Sheet) SetMedia(query string) {
	if query != s.media {
		s.media = query
		s.changed()
	}
}

// Disabled reports whether the sheet's rules are left out of the
// cascade.
func (s *Sheet) Disabled() bool { return s.disabled }

// SetDisabled leaves the sheet's rules out of the cascade, or takes
// them back in, at its place.
func (s *Sheet) SetDisabled(disabled bool) {
	if disabled != s.disabled {
		s.disabled = disabled
		s.changed()
	}
}

// Replace gives the sheet another text, read as the first was: what it
// gets wrong is added to the Document's Warnings.
func (s *Sheet) Replace(text string) {
	d := s.doc
	if d == nil {
		return
	}
	whole, warns := css.Parse(s.name, text, s.parsing()...)
	for _, w := range warns {
		d.warn(w)
	}
	d.addFontFaces(whole.FontFaces(), nil)
	s.whole, s.src, s.rules, s.split = whole, text, nil, false
	s.changed()
}

// Remove takes the sheet out of its Document, which looks as it did
// without it from the next frame. The Sheet does nothing more.
func (s *Sheet) Remove() {
	d := s.doc
	if d == nil {
		return
	}
	s.changed()
	d.authored = slices.DeleteFunc(d.authored, func(o *Sheet) bool { return o == s })
	s.doc = nil
}

// Text is the sheet's text: as it was given, or its rules' once they
// were changed one by one.
func (s *Sheet) Text() string {
	if !s.split {
		return s.src
	}
	return strings.Join(s.Rules(), "\n")
}

// Rules are the sheet's top-level rules as CSS, in order: each style
// rule, and each at-rule with what it holds.
func (s *Sheet) Rules() []string {
	s.byRule()
	out := make([]string, len(s.rules))
	for i, r := range s.rules {
		out[i] = r.Text()
	}
	return out
}

// ErrNoRule is the error of InsertRule and DeleteRule for a place that
// is out of the sheet's rules.
var ErrNoRule = errors.New("ui: no such place among the sheet's rules")

// InsertRule puts one rule in the sheet before its rule i, after the
// last for i the number of its Rules. Only the new rule is read. A rule
// ui reads part of is taken, what it drops added to the Document's
// Warnings, as a browser takes a rule with a property it does not
// know; the error is for a text that is no one rule, and ErrNoRule for
// an i out of range.
func (s *Sheet) InsertRule(i int, text string) error {
	if s.doc == nil {
		return errors.New("ui: the sheet was removed from its Document")
	}
	s.byRule()
	if i < 0 || i > len(s.rules) {
		return fmt.Errorf("%w: %d of %d", ErrNoRule, i, len(s.rules))
	}
	parts, warns := css.ParseRules(s.name, text, s.parsing()...)
	if trimmed := strings.TrimSpace(text); len(parts) != 1 || !strings.HasSuffix(trimmed, "}") &&
		(!strings.HasPrefix(trimmed, "@") || !strings.HasSuffix(trimmed, ";")) {
		return fmt.Errorf("ui: %q is not one rule", text)
	}
	for _, w := range warns {
		s.doc.warn(w)
	}
	s.rules = slices.Insert(s.rules, i, parts[0])
	s.changed()
	return nil
}

// DeleteRule takes the sheet's rule i out; the error is ErrNoRule for
// an i out of range.
func (s *Sheet) DeleteRule(i int) error {
	s.byRule()
	if i < 0 || i >= len(s.rules) {
		return fmt.Errorf("%w: %d of %d", ErrNoRule, i, len(s.rules))
	}
	s.rules = slices.Delete(s.rules, i, i+1)
	s.changed()
	return nil
}

// byRule reads the sheet rule by rule, the first time one is asked for.
func (s *Sheet) byRule() {
	if s.split {
		return
	}
	s.rules, _ = css.ParseRules(s.name, s.src, s.parsing()...)
	s.split, s.whole = true, nil
	s.read = nil
}

func (s *Sheet) parsing() []css.ParseOption {
	var opts []css.ParseOption
	if s.from != "" {
		opts = append(opts, css.From(s.from))
	}
	if s.doc != nil {
		opts = s.doc.parsing(opts...)
	}
	return opts
}

// changed has the Document take its sheets again at the next frame.
func (s *Sheet) changed() {
	s.read = nil
	if s.doc != nil {
		s.doc.resheet = true
	}
}

// sheet is what the cascade reads of s.
func (s *Sheet) sheet() *css.Sheet {
	if s.read != nil {
		return s.read
	}
	read := s.whole
	if s.whole == nil || strings.TrimSpace(s.media) != "" {
		parts := s.rules
		if s.whole != nil {
			parts = []*css.Sheet{s.whole}
		}
		var warns []css.Warning
		read, warns = css.Join(s.name, s.media, parts)
		for _, w := range warns {
			s.doc.warn(w)
		}
	}
	var chain []string
	if s.from != "" {
		chain = []string{s.from}
	}
	s.read = s.doc.imported(read, s.fsys, chain, s.from, s.encoding)
	return s.read
}

// take reads the Document's sheets again after one changed, and has the
// whole tree styled again.
func (d *Document) take() {
	if !d.resheet {
		return
	}
	d.resheet = false
	d.sheets = make([]*css.Sheet, 0, len(d.authored))
	for _, s := range d.authored {
		if !s.disabled {
			d.sheets = append(d.sheets, s.sheet())
		}
	}
	d.trans.SetSheets(d.sheets)
	d.takeFontFaces()
	dom.Data(d.html).Dirty |= dom.DirtyStyle
}

// RegisterProperty registers a custom property as a page's
// CSS.registerProperty does, over any @property rule of its name: its
// values compute as its syntax's type, which transitions and animations
// interpolate, and it inherits only as def says. A name registered so
// before, or an invalid definition, is an error.
func (d *Document) RegisterProperty(def css.PropertyDefinition) error {
	if err := d.trans.RegisterProperty(def); err != nil {
		return err
	}
	d.registered = append(d.registered, def)
	dom.Data(d.html).Dirty |= dom.DirtyStyle
	return nil
}

// RegisteredProperties are the custom properties RegisterProperty
// registered, in order.
func (d *Document) RegisteredProperties() []css.PropertyDefinition { return slices.Clone(d.registered) }

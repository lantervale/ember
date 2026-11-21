package ui

import "github.com/lantervale/ember/ui/internal/dom"

// Language sets the Document's language, a BCP 47 tag such as "fr-BE":
// what :lang() reads where no lang attribute above an element names one,
// as a page's Content-Language. Without it such an element has none.
func Language(tag string) DocumentOption { return func(d *Document) { d.SetLanguage(tag) } }

// Language is the Document's language, "" unless set.
func (d *Document) Language() string { return dom.Data(d.html).Language }

// SetLanguage changes the Document's language; what reads it is styled
// again at the next frame.
func (d *Document) SetLanguage(tag string) { dom.SetLanguage(d.html, tag) }

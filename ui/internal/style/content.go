package style

// ContentKind says what a content property asks for.
type ContentKind uint8

const (
	// ContentNormal is normal: no box for ::before and ::after, the
	// marker's own text for ::marker.
	ContentNormal ContentKind = iota
	// ContentNone is none: no box.
	ContentNone
	// ContentItems is a list of strings, counters, quotes and pictures.
	ContentItems
)

// Content is a computed content property.
type Content struct {
	Kind  ContentKind
	Items []ContentItem
}

// ContentItemKind is one piece of generated content.
type ContentItemKind uint8

const (
	// ContentString is a string, attr() included: Text.
	ContentString ContentItemKind = iota
	// ContentImage is a url(): Text is its source.
	ContentImage
	// ContentCounter is counter(Name, Style); ContentCounters is
	// counters(Name, Text, Style), Text between the values. Style names
	// a counter style, "" for decimal.
	ContentCounter
	ContentCounters
	ContentOpenQuote
	ContentCloseQuote
	ContentNoOpenQuote
	ContentNoCloseQuote
)

// ContentItem is one piece of a content property.
type ContentItem struct {
	Kind  ContentItemKind
	Text  string
	Name  string
	Style string
}

// Dynamic reports content whose text depends on where its element stands
// in the document: counters and quotes.
func (c Content) Dynamic() bool {
	for _, it := range c.Items {
		if it.Kind != ContentString && it.Kind != ContentImage {
			return true
		}
	}
	return false
}

// Counter is one counter of a counter-reset, counter-increment or
// counter-set: its name and the value given or left out (0 for a reset
// or a set, 1 for an increment). Reversed is counter-reset's
// reversed(name), which counts down to 1 when no value is given
// (HasValue).
type Counter struct {
	Name     string
	Value    int
	Reversed bool
	HasValue bool
}

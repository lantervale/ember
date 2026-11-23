package svg

import "sync"

// elementFilters reads a picture's filter elements again for elements
// outside it, one at a time.
type elementFilters struct {
	mu sync.Mutex
	b  *builder
}

// Filter is the filter element id names in the picture, read for an
// element outside it whose bounding box is bbox in its own user space,
// as a page's filter: url(#id) on an HTML element reads it: units of the
// box are of bbox, and percentages of user space are of its size. ok is
// false when the picture has no such element or the filter draws
// nothing.
func (p *Picture) Filter(id string, bbox Box) (*Filter, bool) {
	ef := p.filters
	if ef == nil {
		return nil, false
	}
	ef.mu.Lock()
	defer ef.mu.Unlock()
	n := ef.b.doc.ids[id]
	if n == nil || n.name != elemFilter {
		return nil, false
	}
	vw, vh := ef.b.vw, ef.b.vh
	ef.b.vw, ef.b.vh = bbox.W, bbox.H
	defer func() { ef.b.vw, ef.b.vh = vw, vh }()
	return ef.b.filterElement(n, Identity(), bbox)
}

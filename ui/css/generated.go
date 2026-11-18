package css

import (
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// generate keeps n's ::before and ::after nodes and its ::marker style as
// its style and the sheets ask: a node while the pseudo-element's content
// makes a box, the same one from restyle to restyle.
func (r *restyler) generate(n *dom.Node, data *dom.Pipeline, rem float32) {
	if !r.before && !r.after && !r.marker && data.Before == nil && data.After == nil && data.Marker == nil &&
		!data.HasMarkerText {
		return
	}
	shows := data.Style.Display != style.DisplayNone && generates(n.Tag())
	var before, after *style.Computed
	if r.before && shows && r.may(n, "before") {
		before = r.generated(n, data.Style, rem, "before")
	}
	if r.after && shows && r.may(n, "after") {
		after = r.generated(n, data.Style, rem, "after")
	}
	r.keep(n, data, &data.Before, dom.Before, before)
	r.keep(n, data, &data.After, dom.After, after)
	r.markerOf(n, data, rem)
}

// generated is the style of n's ::before or ::after, nil when its
// content makes no box.
func (r *restyler) generated(n *dom.Node, parent *style.Computed, rem float32, name string) *style.Computed {
	casc := r.pm.cascadeFor(r.pm.matchedFor(n, r.sheets, nil, name), name)
	if casc.longhands[orderIndex["content"]] == nil {
		return nil
	}
	c := initial
	x := &ctx{c: &c, parent: parent, em: parent.Font.Size, rem: rem, overlay: r.overlay, node: n}
	x.cascaded(casc, false)
	if r.browser && casc.longhands[orderIndex["box-sizing"]] == nil {
		// The browser sheet's * rule, which sizes elements by their content,
		// matches no pseudo-element: CSS's initial value does it here.
		c.BoxSizing = style.ContentBox
	}
	if c.Content.Kind != style.ContentItems || c.Display == style.DisplayNone {
		return nil
	}
	if c.Display == style.Contents {
		// A pseudo-element has no children for its box to stand for.
		c.Display = style.Inline
	}
	return &c
}

// keep gives the pseudo-element p of n the style st in the node *slot,
// made when there was none and dropped when st is nil.
func (r *restyler) keep(n *dom.Node, data *dom.Pipeline, slot **dom.Node, p dom.Pseudo, st *style.Computed) {
	g := *slot
	if st == nil {
		if g != nil {
			*slot = nil
			data.Dirty |= dom.DirtyLayout
			r.recount = true
		}
		return
	}
	if g == nil {
		g = dom.NewGenerated(n, p)
		*slot = g
		data.Dirty |= dom.DirtyLayout
		r.recount = true
	}
	gd := dom.Data(g)
	old := gd.Style
	changed := old == nil || !old.Equal(st)
	if changed {
		gd.Style = st
		if r.changed != nil && old != nil {
			r.changed(g, old, st)
		} else {
			setShown(g, st)
		}
		if old == nil || r.layoutChanged(old, st) {
			gd.Dirty |= dom.DirtyLayout
			data.Dirty |= dom.DirtyLayout
			r.recount = true
		}
	} else {
		st = old
	}
	if len(st.CounterReset)+len(st.CounterIncrement)+len(st.CounterSet) > 0 || st.Content.Dynamic() {
		r.counts = true
		r.recount = r.recount || changed
		return
	}
	if r.fill(g, st, nil) {
		data.Dirty |= dom.DirtyLayout
	}
}

// markerOf keeps the style and the text of n's ::marker.
func (r *restyler) markerOf(n *dom.Node, data *dom.Pipeline, rem float32) {
	var st *style.Computed
	if r.marker && data.Style.ListItem && r.may(n, "marker") {
		x := &ctx{c: data.Style, em: data.Style.Font.Size, rem: rem}
		st, _ = x.pseudo(r.pm, n, r.sheets, "marker")
	}
	if !data.Marker.Equal(st) {
		data.Marker = st
		data.Dirty |= dom.DirtyLayout
		r.recount = true
	}
	text, has := "", false
	if st != nil && st.Content.Kind != style.ContentNormal {
		if st.Content.Dynamic() {
			r.counts = true
			return
		}
		text, has = contentText(st.Content, nil), true
	}
	if data.MarkerText != text || data.HasMarkerText != has {
		data.MarkerText, data.HasMarkerText = text, has
		data.Dirty |= dom.DirtyLayout
	}
}

// contentText is the text of content, its pictures left out; counting
// gives the counters and quotes theirs, and may be nil for content
// without any.
func contentText(c style.Content, counting func(style.ContentItem) string) string {
	var b strings.Builder
	for _, it := range c.Items {
		switch it.Kind {
		case style.ContentString:
			b.WriteString(it.Text)
		case style.ContentImage:
		case style.ContentCounter, style.ContentCounters, style.ContentOpenQuote, style.ContentCloseQuote,
			style.ContentNoOpenQuote, style.ContentNoCloseQuote:
			if counting != nil {
				b.WriteString(counting(it))
			}
		}
	}
	return b.String()
}

// fill gives the generated node g the children its content st.Content
// asks for, a text node for each run of text and an img for each
// picture, and reports whether they changed.
func (r *restyler) fill(g *dom.Node, st *style.Computed, counting func(style.ContentItem) string) bool {
	type part struct {
		text  string
		image bool
	}
	var parts []part
	for _, it := range st.Content.Items {
		switch it.Kind {
		case style.ContentImage:
			parts = append(parts, part{text: it.Text, image: true})
			continue
		case style.ContentString:
		case style.ContentCounter, style.ContentCounters, style.ContentOpenQuote, style.ContentCloseQuote,
			style.ContentNoOpenQuote, style.ContentNoCloseQuote:
			it.Text = counting(it)
		}
		if n := len(parts); n > 0 && !parts[n-1].image {
			parts[n-1].text += it.Text
		} else {
			parts = append(parts, part{text: it.Text})
		}
	}
	kids := g.Children()
	same := len(kids) == len(parts)
	for i := 0; same && i < len(kids); i++ {
		src, _ := kids[i].Attr("src")
		same = kids[i].IsText() != parts[i].image && (kids[i].IsText() && kids[i].Text() == parts[i].text ||
			!kids[i].IsText() && src == parts[i].text)
	}
	gd := dom.Data(g)
	if same {
		for _, k := range kids {
			if k.IsText() {
				dom.Data(k).Style, dom.Data(k).Shown = st, gd.Shown
			}
		}
		return false
	}
	kids = make([]*dom.Node, len(parts))
	for i, p := range parts {
		if !p.image {
			kids[i] = dom.NewText(p.text)
			kd := dom.Data(kids[i])
			kd.Style, kd.Shown, kd.Dirty = st, gd.Shown, 0
			continue
		}
		kids[i] = dom.NewElement(dom.Img)
		kids[i].SetAttr("src", p.text)
	}
	g.SetGenerated(kids)
	for _, k := range kids {
		if !k.IsText() {
			// A content picture is an anonymous box: only the user agent's
			// rules style it.
			kd := dom.Data(k)
			kd.Style = r.anonymous(k, st)
			kd.Shown, kd.Dirty, kd.Changed = kd.Style, 0, nil
		}
	}
	gd.Dirty |= dom.DirtyLayout
	return true
}

// anonymous is the style of a box no author rule reaches, under parent.
func (r *restyler) anonymous(n *dom.Node, parent *style.Computed) *style.Computed {
	c := initial
	x := &ctx{c: &c, parent: parent, em: parent.Font.Size, rem: parent.Font.Size, node: n}
	x.cascaded(r.pm.cascade(r.pm.matched(n, r.sheets[:1], nil)), false)
	return &c
}

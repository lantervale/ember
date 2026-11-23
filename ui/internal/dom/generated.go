package dom

// Pseudo is the pseudo-element a generated node stands for.
type Pseudo uint8

const (
	NoPseudo Pseudo = iota
	Before
	After
	FirstLetter
	FirstLine
)

func (p Pseudo) String() string {
	return [...]string{NoPseudo: "", Before: "before", After: "after", FirstLetter: "first-letter",
		FirstLine: "first-line"}[p]
}

// NewGenerated is the node of origin's ::before, ::after, ::first-letter or ::first-line box: a span
// whose parent is origin, not among origin's children, as a Shadow is.
// css makes it, styles it and gives it the content's text nodes and
// pictures as children; layout lays it out first or last among origin's
// children. One node stands for the pseudo-element while it has a box, so
// what is kept by node (boxes, transitions) is kept for it too.
func NewGenerated(origin *Node, p Pseudo) *Node {
	return &Node{tag: Span, parent: origin, gen: p}
}

// Pseudo is the pseudo-element a generated node stands for, NoPseudo for
// any other node.
func (n *Node) Pseudo() Pseudo { return n.gen }

// Generated reports a generated node or one inside it.
func (n *Node) Generated() bool {
	return n.gen != NoPseudo || n.parent != nil && n.parent.gen != NoPseudo
}

// Origin is the element a generated node, or a node inside one, was
// generated for, and n itself for any other node.
func Origin(n *Node) *Node {
	for n != nil && n.Generated() {
		n = n.parent
	}
	return n
}

// Flow is what layout lays out inside n: its ::before node, its children,
// its ::after node.
func (n *Node) Flow() []*Node {
	before, after := n.pipe.Before, n.pipe.After
	if before == nil && after == nil {
		return n.children
	}
	out := make([]*Node, 0, len(n.children)+2)
	if before != nil {
		out = append(out, before)
	}
	out = append(out, n.children...)
	if after != nil {
		out = append(out, after)
	}
	return out
}

// SameFlow reports whether flow is n's Flow.
func (n *Node) SameFlow(flow []*Node) bool {
	before, after := n.pipe.Before, n.pipe.After
	if before != nil {
		if len(flow) == 0 || flow[0] != before {
			return false
		}
		flow = flow[1:]
	}
	if after != nil {
		if len(flow) == 0 || flow[len(flow)-1] != after {
			return false
		}
		flow = flow[:len(flow)-1]
	}
	return equal(flow, n.children)
}

func equal(a, b []*Node) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// SetGenerated gives a generated node its children, the text and the
// pictures of its content, without flagging anything: css flags what its
// change asks for.
func (n *Node) SetGenerated(children []*Node) {
	for _, c := range n.children {
		c.parent, c.pipe.Placed = nil, nil
	}
	for _, c := range children {
		c.parent = n
	}
	n.children = children
}

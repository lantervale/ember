package dom

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/style"
)

// Dirty says what a change needs redone.
type Dirty uint8

const (
	// DirtyStyle: the node's own style is to recompute (its id, classes,
	// attributes, style or state changed, or it was added).
	DirtyStyle Dirty = 1 << iota
	// DirtyTree: siblings or descendants may match other selectors now
	// (a child added or removed; an id, class or attribute changed, which
	// Changed names).
	DirtyTree
	// DirtyLayout: the node's box is to lay out again (its text, children
	// or attributes changed, or css found a change layout reads).
	DirtyLayout
	// DirtyDescendant: some node below is dirty; once css has run, some
	// node below needs layout.
	DirtyDescendant
	// DirtyState: the node's pseudo-class state changed; its descendants
	// may match other selectors when some combinator reads that state.
	DirtyState
)

// Children is the Changed key of a child added or removed.
const Children = "+"

// Pipeline is what the stages keep per node. css writes Style and clears
// DirtyStyle, DirtyTree and DirtyState; layout clears DirtyLayout and
// DirtyDescendant.
type Pipeline struct {
	// Changed names what changed with DirtyTree, until css next runs:
	// ".class", "#id" and "[attribute" for the element's own, and Children
	// for its children added or removed.
	Changed []string
	Dirty   Dirty
	// StyledState is the node's state when css last styled it.
	StyledState State
	// StyledDir is, for an element of dir="auto" or a bdi, the direction
	// its text had when css last styled it, "" for another element.
	StyledDir string
	// Style is the node's computed style; a text node's is its parent's.
	Style *style.Computed
	// Shown is what paint draws: Style itself, or a copy with running
	// transitions' values while any run; a text node's is its parent's.
	Shown *style.Computed
	// Placed is the node's principal box after layout, which Node.Box
	// asks for its border box in the document's space.
	Placed interface{ Rect() geom.Rect }
	// Scroll is a scroll container's offset, kept across layouts, which
	// clamp it to the scrollable overflow and copy it to the Box.
	Scroll geom.Vec2
	// Shadow is the text an input, textarea or select shows: its value,
	// the selected option's label, or its placeholder when Placeholder is
	// set. The Document keeps it; layout lays it out as the element's only
	// content, in place of its children, and export never writes it.
	Shadow      *Node
	Placeholder bool
	// Before and After are the nodes of the element's ::before and ::after
	// boxes (NewGenerated), nil while the pseudo-element generates none.
	// css keeps them; export never writes them.
	Before, After *Node
	// FirstLine is the style of what the element holds on the first line
	// of a block that has a ::first-line style: on that block, the style
	// of its own text there; on an inline element or a ::before or ::after
	// node inside it, the element's style on that line, inheriting from
	// the first line's. A block whose first line is its parent's takes it
	// too. nil elsewhere; css keeps it.
	FirstLine *style.Computed
	// LineBack is, on a block whose first line has a background, the node
	// of the box that paints it behind the line (NewGenerated).
	LineBack *Node
	// FirstLetter is the node of a block's ::first-letter box while one is
	// styled (NewGenerated). Letter is, on that node, the text node whose
	// first letter it takes, and on that text node, the node; LetterEnd is
	// where the letter, the punctuation round it and the spaces before it
	// end in the text, in bytes. The node holds a text node of the letter.
	FirstLetter, Letter *Node
	LetterEnd           int
	// Ordinal is a list item's number, the value of the list-item counter
	// on it, and Counted that css worked it out.
	Ordinal int
	Counted bool
	// Counts is set on the root once something in its tree counts: css
	// then keeps the counters of the whole tree. Firsts is set on it while
	// some sheet styles a ::first-line or a ::first-letter, or did: css
	// then keeps them through the tree.
	Counts, Firsts bool
	// Language is the root's: the Document's language (SetLanguage).
	Language string
	// Marker is the style of a list item's ::marker when a rule styles it,
	// and MarkerText its content when that is not normal (HasMarkerText).
	// MarkerNode stands for that ::marker while a rule styles it, as a
	// generated node does, its transitions kept by it and its Shown what
	// paint draws.
	Marker        *style.Computed
	MarkerText    string
	HasMarkerText bool
	MarkerNode    *Node
	// Markup is what an svg element holds, as SVG, and picture the
	// address of the picture it makes in the colour pictured.
	Markup   string
	picture  string
	pictured color.Color
	// chosen is the picture an img shows in place of its src while
	// choosing, of density device pixels a point, and source the source
	// element it was chosen by, if any (Choose).
	chosen   string
	density  float32
	choosing bool
	source   *Node
	// Near is set on an img that loads lazily once it came near the
	// viewport: it shows its picture from then on.
	Near bool
	// Canvas is a canvas element's machine, made with it since it holds
	// the game's drawing, for the Document to take as the element's.
	Canvas any
	// Frame is the Document an iframe shows, kept by ui; Guarded is
	// whether a panic of that Document stops at the iframe, and Broken
	// the one that did.
	Frame   any
	Guarded bool
	Broken  error
	// Root marks the html element of a Document, which an iframe moved
	// into keeps what it showed.
	Root bool
	// Handlers are the node's callbacks (decision 0057), kept by ui.
	Handlers any
	// Follow is the entity the node follows on screen, kept by ui.
	Follow any
}

// Data is the pipeline data of n.
func Data(n *Node) *Pipeline { return &n.pipe }

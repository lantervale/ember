package ui

import "github.com/lantervale/ember/ui/css"

// AnimationStarted is an animation of Node, from the @keyframes Name,
// past its delay, as the DOM's animationstart: Elapsed is how far into
// it, more than 0 for a negative delay.
type AnimationStarted struct {
	Node    *Node
	Name    string
	Elapsed float32
}

// AnimationIterated is an animation of Node beginning another iteration,
// as animationiteration: Elapsed is the time of the iterations done.
type AnimationIterated struct {
	Node    *Node
	Name    string
	Elapsed float32
}

// AnimationEnded is an animation of Node over, as animationend: Elapsed
// is the time of all its iterations.
type AnimationEnded struct {
	Node    *Node
	Name    string
	Elapsed float32
}

// AnimationCanceled is a running animation of Node stopped before its
// end, as animationcancel: its name gone from the element's
// animation-name, the element display: none or taken out.
type AnimationCanceled struct {
	Node    *Node
	Name    string
	Elapsed float32
}

// ReducedMotion has the Document's sheets answer
// prefers-reduced-motion: reduce, where a page that asks stops or
// shortens its animations (decision 0114).
func ReducedMotion() DocumentOption { return func(d *Document) { d.reducedMotion = true } }

// ReducedMotion reports whether the Document answers
// prefers-reduced-motion: reduce.
func (d *Document) ReducedMotion() bool { return d.reducedMotion }

// SetReducedMotion changes the answer from the next layout, which styles
// the tree again where a rule asks.
func (d *Document) SetReducedMotion(on bool) { d.reducedMotion = on }

// animationEvents queues what the animations did since the last frame,
// to be emitted and dispatched with the Document's other events.
func (d *Document) animationEvents() {
	for _, e := range d.trans.Events() {
		var ev any
		switch e.Kind {
		case css.AnimationStart:
			ev = AnimationStarted{e.Node, e.Name, e.Elapsed}
		case css.AnimationIteration:
			ev = AnimationIterated{e.Node, e.Name, e.Elapsed}
		case css.AnimationEnd:
			ev = AnimationEnded{e.Node, e.Name, e.Elapsed}
		case css.AnimationCancel:
			ev = AnimationCanceled{e.Node, e.Name, e.Elapsed}
		}
		d.events = append(d.events, ev)
	}
}

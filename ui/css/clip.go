package css

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// checkClip takes auto or rect() of four edges, each a length or auto,
// all split by commas or all by whitespace, as Chrome takes them
// (decision 0246).
func checkClip(v []Component) error {
	_, err := readClipRect(nil, v)
	return err
}

func clipRect(x *ctx, v []Component) style.ClipRect {
	r, _ := readClipRect(x, v)
	return r
}

// readClipRect reads a clip value, measuring its lengths on x; with x
// nil it only checks them.
func readClipRect(x *ctx, v []Component) (style.ClipRect, error) {
	var r style.ClipRect
	switch {
	case len(v) == 1 && isWord(v[0], "auto"):
		return r, nil
	case len(v) != 1 || v[0].Kind != Function || !strings.EqualFold(v[0].Text, "rect"):
		return r, errors.New("clip takes auto or rect()")
	}
	edges, err := rectEdges(v[0].Args)
	if err != nil {
		return r, fmt.Errorf("clip: rect(): %w", err)
	}
	r.Set = true
	sides := [4]struct {
		edge *float32
		auto *bool
	}{
		{&r.Edges.Top, &r.Auto.Top}, {&r.Edges.Right, &r.Auto.Right},
		{&r.Edges.Bottom, &r.Auto.Bottom}, {&r.Edges.Left, &r.Auto.Left},
	}
	for i, e := range edges {
		if isWord(e, "auto") {
			*sides[i].auto = true
			continue
		}
		if err = length(e); err != nil {
			return r, fmt.Errorf("clip: rect(): %w", err)
		}
		if x != nil {
			*sides[i].edge = x.length(e).At(0)
		}
	}
	return r, nil
}

// rectEdges are rect()'s four arguments, split by commas or, where it
// has none, by whitespace; a mix of the two is an error.
func rectEdges(args []Component) ([]Component, error) {
	isComma := func(c Component) bool { return c.Kind == Comma }
	var edges []Component
	if slices.ContainsFunc(args, isComma) {
		for _, part := range splitOn(args, isComma) {
			part = items(part)
			if len(part) != 1 {
				return nil, errors.New("expected four values all split by commas or all by spaces")
			}
			edges = append(edges, part[0])
		}
	} else {
		edges = items(args)
	}
	if len(edges) != 4 {
		return nil, fmt.Errorf("expected 4 values, found %d", len(edges))
	}
	return edges, nil
}

// clipRects interpolates two rect()s edge by edge where both have their
// auto edges on the same sides, as Chrome does; any other two flip.
//
//nolint:gochecknoglobals // an interpolation, fixed
var clipRects = interp[style.ClipRect]{
	lerp: func(a, b style.ClipRect, t float32) (style.ClipRect, bool) {
		if !a.Set || !b.Set || a.Auto != b.Auto {
			return a, false
		}
		out := b
		out.Edges = style.Sides[float32]{
			Top: lerpf(a.Edges.Top, b.Edges.Top, t), Right: lerpf(a.Edges.Right, b.Edges.Right, t),
			Bottom: lerpf(a.Edges.Bottom, b.Edges.Bottom, t), Left: lerpf(a.Edges.Left, b.Edges.Left, t),
		}
		return out, true
	},
	equal: same[style.ClipRect],
}

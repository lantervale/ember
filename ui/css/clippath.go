package css

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lantervale/ember/ui/internal/outline"
	"github.com/lantervale/ember/ui/internal/style"
)

//nolint:gochecknoglobals // lookup table
var shapeBoxes = map[string]style.ShapeBox{
	kwBorderBox: style.ShapeBorderBox, kwPaddingBox: style.ShapePaddingBox, kwContentBox: style.ShapeContentBox,
	"margin-box": style.ShapeMarginBox, "fill-box": style.ShapeFillBox, "stroke-box": style.ShapeStrokeBox,
	"view-box": style.ShapeViewBox,
}

// checkClipPath takes none, a basic shape, a reference box, or a shape
// and a box in either order. A url() to an SVG clipPath is refused.
func checkClipPath(v []Component) error {
	_, err := readClipPath(nil, v)
	return err
}

func clipPath(x *ctx, v []Component) style.ClipPath {
	c, _ := readClipPath(x, v)
	return c
}

// shapeReader reads a basic shape's arguments, measuring lengths on x;
// with x nil it only checks them.
type shapeReader struct{ x *ctx }

func (r shapeReader) length(c Component, t term) (style.Length, error) {
	if err := t(c); err != nil {
		return style.Length{}, err
	}
	if r.x == nil {
		return style.Length{}, nil
	}
	return r.x.length(c), nil
}

// readClipPath reads a clip-path value, given without whitespace.
func readClipPath(x *ctx, v []Component) (style.ClipPath, error) {
	var c style.ClipPath
	if len(v) == 1 && isWord(v[0], kwNone) {
		return c, nil
	}
	if len(v) == 0 || len(v) > 2 {
		return c, errors.New("clip-path takes none, a basic shape, a box, or a shape and a box")
	}
	r := shapeReader{x}
	for _, part := range v {
		switch {
		case part.Kind == Ident && c.Box == style.ShapeBoxUnset:
			b, ok := shapeBoxes[strings.ToLower(part.Text)]
			if !ok {
				return c, fmt.Errorf("clip-path: unknown box %q", part.Text)
			}
			c.Box = b
		case urlTerm(part) == nil:
			return c, errors.New("clip-path: url() is not supported; use a basic shape")
		case part.Kind == Function && c.Shape == style.ShapeNone:
			if err := r.shape(&c, part); err != nil {
				return c, fmt.Errorf("clip-path: %s(): %w", strings.ToLower(part.Text), err)
			}
		default:
			return c, fmt.Errorf("clip-path: unexpected %q", part.String())
		}
	}
	if c.Box == style.ShapeBorderBox && c.Shape != style.ShapeNone {
		c.Box = style.ShapeBoxUnset
	}
	return c, nil
}

func (r shapeReader) shape(c *style.ClipPath, f Component) error {
	args := items(f.Args)
	switch strings.ToLower(f.Text) {
	case kwInset:
		c.Shape = style.ShapeInset
		return r.inset(c, args)
	case "xywh":
		c.Shape = style.ShapeInset
		return r.xywh(c, args)
	case "rect":
		c.Shape = style.ShapeInset
		return r.rect(c, args)
	case kwCircle:
		c.Shape = style.ShapeCircle
		return r.ellipse(c, args, 1)
	case "ellipse":
		c.Shape = style.ShapeEllipse
		return r.ellipse(c, args, 2)
	case "polygon":
		c.Shape = style.ShapePolygon
		return r.polygon(c, args)
	case "path":
		c.Shape = style.ShapePath
		return r.path(c, args)
	}
	return errors.New("not a basic shape")
}

// rounded splits off args' round and reads its corners into c.
func (r shapeReader) rounded(c *style.ClipPath, args []Component) ([]Component, error) {
	for i, a := range args {
		if !isWord(a, "round") {
			continue
		}
		radii, err := r.radii(args[i+1:])
		c.Round = radii
		return args[:i], err
	}
	zero := style.Size[style.Length]{Width: style.Px(0), Height: style.Px(0)}
	c.Round = style.Corners[style.Size[style.Length]]{TopLeft: zero, TopRight: zero, BottomRight: zero, BottomLeft: zero}
	return args, nil
}

// radii reads a border-radius: one to four horizontal radii, then after
// a slash one to four vertical ones.
func (r shapeReader) radii(v []Component) (style.Corners[style.Size[style.Length]], error) {
	var out style.Corners[style.Size[style.Length]]
	parts := splitOn(v, isSlash)
	if len(parts) > 2 {
		return out, errors.New("round takes one slash at most")
	}
	var lists [2][4]style.Length
	for k, part := range parts {
		ls, err := r.sides(part, sizeLP)
		if err != nil {
			return out, err
		}
		lists[k] = ls
	}
	if len(parts) == 1 {
		lists[1] = lists[0]
	}
	h, w := lists[1], lists[0]
	out.TopLeft = style.Size[style.Length]{Width: w[0], Height: h[0]}
	out.TopRight = style.Size[style.Length]{Width: w[1], Height: h[1]}
	out.BottomRight = style.Size[style.Length]{Width: w[2], Height: h[2]}
	out.BottomLeft = style.Size[style.Length]{Width: w[3], Height: h[3]}
	return out, nil
}

// sides reads one to four values of t, repeated as CSS repeats a box's
// sides.
func (r shapeReader) sides(v []Component, t term) ([4]style.Length, error) {
	var out [4]style.Length
	if len(v) < 1 || len(v) > 4 {
		return out, fmt.Errorf("expected 1 to 4 values, found %d", len(v))
	}
	var ls [4]style.Length
	for i, c := range v {
		l, err := r.length(c, t)
		if err != nil {
			return out, err
		}
		ls[i] = l
	}
	pick := [4][4]int{{0, 0, 0, 0}, {0, 1, 0, 1}, {0, 1, 2, 1}, {0, 1, 2, 3}}[len(v)-1]
	for i := range out {
		out[i] = ls[pick[i]]
	}
	return out, nil
}

func (r shapeReader) inset(c *style.ClipPath, args []Component) error {
	args, err := r.rounded(c, args)
	if err != nil {
		return err
	}
	s, err := r.sides(args, lengthPercent)
	c.Inset = style.Sides[style.Length]{Top: s[0], Right: s[1], Bottom: s[2], Left: s[3]}
	return err
}

// rest is 100% less the lengths ls.
func rest(ls ...style.Length) style.Length {
	l := style.Length{Kind: style.Fixed, Percent: 100, IsPercent: true}
	for _, o := range ls {
		l = l.Less(o)
	}
	return l
}

// xywh reads x, y, a width and a height, as the inset that leaves them.
func (r shapeReader) xywh(c *style.ClipPath, args []Component) error {
	args, err := r.rounded(c, args)
	if err != nil {
		return err
	}
	if len(args) != 4 {
		return fmt.Errorf("expected 4 values, found %d", len(args))
	}
	var ls [4]style.Length
	for i, a := range args {
		t := lengthPercent
		if i >= 2 {
			t = sizeLP
		}
		if ls[i], err = r.length(a, t); err != nil {
			return err
		}
	}
	c.Inset = style.Sides[style.Length]{Top: ls[1], Right: rest(ls[0], ls[2]), Bottom: rest(ls[1], ls[3]), Left: ls[0]}
	return nil
}

// rect reads the top, right, bottom and left edges from the box's
// top-left, auto standing for the box's own, as the inset that leaves
// them.
func (r shapeReader) rect(c *style.ClipPath, args []Component) error {
	args, err := r.rounded(c, args)
	if err != nil {
		return err
	}
	if len(args) != 4 {
		return fmt.Errorf("expected 4 values, found %d", len(args))
	}
	var ls [4]style.Length
	for i, a := range args {
		if isWord(a, "auto") {
			ls[i] = style.Length{Kind: style.Fixed, IsPercent: true}
			if i == 1 || i == 2 {
				ls[i].Percent = 100
			}
			continue
		}
		if ls[i], err = r.length(a, lengthPercent); err != nil {
			return err
		}
	}
	c.Inset = style.Sides[style.Length]{Top: ls[0], Right: rest(ls[1]), Bottom: rest(ls[2]), Left: ls[3]}
	return nil
}

// centre reads the position after at, from the box's top-left.
func (r shapeReader) centre(c *style.ClipPath, pos []Component) error {
	if len(pos) == 3 || len(pos) == 0 {
		return errors.New("at takes a position of one, two or four values")
	}
	p, err := parsePosition(pos)
	if err != nil {
		return err
	}
	c.AtSet = true
	if c.At[0], err = r.length(p.x, lengthPercent); err != nil {
		return err
	}
	if c.At[1], err = r.length(p.y, lengthPercent); err != nil {
		return err
	}
	if p.right {
		c.At[0] = rest(c.At[0])
	}
	if p.bot {
		c.At[1] = rest(c.At[1])
	}
	return nil
}

// ellipse reads a circle's radius or an ellipse's two, and the centre
// after at.
func (r shapeReader) ellipse(c *style.ClipPath, args []Component, radii int) error {
	at := len(args)
	for i, a := range args {
		if isWord(a, "at") {
			at = i
			break
		}
	}
	if at < len(args) {
		if err := r.centre(c, args[at+1:]); err != nil {
			return err
		}
	} else {
		c.At = [2]style.Length{style.Percent(50), style.Percent(50)}
		c.At[0].IsPercent, c.At[1].IsPercent = true, true
	}
	given := args[:at]
	if len(given) != 0 && len(given) != radii {
		return fmt.Errorf("expected %d radii, found %d", radii, len(given))
	}
	for i, a := range given {
		switch {
		case isWord(a, "closest-side"):
			c.Radius[i] = style.ShapeRadius{Side: style.RadiusClosest}
		case isWord(a, "farthest-side"):
			c.Radius[i] = style.ShapeRadius{Side: style.RadiusFarthest}
		default:
			l, err := r.length(a, sizeLP)
			if err != nil {
				return err
			}
			c.Radius[i] = style.ShapeRadius{Side: style.RadiusLength, Length: l}
		}
	}
	return nil
}

// fillRule reads parts' leading nonzero or evenodd, if any, into c.
func fillRule(c *style.ClipPath, parts [][]Component) [][]Component {
	if len(parts[0]) == 1 && isWord(parts[0][0], "nonzero", "evenodd") {
		c.EvenOdd = isWord(parts[0][0], "evenodd")
		return parts[1:]
	}
	return parts
}

func (r shapeReader) polygon(c *style.ClipPath, args []Component) error {
	if len(args) == 0 {
		return errors.New("a polygon needs a point")
	}
	parts := fillRule(c, splitOn(args, func(c Component) bool { return c.Kind == Comma }))
	if len(parts) == 0 {
		return errors.New("a polygon needs a point")
	}
	for _, p := range parts {
		if len(p) != 2 {
			return errors.New("each point is an x and a y")
		}
		x, err := r.length(p[0], lengthPercent)
		if err != nil {
			return err
		}
		y, err := r.length(p[1], lengthPercent)
		if err != nil {
			return err
		}
		if r.x != nil {
			c.Points = append(c.Points, [2]style.Length{x, y})
		}
	}
	return nil
}

func (r shapeReader) path(c *style.ClipPath, args []Component) error {
	if len(args) == 0 {
		return errors.New("a path needs its data")
	}
	parts := fillRule(c, splitOn(args, func(c Component) bool { return c.Kind == Comma }))
	if len(parts) != 1 || len(parts[0]) != 1 || parts[0][0].Kind != String {
		return errors.New("expected a fill rule and a string of path data")
	}
	_, d, ok := outline.Parse(parts[0][0].Text)
	if !ok {
		return errors.New("bad path data")
	}
	c.Path = d
	return nil
}

// lerpClip is t of the way from a to b, two shapes of a kind side by
// side; any others swap half way, as CSS swaps what does not
// interpolate.
func lerpClip(a, b *style.ClipPath, t float32) style.ClipPath {
	if !(value{kind: clipValue, clip: *a}).interpolates(value{kind: clipValue, clip: *b}) || a.Shape == style.ShapeNone ||
		a.Shape == style.ShapePath {
		if t < 0.5 {
			return *a
		}
		return *b
	}
	out := *b
	sides := func(x, y style.Sides[style.Length]) style.Sides[style.Length] {
		return style.Sides[style.Length]{Top: lerpLength(x.Top, y.Top, t), Right: lerpLength(x.Right, y.Right, t),
			Bottom: lerpLength(x.Bottom, y.Bottom, t), Left: lerpLength(x.Left, y.Left, t)}
	}
	corner := func(x, y style.Size[style.Length]) style.Size[style.Length] {
		return style.Size[style.Length]{Width: lerpLength(x.Width, y.Width, t), Height: lerpLength(x.Height, y.Height, t)}
	}
	out.Inset = sides(a.Inset, b.Inset)
	out.Round = style.Corners[style.Size[style.Length]]{
		TopLeft: corner(a.Round.TopLeft, b.Round.TopLeft), TopRight: corner(a.Round.TopRight, b.Round.TopRight),
		BottomRight: corner(a.Round.BottomRight, b.Round.BottomRight),
		BottomLeft:  corner(a.Round.BottomLeft, b.Round.BottomLeft),
	}
	for i := range out.Radius {
		if a.Radius[i].Side == style.RadiusLength {
			out.Radius[i].Length = lerpLength(a.Radius[i].Length, b.Radius[i].Length, t)
		}
	}
	out.At = [2]style.Length{lerpLength(a.At[0], b.At[0], t), lerpLength(a.At[1], b.At[1], t)}
	out.AtSet = a.AtSet || b.AtSet
	if len(b.Points) > 0 {
		out.Points = make([][2]style.Length, len(b.Points))
		for i := range out.Points {
			out.Points[i] = [2]style.Length{lerpLength(a.Points[i][0], b.Points[i][0], t),
				lerpLength(a.Points[i][1], b.Points[i][1], t)}
		}
	}
	return out
}

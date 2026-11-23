package css

import (
	"maps"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

const (
	propSnapType             = "scroll-snap-type"
	propSnapAlign            = "scroll-snap-align"
	propSnapStop             = "scroll-snap-stop"
	propScrollBehavior       = "scroll-behavior"
	propOverscrollX          = "overscroll-behavior-x"
	propOverscrollY          = "overscroll-behavior-y"
	shortScrollPaddingInline = "scroll-padding-inline"
	shortScrollMarginInline  = "scroll-margin-inline"
	kwContain                = "contain"
)

//nolint:gochecknoglobals // lookup tables
var (
	snapAligns = map[string]style.SnapAlign{kwNone: style.SnapNone, kwStart: style.SnapStart, kwEnd: style.SnapEnd,
		kwCenter: style.SnapCenter}
	overscrolls = map[string]style.Overscroll{kwAuto: style.OverscrollAuto, kwContain: style.OverscrollContain,
		kwNone: style.OverscrollNone}
)

// snapProps are the scroll-snap properties, scroll-padding, scroll-margin,
// scroll-behavior and overscroll-behavior: ui writes horizontal lines
// only, so block is y and inline x.
func snapProps() map[string]prop {
	snap := func(c *style.Computed) *style.Snap { return &c.Snap }
	m := map[string]prop{
		propSnapType: {
			set: func(x *ctx, v []Component) {
				v = items(v)
				s := snap(x.c)
				s.Type, s.Mandatory = 0, false
				switch strings.ToLower(v[0].Text) {
				case "x", kwInline:
					s.Type = style.SnapX
				case "y", kwBlock:
					s.Type = style.SnapY
				case kwBoth:
					s.Type = style.SnapX | style.SnapY
				}
				s.Mandatory = len(v) > 1 && isWord(v[1], "mandatory")
			},
			inherit: func(dst, src *style.Computed) { dst.Snap.Type, dst.Snap.Mandatory = src.Snap.Type, src.Snap.Mandatory },
			anim:    anim{kind: discrete},
		},
		propSnapAlign: {
			set: func(x *ctx, v []Component) {
				v = items(v)
				s := snap(x.c)
				s.AlignY = snapAligns[strings.ToLower(v[0].Text)]
				s.AlignX = snapAligns[strings.ToLower(v[len(v)-1].Text)]
			},
			inherit: func(dst, src *style.Computed) { dst.Snap.AlignX, dst.Snap.AlignY = src.Snap.AlignX, src.Snap.AlignY },
			anim:    anim{kind: discrete},
		},
		propSnapStop: field(func(c *style.Computed) *bool { return &c.Snap.Always },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "always") }, flips[bool]()),
		propScrollBehavior: field(func(c *style.Computed) *bool { return &c.Snap.Smooth },
			func(_ *ctx, v []Component) bool { return isWord(v[0], "smooth") }, flips[bool]()),
		propOverscrollX: keyword(func(c *style.Computed) *style.Overscroll { return &c.Snap.OverscrollX },
			overscrolls),
		propOverscrollY: keyword(func(c *style.Computed) *style.Overscroll { return &c.Snap.OverscrollY },
			overscrolls),
	}
	maps.Copy(m, sideProps("scroll-padding-", "", sizeLength,
		func(c *style.Computed) *style.Sides[style.Length] { return &c.Snap.Padding }))
	maps.Copy(m, sideProps("scroll-margin-", "", anyLength,
		func(c *style.Computed) *style.Sides[style.Length] { return &c.Snap.Margin }))
	return m
}

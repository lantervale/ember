package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func columnsOf(t *testing.T, decls string) *style.Computed {
	t.Helper()
	d := el(dom.Div, "d")
	Compute(page(d), []*Sheet{sheet(t, "#d { font-size: 10px; color: #f00; "+decls+" }")})
	return styleOf(d)
}

func TestColumnProperties(t *testing.T) {
	s := columnsOf(t, "column-count: 3; column-width: 12em; column-gap: 2em; column-fill: auto")
	if s.ColumnCount != 3 || s.ColumnWidth != style.Px(120) || s.Gap.Width != style.Px(20) || !s.ColumnFillAuto {
		t.Errorf("got count %d, width %v, gap %v, fill auto %v", s.ColumnCount, s.ColumnWidth, s.Gap.Width, s.ColumnFillAuto)
	}
	if !s.Multicol() {
		t.Error("a column-count makes a multi-column container")
	}
	s = columnsOf(t, "column-count: auto; column-width: auto; column-fill: balance")
	if s.ColumnCount != 0 || s.ColumnWidth.Kind != style.Auto || s.ColumnFillAuto || s.Multicol() {
		t.Errorf("auto: count %d, width %v, fill auto %v", s.ColumnCount, s.ColumnWidth, s.ColumnFillAuto)
	}
	if s.Gap.Width.Kind != style.Auto {
		t.Errorf("column-gap is normal by default: %v", s.Gap.Width)
	}
	if s = columnsOf(t, "column-width: 80px"); !s.Multicol() {
		t.Error("a column-width makes a multi-column container")
	}
}

func TestColumnsShorthand(t *testing.T) {
	cases := []struct {
		src   string
		count int
		width style.Length
	}{
		{"2", 2, style.Length{}},
		{"100px", 0, style.Px(100)},
		{"3 100px", 3, style.Px(100)},
		{"100px 3", 3, style.Px(100)},
		{"auto 4", 4, style.Length{}},
		{"auto", 0, style.Length{}},
		{"auto auto", 0, style.Length{}},
	}
	for _, c := range cases {
		s := columnsOf(t, "column-count: 9; column-width: 9px; columns: "+c.src)
		if s.ColumnCount != c.count || s.ColumnWidth != c.width {
			t.Errorf("columns: %s: count %d, width %v", c.src, s.ColumnCount, s.ColumnWidth)
		}
	}
}

func TestColumnRule(t *testing.T) {
	s := columnsOf(t, "column-rule: 2px dashed #00f")
	if s.ColumnRuleWidth != 2 || s.ColumnRuleStyle != style.BorderDashed || s.ColumnRuleColor != hex("#00f") {
		t.Errorf("shorthand: %v %v %v", s.ColumnRuleWidth, s.ColumnRuleStyle, s.ColumnRuleColor)
	}
	s = columnsOf(t, "column-rule-style: solid")
	if s.ColumnRuleWidth != 3 || s.ColumnRuleColor != hex("#f00") {
		t.Errorf("a rule is medium and currentColor by default: %v %v", s.ColumnRuleWidth, s.ColumnRuleColor)
	}
	s = columnsOf(t, "column-rule: thin dotted")
	if s.ColumnRuleWidth != 1 || s.ColumnRuleStyle != style.BorderDotted {
		t.Errorf("thin dotted: %v %v", s.ColumnRuleWidth, s.ColumnRuleStyle)
	}
	if s = columnsOf(t, "column-rule-width: 4px"); s.ColumnRuleStyle != style.BorderNone {
		t.Errorf("no rule without a style: %v", s.ColumnRuleStyle)
	}
}

func TestBreakProperties(t *testing.T) {
	p, c := el(dom.Div, "p"), el(dom.P, "c")
	Compute(page(with(p, c)), []*Sheet{sheet(t, `
#p { orphans: 3; widows: 4; break-inside: avoid; break-before: column; break-after: avoid-column; }
`)})
	s := styleOf(p)
	if s.Orphans != 3 || s.Widows != 4 || !s.BreakInsideAvoid || s.BreakBefore != style.BreakColumn ||
		s.BreakAfter != style.BreakAvoid {
		t.Errorf("got %d %d %v %v %v", s.Orphans, s.Widows, s.BreakInsideAvoid, s.BreakBefore, s.BreakAfter)
	}
	if k := styleOf(c); k.Orphans != 3 || k.Widows != 4 || k.BreakInsideAvoid || k.BreakBefore != style.BreakAuto {
		t.Errorf("orphans and widows inherit, breaks don't: %d %d %v %v", k.Orphans, k.Widows, k.BreakInsideAvoid, k.BreakBefore)
	}
	d := el(dom.Div, "d")
	Compute(page(d), []*Sheet{sheet(t, "#d { page-break-inside: avoid; column-span: none; }")})
	if s = styleOf(d); !s.BreakInsideAvoid || s.Orphans != 2 || s.Widows != 2 {
		t.Errorf("page-break-inside is break-inside; orphans and widows are 2: %v %d %d", s.BreakInsideAvoid, s.Orphans, s.Widows)
	}
}

func TestColumnValuesRefused(t *testing.T) {
	for src, want := range map[string]string{
		"column-count: 0":          "column-count",
		"column-count: -1":         "column-count",
		"column-count: 1.5":        "integer",
		"column-width: -4px":       "",
		"column-width: 10%":        "",
		"columns: 2 3":             "columns",
		"columns: 10px 20px":       "columns",
		"columns: 1 2px 3":         "columns",
		"column-fill: balance-all": "",
		"column-rule: 2px 3px":     "",
		"column-span: all":         "column-span: all is not supported",
		"orphans: 0":               "orphans",
		"widows: 1.5":              "",
		"break-inside: sometimes":  "",
	} {
		_, warns := Parse("t.css", "p { "+src+" }")
		if len(warns) != 1 || !strings.Contains(warnings(warns), want) {
			t.Errorf("%s: warnings %q, want one holding %q", src, warnings(warns), want)
		}
	}
}

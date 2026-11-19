package style_test

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"unique"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/kit"
)

// TestEveryFieldOfAStyleCounts changes each field of a style alone, at
// any depth, and each slice, map and pointer between nil, empty and
// filled: none may leave the two equal. A field added to a style without
// its comparison in Equal fails here, and a field of a kind the walk
// doesn't know fails until it does.
func TestEveryFieldOfAStyleCounts(t *testing.T) {
	bases := map[string]style.Computed{"zero": {}, "initial": style.Initial(), "filled": filled(t)}
	for name, base := range bases {
		t.Run(name, func(t *testing.T) {
			a, b := clone(t, &base), clone(t, &base)
			require.True(t, reflect.DeepEqual(a, b))
			require.True(t, a.Equal(b), "a copy")
			n := 0
			change(t, reflect.ValueOf(b).Elem(), "Computed", func(path string) {
				n++
				require.False(t, reflect.DeepEqual(a, b), "%s: not a change", path)
				assert.False(t, a.Equal(b), "%s is not compared", path)
				assert.False(t, b.Equal(a), "%s is not compared, the other way", path)
			})
			require.True(t, a.Equal(b), "the walk puts back what it changes")
			assert.Greater(t, n, reflect.TypeFor[style.Computed]().NumField())
		})
	}
}

func TestStylesOfAPageAreEqualAsDeepEqualFindsThem(t *testing.T) {
	styles := corpus(t)
	require.Greater(t, len(styles), 50)
	distinct := 0
	for i, a := range styles {
		for _, b := range styles[i:] {
			want := reflect.DeepEqual(a, b)
			if !want {
				distinct++
			}
			require.Equal(t, want, a.Equal(b))
			require.Equal(t, want, b.Equal(a))
		}
		c := clone(t, a)
		require.True(t, a.Equal(c), "a copy")
		change(t, reflect.ValueOf(c).Elem(), "Computed", func(path string) {
			require.Equal(t, reflect.DeepEqual(a, c), a.Equal(c), path)
		})
	}
	assert.Positive(t, distinct, "styles that differ")
}

func TestANilStyleIsEqualToNilAlone(t *testing.T) {
	var none *style.Computed
	s := style.Initial()
	assert.True(t, none.Equal(nil))
	assert.False(t, none.Equal(&s))
	assert.False(t, s.Equal(nil))
}

// TestNaNIsEqualOnlyWhereDeepEqualSaysSo: a NaN differs from itself, but
// not through a pointer, a slice or a map both styles share, which
// DeepEqual never looks into.
func TestNaNIsEqualOnlyWhereDeepEqualSaysSo(t *testing.T) {
	nan := float32(math.NaN())
	direct := style.Initial()
	direct.Opacity = nan
	inSlice := style.Initial()
	inSlice.Shadows = []style.Shadow{{X: nan}}
	behindPointer := style.Initial()
	behindPointer.BackgroundImage.Gradient = &style.Gradient{Angle: nan}
	inViewport := style.Initial()
	inViewport.Viewport = &style.Computed{Look: style.Look{Opacity: nan}}
	for name, s := range map[string]*style.Computed{
		"direct": &direct, "in a slice": &inSlice, "behind a pointer": &behindPointer, "in the viewport": &inViewport,
	} {
		shared, copied := *s, clone(t, s)
		for _, o := range []*style.Computed{s, &shared, copied} {
			assert.Equal(t, reflect.DeepEqual(s, o), s.Equal(o), name)
		}
		assert.False(t, s.Equal(copied), name)
	}
	assert.False(t, direct.Equal(&style.Computed{}))
	shared := inSlice
	assert.True(t, inSlice.Equal(&shared), "one slice held by both")
}

func TestFontsAreEqualByTheirFamiliesToo(t *testing.T) {
	f := style.Font{Family: []string{"serif", "sans-serif"}, Size: 12, Weight: 400}
	same := style.Font{Family: []string{"serif", "sans-serif"}, Size: 12, Weight: 400}
	assert.True(t, f.Equal(same))
	change(t, reflect.ValueOf(&same).Elem(), "Font", func(path string) {
		assert.False(t, f.Equal(same), path)
	})
}

// filled is a style with every field set, every slice and map filled and
// every pointer followed, its Viewport one style deep.
func filled(t *testing.T) style.Computed {
	t.Helper()
	var c style.Computed
	fill(t, reflect.ValueOf(&c).Elem(), 1)
	return c
}

//nolint:exhaustive // the kinds a style is made of; another fails the test
func fill(t *testing.T, v reflect.Value, styles int) {
	t.Helper()
	if v.Type() == reflect.TypeFor[unique.Handle[style.Calc]]() {
		v.Set(reflect.ValueOf(unique.Make(style.Calc{N: 1, Steps: [style.CalcSteps]style.CalcStep{{Px: 1.5}}})))
		return
	}
	require.True(t, v.CanSet(), "an unexported field of %s", v.Type())
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(1)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(1.5)
	case reflect.String:
		v.SetString("x")
	case reflect.Array:
		for i := range v.Len() {
			fill(t, v.Index(i), styles)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			fill(t, v.Field(i), styles)
		}
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 2, 2))
		for i := range v.Len() {
			fill(t, v.Index(i), styles)
		}
	case reflect.Map:
		v.Set(reflect.MakeMap(v.Type()))
		v.SetMapIndex(one(t, v.Type().Key(), styles), one(t, v.Type().Elem(), styles))
	case reflect.Pointer:
		if v.Type().Elem() == reflect.TypeFor[style.Computed]() {
			if styles == 0 {
				return
			}
			styles--
		}
		v.Set(reflect.New(v.Type().Elem()))
		fill(t, v.Elem(), styles)
	default:
		t.Fatalf("a %s in a style: say how Equal compares it, then how this test fills and changes it", v.Kind())
	}
}

func one(t *testing.T, typ reflect.Type, styles int) reflect.Value {
	t.Helper()
	v := reflect.New(typ).Elem()
	fill(t, v, styles)
	return v
}

// change makes every single change to v there is, calls changed after
// each with where it is, and puts v back as it was.
//
//nolint:exhaustive // the kinds a style is made of; another fails the test
func change(t *testing.T, v reflect.Value, path string, changed func(path string)) {
	t.Helper()
	old := reflect.New(v.Type()).Elem()
	old.Set(v)
	try := func(what string, to reflect.Value) {
		v.Set(to)
		changed(path + what)
		v.Set(old)
	}
	if v.Type() == reflect.TypeFor[unique.Handle[style.Calc]]() {
		try(" another", reflect.ValueOf(unique.Make(style.Calc{N: 1, Steps: [style.CalcSteps]style.CalcStep{{Px: 7}}})))
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(!v.Bool())
		changed(path)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(v.Int() + 1)
		changed(path)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(v.Uint() + 1)
		changed(path)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(v.Float() + 1)
		changed(path)
	case reflect.String:
		v.SetString(v.String() + "x")
		changed(path)
	case reflect.Array:
		for i := range v.Len() {
			change(t, v.Index(i), fmt.Sprintf("%s[%d]", path, i), changed)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			change(t, v.Field(i), path+"."+v.Type().Field(i).Name, changed)
		}
	case reflect.Slice:
		if v.IsNil() {
			try(" empty for nil", reflect.MakeSlice(v.Type(), 0, 0))
		} else {
			try(" nil", reflect.Zero(v.Type()))
		}
		if v.Len() > 0 {
			try(" shorter", copyOf(t, v.Slice(0, v.Len()-1)))
		}
		try(" longer", reflect.Append(copyOf(t, v.Slice(0, v.Len())), one(t, v.Type().Elem(), 0)))
		for i := range v.Len() {
			change(t, v.Index(i), fmt.Sprintf("%s[%d]", path, i), changed)
		}
	case reflect.Map:
		if v.IsNil() {
			try(" empty for nil", reflect.MakeMap(v.Type()))
		} else {
			try(" nil", reflect.Zero(v.Type()))
		}
		for _, k := range v.MapKeys() {
			less, other := copyOf(t, v), copyOf(t, v)
			less.SetMapIndex(k, reflect.Value{})
			try(fmt.Sprintf(" without %v", k), less)
			e := one(t, v.Type().Elem(), 0)
			e.Set(v.MapIndex(k))
			change(t, e, fmt.Sprintf("%s[%v]", path, k), func(at string) {
				other.SetMapIndex(k, e)
				v.Set(other)
				changed(at)
				v.Set(old)
			})
		}
		more := copyOf(t, v)
		if more.IsNil() {
			more = reflect.MakeMap(v.Type())
		}
		more.SetMapIndex(reflect.ValueOf("a key no style has"), one(t, v.Type().Elem(), 0))
		try(" with one more", more)
	case reflect.Pointer:
		if v.IsNil() {
			to := reflect.New(v.Type().Elem())
			try(" zero for nil", to)
			fill(t, to.Elem(), 0)
			try(" filled for nil", to)
			return
		}
		try(" nil", reflect.Zero(v.Type()))
		change(t, v.Elem(), path, changed)
	default:
		t.Fatalf("%s is a %s: say how Equal compares it, then how this test fills and changes it", path, v.Kind())
	}
	v.Set(old)
}

func clone(t *testing.T, c *style.Computed) *style.Computed {
	t.Helper()
	out, ok := copyOf(t, reflect.ValueOf(c)).Interface().(*style.Computed)
	require.True(t, ok)
	return out
}

// copyOf is v with nothing shared with it: nil stays nil, empty empty.
//
//nolint:exhaustive // the kinds a style is made of; another fails the test
func copyOf(t *testing.T, v reflect.Value) reflect.Value {
	t.Helper()
	out := reflect.New(v.Type()).Elem()
	if v.Type() == reflect.TypeFor[unique.Handle[style.Calc]]() {
		out.Set(v)
		return out
	}
	switch v.Kind() {
	case reflect.Array:
		for i := range v.Len() {
			out.Index(i).Set(copyOf(t, v.Index(i)))
		}
	case reflect.Struct:
		for i := range v.NumField() {
			out.Field(i).Set(copyOf(t, v.Field(i)))
		}
	case reflect.Slice:
		if v.IsNil() {
			return out
		}
		out.Set(reflect.MakeSlice(v.Type(), v.Len(), v.Len()))
		for i := range v.Len() {
			out.Index(i).Set(copyOf(t, v.Index(i)))
		}
	case reflect.Map:
		if v.IsNil() {
			return out
		}
		out.Set(reflect.MakeMap(v.Type()))
		for _, k := range v.MapKeys() {
			out.SetMapIndex(copyOf(t, k), copyOf(t, v.MapIndex(k)))
		}
	case reflect.Pointer:
		if v.IsNil() {
			return out
		}
		out.Set(reflect.New(v.Type().Elem()))
		out.Elem().Set(copyOf(t, v.Elem()))
	case reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		t.Fatalf("a %s in a style: say how Equal compares it, then how this test copies it", v.Kind())
	default:
		out.Set(v)
	}
	return out
}

// corpus is the styles css computes for a page of kit's components and of
// plain elements under a sheet that sets what the kit leaves alone: each
// element's, its ::before, ::after and ::marker's, and the viewport's.
func corpus(t *testing.T) []*style.Computed {
	t.Helper()
	d := ui.NewDocument(kit.Theme(), ui.ScrollingViewport(), ui.CSS(corpusSheet))
	d.Body().Append(
		kit.Panel(kit.Title("Kit"),
			kit.Row(kit.Label("Name"), kit.TextInput("Heron"), kit.Button("Set"), kit.Button("Off", ui.Disabled())),
			kit.Column(kit.Check("Lit"), kit.Switch("Shadows"), kit.Slider(0.4), kit.Progress(0.6), kit.Separator()),
			kit.Table(
				[]*ui.Node{kit.TableColumn("Tile"), kit.TableColumn("Height", kit.Sortable())},
				[]*ui.Node{kit.TableRow("wall", "2"), kit.TableRow("pillar", "3")},
			),
			kit.Tabs(kit.Tab("One", kit.Label("first")), kit.Tab("Two", kit.Label("second"))),
			kit.Menu(kit.MenuItem("Open"), kit.CheckMenuItem("Grid", true), kit.SubMenu("More", kit.MenuItem("Less"))),
			kit.Scroll(kit.Grid(2, kit.Label("a"), kit.Label("b"), kit.Label("c"))),
		),
		ui.Div(ui.Class("areas"),
			ui.H1(ui.Text("Title"), ui.Class("head")),
			ui.Div(ui.Class("side", "fancy"), ui.Text("side")),
			ui.Div(ui.Class("main", "moved"), ui.P(ui.Class("prose"),
				ui.Text("Text with "), ui.A(ui.Attr("href", "#"), ui.Text("a link")), ui.Em(ui.Text(" and ")),
				ui.Strong(ui.Text("weight")), ui.Sub(ui.Text("2")), ui.Sup(ui.Text("3")), ui.Q(ui.Text("a quote")),
			)),
		),
		ui.Ol(ui.Class("counted"), ui.Li(ui.Text("one")), ui.Li(ui.Text("two"), ui.Ul(ui.Li(ui.Text("inner"))))),
		ui.Details(ui.Open(), ui.Summary(ui.Text("More")), ui.Blockquote(ui.Class("cols"), ui.Text("Columns of text."))),
		ui.Table(ui.Caption(ui.Text("Sizes")),
			ui.Tr(ui.Th(ui.Text("w")), ui.Th(ui.Text("h"))),
			ui.Tr(ui.Td(ui.Text("1")), ui.Td(ui.Text("2"))),
		),
		ui.Div(ui.Class("float"), ui.Input(ui.Placeholder("search")), ui.Button(ui.Text("Go"))),
	)
	d.Layout(geom.V2(640, 480))
	require.Empty(t, d.Warnings())
	seen := map[*style.Computed]bool{}
	var out []*style.Computed
	add := func(styles ...*style.Computed) {
		for _, s := range styles {
			if s != nil && !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	var visit func(n *dom.Node) bool
	visit = func(n *dom.Node) bool {
		data := dom.Data(n)
		add(data.Style, data.Shown, data.Marker)
		if data.Style != nil {
			add(data.Style.Viewport)
		}
		for _, g := range []*dom.Node{data.Before, data.After, data.Shadow} {
			if g != nil {
				g.Walk(visit)
			}
		}
		return true
	}
	d.HTML().Walk(visit)
	return out
}

const corpusSheet = `
:root { --ink: #223; --pad: 6px; overflow: auto; }
.areas {
	display: grid; gap: var(--pad) 12px;
	grid-template-columns: repeat(2, minmax(80px, 1fr)) auto; grid-template-rows: auto 1fr;
	grid-template-areas: "head head head" "side main main"; grid-auto-rows: 20px; grid-auto-columns: min-content;
}
.head { grid-area: head; text-shadow: 1px 1px 2px #0008, 0 0 1px red; letter-spacing: 1px; text-transform: uppercase; }
.side { grid-area: side; order: 2; }
.main { grid-column: 2 / span 2; grid-row: 2; }
.fancy {
	background-image: linear-gradient(to right bottom, red, gold 40%, blue 80%); border: 2px dashed currentColor;
	border-radius: 4px 8px; box-shadow: 0 1px 3px #0006, inset 0 0 0 1px white; outline: 1px solid var(--ink);
	transition: background-color 0.2s ease-in 0.1s, opacity 1s; cursor: pointer; opacity: 0.9;
}
.moved {
	transform: translate(4px, 10%) rotate(3deg) scale(1.1); transform-origin: 0 100%;
	background-image: radial-gradient(circle closest-side at 30% 40%, white, black);
	position: relative; z-index: 2; top: 1px; overflow: hidden; user-select: text;
}
.prose { text-indent: 2em; line-height: 1.4; text-align: justify; quotes: "«" "»" "‹" "›"; }
.prose a { text-decoration: underline dotted red 2px; }
.prose a::after { content: " (" attr(href) ")"; color: var(--ink); }
.prose q::before { content: open-quote; }
.prose q::after { content: close-quote; }
.prose::selection { background-color: gold; color: black; }
.counted { counter-reset: item 3 other; list-style-type: upper-roman; }
.counted > li { counter-increment: item 2; counter-set: other 1; }
.counted > li::before { content: counter(item) ". " counters(item, "-", lower-alpha); font-weight: bold; }
.counted li::marker { color: teal; font-style: italic; }
.cols { column-count: 2; column-rule: 1px solid gray; column-gap: 20px; orphans: 3; widows: 1; }
.float { float: right; clear: both; width: calc(50% - 10px); min-height: 20px; max-width: 300px; aspect-ratio: 2; }
.float input::placeholder { color: silver; }
.float input { accent-color: green; caret-color: red; }
table { border-spacing: 2px 4px; caption-side: bottom; table-layout: fixed; }
`

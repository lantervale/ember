package ui

import (
	"math"
	"strconv"

	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/layout"
)

// machine is n's control, made from its attributes the first time it is
// needed; nil for an element without behaviour.
func (d *Document) machine(n *Node) any {
	if n == nil || n.IsText() {
		return nil
	}
	if m, ok := d.machines[n]; ok {
		return m
	}
	m := newMachine(n)
	if m != nil {
		if d.machines == nil {
			d.machines = map[*Node]any{}
		}
		d.machines[n] = m
	}
	return m
}

func has(n *Node, name string) bool { _, ok := n.Attr(name); return ok }

func number(n *Node, name string, or float64) float64 {
	if s, ok := n.Attr(name); ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	}
	return or
}

//nolint:funlen // one case per tag that has a control
func newMachine(n *Node) any {
	value, _ := n.Attr("value")
	disabled := has(n, "disabled")
	switch n.Tag() {
	case dom.Canvas, dom.Iframe:
		if m, ok := dom.Data(n).Canvas.(*control.Canvas); ok {
			return m
		}
		return &control.Canvas{}
	case dom.Button:
		return &control.Button{Disabled: disabled}
	case dom.Textarea:
		t := control.NewTextArea(n.Text())
		t.SetReadOnly(has(n, "readonly"))
		return t
	case dom.Details:
		return &control.Details{Open: has(n, "open")}
	case dom.Dialog:
		return &control.Dialog{Open: has(n, "open")}
	case dom.Select:
		opts, groups, selected := listOf(n)
		s := control.NewSelect(opts...)
		s.Groups = groups
		switch {
		case dom.ListBox(n):
			s.SetListBox(true, has(n, "multiple"), dom.ListBoxRows(n))
			s.SetPicked(selected...)
		case len(selected) > 0:
			s.Selected, s.Highlight = selected[len(selected)-1], selected[len(selected)-1]
		case has(n, "placeholder"):
			s.Selected, s.Highlight = -1, -1
		}
		s.Disabled = disabled
		return s
	case dom.Input:
		typ := inputType(n)
		var t *control.Text
		switch typ {
		case typeCheckbox:
			return &control.Checkbox{Checked: has(n, "checked"), Disabled: disabled}
		case typeRadio:
			return &control.Radio{Checked: has(n, "checked"), Disabled: disabled}
		case "range":
			r := &control.Range{
				Min: number(n, "min", 0), Max: number(n, "max", 100), Step: number(n, "step", 1), Disabled: disabled,
			}
			r.SetValue(number(n, "value", r.Min+(r.Max-r.Min)/2))
			return r
		case typeNumber:
			num := control.NewNumber(value)
			num.Min, num.Max, num.Step = number(n, "min", math.NaN()), number(n, "max", math.NaN()), number(n, "step", 1)
			if v, _ := n.Attr("step"); v == "any" {
				num.Step = 0
			}
			num.SetReadOnly(has(n, "readonly"))
			return num
		case typeButton, typeSubmit, typeReset:
			return &control.Button{Disabled: disabled}
		case typeSearch:
			t = control.NewSearch(value)
		case typePassword:
			t = control.NewPassword(value)
		default:
			t = control.NewText(value)
		}
		t.SetReadOnly(has(n, "readonly"))
		return t
	case dom.TextTag, dom.HTML, dom.Body, dom.Div, dom.Section, dom.Header, dom.Footer, dom.Nav, dom.Main, dom.Aside,
		dom.Article, dom.P, dom.H1, dom.H2, dom.H3, dom.H4, dom.Span, dom.Strong, dom.Em, dom.Code, dom.Label,
		dom.Option, dom.Summary, dom.Table, dom.Thead, dom.Tbody, dom.Tr, dom.Th, dom.Td, dom.Progress, dom.Meter,
		dom.Img, dom.Svg, dom.Hr, dom.Kbd, dom.Samp, dom.Pre, dom.Caption, dom.Tfoot,
		dom.Ul, dom.Ol, dom.Li, dom.Dl, dom.Dt, dom.Dd, dom.H5, dom.H6, dom.Blockquote, dom.Figure, dom.Figcaption,
		dom.Form, dom.Address, dom.Hgroup, dom.Menu, dom.Search, dom.A, dom.B, dom.I, dom.U, dom.S, dom.Small,
		dom.Abbr, dom.Cite, dom.Mark, dom.Time, dom.Var, dom.Del, dom.Ins, dom.Dfn, dom.Bdi, dom.Bdo, dom.DataElement,
		dom.Br,
		dom.Center, dom.Big, dom.Font, dom.Sub, dom.Sup, dom.Q, dom.Fieldset, dom.Legend,
		dom.Col, dom.Colgroup, dom.PictureElement, dom.Source, dom.Optgroup:
		// no control of their own
	}
	return nil
}

// listOf is what the select n offers: its options, its optgroups, each
// label before its options, which a disabled one disables, and the
// options its selected attributes choose.
func listOf(n *Node) ([]control.Option, []control.Group, []int) {
	var opts []control.Option
	var groups []control.Group
	var selected []int
	add := func(o *Node, group int, off bool) {
		v, valued := o.Attr("value")
		opts = append(opts, control.Option{
			Label: o.Text(), Value: v, Valued: valued, Disabled: off || has(o, "disabled"), Group: group,
		})
		if has(o, "selected") {
			selected = append(selected, len(opts)-1)
		}
	}
	for _, c := range n.Children() {
		if c.Tag() == dom.Option {
			add(c, 0, false)
		}
		if c.Tag() != dom.Optgroup {
			continue
		}
		label, _ := c.Attr("label")
		groups = append(groups, control.Group{Label: label, Disabled: has(c, "disabled"), At: len(opts)})
		for _, o := range c.Children() {
			if o.Tag() == dom.Option {
				add(o, len(groups), has(c, "disabled"))
			}
		}
	}
	return opts, groups, selected
}

// buttonLabel is what an input that is a button shows: its value, else
// what Chrome says of a submit or a reset.
func buttonLabel(n *Node) string {
	if v, ok := n.Attr("value"); ok {
		return v
	}
	switch t, _ := n.Attr("type"); t {
	case typeSubmit:
		return "Submit"
	case typeReset:
		return "Reset"
	}
	return ""
}

// textOf is a text field's editing machine, a number's included.
func textOf(m any) *control.Text {
	switch m := m.(type) {
	case *control.Text:
		return m
	case *control.Number:
		return &m.Text
	}
	return nil
}

// sync shows n's machine in its state, attributes and shadow text.
func (d *Document) sync(n *Node, m any) {
	disabled := disabledField(n)
	n.SetState(dom.Disabled, disabled)
	switch m := m.(type) {
	case *control.Button:
		m.Disabled = disabled
		if n.Tag() == dom.Input {
			n.SetShadow(buttonLabel(n), false)
		}
	case *control.Range:
		m.Disabled = disabled
	}
	switch m := m.(type) {
	case *control.Checkbox:
		m.Disabled = disabled
		n.SetState(dom.Checked, m.Checked)
	case *control.Radio:
		m.Disabled = disabled
		radioState(n, m.Checked)
	case *control.Dialog:
		n.SetState(dom.Open, m.Open)
		n.SetState(dom.Modal, m.Open && m.Modal)
		if m.Open {
			n.SetAttr("open", "")
		} else {
			n.RemoveAttr("open")
		}
	case *control.Details:
		n.SetState(dom.Open, m.Open)
		if m.Open {
			n.SetAttr("open", "")
		} else {
			n.RemoveAttr("open")
		}
	case *control.Select:
		m.Disabled = disabled
		m.SetListBox(dom.ListBox(n), has(n, "multiple"), dom.ListBoxRows(n))
		n.SetState(dom.Open, m.Open)
		d.showOptions(n, m)
		if m.ListBox {
			break
		}
		if m.Selected >= 0 && m.Selected < len(m.Options) {
			n.SetShadow(m.Options[m.Selected].Label, false)
		} else {
			p, _ := n.Attr("placeholder")
			n.SetShadow(p, true)
		}
	}
	if t := textOf(m); t != nil {
		t.SetReadOnly(has(n, "readonly"))
		if _, num := m.(*control.Number); !num {
			t.SetMaxLength(max(int(number(n, "maxlength", 0)), 0))
		}
		if t.Empty() {
			p, _ := n.Attr("placeholder")
			n.SetShadow(p, true)
		} else {
			shown, _ := d.shownText(n, t)
			n.SetShadow(shown, false)
		}
	}
}

// Value is a control's value now: a text field's or textarea's text, the
// selected option's value, a range's number, "on" for a checked checkbox,
// a button's value attribute.
func (d *Document) Value(n *Node) string {
	switch m := d.machine(n).(type) {
	case *control.Button:
		v, _ := n.Attr("value")
		return v
	case *control.Checkbox:
		if m.Checked {
			return "on"
		}
		return ""
	case *control.Radio:
		if !m.Checked {
			return ""
		}
		if v, ok := n.Attr("value"); ok {
			return v
		}
		return "on"
	case *control.Select:
		return m.Value()
	case *control.Range:
		return strconv.FormatFloat(m.Value, 'g', -1, 64)
	}
	if t := textOf(d.machine(n)); t != nil {
		return t.Value()
	}
	return ""
}

// SetValue sets a control's value, as the game does, without events. A
// focused text field keeps what the player is typing, until it changes or
// is submitted: then it takes the game's answer, as a chat clearing its
// field on Enter. A callback of the field's own events (OnKey, OnInput…)
// sets it at once, its caret at the end, as a console walking its history
// on Up (decision 0142).
func (d *Document) SetValue(n *Node, v string) {
	switch m := d.machine(n).(type) {
	case *control.Select:
		var found []int
		for i, o := range m.Options {
			if o.Value == v || o.Value == "" && !o.Valued && o.Label == v {
				m.Selected, m.Highlight = i, i
				found = append(found, i)
			}
		}
		if m.ListBox {
			// as the DOM's value: the first option of that value alone, none for no such
			m.SetPicked(found[:min(len(found), 1)]...)
		}
	case *control.Range:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			m.SetValue(f)
		}
	case *control.Checkbox:
		m.Checked = v != ""
	default:
		if t := textOf(m); t != nil && (d.focused() != n || d.typing != n || d.answering == n) {
			if d.typing == n {
				d.typing = nil
			}
			t.SetValue(v)
			d.used(n, set)
			d.uses[n] &^= edited
		}
	}
	d.sync(n, d.machine(n))
}

// SelectionRange is where a text field's selection begins and ends, in
// characters of its value, as the DOM's selectionStart and selectionEnd:
// both the caret's place when nothing is selected, and 0 for what is no
// text field.
func (d *Document) SelectionRange(n *Node) (int, int) {
	if t := textOf(d.machine(n)); t != nil {
		return t.Editor.Selection()
	}
	return 0, 0
}

// SetSelectionRange selects a text field's characters from from up to to,
// kept inside its value, as the DOM's setSelectionRange; the caret stands
// at to. It moves no focus and emits no event.
func (d *Document) SetSelectionRange(n *Node, from, to int) {
	if t := textOf(d.machine(n)); t != nil {
		t.Editor.Select(from, to)
	}
}

// Checked reports whether a checkbox or a radio is checked.
func (d *Document) Checked(n *Node) bool {
	if isMenuItem(n) {
		v, _ := n.Attr("aria-checked")
		return v == attrTrue
	}
	if d.isRadio(n) {
		return d.radioChecked(n)
	}
	c, ok := d.machine(n).(*control.Checkbox)
	return ok && c.Checked
}

// SetChecked checks or unchecks a checkbox, or a radio (checking one
// unchecks the rest of its group), without events.
func (d *Document) SetChecked(n *Node, on bool) {
	if roleOf(n) == roleMenuItemRadio && on {
		for _, o := range menuGroup(n) {
			o.SetAttr("aria-checked", strconv.FormatBool(o == n))
		}
		return
	}
	if isMenuItem(n) {
		n.SetAttr("aria-checked", strconv.FormatBool(on))
		return
	}
	if d.isRadio(n) {
		if on {
			for _, o := range d.radioGroup(n) {
				d.setRadio(o, o == n)
			}
		} else {
			d.setRadio(n, false)
		}
		return
	}
	if c, ok := d.machine(n).(*control.Checkbox); ok {
		c.Checked, c.Indeterminate = on, false
		d.sync(n, c)
	}
}

// Selected is a select's chosen option, a tablist's selected tab, a
// grid's selected row or a listbox's selected option, -1 for none.
func (d *Document) Selected(n *Node) int {
	if s, ok := d.machine(n).(*control.Select); ok {
		return s.Selected
	}
	if roleOf(n) == roleTablist {
		return tabIndexOf(n)
	}
	if isGrid(n) {
		return selectedRow(n)
	}
	if isListbox(n) {
		return selectedOption(n)
	}
	return -1
}

// SetSelected chooses a select's option i, -1 for none, or selects a
// tablist's tab i, a grid's row i (-1 for none) or a listbox's option
// i, without events.
func (d *Document) SetSelected(n *Node, i int) {
	if s, ok := d.machine(n).(*control.Select); ok {
		s.SetPicked(i)
		d.sync(n, s)
		return
	}
	if tabs := tabsOf(n); roleOf(n) == roleTablist && i >= 0 && i < len(tabs) {
		d.selectTab(tabs[i])
	}
	if rows := gridRows(n); isGrid(n) && i < len(rows) {
		if i < 0 {
			for _, r := range rows {
				r.SetAttr("aria-selected", "false")
			}
		} else {
			selectRow(n, rows[i])
			d.revealing = rows[i]
		}
	}
	if opts := listOptions(n); isListbox(n) && i >= 0 && i < len(opts) {
		selectOption(n, opts[i])
	}
}

// SelectedOptions are a select's options selected, in order, as the
// DOM's selectedOptions: a listbox's every one (multiple or a size above
// 1), else the chosen one; nil for none or for what is no select.
func (d *Document) SelectedOptions(n *Node) []int {
	s, ok := d.machine(n).(*control.Select)
	if !ok {
		return nil
	}
	var out []int
	for i := range s.Options {
		if s.Picked(i) {
			out = append(out, i)
		}
	}
	return out
}

// SetSelectedOptions selects a select's options is alone, without
// events: all of them in a select with multiple, else the last.
func (d *Document) SetSelectedOptions(n *Node, is ...int) {
	if s, ok := d.machine(n).(*control.Select); ok {
		s.SetPicked(is...)
		d.sync(n, s)
	}
}

// Values are what a form submits for the select n, as the DOM's
// FormData holds them: the value of each option selected and enabled, in
// order; none for a disabled select.
func (d *Document) Values(n *Node) []string {
	s, ok := d.machine(n).(*control.Select)
	if !ok || disabledField(n) {
		return nil
	}
	var out []string
	for _, i := range d.SelectedOptions(n) {
		if !s.Options[i].Disabled {
			out = append(out, s.Options[i].FormValue())
		}
	}
	return out
}

// showOptions gives a select's options their labels as the player reads
// them, which its list shows and typing matches, and a listbox's
// optgroups their label rows.
func (d *Document) showOptions(n *Node, m *control.Select) {
	for i, c := range dom.Options(n) {
		if i >= len(m.Options) {
			break
		}
		m.Options[i].Shown = ""
		if d.translator != nil {
			m.Options[i].Shown = say(c, d.shown)
		}
		g := c.Parent()
		c.SetState(dom.Checked, m.Picked(i))
		c.SetState(dom.Default, has(c, "selected"))
		c.SetState(dom.Disabled, has(c, "disabled") || g.Tag() == dom.Optgroup && has(g, "disabled"))
	}
	i := 0
	for _, g := range n.Children() {
		if g.Tag() != dom.Optgroup || i >= len(m.Groups) {
			continue
		}
		label, _ := g.Attr("label")
		m.Groups[i].Shown = ""
		if d.translator != nil {
			m.Groups[i].Shown = layout.TranslatedAt(g, label, d.tr())
		}
		g.SetState(dom.Disabled, has(g, "disabled"))
		if m.ListBox {
			g.SetShadow(label, false)
		}
		i++
	}
}

package ui

import (
	"math"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
)

// fieldUse is what the player did to a field: edited is set once they
// typed in it since the program last set its value, which minlength and
// maxlength wait for, and changed once they changed it (:user-valid).
type fieldUse uint8

const (
	edited fieldUse = 1 << iota
	changed
)

// formStates are the states validate keeps, all of them ui's own.
const formStates = dom.Indeterminate | dom.PlaceholderShown | dom.Default | dom.Valid | dom.Invalid | dom.UserValid |
	dom.UserInvalid | dom.InRange | dom.OutOfRange

// radioSet is what validate gathers of a named radio group: whether one
// of it is checked, and one required.
type radioSet struct{ checked, required bool }

// radioKey names a radio group: its radios' form owner and name.
type radioKey struct {
	form *Node
	name string
}

// radioKeyOf is the group of the named radio n.
func radioKeyOf(n *Node) radioKey {
	name, _ := n.Attr("name")
	return radioKey{formOwner(n), name}
}

// validity is what validate keeps from one frame to the next.
type validity struct {
	// forms are the forms and fieldsets the last walk of the tree found,
	// in tree order.
	forms []*Node
	uses  map[*Node]fieldUse
	// patterns are the pattern attributes compiled, nil for one Go cannot
	// read, which then constrains nothing.
	patterns map[string]*regexp.Regexp
	groups   map[radioKey]radioSet
	invalid  map[*Node]bool
	defaults map[*Node]bool
}

// used notes that the player did u to the field n.
func (d *Document) used(n *Node, u fieldUse) {
	if d.uses == nil {
		d.uses = map[*Node]fieldUse{}
	}
	d.uses[n] |= u
}

// validate gives the controls, forms and fieldsets the states of their
// values: :valid and :invalid by constraint validation as Chrome runs it,
// :in-range, :placeholder-shown, :indeterminate, :default, and a
// fieldset's :disabled.
func (d *Document) validate() {
	if d.groups == nil {
		d.groups, d.invalid, d.defaults = map[radioKey]radioSet{}, map[*Node]bool{}, map[*Node]bool{}
	}
	clear(d.groups)
	clear(d.invalid)
	clear(d.defaults)
	for _, n := range d.controls {
		if r, ok := d.machines[n].(*control.Radio); ok {
			if has(n, "name") {
				k := radioKeyOf(n)
				g := d.groups[k]
				g.checked, g.required = g.checked || r.Checked, g.required || has(n, "required")
				d.groups[k] = g
			}
		}
	}
	for _, n := range d.controls {
		s := d.fieldStates(n, d.machines[n], d.groupOf)
		if s&dom.Invalid != 0 {
			for p := n.Parent(); p != nil; p = p.Parent() {
				d.invalid[p] = p.Tag() != dom.Form || d.invalid[p]
			}
			if f := formOwner(n); f != nil {
				d.invalid[f] = true
			}
		}
		if submits(n) {
			if f := formOwner(n); f != nil && !d.defaults[f] {
				d.defaults[f] = true
				s |= dom.Default
			}
		}
		n.SetState(formStates&^s, false)
		n.SetState(s, true)
	}
	for _, f := range d.forms {
		f.SetState(dom.Valid, !d.invalid[f])
		f.SetState(dom.Invalid, d.invalid[f])
		if f.Tag() == dom.Fieldset {
			f.SetState(dom.Disabled, disabledField(f))
		}
	}
}

// groupOf is the group of the named radio n as validate gathered it.
func (d *Document) groupOf(n *Node) radioSet { return d.groups[radioKeyOf(n)] }

// groupNow is the group of the named radio n as it stands, read from the
// tree.
func (d *Document) groupNow(n *Node) radioSet {
	var g radioSet
	k := radioKeyOf(n)
	d.html.Walk(func(c *Node) bool {
		if r, ok := d.machine(c).(*control.Radio); ok {
			if has(c, "name") && radioKeyOf(c) == k {
				g.checked, g.required = g.checked || r.Checked, g.required || has(c, "required")
			}
		}
		return !c.IsText()
	})
	return g
}

// fieldStates are the states the value of the control n, of machine m,
// gives it, but :default for a submit button; group reads a named radio
// group.
func (d *Document) fieldStates(n *Node, m any, group func(*Node) radioSet) dom.State {
	var s dom.State
	kind := inputType(n)
	switch m := m.(type) {
	case *control.Checkbox:
		if m.Indeterminate {
			s |= dom.Indeterminate
		}
		if has(n, "checked") {
			s |= dom.Default
		}
	case *control.Radio:
		checked := m.Checked
		if has(n, "name") {
			checked = group(n).checked
		}
		if !checked {
			s |= dom.Indeterminate
		}
		if has(n, "checked") {
			s |= dom.Default
		}
	}
	if t := textOf(m); t != nil && t.Empty() && placeholds(n, kind) && has(n, "placeholder") {
		s |= dom.PlaceholderShown
	}
	if !d.validates(n, kind) {
		return s
	}
	valid, invalid := dom.Valid, dom.Invalid
	if d.uses[n]&changed != 0 {
		valid, invalid = valid|dom.UserValid, invalid|dom.UserInvalid
	}
	if d.mismatch(n, m, kind, group) {
		s |= invalid
	} else {
		s |= valid
	}
	if num, ok := m.(*control.Number); ok {
		text := num.Value()
		v, err := strconv.ParseFloat(text, 64)
		switch {
		case err != nil || !validFloat(text):
			s |= dom.InRange
		case !math.IsNaN(num.Min) && v < num.Min || !math.IsNaN(num.Max) && v > num.Max:
			s |= dom.OutOfRange
		case !math.IsNaN(num.Min) || !math.IsNaN(num.Max):
			s |= dom.InRange
		}
	}
	if _, ok := m.(*control.Range); ok {
		s |= dom.InRange
	}
	return s
}

// validates reports whether constraint validation reads n: a field, or
// a submit button, neither disabled nor read-only.
func (d *Document) validates(n *Node, kind string) bool {
	reads := n.Tag() == dom.Select || n.Tag() == dom.Textarea || submits(n)
	if n.Tag() == dom.Input {
		reads = kind != typeHidden && kind != typeReset && kind != typeButton
	}
	return reads && !disabledField(n) && !has(n, "readonly")
}

// mismatch reports whether the value of n, a field constraint validation
// reads, breaks one of its constraints.
func (d *Document) mismatch(n *Node, m any, kind string, group func(*Node) radioSet) bool {
	req := has(n, "required")
	switch m := m.(type) {
	case *control.Button, *control.Range:
		return false
	case *control.Checkbox:
		return req && !m.Checked
	case *control.Radio:
		if has(n, "name") {
			g := group(n)
			return g.required && !g.checked
		}
		return req && !m.Checked
	case *control.Select:
		return req && (m.Selected < 0 || m.Selected == 0 && placeholderOption(n))
	}
	t := textOf(m)
	if t == nil || t.Empty() {
		return req && t != nil
	}
	if kind == typeNumber {
		return d.numberMismatch(n, t.Value())
	}
	return d.textMismatch(n, t, kind)
}

// textMismatch reports whether t, the text of the field n holding some,
// is too short or too long since the player edited it, or breaks its
// type or its pattern.
func (d *Document) textMismatch(n *Node, t *control.Text, kind string) bool {
	if d.uses[n]&edited != 0 && (n.Tag() == dom.Textarea || patterned(kind)) {
		size := 0
		for _, r := range t.Editor.Runes() {
			size += utf16.RuneLen(r)
		}
		lo, short := intAttr(n, "minlength")
		hi, long := intAttr(n, "maxlength")
		if short && size < lo || long && size > hi {
			return true
		}
	}
	p, pat := n.Attr("pattern")
	pat = pat && n.Tag() == dom.Input && patterned(kind)
	if !pat && kind != typeEmail && kind != typeURL {
		return false
	}
	values := []string{t.Value()}
	if kind == typeEmail {
		values[0] = strings.TrimSpace(strings.NewReplacer("\n", "", "\r", "").Replace(values[0]))
		if has(n, "multiple") {
			values = strings.Split(values[0], ",")
			for i := range values {
				values[i] = strings.TrimSpace(values[i])
			}
		}
	}
	re := d.pattern(p)
	for _, v := range values {
		if kind == typeEmail && !emailAddress.MatchString(v) || kind == typeURL && !validURL(v) ||
			pat && re != nil && !re.MatchString(v) {
			return true
		}
	}
	return false
}

// numberMismatch reports whether v, a number input's value, is no
// number, or out of its min and max, or off its steps.
func (d *Document) numberMismatch(n *Node, v string) bool {
	if !validFloat(v) {
		return true
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return true
	}
	lo, hasLo := floatAttr(n, "min")
	hi, hasHi := floatAttr(n, "max")
	if hasLo && f < lo || hasHi && f > hi {
		return true
	}
	if s, _ := n.Attr("step"); strings.EqualFold(s, "any") {
		return false
	}
	step, ok := floatAttr(n, "step")
	if !ok || step <= 0 {
		step = 1
	}
	base := lo
	if !hasLo {
		base, _ = floatAttr(n, "value")
	}
	k := (f - base) / step
	return math.Abs(k-math.Round(k)) > 1e-9*max(1, math.Abs(k))
}

// pattern is the pattern attribute p compiled to match a whole value.
func (d *Document) pattern(p string) *regexp.Regexp {
	if re, ok := d.patterns[p]; ok {
		return re
	}
	if d.patterns == nil {
		d.patterns = map[string]*regexp.Regexp{}
	}
	re, err := regexp.Compile("^(?:" + p + ")$")
	if err != nil {
		re = nil
	}
	d.patterns[p] = re
	return re
}

// Valid reports whether n meets its constraints, as the DOM's
// checkValidity: a field's value its required, pattern, min, max, step,
// minlength, maxlength and type; a form or a fieldset whether every
// control in it does. A disabled or read-only control, a button that
// submits nothing, and what is no control are valid.
func (d *Document) Valid(n *Node) bool {
	if n.Tag() == dom.Form || n.Tag() == dom.Fieldset {
		valid := true
		n.Walk(func(c *Node) bool {
			valid = valid && (c == n || c.IsText() || d.Valid(c))
			return valid && !c.IsText()
		})
		return valid
	}
	m := d.machine(n)
	return m == nil || d.fieldStates(n, m, d.groupNow)&dom.Invalid == 0
}

// Indeterminate reports whether a checkbox shows neither checked nor
// unchecked (the DOM's indeterminate).
func (d *Document) Indeterminate(n *Node) bool {
	c, ok := d.machine(n).(*control.Checkbox)
	return ok && c.Indeterminate
}

// SetIndeterminate shows a checkbox neither checked nor unchecked, as a
// "select all" over a list partly selected, until it is clicked or
// checked; it keeps its checked state.
func (d *Document) SetIndeterminate(n *Node, on bool) {
	if c, ok := d.machine(n).(*control.Checkbox); ok {
		c.Indeterminate = on
		d.sync(n, c)
	}
}

// disabledField reports whether n is disabled: by its own disabled
// attribute, or a control in a disabled fieldset but in its first legend.
func disabledField(n *Node) bool {
	if has(n, "disabled") {
		return true
	}
	if !slices.Contains(fieldsetted, n.Tag()) {
		return false
	}
	for c, p := n, n.Parent(); p != nil; c, p = p, p.Parent() {
		if p.Tag() == dom.Fieldset && has(p, "disabled") && c != firstLegend(p) {
			return true
		}
	}
	return false
}

// fieldsetted are the elements a disabled fieldset disables.
//
//nolint:gochecknoglobals // lookup table
var fieldsetted = []dom.Tag{dom.Button, dom.Input, dom.Select, dom.Textarea, dom.Fieldset}

// firstLegend is the fieldset f's first legend child.
func firstLegend(f *Node) *Node {
	for _, c := range f.Children() {
		if c.Tag() == dom.Legend {
			return c
		}
	}
	return nil
}

// formOwner is the form n belongs to: the one its form attribute names,
// none when that is no form, else the nearest above it.
func formOwner(n *Node) *Node {
	if id, ok := n.Attr("form"); ok {
		root := n
		for root.Parent() != nil {
			root = root.Parent()
		}
		var owner *Node
		root.Walk(func(c *Node) bool {
			if owner == nil && !c.IsText() && c.ID() == id {
				owner = c
			}
			return owner == nil
		})
		if owner != nil && owner.Tag() == dom.Form {
			return owner
		}
		return nil
	}
	return formOf(n)
}

// formOf is the form above n, the nearest.
func formOf(n *Node) *Node {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Tag() == dom.Form {
			return p
		}
	}
	return nil
}

// submits reports whether n is a submit button.
func submits(n *Node) bool {
	if n.Tag() == dom.Input {
		t := inputType(n)
		return t == typeSubmit || t == typeImage
	}
	t, _ := n.Attr("type")
	t = strings.ToLower(t)
	return n.Tag() == dom.Button && t != typeButton && t != typeReset
}

// inputType is an input's type in lower case, text for one HTML does not
// know.
func inputType(n *Node) string {
	t, _ := n.Attr("type")
	if t = strings.ToLower(t); slices.Contains(inputTypes, t) {
		return t
	}
	return typeText
}

//nolint:gochecknoglobals // lookup table
var inputTypes = strings.Fields("button checkbox color date datetime-local email file hidden image month number " +
	"password radio range reset search submit tel text time url week")

// patterned reports whether an input of type kind takes pattern,
// minlength and maxlength.
func patterned(kind string) bool {
	switch kind {
	case typeText, typeSearch, typeURL, "tel", typeEmail, typePassword:
		return true
	}
	return false
}

// placeholds reports whether n shows a placeholder.
func placeholds(n *Node, kind string) bool {
	return n.Tag() == dom.Textarea || n.Tag() == dom.Input && (patterned(kind) || kind == typeNumber)
}

// placeholderOption reports whether the select n's first option is its
// placeholder, of no value, which a required select may not keep.
func placeholderOption(n *Node) bool {
	if has(n, "multiple") || number(n, "size", 1) > 1 {
		return false
	}
	for _, c := range n.Children() {
		if c.Tag() == dom.Option {
			v, ok := c.Attr("value")
			if !ok {
				v = strings.TrimSpace(c.Text())
			}
			return v == ""
		}
	}
	return false
}

func floatAttr(n *Node, name string) (float64, bool) {
	s, ok := n.Attr(name)
	if !ok || !validFloat(s) {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// intAttr is a non-negative integer attribute such as minlength.
func intAttr(n *Node, name string) (int, bool) {
	s, ok := n.Attr(name)
	if !ok {
		return 0, false
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	return v, err == nil && v >= 0
}

// validFloat reports whether s is a number as HTML writes one.
func validFloat(s string) bool { return htmlFloat.MatchString(s) }

var (
	htmlFloat    = regexp.MustCompile(`^-?(?:[0-9]+(?:\.[0-9]+)?|\.[0-9]+)(?:[eE][-+]?[0-9]+)?$`)
	emailAddress = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?" +
		`(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
)

// validURL reports whether s is an absolute URL, as type=url asks.
func validURL(s string) bool {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil || u.Scheme == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "ws", "wss", "ftp":
		return u.Host != ""
	}
	return true
}

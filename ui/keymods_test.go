package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/ui"
)

// mods are the modifiers a key event says were held.
type mods struct{ Shift, Alt, Control, Super, Command bool }

func modsOf(e ui.Event) mods {
	return mods{Shift: e.Shift, Alt: e.Alt, Control: e.Control, Super: e.Super, Command: e.Command}
}

func pressedMods(e ui.KeyPressed) mods {
	return mods{Shift: e.Shift, Alt: e.Alt, Control: e.Control, Super: e.Super, Command: e.Command}
}

type modifierCase struct {
	name string
	keys []input.Key
	mac  mods
	pc   mods
}

func modifierCases() []modifierCase {
	return []modifierCase{
		{"none", nil, mods{}, mods{}},
		{"shift", []input.Key{input.KeyLeftShift}, mods{Shift: true}, mods{Shift: true}},
		{"right shift", []input.Key{input.KeyRightShift}, mods{Shift: true}, mods{Shift: true}},
		{"alt", []input.Key{input.KeyLeftAlt}, mods{Alt: true}, mods{Alt: true}},
		{"control", []input.Key{input.KeyRightControl}, mods{Control: true}, mods{Control: true, Command: true}},
		{"super", []input.Key{input.KeyLeftSuper}, mods{Super: true, Command: true}, mods{Super: true}},
		{"all", []input.Key{input.KeyLeftShift, input.KeyRightAlt, input.KeyLeftControl, input.KeyRightSuper},
			mods{Shift: true, Alt: true, Control: true, Super: true, Command: true},
			mods{Shift: true, Alt: true, Control: true, Super: true, Command: true}},
	}
}

// pressWith holds the modifiers, presses key and lets everything go.
func (l *live) pressWith(modifiers []input.Key, key input.Key) {
	pressWith(l, l.frames, modifiers, key)
}

func pressWith(l *live, frames func(int), modifiers []input.Key, key input.Key) {
	in := l.win.Input()
	for _, m := range modifiers {
		in.SetKey(m, true)
	}
	frames(1)
	in.SetKey(key, true)
	frames(1)
	in.SetKey(key, false)
	for _, m := range modifiers {
		in.SetKey(m, false)
	}
	frames(2)
}

func eachPlatform(t *testing.T, run func(t *testing.T, want func(mac, pc mods) mods)) {
	for _, p := range []textedit.Platform{textedit.Mac, textedit.Others} {
		name := "mac"
		if p != textedit.Mac {
			name = "other"
		}
		t.Run(name, func(t *testing.T) {
			ui.SetPlatform(t, p)
			run(t, func(mac, pc mods) mods {
				if p == textedit.Mac {
					return mac
				}
				return pc
			})
		})
	}
}

func TestAFocusedFieldsOnKeyHearsEachModifierHeld(t *testing.T) {
	for _, field := range []struct {
		name string
		make func(...ui.Arg) *ui.Node
	}{{"input", ui.Input}, {"textarea", ui.Textarea}} {
		t.Run(field.name, func(t *testing.T) {
			eachPlatform(t, func(t *testing.T, want func(mac, pc mods) mods) {
				for _, c := range modifierCases() {
					t.Run(c.name, func(t *testing.T) {
						var heard []ui.Event
						n := field.make(ui.OnKey(func(e ui.Event) {
							if e.Key == input.KeyK {
								heard = append(heard, e)
							}
						}))
						l := liveDoc(t, n)
						l.focusField(t, n)
						l.pressWith(c.keys, input.KeyK)
						require.Len(t, heard, 1)
						assert.Equal(t, want(c.mac, c.pc), modsOf(heard[0]))
					})
				}
			})
		})
	}
}

func TestAKeyOnAPlainElementCarriesEachModifierHeld(t *testing.T) {
	eachPlatform(t, func(t *testing.T, want func(mac, pc mods) mods) {
		for _, c := range modifierCases() {
			t.Run(c.name, func(t *testing.T) {
				var heard []ui.Event
				b := ui.Button(ui.Text("Save"), ui.OnKey(func(e ui.Event) { heard = append(heard, e) }))
				l := liveDoc(t, b)
				k := &keyed{l: l}
				l.doc.Focus(b)
				k.frames(1)
				pressWith(l, k.frames, c.keys, input.KeyK)
				require.Len(t, k.got, 1)
				p := k.got[0]
				assert.Equal(t, input.KeyK, p.Key)
				assert.Equal(t, want(c.mac, c.pc), pressedMods(p))
				require.Len(t, heard, 1)
				assert.Equal(t, want(c.mac, c.pc), modsOf(heard[0]))
			})
		}
	})
}

func TestShiftEnterInATextareaIsHeardWithShift(t *testing.T) {
	var heard []ui.Event
	area := ui.Textarea(ui.OnKey(func(e ui.Event) {
		if e.Key == input.KeyEnter {
			heard = append(heard, e)
		}
	}))
	l := liveDoc(t, area)
	l.focusField(t, area)
	l.pressWith([]input.Key{input.KeyLeftShift}, input.KeyEnter)
	l.pressWith(nil, input.KeyEnter)
	require.Len(t, heard, 2)
	assert.True(t, heard[0].Shift)
	assert.False(t, heard[1].Shift)
}

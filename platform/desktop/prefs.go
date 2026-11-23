package desktop

// syncSystem records what the system decides as a frame begins: each
// window's title bar, and the user's preferences.
func (d *driver) syncSystem() {
	for _, n := range d.natives {
		n.syncTitleBar()
	}
	d.syncPreferences()
}

// syncPreferences keeps the World's preferences as the system has them,
// read again only when the system says they changed, or, where it says
// nothing, when one of the program's windows gains the focus. A capture or
// a hidden run asks nothing, so it plays the same on any machine, and
// keeps what the program simulates.
func (d *driver) syncPreferences() {
	if d.scripted {
		return
	}
	if systemChanged(d.focusGained()) {
		d.system = systemPreferences()
	}
	d.tellPreferences()
}

// focusGained reports whether one of the program's windows has the focus
// where none had it the frame before.
func (d *driver) focusGained() bool {
	focused := false
	for _, n := range d.natives {
		focused = focused || n.window.Focused()
	}
	gained := focused && !d.focused
	d.focused = focused
	return gained
}

func (d *driver) tellPreferences() {
	s := d.system
	for e := range d.preferences.All() {
		p := e.GetPreferences()
		p.SyncAppearance(s.Appearance)
		p.SyncMotion(s.Motion)
		p.SyncContrast(s.Contrast)
		p.SyncForcedColors(s.ForcedColors)
		p.SyncHighlight(s.Highlight)
		p.SyncLanguages(s.Languages...)
	}
}

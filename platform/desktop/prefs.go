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
// read again only when the system says they changed. A capture or a
// hidden run asks nothing, so it plays the same on any machine, and keeps
// what the program simulates.
func (d *driver) syncPreferences() {
	if d.scripted {
		return
	}
	if systemChanged(&d.asked) {
		d.appearance, d.languages = systemAppearance(), systemLanguages()
	}
	d.tellPreferences()
}

func (d *driver) tellPreferences() {
	for e := range d.preferences.All() {
		p := e.GetPreferences()
		p.SyncAppearance(d.appearance)
		p.SyncLanguages(d.languages...)
	}
}

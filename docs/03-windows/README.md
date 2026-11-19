# Windows

A window is an entity like any other, made with `platform.NewWindow`: add it to the World to open it, and remove it to take it away with its children. Its close button asks, and the window's `platform.Closing` rule answers: by default the primary window's close ends the program and another's closes that window ([decision 0102](../99-decisions/0102-a-windows-close-is-a-request-its-rule-answers.md)). The first window added is the primary one, then the oldest left once it is gone, and `ember.New` opens a default window only when the game adds none. Each window has its own cameras and its own UI, and shows the camera with the highest `Priority` among those that target it: a camera or a UI element that is the window's child shows in that window, and reads its input, and one that belongs to no window shows in the primary one, unless a camera names its window with `In`.

## The system's appearance and languages

What the user chose on their system is an entity as well, `platform.NewPreferences()`, which the program adds to the World and keeps:

```go
prefs := platform.NewPreferences()
w := ember.New().AddEntities(win, prefs)
```

`prefs.Appearance()` is `platform.AppearanceLight` or `platform.AppearanceDark`, and `prefs.Languages()` the user's languages as BCP 47 tags such as `fr-FR`, the one they prefer first. The platform keeps both up to date while the program runs, and the World emits `platform.AppearanceChanged` and `platform.LanguagesChanged` as the next frame begins, once per change. The first time a value is known is a change too, so a program that only reads the events starts right:

```go
type Theme struct {
	changed world.Reader[platform.AppearanceChanged]
}

func (t *Theme) Update(f world.Frame) {
	for e := range t.changed.Read(f.World) {
		dark := e.Appearance == platform.AppearanceDark
		…
	}
}
```

Before the program runs, and in a test, a server, a hidden run or a capture, nothing is known: the appearance is `platform.AppearanceUnknown` and the languages are empty, so the program needs a look and a language of its own to fall back on. A test says what the system would with `prefs.SyncAppearance(platform.AppearanceDark)` and `prefs.SyncLanguages("fr-FR", "en")`, then runs a frame. macOS and Windows say both; Linux says its languages, from `LANGUAGE`, `LC_ALL`, `LC_MESSAGES` and `LANG`, and not its appearance; a page in a browser says neither yet ([decision 0108](../99-decisions/0108-a-program-follows-the-systems-appearance-and-languages.md)).

## Where they draw

The windows are where the renderer, one of the World's [presenters](../02-world/README.md#presenters), draws; a headless server opens none.

# Windows

A window is an entity like any other, made with `platform.NewWindow`: add it to the World to open it, and remove it to take it away with its children. Its close button asks, and the window's `platform.Closing` rule answers: by default the primary window's close ends the program and another's closes that window ([decision 0102](../99-decisions/0102-a-windows-close-is-a-request-its-rule-answers.md)). The first window added is the primary one, then the oldest left once it is gone, and `ember.New` opens a default window only when the game adds none. Each window has its own cameras and its own UI, and shows the camera with the highest `Priority` among those that target it: a camera or a UI element that is the window's child shows in that window, and reads its input, and one that belongs to no window shows in the primary one, unless a camera names its window with `In`.

## The system's appearance and languages

What the user chose on their system is an entity as well, `platform.NewPreferences()`, which the program adds to the World and keeps:

```go
prefs := platform.NewPreferences()
w := ember.New().AddEntities(win, prefs)
```

`prefs.Appearance()` is `platform.AppearanceLight` or `platform.AppearanceDark`, `prefs.Motion()` is `platform.MotionReduced` where the user asked the system to reduce motion and `platform.MotionFull` where not, `prefs.Contrast()` is `platform.ContrastMore`, `ContrastLess`, `ContrastCustom` or `ContrastNoPreference`, `prefs.ForcedColors()` is `platform.ForcedColorsActive` where a contrast theme forces its colours on programs and `platform.ForcedColorsNone` where not, `prefs.Highlight()` the colours the system selects text in (`Background`, `Inactive` for an unfocused window, `Text`; a colour of no alpha is one it does not say), and `prefs.Languages()` the user's languages as BCP 47 tags such as `fr-FR`, the one they prefer first. The platform keeps them up to date while the program runs, and the World emits `platform.AppearanceChanged`, `platform.MotionChanged`, `platform.ContrastChanged`, `platform.ForcedColorsChanged`, `platform.HighlightChanged` and `platform.LanguagesChanged` as the next frame begins, once per change. The first time a value is known is a change too, so a program that only reads the events starts right:

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

Before the program runs, and in a test, a server, a hidden run or a capture, nothing is known: each answer is unknown (`platform.AppearanceUnknown`, `MotionUnknown`, `ContrastUnknown`, `ForcedColorsUnknown`, an empty `Highlight`) and the languages are empty, so the program needs a look and a language of its own to fall back on. A test says what the system would with `prefs.SyncAppearance(platform.AppearanceDark)`, `prefs.SyncMotion(platform.MotionReduced)`, `prefs.SyncContrast`, `prefs.SyncForcedColors`, `prefs.SyncHighlight` and `prefs.SyncLanguages("fr-FR", "en")`, then runs a frame. Everything is only read, never changed ([decisions 0108](../99-decisions/0108-a-program-follows-the-systems-appearance-and-languages.md), [0121](../99-decisions/0121-a-document-answers-media-features-from-the-preferences-it-is-given.md), [0155](../99-decisions/0155-the-system-says-its-contrast-forced-colours-and-selection-colours-read-only.md)):

| System | What it says |
|---|---|
| macOS | all but forced colours, which it has none of: the app's appearance, Accessibility's Reduce motion and Increase contrast, the colours AppKit selects text in, the preferred languages, told by AppKit when they change |
| Windows | all: the apps' light or dark theme, the animation effects, a contrast theme (its contrast more, less or custom by how far its text stands from its background), `Highlight` and `HighlightText`, the display languages, asked every two seconds |
| Linux | the settings portal's colour scheme, contrast and reduced motion through `gdbus`, GNOME's settings through `gsettings` for what it leaves, KDE's `kdeglobals` (its selection colours too) and GTK's `settings.ini`, asked every two seconds off the frame; the languages from `LANGUAGE`, `LC_ALL`, `LC_MESSAGES` and `LANG`; no forced colours |
| the web | the browser's `prefers-color-scheme`, `prefers-reduced-motion`, `prefers-contrast` and `forced-colors`, followed as they change, its `Highlight` colour and `navigator.languages` |

A ui Document given them with `ui.Preferences(prefs)` answers `prefers-color-scheme`, `prefers-reduced-motion`, `prefers-contrast` and `forced-colors` as the system does, and draws a selection no `::selection` rule colours in the system's selection colours ([media queries](../05-ui/03-css.md#the-users-preferences)).

## Where they draw

The windows are where the renderer, one of the World's [presenters](../02-world/README.md#presenters), draws; a headless server opens none.

# Windows

A window is an entity like any other, made with `platform.NewWindow`. A game can open as many as it needs, each with its own cameras and UI, and can follow what the user chose on their system: light or dark, reduced motion, contrast, languages.

## How windows behave

- **Opening and closing.** Adding a window to the World opens it; removing it takes it away with its children.
- **The close button only asks.** The window's `platform.Closing` rule answers. By default, closing the primary window ends the program and closing another closes only that window ([decision 0102](../99-decisions/0102-a-windows-close-is-a-request-its-rule-answers.md)).
- **The primary window** is the first one added, then the oldest left once it is gone. `ember.New` opens a default window only when the game adds none.
- **What shows where.** A camera or a UI element that is a window's child shows in that window and reads its input. One that belongs to no window shows in the primary one, unless a camera names its window with `In`.
- **One camera per window.** A window shows the camera with the highest `Priority` among those that target it.

The windows are where the renderer, one of the World's [presenters](../02-world/README.md#presenters), draws. A headless server opens none.

## The system's appearance and languages

What the user chose on their system is an entity too, `platform.NewPreferences()`. The program adds it to the World and keeps it:

```go
prefs := platform.NewPreferences()
w := ember.New().AddEntities(win, prefs)
```

### What it says

| Getter | Values | Event when it changes |
|---|---|---|
| `prefs.Appearance()` | `platform.AppearanceLight`, `AppearanceDark` | `platform.AppearanceChanged` |
| `prefs.Motion()` | `platform.MotionReduced` where the user asked the system to reduce motion, `MotionFull` where not | `platform.MotionChanged` |
| `prefs.Contrast()` | `platform.ContrastMore`, `ContrastLess`, `ContrastCustom`, `ContrastNoPreference` | `platform.ContrastChanged` |
| `prefs.ForcedColors()` | `platform.ForcedColorsActive` where a contrast theme forces its colours on programs, `ForcedColorsNone` where not | `platform.ForcedColorsChanged` |
| `prefs.Highlight()` | the colours the system selects text in: `Background`, `Inactive` (for an unfocused window) and `Text`; a colour with no alpha is one the system doesn't say | `platform.HighlightChanged` |
| `prefs.Languages()` | the user's languages as BCP 47 tags such as `fr-FR`, the preferred one first | `platform.LanguagesChanged` |

The platform keeps them up to date while the program runs. The World emits each event as the next frame begins, once per change. The first time a value is known counts as a change, so a program that only reads the events starts right:

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

Everything is only read, never changed ([decisions 0108](../99-decisions/0108-a-program-follows-the-systems-appearance-and-languages.md), [0121](../99-decisions/0121-a-document-answers-media-features-from-the-preferences-it-is-given.md), [0155](../99-decisions/0155-the-system-says-its-contrast-forced-colours-and-selection-colours-read-only.md)).

### When nothing is known

Before the program runs, and in a test, a server, a hidden run or a capture, nothing is known:

- each answer is unknown: `platform.AppearanceUnknown`, `MotionUnknown`, `ContrastUnknown`, `ForcedColorsUnknown`, an empty `Highlight`;
- the languages are empty.

So a program needs a look and a language of its own to fall back on.

A test says what the system would with `prefs.SyncAppearance(platform.AppearanceDark)`, `prefs.SyncMotion(platform.MotionReduced)`, `prefs.SyncContrast`, `prefs.SyncForcedColors`, `prefs.SyncHighlight` and `prefs.SyncLanguages("fr-FR", "en")`, then runs a frame.

### Where each system says it

| System | What it says, and when |
|---|---|
| macOS | All but forced colours, which it has none of: the app's appearance, Accessibility's Reduce motion and Increase contrast, the colours AppKit selects text in, the preferred languages. AppKit tells when they change. |
| Windows | All: the apps' light or dark theme, the animation effects, a contrast theme (its contrast more, less or custom by how far its text stands from its background), `Highlight` and `HighlightText`, the display languages. Asked before the first frame, and again when one of the program's windows gains the focus. |
| Linux | The settings portal's colour scheme, contrast and reduced motion through `gdbus`; GNOME's settings through `gsettings` for what the portal leaves; KDE's `kdeglobals` (its selection colours too) and GTK's `settings.ini`; the languages from `LANGUAGE`, `LC_ALL`, `LC_MESSAGES` and `LANG`. No forced colours. Asked before the first frame, and again, off the frame, a quarter of a second after one of the program's windows gains the focus, so a flurry of focus asks once. The tools are found in `/usr/bin`, `/bin`, `/usr/local/bin` or NixOS's `/run/current-system/sw/bin`, never through `PATH`. |
| The web | The browser's `prefers-color-scheme`, `prefers-reduced-motion`, `prefers-contrast` and `forced-colors`, followed as they change, its `Highlight` colour and `navigator.languages`. |

### In the UI

A ui Document given them with `ui.Preferences(prefs)` answers `prefers-color-scheme`, `prefers-reduced-motion`, `prefers-contrast` and `forced-colors` as the system does. It draws a selection that no `::selection` rule colours in the system's selection colours ([media queries](../05-ui/03-css.md#the-users-preferences)).

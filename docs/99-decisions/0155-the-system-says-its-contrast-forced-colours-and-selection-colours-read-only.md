# 0155. The system says its contrast, forced colours and selection colours, read-only

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Decisions [0108](0108-a-program-follows-the-systems-appearance-and-languages.md) and [0121](0121-a-document-answers-media-features-from-the-preferences-it-is-given.md) left Linux and the web saying neither scheme nor motion, `prefers-contrast` and `forced-colors` the program's alone, and every selection macOS's blue.

## Decision

- `platform.Preferences` also holds `Contrast()` (`ContrastNoPreference`, `More`, `Less`, `Custom`), `ForcedColors()` (`ForcedColorsNone`, `Active`) and `Highlight()` (the colours the system selects in, focused and not, and the text's over them; a colour of no alpha unsaid), each unknown until said, with its `Sync…` stand-in and its `…Changed` event.
- A Document given them answers `prefers-contrast` and `forced-colors` as the system, the program's `ui.PrefersContrast`/`ui.ForcedColors` winning as `ui.ColorScheme` does; a selection no `::selection` colours is drawn in the system's colours, blended with white as Chrome's `Color::BlendWithWhite` blends macOS's, which gives the `#b3d7fe` and `#dcdcdc` measured before.
- macOS: Increase contrast, `selectedTextBackgroundColor` and `unemphasizedSelectedTextBackgroundColor` in the light appearance, observed through AppKit's notifications; no forced colours. Windows: `SPI_GETHIGHCONTRAST` forces colours, its contrast more at 7:1 or above between `COLOR_WINDOWTEXT` and `COLOR_WINDOW`, less at 2.5:1 or under, custom between, as Chromium words it; `COLOR_HIGHLIGHT` and `COLOR_HIGHLIGHTTEXT`.
- Linux, every read off the frame: the settings portal's `org.freedesktop.appearance` (`color-scheme`, `contrast`, `reduced-motion`) through `gdbus`, GNOME's `gsettings` for what it left, KDE's `kdeglobals` (its selection colours, its window's darkness, `AnimationDurationFactor`) and GTK's `settings.ini`. The web: `matchMedia` for the four features, with change listeners, the colour the browser computes for `Highlight`, and `navigator.languages` with `languagechange`.
- Everything is read, nothing written; a hidden run or a capture still asks nothing.

## Alternatives

- A D-Bus client in Go for the portal: none in the module's dependencies, and writing one (auth, marshalling, signals) for four numbers outweighs running the desktop's own tools every two seconds off the frame.
- The CSS `Highlight` system colour following the system too: system colours resolve in the cascade from one table, per scheme, not per Document; left for when the other system colours follow.
- Forcing the system's colours under `forced-colors: active`: ui answers the query only.

## Consequences

Every system says what it can, and tests say it with `Sync…`. What Chrome draws on Windows and Linux can't be checked from the macOS machines, so the selection's blend there follows Chrome on macOS; GNOME's accent colour is not read as a selection colour.

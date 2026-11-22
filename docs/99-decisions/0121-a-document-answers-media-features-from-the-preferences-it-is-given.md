# 0121. A Document answers media features from the preferences it is given

- Status: Accepted
- Date: 2026-11-22
- Area: api
- Owner: engine

## Context

Decision [0086](0086-media-queries-are-answered-for-the-documents-viewport.md) fixed `prefers-color-scheme`, `hover` and `pointer` (light, a mouse), and [0114](0114-animations-run-on-transitions-clock-over-the-cascade.md) left reduced motion to the program. A browser built on ember must answer as the user's system does, and [0108](0108-a-program-follows-the-systems-appearance-and-languages.md) now reads the system's appearance.

## Decision

- A Document follows a `*platform.Preferences` only when the program gives it one: `ui.Preferences(prefs)` (`doc.Preferences`, `SetPreferences`). Its scheme and reduced motion are read at each layout, and the cascade restyles once when an answer changes, as for a resize; no event is read.
- The program's answer wins over the system's: `ui.ColorScheme(a)` / `doc.SetColorScheme(a)` with a `platform.Appearance` (`AppearanceUnknown` follows the preferences again), and `ui.ReducedMotion()` / `doc.SetReducedMotion(on)` as before. `doc.ColorScheme()` and `doc.ReducedMotion()` report the answer: the program's, else the preferences', else the host's for an iframe's Document, else light and no preference.
- `platform.Preferences` gains `Motion()` (`MotionUnknown`, `MotionFull`, `MotionReduced`), `SyncMotion` and the World event `platform.MotionChanged`, as the appearance: macOS reads `accessibilityDisplayShouldReduceMotion` when NSWorkspace says the display options changed, Windows `SPI_GETCLIENTAREAANIMATION` every two seconds, Linux and the web nothing.
- `pointer` is `coarse` and `hover` `none` while the pointer last used is a finger on a touchscreen, until the mouse moves; `any-pointer` then finds both, `any-hover` still hovers. A trackpad's fingers move the mouse's pointer, and keys and pads point with nothing, so neither changes the answer. `prefers-contrast` and `forced-colors` stay fixed.
- A media query's length takes the viewport's units, their small, large and dynamic forms, alone or in `calc()`, `min()`, `max()`, `clamp()`, answered at each viewport.
- The Chrome harness has Chrome emulate the Document's answers (`--blink-settings`), so comparisons no longer follow the machine's dark mode.

## Alternatives

- Read the World's preferences from every Document: a game's look would change with the system where the game never asked (0086's reason), and Chrome comparisons would differ by machine.
- Restyle on `AppearanceChanged`: the cascade already compares answers each layout, and a Document taken from a World would need its reader.
- Answer `pointer` from `LastDevice` for pads and keys too: `pointer: none` would flip a screen's layout each time the player touches a key; `data-input-device` already styles by the last device.
- `color-scheme` and `light-dark()`: they change the browser sheet's colours and form controls, a larger change left in `docs/05-ui/10-missing.md`.

## Consequences

A program that passes its preferences gets a dark page and still pages where the system asks, its tests set both with `Sync…`, and nothing changes for one that doesn't. A `SetReducedMotion` call can't be undone to follow the system again.

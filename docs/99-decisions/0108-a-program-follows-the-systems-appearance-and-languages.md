# 0108. A program follows the system's appearance and languages

- Status: Accepted
- Date: 2026-11-19
- Area: api
- Owner: user

## Context

A browser built on ember must look dark when the system is, and ask pages for the user's languages, and follow both when they change while it runs. Nothing in ember read either.

## Decision

- `platform.NewPreferences()` is an entity the program adds and holds, like a window. `prefs.Appearance()` is `AppearanceLight`, `AppearanceDark` or `AppearanceUnknown`, and `prefs.Languages()` the user's languages as BCP 47 tags, the preferred one first, empty when unknown.
- The platform keeps it up to date, and as the next frame begins the World emits `platform.AppearanceChanged` and `platform.LanguagesChanged`, once per change, the first time a value is known included, so a program that only reads the events still starts right.
- With no backend nothing is known: a test, a server, a hidden run or a capture, which must play the same on any machine. `prefs.SyncAppearance` and `prefs.SyncLanguages` then stand for the system, as `SetKey` stands for a keyboard.
- macOS reads the app's effective appearance, observed by key, and `NSLocale.preferredLanguages`, read again when the locale changes or the user comes back to the program. Windows reads `AppsUseLightTheme` in the registry and `GetUserPreferredUILanguages`, every two seconds. Linux reads the languages from `LANGUAGE`, `LC_ALL`, `LC_MESSAGES` and `LANG` (`internal/sysprefs`), and its appearance stays unknown. The web's stay unknown for now.

## Alternatives

- `platform.Appearance(w)` and `platform.Languages(w)` reading the World: the hidden lookup decision [0015](0015-the-engine-exposes-no-resources.md) turned down.
- On the window, beside its clipboard: the languages are no window's, and a program showing no window still has them.
- A process-wide value: two Worlds in one test run would share what one of them simulates.
- No event for the first value: a program would read at start, before the platform has asked, and get unknown.
- Polling the appearance each frame on macOS: an observer costs nothing while nothing changes.
- Linux's appearance through the settings portal: it speaks D-Bus, a dependency for one boolean, and a `gsettings` child process answers for GNOME alone.
- The same for scrolling: nothing to add, the wheel's direction and speed already come as the system set them, in the deltas GLFW passes.

## Consequences

A program follows the system with one entity and two events, and its tests set what the system would. Nothing on Linux or the web says dark from light yet, and Windows hears a change up to two seconds late.

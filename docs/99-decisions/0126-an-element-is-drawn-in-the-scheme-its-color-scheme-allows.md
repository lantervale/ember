# 0126. An element is drawn in the scheme its color-scheme allows

- Status: Accepted
- Date: 2026-11-22
- Area: api
- Owner: engine

## Context

[0121](0121-a-document-answers-media-features-from-the-preferences-it-is-given.md) has a Document answer `prefers-color-scheme` from the preferences, but a dark scheme changed only what `@media` rules said: system colours, `light-dark()`, the canvas and the browser sheet's controls stayed light, where a browser's go dark for a page that says `color-scheme: dark`.

## Decision

- `color-scheme` is read and inherited; an element's scheme is the Document's preferred one where it names both, the one it names otherwise, light under `normal`. A game's screen that names none stays light, whatever the system or `ui.ColorScheme` say.
- System colours and `light-dark()` are resolved at the computed value, where they are declared, in the properties that take colours, as Chrome resolves them: a child of another scheme inherits the colour, a variable resolves where it is used. A flip restyles the tree only when an element follows the preference, and the changed colours run their transitions.
- The user agent sheet writes its colours as system colours and `light-dark()` with Chrome's dark values, measured from its native controls; a dark root without a background has the canvas `#121212`. Under `ui.BrowserSheets()` a control's text is `FieldText`/`ButtonText`, Chrome's; a game's still takes its parent's colour.

## Alternatives

- Keep the keywords to used-value time, in paint: every colour field would carry its keyword, and paint would need the scheme of each box; Chrome itself resolves them when computing.
- Darken the whole Document under a dark preference, as Chrome's auto dark mode: a game's look would change unasked (0086, 0121).
- A second, dark user agent sheet chosen per Document: an element's scheme is per element, so a light card inside a dark page couldn't keep its light controls.

## Consequences

A page that says `color-scheme: light dark` follows the system with no `@media` rules of its own, and games that say nothing look as before. The deprecated system colour names are colours only in colour properties (`animation-name: mark` stays a name).

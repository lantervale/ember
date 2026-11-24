# 0213. A page's textarea resizes, and sideways lines are spaced as horizontal ones

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Decisions 0207 and 0211 left three gaps: Chrome's textareas take `resize: auto` and ui's none; the viewport's styled bars restyled at once on body's hover, thought to wait as a box's do; sideways lines were left unspaced.

## Decision

- `resize: auto` is valid and computes `both`, as Chrome computes it on any box.
- Under `ui.BrowserSheets()` a textarea takes `resize: auto` (`css.BrowserDefault`), Chrome's grip that drags it; a game's screen gives its textareas none unless its sheet asks, and export writes `textarea { resize: none }` into a game's page so Chrome draws what ui does.
- The viewport's bars keep restyling whole as body's or the root's hover changes: Chrome does so (`TestViewportBarsRestyledByBodysHoverAsChromes`), unlike a box's, which keep their parts until the pointer reaches one; the pointer on them hovers neither.
- Sideways lines (`text-orientation: sideways`, `writing-mode: sideways-*`) are spaced by text-spacing-trim and text-autospace as horizontal ones, by `halt`, as Chrome spaces them; `text-combine-upright` stays unspaced.

## Alternatives

- A textarea resizable in every Document, as Chrome's sheet has it: a game's chat box or notes field would grow a grip it never asked for, and a press in its corner would change its layout; a screen that wants one says `resize: auto`.
- Caching the viewport's parts as a box's (`BarsShown`): Chrome doesn't, so a page would look otherwise under body's hover.

## Consequences

A page's textareas look and drag as Chrome's, a game's stay as they were. Decision 0207's note that the viewport's bars differ from Chrome's no longer holds.

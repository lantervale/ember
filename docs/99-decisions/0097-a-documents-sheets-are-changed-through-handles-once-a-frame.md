# 0097. A Document's sheets are changed through handles, once a frame

- Status: Accepted
- Date: 2026-11-18
- Area: api
- Owner: user

## Context

A Document's stylesheets were its constructor's and stayed as given. A page's scripts add, edit and remove `style` elements, and styled-components inserts its rules one at a time into a sheet, hundreds as a page starts: light could apply none of it, and bbc.com/news stopped at its first rule.

## Decision

- Every stylesheet of a Document is a `*ui.Sheet`: `doc.AddSheet(text, opts…)` adds one after the others, `doc.InsertSheet(before, text, opts…)` before another, `doc.StyleSheets()` lists them in cascade order, those the options gave among them.
- A Sheet is changed whole (`Replace`, `Remove`, `SetMedia`, `SetDisabled`) or rule by rule (`Rules`, `InsertRule`, `DeleteRule`), a rule being a top-level one: a style rule, or an at-rule with what it holds. Only an inserted rule is parsed.
- A run-time sheet is read as the Document's others: `ui.From`, `@media`, `@layer` and `@supports` under `ui.BrowserSheets()`, its warnings in `doc.Warnings()`, an `@font-face` in it a warning as in `ui.CSS`.
- Changes only mark the Document: the next frame joins each changed sheet's rules into the sheet the cascade reads (`css.Join`) and styles the tree once, laying out only what the new styles move. `MatchedRules` and `ComputedStyle` read the sheets as they stand without waiting for it.
- `InsertRule` takes a rule ui reads part of, what it drops a warning, as a browser takes an unknown property; its error is for a text that is no one rule, and `ui.ErrNoRule` for a place out of range.

## Alternatives

- One `doc.SetSheets(texts…)` replacing them all: every inserted rule would parse every sheet again, seconds for a page's start.
- Each inserted rule a sheet of its own in the cascade: no joining, but an element then tries thousands of sheets' buckets.
- Restyle at each change: simple to reason about, and a freeze while a script inserts its rules.
- An incremental index, appending to the buckets in place: cheaper than joining for one rule a frame, but an insertion in the middle moves every rule's order after it, and the join costs under a millisecond for a thousand rules.
- Positions by index (`AddSheetAt(i)`): the caller must then know the options' sheets and the disabled ones; a neighbour's handle says where without counting.

## Consequences

A page's scripts restyle it at one cascade a frame: a thousand inserted rules cost about 5 ms to read and one restyle. A rule inserted alone stands at line 1 of its own text for a tool, and the `css.Sheet` a tool was given for a run-time sheet is another one after a change.

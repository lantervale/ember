# 0152. The old attributes of tables, rules and fonts are hints as Chrome's

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Decision 0092 read the attributes old pages lay out by, and left the rest unread: a table's `border`, `frame` and `rules`, floated tables and pictures, `hr`'s size and shade, `font`'s size, face and colour, a `br`'s `clear`. Pages sent to a simple browser draw their grids, their rules and their headings with them. Around them, a radio group spanned forms of the same name, and a control under `ui.BrowserSheets()` took its parent's font where Chrome's starts from its own.

## Decision

- Each is a hint as 0092's are, mapped as Chrome maps it (`TestPresentationalAttributesStyleAsChromes` compares every computed value): a table's border outset, its cells inset, `frame` and `rules` hidden and solid sides, the later of `border` and `frame` setting the width; `rules` collapsing the borders; `font size` to the seven keywords, `xxx-large` read in CSS too.
- Colour attributes follow HTML's legacy rules for colours, as Chrome: `bgcolor="fff"` is `#0f0f0f`, where 0092 read it as `#fff`.
- A `br`'s `clear` starts the line after it below the floats it names, as Chrome's; `clear` on any other inline element still does nothing.
- A table with a collapsed column keeps, as a flex item, the width its container gave it, its rows too: that is what drew its background "as wide as before".
- `img align="middle"` is `vertical-align: -webkit-baseline-middle`, now a value of the subset: the box's middle on its parent's baseline.
- ember's sheet has a row group and a row hand their border colour down (`border-*-color: inherit`, longhands so restyling stays narrow), as Chrome's, so a cell's border takes its table's `bordercolor`.
- A radio is grouped by its name and its form owner (the form above it, or the one its `form` attribute names), and a form is `:invalid` by the fields it owns.
- Under `ui.BrowserSheets()` a control's font is `13.333333px Arial`, a textarea's `monospace`, at a normal line height, with Chrome's `white-space`.

## Alternatives

- Rules in the user-agent sheet (`table[border] td`): a width that is a number has no selector, and `frame` against `border` depends on their order.
- Keep 0092's colour reading: simpler, and wrong where Chrome reads `fff` or a stray word.
- Resolve `font` sizes to pixels in the hint: Chrome's computed value is the keyword's, and `xxx-large` is CSS any sheet may write.

## Consequences

A game's screen writes none of these, so it changes only where a row's border colour was left to `currentColor`. Arial is Go unless `ui.SystemFonts()` finds the machine's; a page compared with Chrome without system fonts measures its controls' text in Go.

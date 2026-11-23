# 0134. Counter styles are rules, the predefined ones a sheet of them

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

`list-style-type` and `counter()` knew a fixed enum of styles (decimal, the alphabets, roman), formatted in layout. `@counter-style`, the predefined styles Chrome ships and string markers need a style to be a definition looked up by name, and a list item's marker string needs the sheets' rules where it is written.

## Decision

- A counter style is resolved data (`style.CounterStyle`: system, symbols, additive symbols, prefix, suffix, negative, range, pad, fallback), and its algorithms are pure functions of it.
- The predefined styles are CSS Counter Styles' own `@counter-style` rules, an embedded sheet read once, under the author's: an author may redefine any but `decimal`, `disc`, `circle`, `square` and the disclosure triangles, which Chrome keeps too.
- `list-style-type` computes to the shapes paint draws (`disc`, `circle`, `square`, the triangles, `none`), a counter style's name, or a string. css works each item's marker string out while it counts (prefix, representation, suffix) and layout only lays the text out.
- `counter-reset: reversed()` stays refused: Chrome 157 does not read it.

## Alternatives

- Keep an enum and add each predefined style as Go code: no author styles, and two definitions of the same style to keep apart.
- Format in layout: layout has no sheets, and `@counter-style` rules would have to travel with each style.

## Consequences

A style a sheet defines is a name until counting resolves it, so an unknown name falls to decimal as in Chrome; a counter style inside `@media` is answered for the Document's media and restyles the tree when that flips. The complex predefined styles (CJK, Ethiopic) are not in the sheet and write as decimal.

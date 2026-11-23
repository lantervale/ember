# 0178. Spaces break-spaces keeps are units of a line, and break-all reads letters as ideographs

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

`white-space: break-spaces` keeps spaces that take room at a line's end and lets a line break after each one; ui's breaker held every space pending until the next word, hung or dropped at a line's end. A carriage return kept by `pre` was a tab of no width, a place to break, where Chrome draws it as nothing inside its word; and `word-break: break-all` broke only between letters and before a hyphen, by a rule of ui's own.

## Decision

- A space or a tab of `break-spaces` text is a unit of the line: glued to the word before it (no break before the first space), a break after it, measured in the line's width and its min-content. At a line's end such spaces are shown as text, which justify doesn't widen and the bidi algorithm leaves at their own level, as Chrome; between words justify widens them. `break-spaces nowrap` is laid out as `pre`.
- A carriage return of text that keeps spaces is a character of its word, of no width and never drawn: no break before or after it. The page written for Chrome carries it as `&#13;`, which its parser keeps.
- `break-all` breaks where UAX #14 breaks once letters and digits are read as ideographs, and before a hyphen-minus after one, as Chrome; between two elements the break follows the `word-break` of the text before it, and comes before the element's start.

## Alternatives

- Spaces as words of their own, with a break piece after each: the breaker would keep them, but justify and the bidi algorithm would no longer see spaces.
- Blink's own break-all table by line-break class: ui has no line-break classes of its own, and uniseg's are internal; UAX #14 over ideographs gives Chrome's breaks but beside `+` and `|`, which Blink's ASCII table treats apart.

## Consequences

The breaker learns one more kind of unit, a held space, beside words, edges and atoms. Break-all costs a UAX #14 step a grapheme pair in the text that asks for it.

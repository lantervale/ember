# 0188. A styled first line measures its whole block

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: user

## Context

A block whose `::first-line` changes its font was sized to its content in its own styles (decision 0127), and Chrome sizes it otherwise: its min-content and max-content widths are measured with every line in the first line's styles, not only the first. A first letter was also taken from a text node's written text, so a translated text had none, and a press on the letter's box reached no text.

## Decision

- A block's intrinsic widths are measured with every line in the first line's styles, as Chrome's line breaker in its min and max mode does; the inline elements' margins, borders and padding there are resolved in their first-line styles, so `1em` is the first line's em.
- The cascade takes the first letter from the text as layout shows it (`Transitions.ShowText`, the Document's translation), keeping the spaces before it where `white-space` keeps them; a translation changing lays the tree out again and takes the letters again.
- The letter's box stays a generated node of its own; hit testing and selection map its fragments back onto its text node, whose run is kept, of no text, when the letter is all of it.

## Alternatives

- Measure the first line in its styles and the others in their own: what a browser following the spec would do, and wider or narrower than Chrome's box.
- Translate in layout only and leave the letter's node as the cascade cut it: the letter would show the untranslated text.

## Consequences

A block with a much smaller first line can be sized narrower than its other lines need, overflowing as it does in Chrome. Translating costs one more pass over the first letters.

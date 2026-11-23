# 0177. Lines are chosen again after the greedy ones, and calc-size() is a keyword scaled

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Chrome 157 breaks `text-wrap: balance` and `pretty` text with Blink's ScoreLineBreaker (Minikin's optimal breaker) over the lines its greedy breaker made, and bisects a width where it can't score; it lays out `calc-size()` and interpolates a size to and from `auto` under `interpolate-size: allow-keywords`. ui broke lines greedily, and its cascade holds no size from layout.

## Decision

- Lines are broken greedily first, as before; a block whose `text-wrap-style` asks for it then has the lines of a paragraph broken again at the break points of least score, Blink's constants and tie-breaking ported (`balance` the first paragraph of up to six lines, `pretty` the last four of each whose last line is one short word, or whose two lines before it end in hyphens), the greedy lines kept where Chrome keeps them. A balance that can't be scored bisects the width a block of one paragraph is broken at, its lines aligned in the whole width.
- A soft hyphen is a piece of its own: no room mid-line, its line's hyphen (`hyphenate-character`, else U+2010 or a hyphen-minus) at a line's end, drawn after the fragment's text, which never holds the soft hyphen.
- `calc-size(basis, calculation)` computes to a `style.Length` of the basis's kind marked `Sized`, with the multiple of `size` and the rest, the calculation worked with `size` at 0 and 1. Code that doesn't read it takes the keyword; block layout works it on the height the keyword gives. Interpolation turns a keyword and a length into two such lengths of one basis.

## Alternatives

- A Knuth–Plass breaker of ui's own over every paragraph: lines Chrome doesn't break, where the point is to match it.
- A layout size fed back to the cascade for `calc-size()` (decision 0174's note): a second style pass a frame, where a scaled keyword lets layout, which knows the size, finish the sum.
- Hyphenation patterns for `hyphens: auto`: Chrome on macOS hyphenates through the system's dictionary, which no pattern file embedded in ui reproduces; it is read and breaks as `manual`.

## Consequences

Balanced and pretty lines cost a second breaking only where they apply. `calc-size()` is laid out in heights of block containers alone, and a calculation that isn't linear in `size` is read as the line through its values at 0 and 1.

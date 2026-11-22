# 0123. Math functions over percentages are a program layout runs

- Status: Accepted
- Date: 2026-11-22
- Area: api
- Owner: engine

## Context

CSS Values 4's math functions (`round()`, `mod()`, `sign()`, `hypot()`, the trigonometric and exponential ones, typed `calc(10px / 1px)`) may hold a percentage that only layout or paint can resolve, and a computed `style.Length` must stay comparable. The old `Calc` was 16 fixed steps of min, max, sum and scale, and anything longer was worked out as if the percentage were of nothing.

## Decision

- css types every value as CSS Values 4 does (powers of length, angle and percentage), refusing a declaration whose math ends in another type than its property takes, as Chrome refuses it.
- What is known at parse or compute time is worked out at once (units of the font and the viewport included); only what a percentage is in stays, as terms of the value.
- Those terms become a `style.Calc`: a stack program of any length, encoded in a string so the Handle stays comparable, whose ops are the math functions themselves (`style.CalcOp.Apply`, shared by css and layout).
- NaN, infinities and signed zeros follow IEEE 754 inside a function; at the top, NaN is 0 and an infinity the longest length (`style.MaxLength`).
- Percentages of a line-height and a word-spacing resolve at compute time against the font size; vertical-align and gradient stops keep the Calc and resolve it at layout and paint.

## Alternatives

- A tree of handles per node: comparable too, but one allocation per node and a pointer walk per resolution.
- A bigger fixed array: still a limit, and every Length carrying it.
- Resolving the percentage against 0 as before: wrong wherever the reference is known later.

## Consequences

Any math Chrome accepts over a percentage resolves where the percentage does, with no limit on its size; a Calc is decoded at each resolution, which costs a few nanoseconds per step.

# 0106. css asks what a unit stands for, and leaves layout the bounds over a percentage

- Status: Accepted
- Date: 2026-11-19
- Area: rendering
- Owner: engine

## Context

css computed every length to points and a percentage, knowing only font sizes: `ex`, `ch`, `lh` and the viewport units were refused, and so were `min()`, `max()` and `clamp()`, which pages of the web write in most of their sheets (`width: min(100%, 500px)`, `font-size: clamp(1rem, 2vw, 2rem)`). The font units need a face's metrics, which css must not load; the viewport units need the viewport and a restyle when it changes; and a bound over a percentage is known only to layout.

## Decision

- css stays without fonts: the Document tells its `Transitions` how to measure a font (`MeasureFonts`, a `FontUnits` of ex, ch, cap, ic and the normal line), and the viewport comes with the media it already gives (`SetMedia`).
- The font's family, style and weight, its size and its line height are applied before every other longhand, so a length in `ex` or `lh` reads the element's own; in `font-size`, and `lh` in `line-height`, they read the parent's, and the root's units the root's.
- css notes which of these units a tree reads: a viewport that changes, or a font that lands, styles the tree again only where some length is in them.
- A bound with no percentage in it is worked out by css and is a plain length. One over a percentage is a `style.Calc`, the steps to work it out against what the percentage is of, held by a `unique.Handle`: equal values are one handle, so a `Length` still compares as its fields and a style styled again is equal to itself.

## Alternatives

- Give css the fonts: the pure package would load faces, and a server would too.
- Leave the units for layout, as percentages: every property that takes a length (borders, shadows, font sizes) would wait for layout, not only those that take a percentage.
- A pointer to an expression in `Length`: lengths compare by `==` all over layout and in `Computed.Equal`, and two equal values would differ; an index into a table of expressions never frees them.
- A lower bound, a value and an upper bound as three fields of `Length`: says `clamp()` but not a bound in `calc()` or in another, and widens every length of every style.

## Consequences

The units of fonts and of the viewport and the three functions work wherever a length does, checked against Chrome. A `Calc` holds 16 steps: a longer one, and one in the few values that keep their own percentage (`line-height`, `word-spacing`, `vertical-align`, a gradient's stop), is worked out against nothing, as [What is missing](../05-ui/10-missing.md) says.

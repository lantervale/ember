# 0210. Marks of vertical lines are measured in their flow, and a fallback stands by its own metrics

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Decision [0193](0193-vertical-lines-are-selected-in-the-flow-of-their-lines.md) selected vertical lines but left three marks out: the empty lines of preformatted text, a br alone on its line and the room of tabs kept between fragments, all measured on horizontal lines only. Decision [0180](0180-characters-stand-by-uax-50-in-their-faces-vertical-metrics.md) set a standing character by its face's vertical metrics, but a fallback face without them set it by the first face's ascent and descent, where Chrome takes the fallback's.

## Decision

- The empty lines' marks and the tabs' room are found as on horizontal lines, on the fragments turned into the flow of their lines (`flowRect`), and an empty line's mark is turned back onto the page; paint draws a vertical run's marks that are on none of its lines as they stand.
- A br's empty line is marked across the line, as wide as the line is high and centred on the br's box as the central baseline centres it, a space long from the line's start: down the line, or up it in `sideways-lr`. Whether a br is alone on its line is asked across the lines.
- A character standing takes the vertical metrics of the face that draws it, the first face's, a later family's or the Fallback's (Go Mono); without them, that face's ascent and descent, as Blink's vertical data of a face without `vhea` and `vmtx`.

## Alternatives

- Write vertical twins of the marks' code: three more places where the writing modes' sides and directions must agree with the flow that hit testing already uses.
- Keep the first face's ascent and descent for a fallback: a symbol of a taller fallback face overlaps its neighbours where Chrome gives it room.

## Consequences

A selection over vertical lines marks what one over horizontal lines marks, where Chrome draws it. A tab that starts a line of kept white space is still unmarked, on both kinds of lines (10-missing).

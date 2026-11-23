# 0197. Text right to left and sideways is shaped as one, and a reshaped end keeps its joins

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Text shaped as one across elements ([decision 0183](0183-text-is-shaped-as-one-across-elements-of-one-font-and-spacing.md)) and its reshaped line ends ([decision 0190](0190-line-ends-are-shaped-again-and-carets-share-a-ligature.md)) covered text left to right and horizontal only. Chrome shapes a right-to-left run across element edges the same way: `ل<span>ا</span>` is one lam-alef, the first element's, the span empty at its left; a Hebrew pair is kerned across an edge; `calt` reads the next element's letters. In vertical lines it does so only where no character stands (`text-orientation: sideways`, `sideways-rl`, `sideways-lr`); under `mixed` and `upright` it shapes element by element. A line end it shapes again keeps the rest of the text as context, so an Arabic letter cut by `overflow-wrap: anywhere` keeps its joined form.

## Decision

- Layout numbers runs of one bidi level as one (`shapingRuns`), whatever the level, and in vertical lines unless characters may stand. Pieces are measured in their logical order as before; a fragment shown right to left takes the text before it from its right (`besideShown`), and the text past the line's end where it holds the line's last word, whichever side that is.
- twod's `Amid` shapes text right to left as one with its lead and trail, mirrored as the text is, and draws from the end the glyphs whose clusters start in it.
- A word or grapheme shaped again at a line's start or end keeps the text it is cut from as HarfBuzz context; a fragment cut by a line keeps the text round it as its context where it has no lead or trail. A word at a run's edge gains only what the text past its run changes, not its own word's joins again.

## Alternatives

- Reordering lead and trail into visual order and shaping them with the fragment's text as shown: twod shapes text right to left in its logical order, so this would reverse it twice.
- Shaping every vertical run as one, standing characters included: Chrome does not, and its standing clusters are set one by one by `vmtx`.

## Consequences

Arabic ligatures, alternates and marks, and Hebrew kerning, cross element edges right to left and inside left-to-right lines as in Chrome; sideways lines kern and ligate across edges. Chrome breaks a line inside a later element's ligature to fit it, which ember does not, and drops a bidi-split inline's empty part from its box ([missing](../05-ui/10-missing.md)).

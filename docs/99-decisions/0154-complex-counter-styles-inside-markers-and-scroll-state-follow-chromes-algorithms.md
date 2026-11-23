# 0154. Complex counter styles, inside markers and scroll state follow Chrome's algorithms

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Several features ui refused have a spec algorithm Chrome departs from: the Chinese and Korean counter styles, the place and width of a marker inside its item, `-webkit-mask-composite`'s compositing operators, and when a `scroll-state()` query is answered. ui follows Chrome where the two differ, as every ui page is checked against it.

## Decision

- The complex predefined counter styles are internal systems of the predefined sheet (`-ember-…`), which author sheets can't name and `extends` copies. Japanese writes -9999 to 9999 as CSS Counter Styles does; Chinese and Korean write any number in groups of four digits with their markers (万, 億, 만…), a space after a Korean group's, a group's zeros collapsed and a leading ten's 1 dropped as Chrome's CJK ideographic algorithm does. `symbols()` is kept by its serialisation, which names the anonymous style it made.
- A marker inside its item is a piece of the item's first line, as the summary's triangle already was: its text as wide as written, a picture and 7 points, a bullet two points wider and the font's size after it, as Chrome's margins put them; a first block puts it on a line of its own.
- `-webkit-mask-composite`'s operators are mask-composite values of their own, kept as written, joined by the picture operations the plain words use (clips by a picture, sums).
- `scroll-state()` is answered after each layout and each scroll, the same frame, from the boxes layout or a refresh left: a sticky box's offset, a scroller's offset within its overflow. ui snaps no scroll, so `snapped` is always `none`.

## Alternatives

- The spec's Chinese algorithm (range -9999 to 9999, a 1 dropped after any zero): Chrome writes 10000 and 1010 otherwise, and pages are compared with Chrome.
- A marker inside as a generated inline element, like `::before`: the counter text is worked out by counting, after the tree is built, which the piece the summary used already takes.
- Answering `scroll-state()` a frame late, as Chrome's snapshot does: ui lays out and draws in one step, so the answer would show a frame after the scroll for nothing.

## Consequences

The complex styles' glyphs come from the faces' fallbacks, so a game shows them only with fonts that hold them. A scroll that moves a scroll-state answer restyles and lays out the whole tree that frame, where a plain scroll only refreshes offsets.

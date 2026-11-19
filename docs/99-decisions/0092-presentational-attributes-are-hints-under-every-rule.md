# 0092. Presentational attributes are hints under every rule

- Status: Accepted
- Date: 2026-11-18
- Area: api
- Owner: engine

## Context

Pages older than CSS, and the ones sites still send a browser they do not know (Google's search page), are laid out by attributes: `width="25%"` and `align="center"` on a cell, `valign` on a row, `cellpadding` on a table, a `center` round it all. ui read an `img`'s `width` and `height` and nothing else (decision 0082 left `font`'s out), so such a page stood at the left edge in columns as wide as their text.

## Decision

- ui reads them as HTML maps them to CSS: `width`, `height`, `bgcolor`, `align`, `valign`, `nowrap`, `cellspacing`, `cellpadding` and body's `text`, on the elements `docs/05-ui/03-css.md` lists.
- They are hints, as an `img`'s size already was: declarations of the author's, under every rule of every sheet, so any rule wins over an attribute.
- What they need that the subset lacked is CSS Chrome reads too: `text-align: -webkit-left | -webkit-right | -webkit-center`, which place blocks as well as lines (`center`'s default now), and a percentage width on a cell in the automatic table layout. `valign` is `align-content` on the cell, `vertical-align` staying refused.
- A cell is hinted by its table's `cellpadding` and its row's or row group's `valign`; a change of either restyles the cells below.
- The rest (`border` on a table, `font`'s, `hr`'s, `clear`…) stays unread, listed in `docs/05-ui/10-missing.md`.

## Alternatives

- Have the web explorer write them into each element's style attribute: a style attribute wins over the page's sheets where a hint must lose to them, and a game showing old HTML gets nothing.
- Rules in the user-agent sheet (`td[align="center"] { … }`): values that are numbers (`width`, `cellpadding`) have no selector.
- Keep refusing them: pages laid out by attributes are the ones a simple browser is sent.

## Consequences

A game's screen never writes these attributes, so nothing of it changes but `center`, which now centres its narrow blocks as Chrome does. Chrome compares them under its own user-agent sheet (`TestPresentationalAttributesLayOutAsInChrome`): sent ember's as the page's sheet, its rules would win over the attributes.

# 0082. ui has the text elements pages are written with

- Status: Accepted
- Date: 2026-11-17
- Area: api
- Owner: user

## Context

ui had the elements a game screen is built from and left out lists, links, line breaks and the presentational ones (`b`, `i`, `center`, `font`…). The web explorer stood divs and spans of a class in for them and rewrote every site's selectors to match, and a game's own prose (a quest log, a changelog, credits) had no list or link.

## Decision

- ui has `ul ol li menu dl dt dd h5 h6 blockquote figure figcaption form fieldset legend address hgroup search center br a b i u s small big sub sup abbr cite q mark time var del ins dfn bdi bdo data font`, each a constructor (`ui.Ul`, `ui.A`…) and a type selector.
- Their default look is Chrome's own user-agent sheet, compared with Chrome's and not with ember's sheet sent to it (`TestTheTextElementsDefaultsMatchChromes`).
- What CSS alone cannot say is laid out by ui: a list item's marker outside its box, a `br`'s line break, a `q`'s quotation marks, a fieldset's legend on its top border.
- An `a` with an `href` matches `:link`, takes the focus and clicks as a button does; ui opens nothing: what a link does is its `OnClick`.
- Elements that only name a meaning (`time`, `data`, `address`, `search`…) are plain boxes; `font`'s attributes, `bdo`'s direction, a fieldset's `disabled`, a form's submission and an `ol`'s `reversed` and `type` do nothing.

## Alternatives

- Keep emulating them in the explorer: every page's selectors are rewritten, `:link` and list markers cannot be had, and games get nothing.
- Add only the semantic ones: pages use `b`, `i` and `center` all the same, and each costs a line of the default sheet.

## Consequences

The kit stays the way to build a game's screen; these elements are for prose and for pages ui did not write. Each new element is one more entry in the per-tag tables (controls, automation, paint, layout), which stay exhaustive.

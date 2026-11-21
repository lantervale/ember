# 0113. :has() flags its anchors before the cascade

- Status: Accepted
- Date: 2026-11-21
- Area: api
- Owner: user

## Context

`:has()` makes an element's style depend on what is below it or after it, where every other selector reads above and before its subject, which is all the one-cascade-a-frame restyle (decision 0097) knew how to invalidate. `:target`, `:focus-within`, `:empty` and `:lang()` read things the tree did not hold.

## Decision

- Before visiting the tree, css walks its dirty part once: an element whose class, id, attribute or state some `:has()` reads, or whose children changed, flags the elements its change can turn: its ancestors, and the elements before it and before each ancestor when a `:has()` starts with `+` or `~`, each only when it has what the compound holding the `:has()` names. A flagged element restyles, its subtree or the elements after it too when a `:has()` stands above or before the subject. Matching walks the anchor's subtree or its later siblings, stopping at the first match.
- `:empty`, `:target` and `:focus-within` are states (`dom.Empty`, `Target`, `FocusWithin`), so the state invalidation already in place reads them: the tree keeps `Empty` itself, a change of it restyles an element only where a sheet reads `:empty`; `doc.SetTarget(n)` sets the target, separate from `ScrollToStart`; `:focus-within` is the focused element and its ancestors, an iframe holding its Document's focus included, as in Chrome.
- The Document's language is `ui.Language(tag)` / `doc.SetLanguage(tag)`, a page's Content-Language, kept on the root and read by `:lang()` where no `lang` attribute is above an element; `lang` and `contenteditable` changes restyle the subtree under them where a sheet reads `:lang()`, `:read-only` or `:read-write`.

## Alternatives

- Restyling the whole Document on any change while a sheet has a `:has()`: simple, but every hover would restyle every element of the page.
- Chrome's per-feature invalidation sets and a `:has()` result cache: faster still on huge pages, far more code; the walk above restyles a hover on a 6,000-element page with five `:has()` rules in about 0.3 ms.
- `ScrollToStart` setting the target: a program scrolling to an element for its own reasons would restyle the page.

## Consequences

A page without `:has()` pays nothing. One with a `:has()` reading a frequent state (`:has(:hover)`) pays a walk up the changed element's ancestors each change, and their earlier siblings for `+` and `~`; a `:has()` whose element names nothing (`:has(+ .menu .label:hover)` on a bare sibling) rescans that subtree each time it is flagged.

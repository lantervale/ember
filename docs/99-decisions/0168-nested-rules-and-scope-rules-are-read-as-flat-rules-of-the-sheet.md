# 0168. Nested rules and @scope rules are read as flat rules of the sheet

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Pages and component libraries write CSS Nesting (`.card { &:hover { … } > h2 { … } }`) and `@scope`, which ui read only as `@starting-style` in a style rule: every other nested rule was a warning, and its declarations were lost. Chrome reads a nested rule's `&` as `:is()` of its parent's list, keeps declarations written after a nested rule in their place (`CSSNestedDeclarations`) with the parent's own weight, and ranks a scoped declaration by how near its root is, after specificity and before the order.

## Decision

- The parser unfolds a style rule into flat `Rule`s of the sheet, in order: the rule with its leading declarations, then each nested rule and each later run of declarations where it stands. A nested selector without `&` is relative (`& ` before it); `&` is the parent's compound itself where that matches the same with the same weight (one compound, no pseudo-element), `:is()` of its list otherwise, `:scope` outside any rule. A run of declarations takes its parent's own selectors, so `.a, #b { .x {} color: red }` weighs `.a` where `.a` matches, as in Chrome.
- `@media`, `@container`, `@starting-style`, `@scope`, and under `ui.BrowserSheets()` `@supports` and `@layer`, nest in style rules and in each other, their declarations those of the rule round them; any other at-rule in a style rule is dropped with a warning.
- `@scope (start) to (end)` gives its rules a `scopeRule`: the cascade walks the element's ancestors for a root, leaves out what an end cuts off, and ranks the declaration by its generations under the nearest root, after specificity and before the order. Rules in it are relative to `:where(:scope)`, `&` there weighing nothing; `:scope` weighs as a class and is the root element outside an `@scope`. An `@scope` of no start is rooted at the root element, as a sheet of no owner element.
- The outermost rule holding others keeps its text, which `Sheet.String` writes back, so an exported page and `Sheet.Rules` show the nesting as written; the CSSOM edits a whole top-level rule (decision 0097).

## Alternatives

- Keep the tree of rules and match through it: closer to the CSSOM, but every consumer of `Sheet.Rules` (the index, `@media` and `@container` switches, imports, joins, the inspector) would learn a second shape.
- `&` always as `:is()`: simplest, but every `&:hover` falls out of the class buckets into those every element tries.
- Hoist the declarations after nested rules into the rule, as Chrome did before version 130: they would then lose to nested rules written before them.

## Consequences

A nested sheet costs what its unfolded flat sheet costs, and the cascade, the restyle reads and the inspector work on it unchanged. Rules nested in a rule are not each an object of the CSSOM, and an `@scope` of no start cannot be rooted at the element of a `style` that ui does not have.

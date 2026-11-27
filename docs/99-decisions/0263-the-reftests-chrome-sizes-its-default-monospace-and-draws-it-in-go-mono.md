# 0263. The reftests' Chrome sizes its default monospace, and draws it in Go Mono

- Status: Accepted
- Date: 2026-11-27
- Area: tools
- Owner: engine

## Context

[0259](0259-a-browsers-default-monospace-is-13px-where-no-absolute-size-stands-above-it.md) sizes a lone `monospace` at Chrome's 13px and scales it in and out of the generic, but the reftest runner named `monospace` Go Mono for Chrome, which then drew it at 16px: every test whose author monospace takes its size from a keyword stopped matching the runner's Chrome, though ember draws it as Chrome would.

## Decision

- Everything 0259 decides stands, but for its last point.
- The runner's handler leaves a `monospace` alone in its family list as it is, so Chrome sizes it by its own rules (13px for `medium`, the 13/16 scaling going in and out); one among other families is still named Go Mono, as Chrome draws it at the size given there.
- Once a page has loaded, the runner has Chrome run `reftest.MonospaceFace` before shooting it: every element's and pseudo-element's computed size is set on it as Chrome computed it, then those whose family is `monospace` alone are drawn in Go Mono.

## Alternatives

- Naming it Go Mono in the sheets, scaled by 13/16 in the same rule: right only where no absolute size stands above it, and wrong for what inherits from it.
- `Page.setFontFamilies` with Go Mono as Chrome's fixed font: Chrome looks the name up among the machine's fonts only, not the page's `@font-face`, and draws Times.
- Installing Go Mono on the machine: the runner changes nothing outside its own processes.

## Consequences

The runner's Chrome now draws a lone `monospace` at the size ember gives it, in ember's face: content-172 and -174, word-spacing-characters-001 and -003, flexbox_columns-flexitems and float-nowrap-9 match Chrome again, and the 153 tests marked `chrome` that name monospace still do. A size Chrome resolves after the script runs (an animation, a style changed later) keeps the size fixed by the script, and a quoted `"monospace"` stays a family Chrome lacks, drawn in its default face.

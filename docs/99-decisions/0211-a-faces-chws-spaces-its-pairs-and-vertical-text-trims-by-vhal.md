# 0211. A face's chws spaces its pairs, and vertical text trims by vhal

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Decision 0187 spaced horizontal text only, by Chrome's rules alone. Chrome also trims vertical text (by `vhal`), text fields and a face's own `chws` pairs, and its `trim-start` at a block's start looked inconsistent.

## Decision

- Where one face draws two characters side by side at one size and has `chws` and `halt`, the pair takes what the face's `chws` does to it, shaped as the pair, across elements; Chrome's rules apply where it has none or either says `space-all`. A trim a character takes for the one after it goes where a line ends after it, and a closing bracket a line parts from punctuation its rules would pair it with is never trimmed to fit, as Chrome measures it.
- In vertical lines the characters standing upright trim by the same classes and rules through `vhal`, read by shaping the glyph's `vert` form with and without it (internal/shape now keeps a value record's vertical advance); text-autospace adds its room beside turned letters and none under `text-orientation: upright`; sideways lines stay unspaced.
- An input's caret, selection and scroll width, a textarea's rows and a select's open list are measured through the same spacing, one line of the field's text laid out as layout lays it out (`layout.LineSpacing`), the rows from their fragments.
- `trim-start` trims an opening bracket starting a block or following a forced break in every face and element, as Chrome 157 does with the faces a page embeds; the full width 0187 saw came from Chrome's own system face, which also trims under `normal`, a case no page of ui's reaches.

## Alternatives

- Shaping each run with `chws` over its ranges in internal/shape: the pairs of real faces' `chws` reach one neighbour, which shaping the pair reproduces with the per-character room layout already carries.
- Testing `chws` in a system face: none here has it (Hiragino, PingFang UI), so a stand-in, go-han-chws.ttf, has a `chws` that spaces some pairs otherwise than Chrome's rules, so its Chrome suite tells whose rules spaced them.

## Consequences

A face with `chws` costs one shaping per new pair of characters, cached by face. Pairs across two faces or sizes and sideways lines keep the rules or nothing, and the open list is uncompared, as Chrome on macOS shows a native menu.

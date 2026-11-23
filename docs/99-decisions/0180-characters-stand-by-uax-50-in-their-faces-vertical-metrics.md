# 0180. Characters stand by UAX #50, in their faces' vertical metrics

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Decision [0176](0176-vertical-writing-modes-are-laid-out-turned-into-their-own-frame.md) stood the CJK scripts of vertical lines upright by a hand-written list of blocks, each in a room of its font's ascent and descent, and aligned a vertical line's inline boxes on their alphabetic baseline. Chrome stands what Unicode's Vertical_Orientation (UAX #50) doesn't turn, sets each standing glyph by its face's `vhea`, `vmtx` and `VORG` in its `vert` alternate, and aligns the boxes of vertical lines on the central baseline.

## Decision

- `ui/internal/orient` holds UAX #50's table, generated from the UCD's `VerticalOrientation.txt` by `go generate` (Unicode 17.0.0), never read at build or test time. Under `mixed` a grapheme cluster stands when its first character is U, Tu or Tr.
- A standing cluster is shaped with `vert` before the text's own features and takes the sum of its glyphs' `vmtx` advances down the line; its baseline stands below its room's top by `VORG`'s origin, else its first glyph's top side bearing above its ink, as Blink's `OpenTypeVerticalData`. A face without `vhea` and `vmtx` keeps its ascent and descent. `vrt2` is left alone, as HarfBuzz and Chrome leave it.
- Under `mixed` and `upright`, layout measures each face about the central baseline: half its ascent and descent each side, in whole device pixels, the odd one above. A fragment's baseline stays the alphabetic one paint draws on. An atomic inline without a baseline makes one up across the middle of its margin box, and `vertical-lr` mirrors atomics across their line as it mirrors the text.

## Alternatives

- Keep the block list: emoji, ©, § and the punctuation UAX #50 stands turn, and every Unicode release drifts further.
- Fetch the table at build or test time: the build would need the network.
- Apply `vrt2` to the Tr characters: Chrome doesn't, so the lines would differ from its.

## Consequences

Symbols, emoji, CJK brackets and stops stand where Chrome stands them, at the advances vertical faces give, and a vertical line of several font sizes or inline-blocks lines up as in Chrome. Regenerating the table for a new Unicode version is one `go generate ./ui/internal/orient`.

# 0217. Line metrics round to device pixels as Chrome's

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

ui rounded a face's ascent and descent to device pixels but kept its line gap whole, floored half the leading in points and took a line height as computed. Against headless Chrome on macOS, from 8 to 48 px by halves at one and two device pixels a point, lines of Arial or Times New Roman were up to a pixel off in height, and at two pixels a point baselines of every face sat half a point high wherever the leading was odd. The machine's Helvetica, Times and Courier stood about 15% shorter than Chrome's.

## Decision

- A face's ascent, descent and line gap (its own or its `@font-face` overrides', after `size-adjust`) round each to whole device pixels, as Blink's `SimpleFontData` rounds Skia's metrics (`SkScalarRoundToScalar`, `lroundf`); `line-height: normal` is their sum.
- On macOS, the machine's faces of the families Times, Helvetica and Courier add to their ascent 15% of their rounded ascent and descent, rounded, as Blink's `FontMetrics::AscentDescentWithHacks` does after Safari; a program's or a page's face of that name keeps its metrics.
- A `line-height` of a number or a length floors to the layout unit, 1/64 of a device pixel (`LayoutUnit::FromFloatFloor`). Half the leading above the glyphs is halved in layout units, truncated toward zero as C++ divides, then floored to whole device pixels (`CalculateLeadingSpace`). `vertical-align: sub` and `super` move a box by a fifth or a third of its parent's font size and one device pixel more, cut to the layout unit, as Blink works them out in device pixels.
- On Linux, Chrome's metrics are FreeType's through Skia: hhea's (OS/2's typo metrics when `USE_TYPO_METRICS` says so), rounded alike, but with subpixel positioning (on at more than one device pixel a point) a descent rounded down takes a pixel from the ascent. On Windows they are DirectWrite's, from OS/2's `usWinAscent` and `usWinDescent` unless `USE_TYPO_METRICS`, the line gap what makes up hhea's line spacing. ui measures alike everywhere, from hhea, so a face whose tables disagree may stand a device pixel apart from Chrome there; Go and Go Mono agree.

## Alternatives

- Rounding the sum of the metrics once: Chrome rounds each, and the sums differ by a pixel at many sizes.
- Platform switches for Linux's and Windows' metrics: ui would lay a page out differently by machine, which no test here could check.

## Consequences

`TestLineHeightsAndBaselinesMatchChromesAtEverySize` holds to the layout unit for Go, Go Mono, Noto Sans with and without overrides and size-adjust, and macOS's Arial, Georgia, Times New Roman, Verdana, Menlo, Times and Helvetica, with `line-height` normal, 1.25, 1.37 and 13.3px at one and two device pixels a point. Text moves by up to a device pixel where these differed. Chrome's own Courier on macOS changes height by a device pixel with the faces the page holds, so it stays out of the test. The generic families remain the engine's faces, where Chrome's are the machine's.

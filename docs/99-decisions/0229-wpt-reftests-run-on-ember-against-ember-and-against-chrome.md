# 0229. WPT reftests run on ember against ember, and against Chrome

- Status: Accepted
- Date: 2026-11-24
- Area: tools
- Owner: engine

## Context

ui is checked against Chrome page by page, on pages written for it; web-platform-tests holds thousands of CSS reftests, a test and a reference drawn alike when the property works, which no harness of ember ran.

## Decision

- `ui/internal/reftest` reads the reftests of a WPT checkout's MANIFEST.json (light's, at its `wpt.lock`; never downloaded) and builds each page and reference as a Document with `ui.BrowserSheets()`, a scrolling viewport and overlay scrollbars, and Ahem from the checkout's `fonts/`; no system fonts, so the pictures are the same on every machine.
- ember draws both, at WPT's 800×600 and 1 pixel a point, in hidden worker processes that each draw many pages through the window capture; a page that hangs past its time or crashes fails alone and a new worker takes the next. The pictures are compared under the test's `fuzzy` rules, references chained and alternatives tried as WPT's runner does.
- What ui can't load is skipped with its reason: scripts (and `reftest-wait`), testdriver, print (`paged`), frames, canvas, video, MathML, references not in the sparse checkout.
- Chrome stays the reference: under the `chrome` tag each test drawn by ember is also shot by headless Chrome through `ui/internal/chrome`'s sessions, served from 127.0.0.1 with ember's own faces for the generic families, and held to the pixel pages' rule (mean difference at most 0.002). Where a reference and Chrome disagree, Chrome is right.
- `testdata/expected.txt` lists the tests that pass, `ref` and `chrome` for each comparison; `task reftest` (gpu tag) fails when a `ref` one stops matching its references, `task reftest:chrome` when a `chrome` one stops matching Chrome. Neither runs in CI yet.

## Alternatives

- Comparing every WPT page with Chrome only: a reference is free of Chrome's version and fonts, and catches what both drawings of a test share; Chrome's comparison catches what ember draws wrong alike in the test and its reference.
- Running WPT's own wptrunner with light as the product: it needs Python, a server and a browser shell around ember, and tests ember through light's page loader rather than ui's.

## Consequences

2,400 CSS tests guard ui in under half a minute without Chrome; the survey ranks what fails by property. Chrome's comparison takes hours, a Chrome a page, and its mean threshold lets a few hundred wrong pixels of an 800×600 page through, so the reference catches small faults Chrome's rule misses. Text in a face ember lacks is drawn in its own in both pictures, so a test about font matching may pass against its reference and fail against Chrome.

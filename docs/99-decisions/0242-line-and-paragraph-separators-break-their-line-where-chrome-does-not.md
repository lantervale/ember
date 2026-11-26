# 0242. Line and paragraph separators break their line, where Chrome does not

- Status: Accepted
- Date: 2026-11-26
- Area: rendering
- Owner: engine

## Context

U+2028 LINE SEPARATOR and U+2029 PARAGRAPH SEPARATOR are mandatory breaks in UAX #14 (class BK), and U+2029 ends a bidi paragraph in UAX #9 (class B). WPT's css/CSS2/bidi-text/bidi-breaking-003 holds a page to that; Chrome 158 breaks at neither, drawing each as a space of its own, and ember drew the face's missing glyph.

## Decision

- Both end their line under every white-space, as a preserved newline does, and draw nothing; collapsible spaces after them go as after a newline.
- U+2029 also ends the bidi paragraph it is in, so the text after it is ordered apart.

## Alternatives

- Chrome's spaces: ember follows Chrome in what CSS leaves open, but here the Unicode algorithms CSS defers to say otherwise, and the owner's goal is the CSS 2.1 suite whole.
- Breaking only where white-space keeps newlines: no rule says so, and the test's pre-lines falls back to normal in Chrome and Firefox alike.

## Consequences

Pages with these separators lay out as in Firefox and the specifications, not as in Chrome; no Chrome comparison holds a case of them.

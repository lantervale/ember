# 0081. System fonts are an option of a Document

- Status: Accepted
- Date: 2026-11-17
- Area: api
- Owner: user

## Context

A Document drew only its own fonts: ember's faces and the files its sheets name in `@font-face`. A page written for browsers names installed families (`font: 12px georgia, serif`), and showed in ember's sans-serif, its lines breaking elsewhere than in Chrome.

## Decision

- `ui.SystemFonts()` is a Document option, off by default: with it `font-family` also resolves family names, in any case, against the fonts installed on the machine, after the Document's own families.
- The fonts folders are scanned once per process, off the frame, reading only names and weights; a face's file is read when text first asks for it. Text laid out before the scan lands shows in the next family of its list, then is laid out again.
- The generic families (`serif`, `sans-serif`, `monospace`) stay ember's faces, so a Document looks the same with or without the option until a sheet names an installed family.
- The reader is ember's own, in pure Go over x/image's sfnt, with collections (`.ttc`) unpacked; no font library of the system is called. In the browser the option finds nothing.

## Alternatives

- On for every Document: a game's screen would change with the machine it runs on, and a typo in a family name would be hidden on the author's.
- Map the generic families to the system's too: the root's `sans-serif` would turn every Document with the option to Helvetica or Segoe UI.
- Ask the system (CoreText, DirectWrite, fontconfig) for a family: cgo in `ui`, which servers and the browser build must not have.

## Consequences

A screen that uses the option depends on the machine: it is for tools that show other people's pages, such as the web explorer, and a game ships its fonts with `@font-face`. A font x/image cannot read (some CJK collections, bitmap fonts) is a Document warning and its family falls to the next name; variable fonts draw their default instance; Chrome's own ascent adjustment for Times, Helvetica and Courier on macOS is not reproduced.

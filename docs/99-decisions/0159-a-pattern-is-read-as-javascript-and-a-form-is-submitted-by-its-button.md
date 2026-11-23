# 0159. A pattern is read as JavaScript, and a form is submitted by its button

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

A `pattern` went through Go's `regexp`, which reads neither lookarounds nor backreferences and takes classes the v flag refuses, so a page's pattern constrained nothing or the wrong values; an email typed with a non-ASCII domain was invalid; and with no notion of a form being submitted, `:user-invalid` never marked the fields a player had not touched.

## Decision

- `ui/internal/pattern` reads a pattern as HTML asks, JavaScript's `RegExp(pattern, "v")`, and matches a whole value by backtracking over code points: lookaheads and lookbehinds, backreferences by number and name, `(?ims-ims:…)`, `\p{…}` from Go's `unicode` tables, the v flag's nested classes, `&&`, `--` and `\q{…}`, and its syntax errors, which leave the pattern constraining nothing as in Chrome. A match undecided after a million steps is taken to match.
- An email's domain goes to punycode before it is checked, through `golang.org/x/net/idna` as Chrome's ICU does (UTS 46 nontransitional, the bidi rule, no joiner or STD3 rules), for a typed value and a `value` attribute; a value the program set with `SetValue` is checked as written, as a script's in Chrome.
- A form is submitted by a click on its submit button (pointer, keys, automation), by Enter in a one-line field (its first submit button clicked, or the form submitted when that field is its only one blocking implicit submission), and by `doc.RequestSubmit(form)`. Every field of it then matches `:user-valid` or `:user-invalid`; unless `novalidate` or the button's `formnovalidate`, an invalid field stops it and takes the focus; else the form emits `Submitted` (`OnSubmit` on the form). Nothing is sent anywhere.

## Alternatives

- Translating a pattern to Go's syntax: lookarounds and backreferences have no translation, and the v flag's errors would still be Go's.
- A copy of the IDNA tables in ui: `x/net` is already in the module's requirements, and its `idna` package is the maintained UTS 46.
- Submitting through a new event: `Submitted` already means "the player asked to go"; on the form it is that for the whole form.

## Consequences

Patterns written for browsers constrain as they do there, at the cost of a matcher of our own; Go's tables lack the emoji and case-mapping properties, script short names and `Script_Extensions`, which leave such a pattern unread. A program reacting to Enter in a field still hears the field's `Submitted`, and now its form's.

# 0124. A field's validity is a state ui keeps

- Status: Accepted
- Date: 2026-11-22
- Area: api
- Owner: user

## Context

`:valid`, `:invalid`, `:in-range`, `:placeholder-shown`, `:indeterminate`, `:default` and `:user-invalid` read a field's value and what the player did to it, which live in ui's controls, not in the tree css matches; and a form or fieldset is `:invalid` by the fields under it.

## Decision

- ui works out constraint validation as Chrome does (`required`, `pattern`, `min`, `max`, `step`, `minlength`, `maxlength` once the player edited, `type=email`, `url`, `number`, a required select left on its empty first option, a radio group none of which is checked) once a frame after syncing its controls, and keeps the answers as states on the elements (`dom.Valid`, `Invalid`, `UserValid`, `UserInvalid`, `InRange`, `OutOfRange`, `PlaceholderShown`, `Indeterminate`, `Default`), as it keeps `:checked`; a form or fieldset takes `Invalid` when a field under it has it.
- css reads those states like `:hover`, so the state invalidation already in place (decision 0113 for `:has()`) restyles what a change of them reaches. What attributes alone answer stays in css: `:required`, `:optional`, `:dir()`, a progress without a value as `:indeterminate`.
- A program reads it with `doc.Valid(n)`, a form or fieldset by its fields, and sets a checkbox `doc.SetIndeterminate(n, on)`.

## Alternatives

- css asking ui for a field's validity through a hook while it matches: every match of `:valid` would rerun the checks, and css could not tell which elements a typed letter restyles.
- Validity worked out in css from attributes alone: it never sees the typed value.

## Consequences

Every frame checks every control, cheaply: no string is made for a field whose constraints need none, so a focused plain field still allocates nothing. Validity follows the frame: a `Matches` between two frames reads the last frame's answer, `doc.Valid` the value as it is.

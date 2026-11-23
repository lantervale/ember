# 0195. line-break strictness reclasses characters before UAX #14

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Chrome breaks lines with ICU, whose rules for `line-break: auto`, `loose`, `normal` and `strict` differ on small kana, iteration marks, CJK hyphens, inseparable characters, centred punctuation, postfixes and prefixes, and pick a set by the element's language. ui broke every value by uniseg's UAX #14 rules, which are `strict`'s, so even `auto` kept small kana off a line's start where Chrome lets them start one. `line-break: anywhere` broke inside a text run but not where one element's text met the next.

## Decision

- The computed style carries a `Strictness` resolved from `line-break` and the element's language (its `lang`, else the Document's), as measured in Chrome: Chinese, Japanese and Korean take the CJK rules of `normal` and `loose`, Chinese those of `normal` for `auto` too, Korean has no `strict`, every other language takes the plain ones. A `lang` changed above an element restyles it.
- Before uniseg reads a block's text, the characters a strictness breaks around as around an ideograph (UAX #14's CJ but in `strict`; iteration marks in `loose`; 〜 and ゠ in the CJK rules; centred punctuation in loose CJK) are replaced by an ideograph of as many bytes, so the byte offsets of its breaks stand.
- What loose changes without a class of its own is a pass after it: between two inseparable characters; in loose CJK before a CJK hyphen after an ideograph (and none after it before a letter), before a postfix after an ideograph or a digit, before a full-width prefix after a letter and after it.
- `anywhere` also breaks where its text ends and another element's starts, as Chrome breaks between elements by the rules of the text before.

## Alternatives

- ICU's rule files compiled to Go: exact, but a second segmenter beside uniseg for a handful of characters.
- Following the specification's lists rather than Chrome: Chrome's sets differ from them (`auto` is `normal` but in Chinese, `%` and `$` stay put, ° breaks only after an ideograph), and ui is checked against Chrome.

## Consequences

Japanese text under the default `auto` now starts lines with small kana and prolonged sound marks as Chrome does. A text with no character from U+2000 on costs nothing more. Blink's own table for pairs of ASCII characters, which overrides ICU between them, is not followed: such pairs break as UAX #14 has them.

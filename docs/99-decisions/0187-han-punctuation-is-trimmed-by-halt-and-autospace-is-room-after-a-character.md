# 0187. Han punctuation is trimmed by halt, and autospace is room after a character

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Chrome sets the full-width punctuation of Chinese and Japanese half wide where two stand together (`text-spacing-trim`, its HanKerning, `normal` by default) by the face's `halt` feature, and with `text-autospace: normal` puts an eighth of the font size between an ideograph and a letter or a digit (its TextAutoSpace). Both reach across elements and change where a line starts and ends. ui read the two properties and laid out neither.

## Decision

- Once a block's text is cut into words, one pass reads its characters across elements as Chrome types them: brackets by general category and East Asian width, the dots, colons and curly quotes by the ink and advance of their glyph in the face drawing them (left: a dot; centred: a middle dot; an em wide: a full-width quote). Where Chrome's rules set one of a pair half wide, the character takes what `halt` does to its glyph, read by shaping the glyph alone with and without it, unless its own `text-spacing-trim` is `space-all` or its face has no `halt`; where an ideograph and a letter or digit meet and both say `normal`, the first takes an eighth of its font size after it. A word's width takes the room.
- A line's start and end set what Chrome sets there: an opening bracket starting a wrapped line is full width under `normal` and `space-all`, half under `space-first` and `trim-start` (`trim-start` also on the block's first line and after a forced break); the room after the last character goes where the line breaks inside its element's text; a closing bracket is trimmed where that alone fits a line, the breaker measuring the line so.
- A fragment carries the room by byte; paint draws its text in pieces moved by it, and selection measures through it.
- A line may break before an opening curly quote and after a closing one between wide characters (Unicode 15.1's LB19a), as Chrome breaks them; uniseg's rules are 15.0's.

## Alternatives

- `halt` as an OpenType feature over ranges of a run, in internal/shape: drawing and measuring would shape across a fragment's words and know their line's ends; `halt` is a single adjustment a glyph, which room by character reproduces without it.
- A face's own `chws`, which Chrome hands the middle of a run to: neither machine face here has it, and Chrome's rules are what a face's `chws` follows; pairs are decided by the rules alone.

## Consequences

Text without ideographs, curly quotes or middle dots costs one scan. Vertical text (`vhal`) and text fields are left untrimmed and unspaced, and where Chrome itself differs between equal cases (an opening bracket starting a block under `trim-start`) ui follows the specification. A closing bracket the content's last line would overflow by is trimmed as Chrome 158 trims it, where Chrome 157 and older measure the line fitting by the trim but draw the bracket whole: the Chrome tests of it skip an older Chrome.

# 0089. A generated box is a node of its element

- Status: Accepted
- Date: 2026-11-17
- Area: api
- Owner: user

## Context

ui read two pseudo-elements, `::placeholder` and `::selection`, one property each, and refused the rest (decision 0049's subset). Pages written for browsers draw with `::before` and `::after`: CSS Zen Garden 221 shows its numerals and a photograph that way, and the web explorer showed neither. Lists numbered themselves in layout and `q` wrote its marks there, out of reach of a sheet.

## Decision

- Pseudo-elements are part of the CSS subset, always on, as Chrome has them. `::before`, `::after` and `::marker` are supported, `::placeholder`, `::selection` and `::backdrop` take what ui can show, and every other one Chrome reads is in one table (`ui/css/pseudo.go`): refused with a warning, or read and matching nothing. A test asks Chrome for the table.
- A `::before` or `::after` box is a node the cascade makes and keeps on its element, one per element and pseudo-element, outside the element's children. Layout lays out an element's flow: the `::before` node, the children, the `::after` node. So the box keeps its layout, its transitions and its place in the box tree as any node does.
- The node is no element. Selectors, `Children()`, `ByID`, automation, selection and translation never see it, and a pointer over it is over its element.
- `content` takes strings, `attr()`, `url()`, counters and quotes. Counters are one pass over the tree after the cascade, run only in a Document that counts and only when what it reads changed. A list's numbers are the `list-item` counter of that pass, and `q`'s marks are `q::before` and `q::after` in the user-agent sheet: one mechanism each, a sheet's to change.
- `::first-line` and `::first-letter` are refused with a warning.

## Alternatives

- Make the boxes in layout, with no node: every cache keyed by node (retained layout, transitions, what paint shows) would need a second key, and a hover that restyles a box would lay its element out again.
- Add real child elements: `:first-child`, `:empty`, `Children()` and automation would see them, and a game's own tree would change under it.
- Keep list numbers and quotation marks in layout beside counters: two answers to one question, and `counter(list-item)` would disagree with the marker.

## Consequences

A Document whose sheets have no pseudo-element rule for an element's bucket looks for none, and one that counts nothing skips the counter pass; the cascade pays for five more properties. A line or a letter can't be styled apart, `::marker` shows colour, font and content only, and a generated box on an `img` or a table part never appears, as or beyond Chrome.

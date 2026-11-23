# 0212. A picture takes its natural size in its own writing mode

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Pictures (img, svg, canvas, iframe) had their natural size turned into their parent's frame. A picture of another writing mode than its parent, or one positioned out of a vertical box into a horizontal containing block ([0204](0204-a-positioned-box-is-laid-out-in-its-containing-blocks-frame.md)), is sized in its own frame, so it stood turned: a 40×16 picture took 16×40. An absolutely positioned picture's min and max sizes were clamped one side at a time, never carried through its ratio, an `aspect-ratio` alone set nothing, and a range's sizes were rules of ui's sheet, so `height: auto` left it empty and `slider-vertical` stayed 16 tall.

## Decision

- A replaced box's natural size and proportions are turned into its own writing mode, where it is laid out whatever its parent's or its containing block's frame.
- An absolutely positioned picture with neither size set is sized as an inline one (its min and max sizes carried through its ratio); an `aspect-ratio` that isn't `auto` gives a picture its natural inline size and the block size its ratio makes.
- A range's sizes are no rules of ui's sheet but its natural content, as Chrome's shadow tree gives it: 129 along its track, 16 across a range of its own look, 129 across one of `appearance: slider-vertical` in horizontal lines; a block shrinks it to them, a grid or a flexbox may stretch it, and one laid out as a row round its `::before` and `::after` keeps its track's length whatever they hold. Export writes those a range of ui's own look leaves auto, as Chrome's bare ranges lack them.

## Alternatives

- Turn the natural size again where a positioned picture is placed in another frame: every frame it passes through would need its own turn, where a picture's own frame is the one it is measured in.
- Keep the range's sizes in ui's sheet with a pseudo-class for slider-vertical: a second cascade like `:-ember-native`'s, and an author's `height: auto` still empty.
- Fill a range's auto sizes in the cascade: it would never stretch in a grid or a flexbox.

## Consequences

Pictures across writing modes, positioned or not, and slider-vertical ranges take Chrome's sizes. A range's computed size stays `auto` as Chrome's; its natural size lives in layout, beside a picture's.

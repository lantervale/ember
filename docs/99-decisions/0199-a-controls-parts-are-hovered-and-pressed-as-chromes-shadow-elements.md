# 0199. A control's parts are hovered and pressed as Chrome's shadow elements

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Pages style a range's thumb and track, a progress's and a meter's value, on `:hover` and `:active`, and Chrome shades its own controls' accent as the pointer moves over them. ui's parts are styles of their element, not elements (decision 0160), so nothing held a part's hover or press; a range only ran along its width, and its thumb of ui's own look ignored the accent.

## Decision

- An element carries its parts' states beside its pseudo-classes (`dom.ContainerHover`, `TrackHover`, `ThumbHover` and their Active ones), nested as Chrome's shadow tree: over the thumb is over the track and the container. The Document finds them under the pointer from paint's layout of the parts (`paint.ControlPartsAt`) and keeps those a press began on until it ends; the cascade matches a part's `:hover` and `:active` against them, and `::-webkit-slider-container` is a part of its own.
- A range is laid out in its own flow: its parts' layout is written for a horizontal range and read turned for a vertical one (writing-mode, rtl, sideways-lr, slider-vertical of ui's own look), the same layout giving paint, the pointer's parts and the value a press or a drag sets, a press on the thumb keeping where it was taken.
- An accent is shaded as Chrome shades it: its HSL lightness 0.11 darker hovered and lighter pressed, the other way round for a light accent, which a range judges by luminance from 0.256 (measured between two greys) and a checkbox by the tick it takes. A range of ui's own look draws its thumb in the accent when one is set.

## Alternatives

- Make the parts real elements of a shadow tree: what decision 0184 turned down, for three or four elements more per control.
- Pin the parts' states on hover of the whole control: a track hovered by the pointer off it, a thumb never alone, unlike Chrome.

## Consequences

Hovered and pressed parts, vertical ranges and the accent's shades match Chrome's; the kit's slider shows its accent thumb at rest. The states follow the mouse only: a part has no `:focus`, and touch presses parts as the mouse does.

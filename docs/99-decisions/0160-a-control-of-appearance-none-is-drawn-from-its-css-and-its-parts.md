# 0160. A control of appearance: none is drawn from its CSS and its parts

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Pages restyle their controls the way Chrome lets them: `appearance: none` on a checkbox, a select or a button, and a range's, a progress's or a meter's parts through `::-webkit-slider-thumb`, `::-webkit-progress-value`, `::-webkit-meter-bar` and the rest. ui read `appearance` and changed nothing, refused the parts, and drew `accent-color` only in a range's fill and a progress's value.

## Decision

- `appearance` computes as Chrome's, the user agent's sheet giving controls `auto`. Its rules of a control's own look (the 13-point checkbox, its blue when checked, the select's chevron, the fields' thin rounded borders) carry a user-agent pseudo-class, `:-ember-native`, that holds only while the control's appearance isn't `none`: the cascade reads the element's appearance first, and matches again without those rules when it is `none`. What is left is Chrome's sheet for every control: 2-point inset fields, outset buttons, an empty 0 by 0 checkbox.
- A control of appearance `none` draws no mark: no tick, no dot, no switch thumb, no range track.
- The parts of a range, a progress and a meter are pseudo-elements of their own, read in every sheet under their prefixed names, each inheriting from the part it is in. A progress or a meter of appearance `none` is its inner element, bar and value boxes, a meter's bar half its height and its value styled by its region; a range of appearance `none` is its track, centred and as tall as its thumb unless sized (and the range as tall as the track when its height is auto), and its thumb at the value along the track less its width, Chrome's own disc unless its appearance is `none`. A meter of its own look takes its styled parts' colours.
- A checked checkbox and radio of their own look whose `accent-color` is set are filled and ringed in it, a checkbox's tick white or Chrome's dark grey, whichever contrasts more.
- The exported page keeps Chrome from drawing its own controls in a style element of its own, and writes `:-ember-native` as `:not([data-ember-bare])`, marking the controls of appearance `none`.

## Alternatives

- Drop ui's own looks from the user agent's sheet and paint them all, as Chrome's theme does: every look ui draws in CSS today (and the kit restyles) would move into Go.
- Leave the look rules unconditional and have paint ignore them under `none`: the 13-point checkbox would keep its size and borders, and the boxes wouldn't be Chrome's.

## Consequences

The boxes and pictures of these controls match Chrome's; a range's thumb of ui's own look keeps its `color`, the convention the kit's slider is built on, and the parts' hovered and pressed states aren't drawn.

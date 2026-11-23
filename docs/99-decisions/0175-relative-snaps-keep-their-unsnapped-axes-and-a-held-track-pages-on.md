# 0175. Relative snaps keep their unsnapped axes, and a held track pages on

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Probing Chrome 157 after [0169](0169-the-users-scrolls-glide-on-chromes-curves-and-two-axis-snaps-land-where-both-areas-show.md), diagonal `scrollBy`s and first layouts of two-axis containers landed elsewhere than ui, a held click on a track paged once, and only macOS told a trackpad's turn from a mouse's. `scroll-snap-align: end start` looked swapped, but Chrome reads block then inline as the spec, as ui did: the difference was the relative scroll's.

## Decision

- A relative scroll (`ScrollBy`, a notch, a key, a track) that snaps on some axis keeps where it began on the others, as Chromium's `FindSnapPosition` starts from the strategy's current position; one that finds no area on any axis, even nearest to where it began, lands where it was sent, unsnapped. The mutual search for an area aligned on both axes stays the absolute scroll's.
- The snap after a layout, the first included, keeps the block axis's area when the two found aren't seen together, or the axis's the container alone was snapped on, as Chrome's target-prioritizing strategy; `SetScroll` keeps the nearer.
- A click held on a track pages on after 250 ms at twenty pages a second, linear, while the pointer stays on the track past the thumb, as `ScrollbarController`'s autoscroll; let go, it snaps ahead from where the repeat began.
- A wheel turn is precise where the platform shows it: a Windows `WM_MOUSEWHEEL` of parts of `WHEEL_DELTA` (at Chrome's 100 points a notch), a Wayland axis off the compositor's notches of ten, a web `WheelEvent` of pixels whose `wheelDelta` isn't whole notches of 120. X11 stays notches.

## Alternatives

- Read Windows' fractional turns as fractional notches, as Chrome does for `WM_MOUSEWHEEL`: a precision touchpad, which Chrome reads through DirectManipulation that GLFW doesn't have, would snap a container a whole item at each of its small turns.
- Read X11's smooth scrolling through XInput2 ourselves: GLFW owns the event loop and reads the core buttons only.
- Port Chrome's covering candidates that dodge intruding areas: a large algorithm of its own for areas as big as the snapport, left in 10-missing.

## Consequences

Scattered two-axis containers land where Chrome 157 lands, scrollTo and diagonal scrollBy alike, their areas smaller than the snapport. A high-resolution wheel on Windows moves at once by its points where Chrome glides it.

# 0169. The user's scrolls glide on Chrome's curves, and two-axis snaps land where both areas show

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

[0163](0163-scrolls-snap-and-glide-as-chromes-strategies-on-the-documents-clock.md) glided the program's scrolls only: the wheel, the keys and a click on a track jumped, a trackpad's two fingers moved three lines a tenth of a point, and a container snapping on both axes picked each axis's nearest area apart, landing where neither might show. Chrome on the desktop glides the user's scrolls and snaps a two-axis container to areas seen together; a game reading a list the frame after it scrolls wants the old jump.

## Decision

- The wheel's notches, the keys and a click on a track glide on Chromium's `ScrollOffsetAnimationCurve`: ease-in-out (`cubic-bezier(0.42, 0, 0.58, 1)`), 9 frames of 1/60 s for the keys and the track, 12 frames down to 6 for a notch the farther it goes (from 120 to 480 points). A scroll during a glide goes on from where the glide is going, keeping its speed through the curve's first slope and lasting no longer than that speed takes, as `UpdateTarget`. `ui.InstantScrolls()` (`doc.SetInstantScrolls`) has them move at once; the window's Document rules its iframes, as `WheelScroll`.
- `input.State.WheelPrecise()` says a frame's wheel came from a trackpad or a smooth wheel: AppKit's `hasPreciseScrollingDeltas`, seen by an event monitor before GLFW scales the points to tenths. Such a turn moves at once by its points, and its gesture snaps once it ends, as 0163; a notch snaps as it turns, as a `ScrollBy`.
- A page key, Space and a track page by Chrome's `CalculatePageStep`, seven eighths of the snapport (at least all but 40 points on macOS), and snap preferring the positions 0.4 to 1 page ahead; an arrow scrolls 40 points and snaps to the first position ahead. With nothing focused, the arrows and Space scroll only a Document with `ScrollingViewport()`: on a game's screen they stay the game's.
- A two-axis container searches each axis among the areas meeting the snapport where the scroll goes on the other; when the two found don't show together, the axis nearer its aim is kept and the other searched again among the areas seen with it; with none in either corridor, an area aligned on both axes nearest it, as `FindSnapPosition` and `FindSnapPositionForMutualSnap`. Of areas lined up alike, one aligned on both axes is the one snapped to. A mandatory scroll with nothing ahead takes the nearest to where it began.
- A drag of the selection past a snap container's edge scrolls it freely; let go, it snaps as `SetScroll` does.

## Alternatives

- Instant by default, gliding by an option: what a page's reader sees in Chrome would be the exception.
- Tell a trackpad by its fractional turns: a mouse's accelerated notches on macOS are fractional too.
- Arrows and Space scrolling on every Document, as a browser: a game's hero walking by the arrows would stop whenever the pointer is over a list.

## Consequences

The final offsets of scrollTo and scrollBy on two-axis containers match Chrome's; the curves are Chromium's code, checked headless frame by frame. Tests that read a user's scroll the next frame run the glide out or set `InstantScrolls`. Windows' and Linux' wheels, and the browser's, always count as notches.

# 0167. Pictures load, play and choose again when a desktop Chrome does

- Status: Accepted
- Date: 2026-11-23
- Area: assets
- Owner: user

## Context

Every picture was asked for when laid out, every animation played from the moment it landed whether shown or not, and an `img` chose again whenever the density changed ([0138](0138-animated-pictures-play-as-one-texture-per-address.md), [0115](0115-an-img-chooses-its-picture-as-chrome-does.md)). Headless Chrome, ui's reference, can't say otherwise: it loads every lazy picture at once and dumps the page on a virtual clock.

## Decision

- What a desktop Chrome does is measured in headless Chrome driven over its DevTools pipe in real time (`chrome.Open`), told to load lazily (`--blink-settings=lazyLoadEnabled=true`, which headless turns off), its pictures served from 127.0.0.1.
- An `img` with `loading="lazy"` shows no picture, and asks for none, until laid out within 1250 points along and 625 across the viewport and every scroll container round it (their scrollports, scrollbars out), as Chrome for Testing 156 measured on a fast network; once near it keeps its picture.
- An animation begins when first drawn, and moves only while the last paint drew it. Drawn again within the play it left, it comes back in step; past that play's end, the next play begins then, as Chrome's compositor resumes it.
- Frames past `animated.MaxPixels` are drawn as they show, from the canvas as the last held frame left it, so every frame plays; a GIF frame past the picture is cut, and a first one grows it to where it reaches, as Chrome draws them.
- An `img` chooses again when the viewport resizes, one in a `picture` with a `source` of `media` when another feature changes, and keeps its picture when only the density does: each `img` keeps the density it chose for.
- `fetchpriority="high"` pictures are asked for first among those a layout asks for, `low` last; `decoding` changes nothing, a picture never showing before it is decoded.

## Alternatives

- Chrome's documented margins without measuring: they differ by network type and across, so the measurement is the reference.
- Pausing an animation where it left and resuming from there: Chrome doesn't.
- Decoding frames past the budget again from the file each play: no reader keeps its place, and the frames held already cover the start.
- Following Chrome's whole request scheduler (its tight mode, boosting pictures in view): its order hangs on when its parser finds each picture; only the priority classes carry over.

## Consequences

A long page asks only for what is near, and animations out of view cost nothing; a picture with `loading="lazy"` shows later than one without, as in Chrome. A capture that draws only its last frame starts its animations there (`EMBER_CAPTURE_DRAW=all` plays them before).

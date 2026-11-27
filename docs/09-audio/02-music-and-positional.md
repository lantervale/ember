# Music and positional sound

Loop music in the background, crossfade between tracks, and place sounds in the world so they get louder as the hero comes near and come from his side. Here a day forest ambience loops over the meadow, and M crossfades it to a night one with owl calls, the meadow darkening with it, and back. A campfire crackles on the left and a river runs on the right: walking to one makes it louder, and each is heard from the side it is on, which changes as the hero turns. The lines at the top show what the hero hears of each, since a GIF can't. Run it to hear it.

![The hero turns a full circle between the campfire and the river, the pans swapping sides, then walks to the river](music-and-positional.gif "ember · music and positional sound")

## Streams and buses

`audio.Open` opens a stream, here an Ogg Vorbis recording. A stream is decoded as it plays rather than all at once, which suits anything long.

The mixer is made with buses, `WithBus("music", 0.5)` and `WithBus("sfx", 1)`:

- Each voice plays on one bus. A bus's volume and mute apply to all its voices at once, which is how a settings menu turns music down: `mixer.Bus("music").SetVolume(v)`, `SetMuted(true)`.
- `master`, over every bus, always exists: `mixer.Master()`. `mixer.Bus(name)` is nil for a bus the mixer wasn't made with.

## Music

`mixer.PlayMusic(stream, opts…)` plays a track on the `music` bus, or the one `OnBus` names, one track at a time. Here it loops and fades in over a second.

A track needn't be music: here the day and night ambiences are tracks, since one plays at a time in the background. `mixer.Music()` is the track playing, and `mixer.StopMusic(audio.FadeOut(d))` stops it.

[main.go](../../examples/54-music-and-positional/main.go)

## The crossfade

Playing another track with `Crossfade(d)` fades the one playing out while the new one fades in, at equal power, here over the same two seconds as the meadow's colour. Without it, the old track stops at once, over 10 ms so it doesn't click. Each track plays at its own `Volume`, to balance them.

[music.go](../../examples/54-music-and-positional/music.go)

## A listener and emitters

- **The listener** is where sound is heard from. The hero carries an `audio.Listener`, so the meadow is heard from where he stands. The first listener in the World is used; without one, emitters play as set.
- **An emitter** is a sound that plays where its entity is. The campfire and the river each carry an `*audio.Emitter`, made by `audio.NewEmitter(sound, opts…)`.

Both sounds here are streams, opened with `audio.Open` like the ambiences, since long recordings are better read as they play. An emitter takes a clip from `audio.Load` just the same.

An emitter takes the voice options and three of its own:

| Option | Default | |
|---|---|---|
| `Autoplay()` | | starts it as soon as it is in a World with a mixer; else `Play()` starts it from the beginning, on the next frame, and `Stop()` stops it, over `FadeOut` when given |
| `Distance(min, max)` | 1, 100 | heard in full up to min, silent from max, in world units: points in 2D, metres in 3D |
| `Rolloff(r)` | 1 | how quickly it fades between them |

Here `Loop` keeps each sound going and `OnBus("sfx")` plays it on the sound effects bus. Tune the distances per source: the fire, `Distance(30, 500)` and `Rolloff(1)`, carries less far than the river, `Distance(80, 700)` and `Rolloff(0.7)`, so each one dominates near itself.

An emitter's pan comes from where it is beside the listener. `Heard()` says how loud and from which side the listener hears it, which the HUD prints.

[scene.go](../../examples/54-music-and-positional/scene.go)

## Turning the listener

A listener hears from where its entity faces, so turning the hero turns his ears:

- **In 2D**, a listener at rotation 0 faces up the screen, and turning it clockwise turns its ears with it: after a half turn, the fire on the left is heard on the right.
- **In 3D**, a listener faces down its transform's -Z, so it goes on the camera.

Q and E turn the hero, and walking turns him to face his way. A yellow arrow shows where he faces, and the HUD's pans change as he turns. `Face` turns the hero but keeps his sprite upright, mirrored to the side he faces, so only the ears and the arrow turn. The hero walks as on the [Keyboard](../08-input/01-keyboard.md) page, kept inside the window.

[walk.go](../../examples/54-music-and-positional/walk.go)

## Credits

The sounds are recordings from Freesound, all under CC0 1.0:

- the river: ["Little river" by AudioPapkin](https://freesound.org/people/audiopapkin/sounds/815124/)
- the campfire: ["Campfire 01" by HECKFRICKER](https://freesound.org/people/heckfricker/sounds/729395/)
- the day: ["ForestDay_BW.66" by sonidosreales245](https://freesound.org/people/sonidosreales245/sounds/517042/)
- the night: ["Forest owl calls in the night city atmosphere" by Garuda1982](https://freesound.org/people/garuda1982/sounds/721046/)

## Try it

[54-music-and-positional](../../examples/54-music-and-positional)

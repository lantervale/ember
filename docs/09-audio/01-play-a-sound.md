# Play a sound

Play a short sound when something happens, as many times at once as needed, higher or lower and to the left or right. Here Space plays a short pling, a little higher or lower each time, and a click plays it from the side of the window it lands on. A ring shows where each one played, and the line at the top counts the sounds playing and finished. Run it to hear it.

![Rings where the sound plays: at a click on the left, in the middle, at a click on the right](play-a-sound.gif "ember · play a sound")

## Loading a clip and adding the mixer

`audio.Load` reads a clip, here a WAV file embedded with the example, which the `audio/wav` import decodes. The mixer is an entity, added to the World with the window.

`audio.NewMixer` takes options of its own:

| Option | Default | |
|---|---|---|
| `MaxVoices(n)` | 64 | the most voices playing at once |
| `SampleRate(hz)` | 48000 | the output rate, from 8000 to 192000 |
| `WithBus(name, volume, …)` | | a bus: see [Music and positional sound](02-music-and-positional.md) |

[main.go](../../examples/53-play-a-sound/main.go)

## Playing it

`mixer.Play(clip, opts…)` starts a voice of the clip. The same clip can play as many voices at once as needed, and `Voices()` counts those playing, paused or fading out.

The options shape the voice:

| Option | Default | |
|---|---|---|
| `Volume(v)` | 1 | louder or quieter |
| `Pitch(p)` | 1 | faster and higher, or slower and lower: 2 is an octave up; from 1/16 to 16 |
| `Pan(p)` | 0 | from the left ear, -1, to the right, 1 |
| `Loop()` | | plays again each time it ends, until stopped |
| `FadeIn(d)` | | rises from silence over d |
| `OnBus(name)` | master | the bus it plays on; an unknown name plays on master |
| `Priority(n)` | 0 | past `MaxVoices`, a new sound takes the place of the oldest of the lowest priority, or is dropped if every voice outranks it |
| `Effects(…)` | | effects on this voice alone: see [Effects](03-effects.md) |

[play.go](../../examples/53-play-a-sound/play.go)

## Changing a voice while it plays

`Play` returns the `*audio.Voice`:

- `Stop()` stops it, with `audio.FadeOut(d)` if wanted.
- `Pause()` and `Resume()` hold it and let it go on.
- `SetVolume`, `SetPan` and `SetPitch` change it while it plays.

## When a voice ends

The World gets an `audio.VoiceFinished` event, read like any other. Its `Voice` is the voice, and `Stopped` is true when it was stopped, or gave its place to another, rather than played to its end. A looping voice sends `audio.VoiceLooped` each time it starts over.

## Try it

[53-play-a-sound](../../examples/53-play-a-sound)

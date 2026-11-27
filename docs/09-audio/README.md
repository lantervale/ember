# Audio

Play sound effects, music and sounds placed in the world, through effects such as reverb. A page can't play sound: its GIF shows what the example draws while it plays, so run the example to hear it.

## The mixer, clips and streams

- **The mixer** plays every sound. `audio.NewMixer()` is an entity: the game adds it to the World next to its window.
- **A clip** is a short sound, decoded whole into memory by `audio.Load`.
- **A stream** is music or a long ambience, opened by `audio.Open` and decoded as it plays.
- **Formats** come as imports, `_ "github.com/lantervale/ember/audio/wav"` and `_ "github.com/lantervale/ember/audio/vorbis"`, so a game only builds in the decoders it uses.

## The pages

1. [Play a sound](01-play-a-sound.md): a clip, its pitch and pan, voices.
2. [Music and positional sound](02-music-and-positional.md): buses, music and crossfades, emitters heard from a listener.
3. [Effects](03-effects.md): reverb, filters and echo, on a bus or on one sound.

## No sound device needed

`ember.New` installs audio. The sound device opens only when a game runs in a window. A server, a test or a hidden capture mixes into a silent output at the World's pace instead, so voices still play, end and send their events.

In a test, `audio.NewRecorder(mixer)` keeps what was mixed:

- after `RunFrames`, or `Mix(frames)` to mix at once, `Samples()` holds it, interleaved stereo, and `Frames()` counts it;
- `Peak(from, to)` says how loud it was, and `PeakLR(from, to)` on which side;
- `Reset()` forgets it.

Streams decode at once there, so a test hears the same thing every run.

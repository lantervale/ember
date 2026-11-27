# Keyboard

Read keys for movement and shortcuts, and typed text for names and chat. Here the hero walks with WASD or the arrows and runs while Shift is held. Enter starts naming him: what the player types goes into his name tag, Backspace removes a letter, Ctrl or Cmd with V pastes, and Enter again ends it.

![The hero walking, running, then being renamed Heron](keyboard.gif "ember · keyboard")

## Reading keys

A window's input, `win.Input()`, answers three questions about a key:

| Method | True when |
|---|---|
| `Down(key)` | the key is held |
| `Pressed(key)` | the key went down this frame |
| `Released(key)` | the key came up this frame |

A press lasts one frame, however long the key is held.

Two helpers read several keys as one value:

- `Direction(left, right, up, down)` reads four keys as a direction. It is normalised, so diagonals are no faster, and y points down, as on screen.
- `Axis(neg, pos)` reads two keys as one number: -1, 0 or 1.

## Keys are places, not letters

A key is named after where it sits on a US keyboard. `KeyW` is the key AZERTY prints Z on, so WASD walks with the same fingers on any layout.

A modifier can be read two ways:

- `KeyShift`, `KeyControl`, `KeyAlt` and `KeySuper` match the key on either side.
- `KeyLeftShift`, `KeyRightShift`, `KeyLeftControl`… match one side only.

[walk.go](../../examples/47-keyboard/walk.go)

## Reading typed text

What the player types is text, not keys. `Text()` is the characters typed this frame, as the layout and the system make them: accents and capitals included, repeated while a key is held.

So the example reads the name from `Text()`, and reads Enter and Backspace, which type nothing, as keys.

The clipboard is on the window: `win.Clipboard()` reads its text and `win.SetClipboard(text)` writes it.

## Keeping typing out of gameplay

While the name is typed, walking stops, or every W in the name would walk the hero. The UI does the same for the game:

- `KeyboardCaptured()` is true while a UI text input has the keyboard. Every key then reads as up.
- `Captured(key)` is true for the keys a focused UI menu moves by. It takes only those; the others still reach the game. See the [Controls](../05-ui/09-walkthroughs/03-controls.md) page.

[naming.go](../../examples/47-keyboard/naming.go)

## The window

The name tag is the hero's child, so it walks with him.

[main.go](../../examples/47-keyboard/main.go)

## Testing

`SetKey(key, down)` holds or lets go of a key, and `AddText(r)` types a character. The next frame sees both.

## Try it

[47-keyboard](../../examples/47-keyboard)

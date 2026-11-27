# Input

Read what the player does with keys, the mouse, gamepads and the trackpad, from any system.

## One snapshot a frame

Each window has its input, `win.Input()`: what its keys, mouse buttons, pointer, wheel, gamepads and trackpad did this frame.

- It is a snapshot taken before the frame's first system runs, so every system sees the same keys down.
- A press lasts exactly one frame.
- Input that arrives between frames, from the player or from a test, shows in the next one.

## The pages

They build on the [2D](../06-2d/README.md) section's sprites and cameras.

1. [Keyboard](01-keyboard.md): keys by place, typed text, the clipboard.
2. [Mouse](02-mouse.md): the pointer in the world, buttons, the wheel, cursors.
3. [Actions](03-actions.md): what the player means, whatever key, button or pad says it, and rebinding.
4. [Gamepads](04-gamepads.md): buttons, sticks, triggers, several players, rumble.
5. [Touch](05-touch.md): the trackpad's fingers.
6. [Gestures](06-gestures.md): pinch, turn, pan and swipe.

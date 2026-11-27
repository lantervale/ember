# Actions

An action is what the player means, such as walk left or plant, bound to the buttons that say it. The game reads actions instead of buttons, so the same code works with keys, a mouse or a pad, and the player can rebind them. Here the hero walks with WASD, the arrows or a pad's left stick, a ghost tree follows the pointer into the world, a click or Enter plants it there, and the wheel zooms.

![The hero walking, trees planted where the pointer clicks, the view zooming in and out](actions.gif "ember · actions")

## The action map

The actions are values of the game's own type. `input.NewActions` takes one `input.Bind(action, buttons…)` per action:

- Keys, mouse buttons and pad buttons bind alike, so planting answers the left button, Enter and the pad's south button.
- Pad buttons are named by where they sit: `GamepadSouth`, `GamepadEast`, `GamepadWest` and `GamepadNorth`. One binding means the same button on every brand of pad.
- Each way a stick leans is a button too, such as `GamepadLeftStickLeft`.
- The action type's `String` names each action when the bindings are saved.

[controls.go](../../examples/49-actions/controls.go)

## Reading actions

The map reads an `input.Reader`: a window's input, or one pad for one player ([Gamepads](04-gamepads.md)). It has the same reads as the input itself:

| Method | Gives |
|---|---|
| `Down(in, action)` | true while one of its buttons is held, a trigger or stick direction past halfway |
| `Pressed(in, action)` | true on the frame it starts: one of its buttons goes down while none was held |
| `Released(in, action)` | true on the frame it ends |
| `Value(in, action)` | how far it is held, 0 to 1: the most any of its buttons is |
| `Axis(in, neg, pos)` | two actions as one number |
| `Direction(in, left, right, up, down)` | four actions as a direction, y down as on screen |

`Direction` gives keys a full step, and diagonals are no faster. A stick gives its lean, so half a push walks at half speed. The hero is the walker of [A 2D camera](../06-2d/05-a-2d-camera.md), with the sheet's rows facing up and down too.

[walk.go](../../examples/49-actions/walk.go)

## Planting under the pointer

`Mouse()` is where the pointer is, in window points. The camera's `ScreenToWorld` turns it into the world under it, through the camera's follow and zoom, so the ghost tree stays under the pointer while the hero walks. `Wheel()` is in lines, up negative.

[garden.go](../../examples/49-actions/garden.go)

## Hints for the device in hand

A hint on screen should name what the player holds. `win.Input().LastDevice()` is the device used last:

- `input.KeyboardDevice`, the value at the start.
- `input.MouseDevice`, `input.GamepadDevice` or `input.TouchDevice`.

As each frame begins, the device changes on:

- a key press or typed text;
- a mouse button, the wheel, or a pointer move of more than 3 points;
- a pad button, or a stick or trigger past halfway;
- a new finger.

A twitch of the mouse or a resting stick doesn't change it. So a menu's footer reads "Ⓐ Choose" after a pad press and "Enter Choose" after a key:

```go
hint.SetText("Enter Choose")
if win.Input().LastDevice() == input.GamepadDevice {
	hint.SetText("Ⓐ Choose")
}
```

## The window

The ghost tree stands for the pointer, so `SetCursor(platform.CursorHidden)` hides the system one. The other cursors:

- `CursorText`, `CursorHand`;
- the resize arrows `CursorResizeEW`, `CursorResizeNS` and `CursorResizeNWSE` (on the desktop, NWSE shows as a crosshair for now);
- `CursorArrow`, which hands the cursor back to the UI: a hand over what clicks, a text cursor over inputs, the resize arrows over a resizable panel's edges.

[main.go](../../examples/49-actions/main.go)

## Saving and rebinding

- `json.Marshal(controls)` writes the bindings by name: each action as its `String` prints it, each button as it prints (`"Space"`, `"MouseLeft"`).
- `json.Unmarshal` reads them back into the same actions. Actions the map doesn't bind are ignored, so an old settings file stays readable. `input.ParseButton(name)` reads one button's name.
- `controls.Bind(PlantTree, input.KeySpace)` replaces an action's buttons, for a rebinding menu.
- `Bindings(action)` lists an action's buttons, `Bound(button)` finds the action a button already does, and `Actions()` lists the actions in the order they were bound.

## Try it

[49-actions](../../examples/49-actions)

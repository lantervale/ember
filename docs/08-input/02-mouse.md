# Mouse

Read the pointer, the buttons and the wheel, and change the cursor. Here three crates are dragged with the left button, the wheel zooms, and the mouse's Back and Forward buttons undo and redo the moves. The pointer becomes a hand over a crate, and a picture of the game's own while one is carried.

![Two crates dragged, the moves undone and redone, the view zooming in and out](mouse.gif "ember · mouse")

## From the pointer to the world

`Mouse()` is where the pointer is, in window points from the top-left corner. `MouseDelta()` is how far it moved this frame.

The world under the pointer depends on the camera. `ScreenToWorld` turns window points into world units through the camera's position and zoom: at zoom 2, a hundred points across the window are fifty in the world.

## Buttons and the wheel

Mouse buttons read like keys, with `Down`, `Pressed` and `Released`: `MouseLeft`, `MouseRight` and `MouseMiddle`, and `MouseBack` and `MouseForward` for the side buttons.

`Wheel()` is this frame's turn of the wheel, in lines, as a `geom.Vec2`: Y up is negative. A trackpad's two-finger scroll arrives there too. `WheelPrecise()` says the turn came from a device that scrolls by points, a trackpad or a smooth wheel, rather than by notches; `Wheel` then counts ten points a line.

## Cursor shapes

`win.SetCursor` gives the pointer a shape:

- `CursorHand` over what can be dragged.
- `CursorHidden` hides it, for a game that draws its own.
- `CursorArrow` elsewhere. It also hands the cursor back to the UI, which shows a hand over buttons and a text cursor over inputs by itself.

[drag.go](../../examples/48-mouse/drag.go)

## A cursor of your own

![The move cursor: four arrows from a centre](move-cursor.png)

`platform.NewCursor(img, hotX, hotY, opts…)` makes a cursor from a picture:

- One pixel of the picture is one point.
- The hot spot is in the picture's pixels, from its top-left.
- `platform.HiDPI(img2x)` gives the same picture at twice the size, for 2× screens on the web, Windows and Linux. macOS scales the plain one.

`win.SetCursor` takes it like `CursorHand`. Make each cursor once, like a texture. While a game's cursor is set, the UI's hand and text cursors stand aside; `CursorArrow` gives them back ([decision 0061](../99-decisions/0061-picture-cursors-are-cursor-values.md)).

Here `moveCursor` reads the example's 24-pixel move cursor and its 48-pixel double, with the hot spot at 12, 12, the centre. The camera follows nothing, so it looks at the world's origin, and the crates are placed around it.

[main.go](../../examples/48-mouse/main.go)

## The UI and dropped files

- `PointerCaptured()` is true while the UI holds the pointer, over a button for instance, so gameplay can leave that click alone.
- `Dropped()` lists the paths of the files and folders dropped on the window this frame.

## Testing

`SetMouse`, `SetMouseButton`, `AddWheel`, `AddPreciseWheel` and `AddDrop` play the player. The next frame sees them.

## Try it

[48-mouse](../../examples/48-mouse)

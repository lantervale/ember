# Multiple windows

The primary window shows the scene, and a second window shows it from above. Each window has its own camera and its own label. Drag the mouse in the scene window to turn around the crates, and scroll to zoom: the map stays still.

![The scene in the primary window](scene.png "ember · scene") ![The map window, from above](map.png "ember · map")

## Two windows

Each window is made by `platform.NewWindow`: adding it opens it, removing it takes it away. The first window added is the primary one.

Each window's camera and label are its children; the label is a `kit.Label` in a ui Document of its own. A camera or a UI element that is a window's child shows in that window and reads its input:

- the scene window's orbit camera follows that window's mouse;
- the map window's fixed camera draws there;
- closing a window takes its camera and label with it.

A camera outside any window can still pick one with `In(win)`.

[window.go](../../examples/10-multi-window/window.go)

## Closing a window

A close button only asks: the window's `platform.Closing` rule decides what happens.

| Rule | Closing the window |
|---|---|
| `platform.CloseQuitsIfPrimary`, the default | Ends the program if it is the primary window, else closes only that window. Suits a game with a tool window. |
| `platform.CloseQuitsIfLast` | Closes the window, and ends the program when it was the last one. |
| `platform.CloseQuits` | Ends the program, from any window. |
| `platform.CloseOnly` | Closes the window and lets the program run on, even with none left, until it adds a window again or quits. |
| `platform.CloseAsks` | Does nothing: the program reads `platform.WindowCloseRequested` and decides. |

Here both windows are equals, so each has `platform.CloseQuitsIfLast`: close either and the other stays, and the program ends with the last one. Once the scene window is gone, the map is the primary window.

A program that asks before closing, for unsaved work:

```go
doc := platform.NewWindow(platform.Closing(platform.CloseAsks))

var asked world.Reader[platform.WindowCloseRequested]
w.On(world.Update, "close", func(f world.Frame) {
	for r := range asked.Read(f.World) {
		if saved(r.Window) {
			f.World.RemoveEntity(r.Window)
		}
	}
})
```

`win.SetClosing` changes the rule while the window is open, and `win.RequestClose()` asks as the close button does, from a menu or a test.

## The World

`main` adds the scene window first, so it is the primary one, then the ground, the crates and the map window, like any other entities.

[main.go](../../examples/10-multi-window/main.go)

[scene.go](../../examples/10-multi-window/scene.go)

## Try it

[10-multi-window](../../examples/10-multi-window)

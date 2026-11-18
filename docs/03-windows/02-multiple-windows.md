# Multiple windows

The primary window shows the scene, and a second window shows it from above. Each window has its own camera and its own label. Drag the mouse in the scene window to turn around the crates, and scroll to zoom: the map stays still.

![The scene in the primary window](scene.png "ember · scene") ![The map window, from above](map.png "ember · map")

## Two windows

Each window is made by `platform.NewWindow`: adding it opens it, and removing it takes it away. The first window added is the primary one. Each window's camera and label are its children; the label is a `kit.Label` in a ui Document of its own. A camera or a UI element that is a window's child shows in that window and reads its input: the scene window's orbit camera follows that window's mouse, and the map window's fixed camera draws there. Closing a window takes its camera and label with it. A camera outside any window can still pick one with `In(win)`.

[window.go](../../examples/10-multi-window/window.go)

## Closing a window

A close button only asks: what happens is the window's `platform.OnClose` rule. Without one, closing the primary window ends the program and closing another only closes it, which suits a game with a tool window. Here both windows are equals, so each has `platform.CloseQuitsIfLast`: close either and the other stays, and the program ends with the last one. Once the scene window is gone the map is the primary window.

The other rules are `platform.CloseQuits`, which ends the program from any window; `platform.CloseOnly`, which closes the window and lets the program run on, even with none left, until it adds a window again or quits; and `platform.CloseAsks`, which does nothing, for a program that wants to ask first:

```go
doc := platform.NewWindow(platform.OnClose(platform.CloseAsks))

var asked world.Reader[platform.WindowCloseRequested]
w.On(world.Update, "close", func(f world.Frame) {
	for r := range asked.Read(f.World) {
		if saved(r.Window) {
			f.World.RemoveEntity(r.Window)
		}
	}
})
```

`win.SetOnClose` changes the rule while the window is open, and `win.RequestClose()` asks as the close button does, from a menu or a test.

## The World

`main` adds the scene window first, so it is the primary one, then the ground, the crates and the map window, like any other entities.

[main.go](../../examples/10-multi-window/main.go)

[scene.go](../../examples/10-multi-window/scene.go)

## Try it

[10-multi-window](../../examples/10-multi-window)

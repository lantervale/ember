# A window

One window and what it can do: a title, a size and a background, size limits, fullscreen, its own title bar, and what it shows. A label in its corner shows the window's size, and the ground is seen through an orbit camera.

![The window, with its size in the corner](window.png "ember · window")

## The window

A window is an entity: `platform.NewWindow` makes it from its options, and adding it to the World opens it.

- `MinSize(320, 180)` stops it from being resized smaller.
- `AspectRatio(16, 9)` keeps its shape while resizing.
- Its children show in it: the label, a `kit.Label` in a small ui Document padded 16 points from the corner by its sheet; the ground; and `camera.NewOrbit`, which turns around the ground as you drag the mouse in the window.

[main.go](../../examples/09-window/main.go)

### Vsync

A window presents its frames in step with the display's refresh, unless it's given `platform.VSync(false)`. Then frames run as fast as they come, which suits measuring. The window still shows at most one frame each refresh and draws the others without showing them, so the window system never stalls the game.

## Something to see

The ground is a prop: a transform and a model.

[ground.go](../../examples/09-window/ground.go)

## Reading the window's size

`ShowSize` holds the window and its label, reads the window's size and sets the label's text every frame.

[size.go](../../examples/09-window/size.go)

## Fullscreen

Every option has a setter for while the game runs. `ToggleFullscreen` reads the window's own input and switches fullscreen with `SetFullscreen` when F is pressed.

[fullscreen.go](../../examples/09-window/fullscreen.go)

## Drawing your own title bar

A program that draws its own header, as a browser draws its tabs, gives its window `platform.OwnTitleBar()`. The content fills the title bar and the title is hidden, while the window keeps the system's buttons, edges, shadow and corners ([decision 0091](../99-decisions/0091-a-window-keeps-the-systems-frame-under-its-own-title-bar.md)).

```go
win := platform.NewWindow(platform.Title("explorer"), platform.OwnTitleBar())

area := win.TitleBarArea() // read each frame: zero in fullscreen
header.SetStyle(fmt.Sprintf("height: %gpx; padding-left: %gpx", area.Height, area.Left)) // 40 and 84 on macOS 26
```

| Method | What it does |
|---|---|
| `win.TitleBarArea()` | What the system still draws over the content, in points: the bar's `Height`, and the widths its buttons take at the `Left` and the `Right`. Zero when nothing lies over the content. |
| `win.StartDrag()` | The system moves the window with the pointer, from the left press being handled to its release. The window's input reads the button as up from then. |
| `win.DoubleClickTitleBar()` | What the system does for a double click on a title bar: on macOS zoom, minimise or nothing, as the user set it. |

A ui screen calls neither method: it marks its header `app-region: drag` in its sheet ([CSS](../05-ui/03-css.md)). A press on the header's empty part then drags the window and a double click zooms it, while its buttons and tabs still click.

Only macOS does this. Windows and Linux keep their title bar above the content, and a page in a browser has none: there the area is zero and the two methods do nothing, so the same header works everywhere without a branch. A hidden run reports zero too; a test sets an area with `win.SyncTitleBar(area)`. [light](https://github.com/lantervale/light), a small web browser built on ember, draws its tabs this way.

## When it fails

If the window can't open, or a panic escapes a frame, ember writes a `crash.log`. A player who reports a crash or a wrong picture sends it, with the `errors.log` beside it.

### What the log holds

- the reason, and the panic's stack;
- the Go, GLFW and wgpu-native versions;
- a "Graphics:" line with the backend (Metal, Direct3D 12, Vulkan or OpenGL), and the graphics card;
- the error lines the engine printed before the crash.

`errors.log`, when there is one, holds the error lines the engine printed during that run, such as a shader the graphics card refused.

Two failures get a reason written for players:

- no graphics adapter WebGPU can draw with: "Ember found no graphics adapter to draw with", naming what it tried, with advice on drivers, Remote Desktop and virtual machines;
- no WebGPU library: "ember couldn't load its graphics library, which ships beside the game. Reinstalling the game puts it back."

### Where it goes

The log lands in the player's config folder, under the primary window's title made into a name. For a window titled "Ember Quest":

| System | Path |
|---|---|
| macOS | `~/Library/Application Support/ember-quest/crash.log` |
| Windows | `%AppData%\ember-quest\crash.log`; a message box also gives the reason and the log's path, so a game built without a console never fails in silence |

With no config folder, it goes to the temporary folder. `ember.CrashLog(path)` writes it elsewhere.

### What a clean run leaves

Nothing. The log's folder is made as the game starts, since the log must be open before a crash can be written into it. A clean exit removes that folder when the run made it and nothing is in it. A folder that was there before, or that holds any file (an `errors.log`, an earlier crash's log, the game's own saves), is never touched.

A run killed from outside has no clean exit: it leaves the folder with an empty `crash.log`. If the game had crashed before, that earlier crash's log is beside it as `crash.log.last`, and that is the file a player sends.

### In tests and captures

A test or a capture that runs a program which may fail keeps the log out of the player's folder with `EMBER_CRASH_DIR=dir`: `dir` stands in for the config folder, so the log lands in `dir/ember-quest/crash.log`. A path given to `ember.CrashLog` still wins. `golden.Run` sets it for each scene, to `crash` in the scene's test folder, unless the scene's own `Env` names another.

## Try it

[09-window](../../examples/09-window)

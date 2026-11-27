# Hello cube

The smallest program that opens a window and draws: a spinning cube, each corner coloured by its position. [The World](../02-world/README.md) explains the entities and systems it uses.

![The spinning RGB cube](hello-cube.png "ember · hello cube")

## The program

- `platform.NewWindow` makes the window, an entity, with its title, size and background.
- `ember.New` returns a World. The window, the cube and the camera are entities added to it.
- `On` adds a function that turns the cube a little every frame.
- The cube's material is `Unlit`, so it shows its vertex colours as they are.

[main.go](../../examples/01-hello-cube/main.go)

## Try it

[01-hello-cube](../../examples/01-hello-cube)

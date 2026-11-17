# menu

![Use, Drop and Inspect buttons, one a line, a disc before each](menu.png "ember · menu")

```go
ui.Menu(
	ui.Li(ui.Button(ui.Text("Use"))),
	ui.Li(ui.Button(ui.Text("Drop"))),
	ui.Li(ui.Button(ui.Text("Inspect"))),
)
```

A list of commands, laid out as a [`ul`](43-ul.md): discs, circles in another list, squares two lists deep. It is not a menu that opens: that is `role="menu"`, or kit's menu.

| Default CSS | |
|---|---|
| `menu` | `display: block; margin: 1em 0; padding-left: 40px; list-style-type: disc` |
| a `menu` in a `ul`, `ol` or `menu` | `list-style-type: circle` |
| a `menu` in two of them | `list-style-type: square` |
| any list in a list | `margin-top: 0; margin-bottom: 0` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element menu
```

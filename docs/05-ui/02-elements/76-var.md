# var

![Damage is strength times level, strength and level in italics](var.png "ember · var")

```go
ui.P(ui.Text("Damage is "), ui.Var(ui.Text("strength")), ui.Text(" times "), ui.Var(ui.Text("level")), ui.Text("."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

A variable of a formula or a program.

| Default CSS | |
|---|---|
| `var` | `font-style: italic` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element var
```

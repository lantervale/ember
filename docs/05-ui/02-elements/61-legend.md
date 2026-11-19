# legend

![Sound, the legend standing on the top line of the frame round two checkboxes](fieldset.png "ember · fieldset")

```go
ui.Fieldset(
	ui.Legend(ui.Text("Sound")),
	ui.Div(ui.Label(ui.Input(ui.Type("checkbox"), ui.Checked(true)), ui.Text(" Music"))),
	ui.Div(ui.Label(ui.Input(ui.Type("checkbox")), ui.Text(" Effects"))),
)
```

A [`fieldset`](60-fieldset.md)'s title. Only as the fieldset's first child, in the flow and not floated, it stands on the frame's top line, as wide as its text, the line leaving a gap for it. Anywhere else it is a plain block.

| Default CSS | |
|---|---|
| `legend` | `display: block; padding: 0 2px` |
| `fieldset > legend` | `width: fit-content` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element legend
```

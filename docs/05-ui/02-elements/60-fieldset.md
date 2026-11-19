# fieldset

![Music and Effects checkboxes in a grooved grey frame, Sound standing on the frame's top line](fieldset.png "ember · fieldset")

```go
ui.Fieldset(
	ui.Legend(ui.Text("Sound")),
	ui.Div(ui.Label(ui.Input(ui.Type("checkbox"), ui.Checked(true)), ui.Text(" Music"))),
	ui.Div(ui.Label(ui.Input(ui.Type("checkbox")), ui.Text(" Effects"))),
)
```

A framed group of controls. When its first child is a [`legend`](61-legend.md), the legend stands on the frame's top line and the line leaves a gap for it. Its `disabled` attribute disables nothing: disable each control.

| Default CSS | |
|---|---|
| `fieldset` | `display: flow-root; margin: 0 2px; padding: 0.35em 0.75em 0.625em; border: 2px groove #efefef; min-width: min-content` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element fieldset
```

# form

![A Hero label, a text field holding Aldric and a Create button](form.png "ember · form")

```go
ui.Form(ui.ID("new-hero"),
	ui.Label(ui.For("hero"), ui.Text("Hero")),
	ui.Input(ui.ID("hero"), ui.Value("Aldric")),
	ui.Button(ui.Text("Create")),
)
```

A plain block round a group of controls. It submits nothing: its `action` and `method` are not read, a button in it is not a submit button, and Enter in one of its fields emits `ui.Submitted` for the field alone, which does not reach an `OnSubmit` on the form. Put `ui.OnSubmit` on the field and `ui.OnClick` on the button.

| Default CSS | |
|---|---|
| `form` | `display: block` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element form
```

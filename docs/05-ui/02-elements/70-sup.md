# sup

![You finished 3rd of 40, the rd smaller and above the line](sup.png "ember · sup")

```go
ui.P(ui.Text("You finished 3"), ui.Sup(ui.Text("rd")), ui.Text(" of 40."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

A superscript: smaller text whose baseline is a third of its parent's font size and a pixel above the line's, as Chrome's. The line grows to hold it.

| Default CSS | |
|---|---|
| `sup` | `font-size: smaller; vertical-align: super` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element sup
```

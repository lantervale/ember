# hgroup

![The Crypt as a title with Floor 3, the flooded halls under it, in one dashed box](hgroup.png "ember · hgroup")

```go
ui.Hgroup(
	ui.H2(ui.Text("The Crypt")),
	ui.P(ui.Text("Floor 3, the flooded halls")),
)
```

A heading with the lines that go with it: a plain block, its children keeping their own margins. The dashed box in the picture is the example's.

| Default CSS | |
|---|---|
| `hgroup` | `display: block` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element hgroup
```

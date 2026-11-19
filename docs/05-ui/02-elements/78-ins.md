# ins

![Torch radius: 200, then 300, the new value underlined](ins.png "ember · ins")

```go
ui.P(ui.Text("Torch radius: 200 "), ui.Ins(ui.Text("300")))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

Text put into a document: underlined. Its `cite` and `datetime` attributes show nothing.

| Default CSS | |
|---|---|
| `ins` | `text-decoration: underline` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element ins
```

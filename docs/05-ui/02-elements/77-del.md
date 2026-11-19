# del

![Torch radius: 200, then 300, the old value struck through](del.png "ember · del")

```go
ui.P(ui.Text("Torch radius: "), ui.Del(ui.Text("200")), ui.Text(" 300"))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

Text taken out of a document: struck through. Its `cite` and `datetime` attributes show nothing.

| Default CSS | |
|---|---|
| `del` | `text-decoration: line-through` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element del
```

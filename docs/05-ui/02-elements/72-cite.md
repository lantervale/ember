# cite

![As told in The Lantern Keeper's Almanac, the book's title in italics](cite.png "ember · cite")

```go
ui.P(ui.Text("As told in "), ui.Cite(ui.Text("The Lantern Keeper's Almanac")), ui.Text("."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

The title of a work.

| Default CSS | |
|---|---|
| `cite` | `font-style: italic` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element cite
```

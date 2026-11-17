# big

![Critical hit: 128 damage, the number larger than the words](big.png "ember · big")

```go
ui.P(ui.Text("Critical hit: "), ui.Big(ui.Text("128")), ui.Text(" damage"))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

The old element for larger text: one size larger than the text round it.

| Default CSS | |
|---|---|
| `big` | `font-size: larger` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element big
```

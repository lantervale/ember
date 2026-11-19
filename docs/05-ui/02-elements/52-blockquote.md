# blockquote

![A line of text, then what the innkeeper said, set in on both sides](blockquote.png "ember · blockquote")

```go
ui.P(ui.Text("The innkeeper lowers her voice:"))
ui.Blockquote(ui.Text("Nobody who went down to the crypt after dark came back the same."))
```

A quotation that stands alone: a block set in on both sides. Its `cite` attribute shows nothing.

| Default CSS | |
|---|---|
| `blockquote` | `display: block; margin: 1em 40px` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element blockquote
```

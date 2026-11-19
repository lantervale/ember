# time

![Saved on 17 November, 21:40](time.png "ember · time")

```go
ui.P(ui.Text("Saved on "), ui.Time(ui.Attr("datetime", "2026-11-17T21:40"), ui.Text("17 November, 21:40")), ui.Text("."))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

A date or a time. Plain semantics: it shows its text, and its `datetime` attribute is the game's to read with `n.Attr`.

| Default CSS | |
|---|---|
| `time` | none: inline, as [`span`](14-span.md) |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element time
```

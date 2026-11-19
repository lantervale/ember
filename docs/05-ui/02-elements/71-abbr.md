# abbr

![HP 80, MP 35, HP and MP over a dotted line](abbr.png "ember · abbr")

```go
ui.P(ui.Abbr(ui.Attr("title", "Hit points"), ui.Text("HP")), ui.Text(" 80"))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

An abbreviation, its `title` what it stands for: with a `title`, a dotted line runs under it. The `title` shows no tooltip: use `kit.Tooltip` for one.

| Default CSS | |
|---|---|
| `abbr` | none: inline, as [`span`](14-span.md) |
| `abbr[title]` | `text-decoration: underline dotted` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element abbr
```

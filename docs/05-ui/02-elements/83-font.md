# font

![Gold: 1,250, the number larger and in gold](font.png "ember · font")

```go
ui.P(ui.Text("Gold: "), ui.Font(ui.Attr("color", "#b8860b"), ui.Attr("size", "5"), ui.Text("1,250")))
```

| Attribute | |
|---|---|
| `ui.Text(s)` | its text; children may be any inline elements |

The old font element, kept so old pages build. Its attributes style it as Chrome reads them, under every rule of the sheets ([decision 0152](../../99-decisions/0152-the-old-attributes-of-tables-rules-and-fonts-are-hints-as-chromes.md)):

| Attribute | Sets |
|---|---|
| `size` | `font-size`: 1 to 7, or a step from 3 (`+1`, `-2`) |
| `face` | `font-family` |
| `color` | `color`, read by HTML's legacy colour rules |

In a game's own screens, use a [`span`](14-span.md) with a `Style` or a class.

| Default CSS | |
|---|---|
| `font` | none: inline, as [`span`](14-span.md) |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element font
```

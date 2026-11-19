# search

![A search field with the placeholder Search the bestiary and a Find button, in one dashed box](search.png "ember · search")

```go
ui.Search(
	ui.Input(ui.Type("search"), ui.Placeholder("Search the bestiary")),
	ui.Button(ui.Text("Find")),
)
```

The part of a screen that searches or filters: a plain block. The searching is the game's. The dashed box in the picture is the example's.

| Default CSS | |
|---|---|
| `search` | `display: block` |

Automation: recorded, as a label, only when it has an `id`.

```
go run ./examples/93-html-elements -element search
```

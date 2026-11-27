# Automation

Every element a Document shows is recorded for automation, under an ID made from its name. A test, or a tool outside the game, finds an element by that ID and acts on it: presses a button, types in a field, picks an option.

```go
auto := automation.New(automation.Recording())
ember.New(ember.Automation(auto)).AddEntities(win)
```

`automation.Recording()` records without serving, for tests. Every ember app serves automation over Connect when `EMBER_AUTOMATION=host:port` is set.

## An element's ID

The first name it has is its ID:

| Name | Example |
|---|---|
| its `id` | `ui.ID("save")` → `save` |
| else its `data-name` | `ui.Attr("data-name", "slot-1")` → `slot-1` |
| else its `aria-label` | `ui.Attr("aria-label", "Close")` → `close` |
| else the slug of its text, label or placeholder | `kit.Button("Save tileset")` → `save-tileset` |

- The ID is prefixed by its ancestors' ids and data-names: `settings/save`.
- Text that a `data-name` names is a label: `ui.Span(ui.Attr("data-name", "stock"), ui.Text("12"))` is a label `stock` reading `12`. A named box of elements only scopes the IDs inside it.
- An element with none of these is not recorded: a pop-up with neither id nor label, for instance.
- A label's text never includes the control it labels.

## Values an element draws itself

An element that draws its own value, such as a `ui.Canvas`, tells automation what it shows with WAI-ARIA:

```go
orb := ui.Canvas(draw, ui.ID("health"), ui.Attr("role", "progressbar"), ui.Attr("aria-valuenow", "75"))
```

| `role` with `aria-valuenow` | Recorded as |
|---|---|
| `progressbar` | a bar, its value the share of `aria-valuemin` to `aria-valuemax` (0 to 100 by default): `0.75` |
| `meter` | a meter, the value as is |
| `slider` | a slider, the value as is |

## Actions

| Action | On | Does |
|---|---|---|
| `press` | buttons, checks, summaries, menu items, rows, options | presses it; on a row or an option, selects and activates it |
| `set` | a control | sets its value |
| `type` | a text field | types text at its caret |
| `submit` | a one-line text field | submits it, as Enter |
| `select-all` | a text field, a label whose text takes a selection | selects its text |
| `select` | an option, a tab, a row | selects only |
| `context` | an element with `OnContextMenu` | right-clicks it |
| `drag` | a `Draggable` element | drags it onto the element whose path is the action's value |

```
go run ./examples/92-ui-kit -element settings
```

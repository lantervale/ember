# optgroup

![A weapon select open: Fists, then Blades with Sword and Dagger indented, Magic with Wand and Staff, and Cursed greyed with Bone flail](optgroup.png "ember · optgroup")

```go
ui.Select(
	ui.Option(ui.Text("Fists")),
	ui.Optgroup("Blades", ui.Option(ui.Text("Sword")), ui.Option(ui.Text("Dagger"), ui.Selected())),
	ui.Optgroup("Cursed", ui.Disabled(), ui.Option(ui.Text("Bone flail"))),
)
```

| Attribute | |
|---|---|
| `label` (`ui.Optgroup`'s first argument) | the group's name, a row of the open list of its own, bold and never chosen |
| its [`option`](22-option.md) children | the select's options, counted with the others in order: `doc.Selected(n)` is their index across groups, as the DOM's `selectedIndex` |
| `ui.Disabled()` | disables its options; it and they match `:disabled`, an enabled group `:enabled` |

| Default CSS | |
|---|---|
| `optgroup` | `display: block; font-weight: bolder` |
| `optgroup:disabled` | `color: gray` |

In the open list, the arrows, typing and the pointer pass over the labels, and a group's options stand behind four spaces, which the select's width counts, as Chrome's does; the labels never widen the select.

In a [listbox](21-select.md#a-listbox-multiple-or-a-size) (a select with `multiple` or a `size` above 1), the label is a bold row of the page before the group's options, which stand 20px in; it takes no press, and the keys pass over it.

Automation: through its select's `select`, by an option's index, value or text; a label is never one.

```
go run ./examples/93-html-elements -element optgroup
```

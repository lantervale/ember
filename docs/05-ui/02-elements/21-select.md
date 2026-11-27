# select

![A difficulty select showing Normal, and a class select showing its placeholder, Pick one](select.png "ember · select")

```go
ui.Select(ui.Placeholder("Pick one"),
	ui.Option(ui.Value("warrior"), ui.Text("Warrior")),
	ui.Option(ui.Value("mage"), ui.Text("Mage")),
)
```

| Attribute | |
|---|---|
| its [`option`](22-option.md) children, and those of its [`optgroup`](88-optgroup.md) children | what it offers, in order across groups; the chosen one's text shows; `doc.Selected(n)`, `doc.SetSelected(n, i)`, `doc.Value(n)` |
| `ui.Placeholder(s)` | a ui extension: it starts with nothing chosen and shows `s`, dimmed |
| `ui.Disabled()` | takes no press and no focus; `:disabled` |

| Default CSS | |
|---|---|
| `button, input, select, textarea` | `appearance: none; margin: 0; padding: 0; border: 0; background: none`, font, colour, line height and text alignment inherited; `display: inline-block` |
| `input, select, textarea` | `padding: 1px 2px; border: 1px solid #767676; border-radius: 2px; background-color: #fff` |
| `input, select` | `white-space: nowrap; overflow: hidden` |
| `select` | `padding-right: 20px; background-image: url(ui:select-arrow); background-repeat: no-repeat; background-position: right 6px center` |
| `select:disabled` | grey text, border at 30%, `opacity: 0.7` |

Its width is its widest option's, an option of a group behind four spaces, its placeholder counted while it shows. Open, its highlighted option takes its `accent-color`.

| Event | When |
|---|---|
| `ui.Edited{Node}`, `ui.Changed{Node}` | another option chosen: a click opens the list and a click chooses, or a press opens it and releasing over an option chooses; or typed, as in a listbox below, which while the list is open moves its highlight |

Automation: its `id`, else `data-name`, `aria-label`, its label's text, then its placeholder (`difficulty`); `press` opens it, `select` chooses an option by index, value or text.

## A listbox: `multiple` or a `size`

```go
ui.Select(ui.Attr("multiple", ""), ui.Attr("size", "5"),
	ui.Option(ui.Text("Sword"), ui.Selected()),
	ui.Optgroup("Magic", ui.Option(ui.Text("Wand")), ui.Option(ui.Text("Staff"), ui.Selected())),
	ui.Option(ui.Text("Bow"), ui.Disabled()),
)
```

With `multiple`, or a `size` above 1, the select is a listbox, as in a browser: its options are rows of the page, `size` of them in view (4 for `multiple` alone), each row as tall as the tallest option or group label, and it scrolls through the rest. A group's label is a bold row of its own; its options stand 20px in. Nothing is selected until a `selected` attribute, the player or the program says so.

| | |
|---|---|
| a press | selects its option alone; with Command (Control off macOS) turns it on or off, with Shift selects the run from the last pressed; a drag selects the run it crosses, and within 20 points of the top or bottom edge, or past it, scrolls the listbox on each frame, the faster the farther in |
| the arrows, Home, End, Page Up and Down | select the enabled option they reach alone, scrolled into view at once; with Shift the run from the anchor; with Command they move without selecting, and Space turns the option reached on or off |
| typing | letters typed within a second of each other select the option starting with them all, from the first selected on; one letter typed again goes on to the next option starting with it (decision 0206) |
| `doc.SelectedOptions(n)`, `doc.SetSelectedOptions(n, is...)` | every selected option, as the DOM's `selectedOptions`; without `multiple` the last of `is` alone |
| `doc.Values(n)` | what a form submits: each selected, enabled option's value |
| `doc.Selected(n)`, `doc.Value(n)` | the first selected, as `selectedIndex` and `value` |

A run passes over disabled options, and a group's label takes no press.

- `required` holds once an option is selected.
- Each change fires `ui.Edited` then `ui.Changed` (`OnInput`, `OnChange`), once a press lets go.
- Selected rows are grey, light blue while the listbox has the focus (`select:-ember-listbox option:checked` in the default sheet); under forced colours `Highlight` under `HighlightText`, a disabled one's `GrayText`.
- Typing fires a change alone for each option found until a press or a key has moved the listbox, as Chrome's.
- Automation's `select` takes a list of options, which it selects alone, and a `multiple` listbox's value is the list of those selected.

```
go run ./examples/93-html-elements -element select
```

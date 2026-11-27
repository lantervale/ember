# Events

An element tells what the player did through callbacks you give it when you build it. Screens, tools and examples use them, so the reaction reads beside the element:

```go
ui.Button(ui.Text("Save"), ui.OnClick(save))
```

Every event is also a World event, for code that isn't the element's builder: a system elsewhere, or a test.

```go
type Settings struct{ clicks world.Reader[ui.Clicked] }

func (s *Settings) Update(f world.Frame) {
	for e := range s.clicks.Read(f.World) {
		if e.Node.ID() == "save" {
			save()
		}
	}
}
```

## Every event

| Callback | World event | When |
|---|---|---|
| `ui.OnClick` | `ui.Clicked{Node, Clicks}` | a button, a summary, a canvas or an element with a `tabindex` pressed: by the pointer, a key, a pad or automation. `Clicks` is 2 or 3 for a double or triple click |
| `ui.OnInput` | `ui.Edited{Node}` | a value changing while edited: each keystroke, each step of a range, a checkbox toggled |
| `ui.OnChange` | `ui.Changed{Node}` | a value committed: a checkbox toggled, an option chosen, a text field left or submitted after an edit |
| `ui.OnSubmit` | `ui.Submitted{Node}` | Enter in a one-line text field, or a textarea's submit shortcut; on a `form`, the form submitted and valid ([below](#forms)) |
| `ui.OnToggle` | `ui.Toggled{Node}` | a `details` element opened or closed |
| `ui.OnClose` | `ui.Closed{Node, Value}` | a dialog closed, with its return value, "" for Escape |
| `ui.OnSort` | `ui.Sorted{Node, Column, Descending}` | a table's column header pressed |
| `ui.OnKey` | `ui.KeyPressed{Node, Key, Text, Command, Shift, Alt, Control, Super, Pad}` | a key nothing took, on the focused element or the open modal dialog ([below](#keys)) |
| `ui.OnBack` | `ui.BackPressed{Node}` | Escape or the pad's East that nothing took: a screen's way back |
| `ui.OnPointerEnter` | `ui.PointerEntered{Node}` | the pointer coming over it, once, not again for its children (mouseenter) |
| `ui.OnPointerLeave` | `ui.PointerLeft{Node}` | the pointer leaving it and all its children (mouseleave) |
| `ui.OnPointerDown` | `ui.PointerPressed{Node, At, Button}` | a mouse button pressed over it (pointerdown); the press then focuses, clicks and selects as ever |
| `ui.OnPointerMove` | `ui.PointerMoved{Node, At}` | the pointer moved over it, or anywhere while it holds the pointer (pointermove) |
| `ui.OnPointerUp` | `ui.PointerReleased{Node, At, Button}` | a mouse button released over it, or anywhere while it holds the pointer (pointerup) |
| `ui.OnContextMenu` | `ui.ContextClicked{Node, At}` | the right button pressed and released on it, or on macOS the left one with Control held. `At` is the pointer's place |
| `ui.OnDrop` | `ui.Dropped{Target, Dragged}` | a draggable element let go over it, a drop target |
| `ui.OnDragEnd` | `ui.DragEnded{Source, Dropped}` | its drag ended, taken by a target or cancelled |
| `ui.OnMove` | `ui.WindowMoved{Node, At}` | a window dragged by its bar and let go elsewhere |
| `ui.OnResize` | `ui.WindowResized{Node, Size}` | a window's edge drag let go with a new size |
| `ui.OnFold` | `ui.WindowFolded{Node, Folded}` | a window folded to its bar or unfolded |
| `ui.OnSelectionChange` | `ui.SelectionChanged{Node}` | the Document's selected text set, moved or cleared. `Node` is `body`, so the callback goes on `body` or `html` |
| `ui.OnRightDragSelect` | `ui.RightDragSelected{Node}` | text selected by a drag of the right button, let go, in a Document with `ui.RightDragSelects()`. `Node` is `body`, as above |
| `ui.OnAnimationStart` | `ui.AnimationStarted{Node, Name, Elapsed}` | a CSS animation past its delay ([CSS](03-css.md#animations)) |
| `ui.OnAnimationIteration` | `ui.AnimationIterated{Node, Name, Elapsed}` | a CSS animation beginning another iteration |
| `ui.OnAnimationEnd` | `ui.AnimationEnded{Node, Name, Elapsed}` | a CSS animation over |
| `ui.OnAnimationCancel` | `ui.AnimationCanceled{Node, Name, Elapsed}` | a running CSS animation stopped before its end |

`doc.Value(e.Node)` reads what a control now holds.

## Writing a callback

A callback takes one of three forms:

| Form | Told |
|---|---|
| `func()` | nothing |
| `func(ui.Event)` | the event: `Type` (`ui.EventClick`…), `Node`, and what its World event carries (`Clicks`, `Value`, `Column`, `Descending`, `Key`, `Text`, `Command`, `Shift`, `Alt`, `Control`, `Super`, `Pad`, `At`, `Size`, `Folded`, `Dragged`, `Dropped`, `Button`, and an animation's `Animation` name and `Elapsed`) |
| `func(*world.World, ui.Event)` | the event and the World |

`ui.On` adds one to an element already in the Document, and returns the function that takes it off:

```go
remove := ui.On(slot, ui.EventClick, equip)
```

## When callbacks run

- Every event is emitted as a World event, whether or not it has callbacks. World events are emitted late in a frame and read the next.
- Callbacks run right after their event, in the Document's Update, in the order they were added, and only while their element is in the Document.
- A point on a box's right or bottom edge is outside it, as Chrome's `elementFromPoint` has it.

### Which events bubble

Most events stay on their element. These bubble: `OnBack`, `OnKey`, `OnContextMenu`, `OnPointerDown`, `OnPointerMove`, `OnPointerUp`, `OnSelectionChange` and `OnRightDragSelect`.

- The focused element's handlers run, then each ancestor's up to `html`, innermost first. `e.Node` stays the focused element.
- Any handler stops it with `e.StopPropagation()`.
- So one `OnBack` on a screen's root hears its Escape and the pad's East. On `body` or `html` it also hears a Back with nothing focused, which emits no World event ([decision 0060](../99-decisions/0060-back-and-key-callbacks-bubble.md)).
- In a Document shown by an iframe, a key or a Back nothing stopped goes on from the iframe into the Document round it, with `e.Node` the iframe there. That includes a key in a focused field, whose default `e.PreventDefault()` still holds back ([decision 0141](../99-decisions/0141-keys-and-the-arrows-cross-a-frame-the-frame-fades-and-moves-with-its-iframe.md)).

### Keys

- `KeyPressed` carries the modifiers held, either side's. `Command` is Super on macOS and Control elsewhere.
- A pad button that focus navigation leaves (West, North, the shoulders, the triggers, Back, Start) is a `KeyPressed` too, as `Pad`, with `Key` `input.KeyUnknown`.
- In a focused text field, `OnKey` hears every key before the field, as a keydown, and `e.PreventDefault()` keeps it from the field. Those keys emit no World event. [Focus and keys](05-focus-and-keys.md) has more.

### Forms

A `form` is submitted by its submit button, by Enter in its field, or by `doc.RequestSubmit(form)`. Enter clicks the form's first submit button, or submits the form when it has no button and that field is its only text field.

- An invalid form is not submitted: its first invalid field takes the focus, unless the form has `novalidate` or the button `formnovalidate`.
- Either way, its fields match `:user-valid` or `:user-invalid` from then on.
- ui sends nothing anywhere: `OnSubmit` is where the game acts ([decision 0159](../99-decisions/0159-a-pattern-is-read-as-javascript-and-a-form-is-submitted-by-its-button.md)).

## Selected text

Text whose `user-select` allows it ([CSS](03-css.md#selecting-text)) is selected with the pointer, as in Chrome on macOS. A Document has one selection, apart from its fields' own.

```go
doc := ui.NewDocument(kit.Theme(), ui.CSS(`#letter { user-select: text; }`))
ui.OnSelectionChange(func() { copyButton.SetAttr("data-count", strconv.Itoa(len(doc.Selection().Text))) }).Apply(doc.Body())
```

### What the player does

| Input | Does |
|---|---|
| a press and a drag | selects from the place nearest the press to the place nearest the pointer, over elements, lines, blocks, columns and scrollers. Held past the edge of a box that scrolls the text, it scrolls it |
| a double click, a triple click | selects the word, the block; a drag then goes on by words or blocks |
| Shift and a click | moves the selection's end there |
| a click in text or beside it | clears the selection |
| a press inside the selection, moved 3 points | drags its text ([below](#dragging-selected-text)) |
| a press inside the selection, let go | clears it, as a click elsewhere |
| a finger held half a second on a touchscreen | selects the word under it, in the page or a field. The page's selection shows a handle at each end that a finger drags along, and a tap clears it |
| a press on a button, a link, a control, a `summary`, a `label`, an element with a `tabindex`, or on `user-select: none` text | selects nothing and keeps the selection, as Chrome; the click does what it does |
| Cmd+C (Ctrl+C elsewhere) | copies it, while no field has the focus |
| Cmd+A (Ctrl+A elsewhere) | selects all the Document's text that can be, while no field has the focus |
| Shift and an arrow, Home or End (with the platform's word and line keys) | moves the end of a selection that exists, by a character, a word, a line, to the line's or the text's end |
| Escape | leaves it, as Chrome |

Over text a press would select, the pointer is `platform.CursorText` while the element's `cursor` is `auto`; an author's `cursor` wins.

### Dragging selected text

A press inside the selection, moved 3 points, drags its text.

- Let go over a text field, the text goes in where the pointer is, selected, and the field takes the focus.
- Text from the page is copied; text from a field is moved, or copied with Option (macOS) or Control held.
- Let go elsewhere, it does nothing. Escape cancels it.
- On macOS the press must be held 0.15 s first, as in Chrome; a quicker move selects.

### Vertical lines

On vertical lines ([writing modes](03-css.md#writing-modes)) the same works down them, as in Chrome ([decision 0193](../99-decisions/0193-vertical-lines-are-selected-in-the-flow-of-their-lines.md)):

- a press finds the place along the line it falls on;
- a drag and a double click select down and across lines, and the selection is drawn along them;
- with Shift, the arrows along the lines step through the text, and those across them go to the next or the previous line: Left the next in `vertical-rl`, Right in `vertical-lr` and `sideways-lr`;
- the empty lines of preformatted text, a `br` alone on its line and the room of tabs kept between words are marked across the lines, as on horizontal ones ([decision 0210](../99-decisions/0210-marks-of-vertical-lines-are-measured-in-their-flow-and-a-fallback-stands-by-its-own-metrics.md)).

### Reading and setting it

| | |
|---|---|
| `doc.Selection()` | `ui.Selection{Anchor, Focus, Text}`: where it began, where it ends (before `Anchor` when made backwards), and what a copy writes. `Empty()` when nothing is selected |
| `doc.Select(anchor, focus)` | selects between two `ui.Position{Node, Offset}`: a text node and a character offset in its text as shown, before white space collapses |
| `doc.SelectAll()`, `doc.SelectContents(n)` | all the selectable text of the Document, or of an element; false when there is none |
| `doc.ClearSelection()` | selects nothing |
| `doc.SelectionRects()` | the rectangles it is drawn in, in the Document's space |
| `doc.TextDragImage()`, `doc.SelectionHandles()` | the ghost of the text a drag carries (`.ember-drag-image`) and the touch selection's handles (`.ember-selection-handle`, `data-side` `start` or `end`); nil while none shows. A sheet restyles them ([decision 0222](../99-decisions/0222-a-press-inside-the-selection-drags-its-text-and-a-long-press-selects-a-word.md)) |
| automation's `select-all` | on a recorded label whose text takes a selection, selects it; on a text field, its text |

### How long it lasts

- The selection is kept by node and offset, so it outlives a restyle, a relayout and a scroll. It goes when one of its ends leaves the Document.
- A field that takes the focus takes the selection: the page's is cleared, and the field's own works as ever.
- Keys reach the selection only while no field of any Document in the window has the keyboard. So a page drawn inside another Document's canvas and that Document's fields each keep their own copy and select all.

### Selecting with the right button

Only the left button selects, as in Chrome. A Document made with `ui.RightDragSelects()` also selects with the right one ([decision 0109](../99-decisions/0109-a-right-drag-selects-text-only-where-a-document-asks.md)). A program can then answer a selection made that way, with a card of what to do with it, and leave the left button's a plain one:

```go
page := ui.NewDocument(ui.BrowserSheets(), ui.RightDragSelects())
ui.OnRightDragSelect(func() { card.Open(page.Selection().Text) }).Apply(page.Body())
```

- Pressed where a left press would begin a selection and moved 4 points away, the right button selects by characters from the press to the pointer, scrolling as a left drag does.
- Let go with text selected, it emits `ui.RightDragSelected` (`ui.OnRightDragSelect` on `body`) and no `ContextClicked`, in the Document and in those showing it through an iframe.
- Let go where it was pressed, or with nothing selected, it is the right click it is without the option, and leaves the selection alone.

## Right clicks and drag and drop

```go
slot := kit.Panel(ui.Draggable(), ui.OnDragEnd(func(e ui.Event) { if !e.Dropped { sfx.Play(thud) } }))
bag := kit.Panel(ui.OnDrop(func(e ui.Event) { equip(e.Dragged) }))
card := kit.Panel(ui.OnContextMenu(func(e ui.Event) { doc.ShowModal(menu) }))
```

- A `ui.Draggable()` element drags once the left button has moved it 4 points; less is a click.
- Let go over an element with `OnDrop`, the target's `OnDrop` runs, then the source's `OnDragEnd`. Let go over no target, or Escape, cancels it.
- While it drags, the source carries `data-dragging` and the target under the pointer `data-drag-over`, for CSS, and the pointer is a hand.
- A right click on the page's `html` or `body` stays the game's.
- Not yet: a picture following the pointer, drags by keys or pad, and drags between Documents or windows ([decision 0062](../99-decisions/0062-ui-elements-take-a-right-click-and-drag-and-drop-of-their-own.md)).

## Holding the pointer

A drag of the page's own, such as a panel's edge, takes the pointer from its `OnPointerDown`. `doc.SetPointerCapture(n)` sends `n` the pointer's moves and its release wherever the pointer goes, and lets go once no button is down; `doc.PointerCapture()` reads it.

```go
edge := ui.Div(ui.Class("edge"), ui.OnPointerDown(func(e ui.Event) { doc.SetPointerCapture(e.Node) }),
	ui.OnPointerMove(func(e ui.Event) {
		if doc.PointerCapture() == e.Node {
			resize(e.At.X)
		}
	}))
```

The pointer's events skip the page's `html` and `body`, which are the game's, unless one holds the pointer. Places are in the Document's points ([decision 0142](../99-decisions/0142-the-pointer-and-the-keys-reach-callbacks-as-a-pages-dom-events.md)).

## Title tooltips

An element's `title` shows as Chrome's tooltip once the pointer rests half a second over it.

- It shows below and right of the pointer, or above it near the window's foot, its lines kept.
- It stays until the pointer leaves for another title, a button or a key is pressed, or ten seconds pass.
- An empty `title` shows none, not its ancestor's.
- Its box is `.ember-title-tip`, the last child of `html` while it shows: Chrome's look by default, a kit tip's under kit's theme. `doc.TitleTip()` returns it.

```
go run ./examples/92-ui-kit -element settings
```

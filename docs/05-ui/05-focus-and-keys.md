# Focus and keys

![The settings panel, Save focused by Tab](focus.png "ember · settings")

| Input | Does |
|---|---|
| Tab, Shift-Tab | the next or previous focusable element, in `tabindex` order, once something has the focus |
| the arrows | the focus to the nearest element that way |
| the pad's cross or stick | the same as the arrows |
| Enter, Space, the pad's South | press the focused element |
| Escape, the pad's East | close a menu or a modal dialog, else `BackPressed` |
| a click | focuses quietly, without the outline |
| Cmd/Ctrl+C, Cmd/Ctrl+A, Shift and a moving key | copy, select all and extend the [selected text](04-events.md#selected-text), while no field has the focus and there is text to select; otherwise the game's |

| | |
|---|---|
| `ui.Attr("autofocus", "")` | the first focus, as on a page's load |
| `doc.Focus(n)`, `doc.Focused(n)` | give or read the focus, shown as from Tab; an element shown this frame gets it at its first layout (a later Focus replaces it, leaving the Document drops it), so a screen is shown and focused in one call |
| `doc.Blur()` | take the focus away, as an address field after Enter: a text field commits its edit, and the keys are the game's again |
| a modal dialog | keeps the focus inside until it closes, and owns its window's input: other Documents and twod widgets get no hover, clicks or keys, and the game reads `PointerCaptured` and `KeyboardCaptured` true ([decision 0053](../99-decisions/0053-a-modal-owns-its-windows-input.md)) |
| `:focus-visible` | matches only focus from keys or a pad |
| `html[data-input-device="keyboard"]` | follows the last device, keyboard, mouse, gamepad or touch, so CSS shows its glyphs |

A key nothing takes is a `KeyPressed` on the focused element; with nothing focused and no modal dialog, the keys are the game's. In a focused text field, `OnKey` hears each key pressed or repeated before the field acts on it, as a page's keydown, and `e.PreventDefault()` keeps the field from it: a console walks its history on Up, or takes Shift+Enter for a new line. A callback of the field's own events sets its value with `doc.SetValue` at once, the caret at the end; the game's other `SetValue`s wait while the player types ([decision 0142](../99-decisions/0142-the-pointer-and-the-keys-reach-callbacks-as-a-pages-dom-events.md)).

A window can hold several Documents, a screen and a dialog of its own, for instance. While a modal dialog of another Document is open in the window, a Document's focused element keeps the focus: `doc.Focused(n)` stays true and no event is sent. It matches neither `:focus` nor `:focus-visible`, though, so it doesn't look focused under the dialog; both return when the dialog closes or its Document is hidden ([decision 0073](../99-decisions/0073-a-focused-element-under-anothers-modal-looks-unfocused.md)).

A Document takes no key or pad press in its first frame, nor in the first frame it draws after being hidden: the Escape that opened a pause menu is never also that menu's `BackPressed`.

```
go run ./examples/92-ui-kit -element settings
```

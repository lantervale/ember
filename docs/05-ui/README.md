# UI

`ember/ui` builds a game's screens, menus and HUD from Go. You write HTML elements, style them with a subset of CSS, and ember lays them out as Chrome lays out the same page, then draws them through twod ([decision 0049](../99-decisions/0049-ui-is-modern-html-elements-and-a-css-subset-written-from-go.md)). It replaced the first `ember/ui` in [decision 0059](../99-decisions/0059-ember-ui-is-uiv2.md).

![A settings panel: a title, a form of a name, a volume slider and a music switch, then Cancel and Save](settings.png "ember · settings")

```go
doc := ui.NewDocument(kit.Theme())
doc.Body().Append(
	kit.Form(kit.Field("Name", kit.TextInput("Ayla"))),
	kit.Button("Save", kit.Primary()),
)
win.Add(doc)
```

- A **Document** is one screen: the tree of elements a window shows. It is an entity, added to its window.
- **Elements** are built with functions named after their HTML tags (`ui.Div`, `ui.Button`…) and arguments (`ui.Arg`): attributes, classes, styles and callbacks.
- The **kit** (`ui/kit`) is ember's own components and look, built from those elements.

## In this section

| Page | |
|---|---|
| [Document](01-document.md) | a screen: its options, pictures, fonts and SVG |
| [Elements](02-elements/README.md) | every HTML element, how to build and change it |
| [CSS](03-css.md) | what the CSS subset styles, and what it refuses |
| [Events](04-events.md) | callbacks and World events: what the player did |
| [Focus and keys](05-focus-and-keys.md) | Tab, the arrows and pads; keys in text fields |
| [Automation](06-automation.md) | element IDs and the actions tests and tools take |
| [Kit](07-kit/README.md) | ember's components: buttons, forms, tables, windows… |
| [Matches Chrome](08-chrome.md) | how close each component is to Chrome's, measured |
| [Walkthroughs](09-walkthroughs/README.md) | small programs, file by file |
| [What is missing](10-missing.md) | the known gaps, where a page written for a browser expects more |

## Try it

The kit gallery shows every component alone, its events logged: `go run ./examples/92-ui-kit -element button`.

[main.go](../../examples/92-ui-kit/main.go)

[elements.go](../../examples/92-ui-kit/elements.go)

[core.go](../../examples/92-ui-kit/core.go)

[fetched.go](../../examples/92-ui-kit/fetched.go)

[controls.go](../../examples/92-ui-kit/controls.go)

[layout.go](../../examples/92-ui-kit/layout.go)

[inputs.go](../../examples/92-ui-kit/inputs.go)

[lists.go](../../examples/92-ui-kit/lists.go)

[windows.go](../../examples/92-ui-kit/windows.go)

[92-ui-kit](../../examples/92-ui-kit)

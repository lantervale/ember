# Kit

The kit (`ember/ui/kit`) is ember's own components, so a screen needs no stylesheet of its own: buttons, fields, forms, tables, menus, windows and dialogs. Each component is a ui element with a `kit-…` class, or a small composite where HTML has none, and `kit.Theme()` styles them all.

```go
doc := ui.NewDocument(kit.Theme())
doc.Body().Append(kit.Button("Save", kit.Primary()))
```

- A component takes `ui.Arg`s like any element, and kit's own options are `Arg`s too: `kit.Button("Save", kit.Primary(), ui.ID("save"))`.
- Each page shows the component in every state, its options, its events, the classes a sheet restyles, and how automation records it.
- Each page also says how close it is to Chrome drawing the same page; [Matches Chrome](../08-chrome.md) has them all.

## Text and pictures

| Component | Example |
|---|---|
| [Label](01-label.md) | `kit.Label("Health: 100")` |
| [Rich text](02-rich-text.md) | `kit.RichText("Reward: [color=#ffce76]250 gold[/color], [b]Ember Blade[/b]")` |
| [Icons](03-icons.md) | `set, err := kit.LoadIcons(files, "lucide.ttf", "codepoints.json")` |
| [Image](04-image.md) | `kit.Image("panel.png", kit.KeepAspectCentered())` |
| [Separator](05-separator.md) | `kit.Column(open, save, kit.Separator(), quit)` |

## Layout

| Component | Example |
|---|---|
| [Row and Column](06-row-and-column.md) | `kit.Footer(kit.JustifyEnd(), cancel, apply)` |
| [Grid](07-grid.md) | `kit.Grid(2, kit.Columns("max-content 1fr"), name, grass, solid, no)` |
| [Panel](08-panel.md) | `kit.Panel(kit.Title("Settings"), content)` |
| [Scroll](21-scroll.md) | `kit.Scroll(ui.Style("height: 200px"), lines...)` |

## Controls

| Component | Example |
|---|---|
| [Button](09-button.md) | `kit.Button("Save", kit.Primary())` |
| [Check](10-check.md) | `kit.Check("Show grid", ui.Checked(true))` |
| [Switch](11-switch.md) | `kit.Switch("Music", ui.Checked(true))` |
| [Slider](12-slider.md) | `kit.Slider(70, ui.ID("music"))` |
| [Progress](13-progress.md) | `kit.Progress(65)` |
| [Text input](14-text-input.md) | `kit.TextInput("", ui.Placeholder("Hero name"), kit.MaxLength(16))` |
| [Text area](15-text-area.md) | `kit.TextArea("", ui.ID("quest-notes"), ui.Placeholder("Quest notes"), kit.Rows(6))` |
| [Number field](16-number-field.md) | `kit.NumberField(32, kit.Range(1, 256), ui.ID("width"))` |
| [Choice](17-choice.md) | `kit.Choice(0, []*ui.Node{kit.Label("Tiles"), kit.Label("Objects"), kit.Label("Lights")})` |
| [Pop-up](18-pop-up.md) | `kit.PopUp(0, []string{"Windowed", "Borderless", "Fullscreen"})` |
| [Path pop-up](19-path-pop-up.md) | `kit.PathPopUp("/Users/ada/assets/survivors/crypt")` |
| [Tooltip](20-tooltip.md) | `kit.Button("Save", kit.Tooltip(kit.Label("Saves the tileset")))` |
| [Hint](33-hint.md) | `kit.Hint(kit.KeyboardHint("Arrows to look · Enter to choose"), kit.MouseHint(…))` |

## Forms, lists and navigation

| Component | Example |
|---|---|
| [Form](22-form.md) | `kit.Form(kit.Field("Name", kit.TextInput("Ayla")), …)` |
| [Tabs](23-tabs.md) | `kit.Tabs(kit.Tab("Video", videoPage), kit.Tab("Audio", audioPage))` |
| [Table](24-table.md) | `kit.Table([]*ui.Node{kit.TableColumn("Name", kit.Sortable()), …}, …)` |
| [Sidebar](25-sidebar.md) | `kit.Sidebar(kit.SidebarSection("Places", kit.SidebarItem("Home")), …)` |
| [Thumbnails](26-thumbnails.md) | `kit.Thumbnails(kit.Thumbnail("Crypt", "crypt.png"), kit.Thumbnail("Sanctuary", "sanctuary.png"))` |
| [Status bar](27-status-bar.md) | `kit.StatusBar("walls.tsj · 41 tiles · 3 shapes", kit.Label("100%", kit.Caps()))` |

## Menus, windows and dialogs

| Component | Example |
|---|---|
| [Menu](28-menu.md) | `kit.MenuButton("Tile actions", kit.Menu(kit.MenuItem("Duplicate", kit.Shortcut("Cmd+D")), …))` |
| [Menu bar](29-menu-bar.md) | `kit.MenuBar(kit.MenuBarItem("File", kit.Menu(kit.MenuItem("Quit"))), …)` |
| [Window](30-window.md) | `kit.Window("Inspector", kit.Toolbar(kit.Button("Save")), …)` |
| [Alert](31-alert.md) | `kit.Alert("Save the tileset?", "Your changes will be lost.", kit.Button("Cancel"), …)` |
| [File dialog](32-file-dialog.md) | `kit.NewFileDialog(doc, os.DirFS(dir), kit.FileType("Pictures", ".png", ".jpg"))` |

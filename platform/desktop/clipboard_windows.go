package desktop

import "github.com/lantervale/ember/internal/clipboard"

func setClipboardHTML(html, text string) bool { return clipboard.SetHTML(html, text) == nil }

func clipboardHTML() string { return clipboard.HTML() }

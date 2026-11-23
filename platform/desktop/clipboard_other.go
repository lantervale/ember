//go:build !darwin && !windows

package desktop

// setClipboardHTML takes no HTML: a Linux desktop's clipboard offers its
// targets through GLFW, which offers text alone.
func setClipboardHTML(string, string) bool { return false }

func clipboardHTML() string { return "" }

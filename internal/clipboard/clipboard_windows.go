package clipboard

import (
	"errors"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

//nolint:gochecknoglobals // the system's functions, found once
var (
	user32         = syscall.NewLazyDLL("user32.dll")
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	openClipboard  = user32.NewProc("OpenClipboard")
	closeClipboard = user32.NewProc("CloseClipboard")
	emptyClipboard = user32.NewProc("EmptyClipboard")
	setData        = user32.NewProc("SetClipboardData")
	getData        = user32.NewProc("GetClipboardData")
	registerFormat = user32.NewProc("RegisterClipboardFormatW")
	globalAlloc    = kernel32.NewProc("GlobalAlloc")
	globalFree     = kernel32.NewProc("GlobalFree")
	globalLock     = kernel32.NewProc("GlobalLock")
	globalUnlock   = kernel32.NewProc("GlobalUnlock")
	globalSize     = kernel32.NewProc("GlobalSize")
	moveMemory     = kernel32.NewProc("RtlMoveMemory")
)

const (
	unicodeText = 13 // CF_UNICODETEXT
	moveable    = 2  // GMEM_MOVEABLE
)

var errBusy = errors.New("clipboard: another program holds the clipboard")

// htmlFormat is the clipboard format Windows registers as HTML Format.
func htmlFormat() uintptr {
	name, err := syscall.UTF16PtrFromString("HTML Format")
	if err != nil {
		return 0
	}
	f, _, _ := registerFormat.Call(uintptr(unsafe.Pointer(name)))
	return f
}

// open opens the clipboard, a few times over a short while when another
// program holds it.
func open() error {
	for range 5 {
		if ok, _, _ := openClipboard.Call(0); ok != 0 {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return errBusy
}

// SetHTML puts html, as HTML Format, and text, its plain version, on the
// clipboard together.
func SetHTML(html, text string) error {
	format := htmlFormat()
	if format == 0 {
		return errors.New("clipboard: no HTML Format")
	}
	if err := open(); err != nil {
		return err
	}
	defer closeClipboard.Call() //nolint:errcheck // nothing to do about a clipboard that won't close
	emptyClipboard.Call()       //nolint:errcheck // what it held is replaced either way
	wide := utf16.Encode([]rune(text + "\x00"))
	if err := put(unicodeText, unsafe.Pointer(&wide[0]), uintptr(len(wide)*2)); err != nil {
		return err
	}
	data := append(CFHTML(html), 0)
	return put(format, unsafe.Pointer(&data[0]), uintptr(len(data)))
}

// put copies size bytes from at into memory the clipboard then owns, as
// format.
func put(format uintptr, at unsafe.Pointer, size uintptr) error {
	h, _, _ := globalAlloc.Call(moveable, size)
	if h == 0 {
		return errors.New("clipboard: out of memory")
	}
	p, _, _ := globalLock.Call(h)
	if p == 0 {
		globalFree.Call(h) //nolint:errcheck // freed on the way out
		return errors.New("clipboard: memory not locked")
	}
	moveMemory.Call(p, uintptr(at), size) //nolint:errcheck // a copy returns nothing
	globalUnlock.Call(h)                  //nolint:errcheck // unlocked whatever it says
	if ok, _, _ := setData.Call(format, h); ok == 0 {
		globalFree.Call(h) //nolint:errcheck // the clipboard did not take it
		return errors.New("clipboard: format not taken")
	}
	return nil
}

// HTML is the fragment of the HTML Format on the clipboard, empty when
// there is none.
func HTML() string {
	format := htmlFormat()
	if format == 0 || open() != nil {
		return ""
	}
	defer closeClipboard.Call() //nolint:errcheck // nothing to do about a clipboard that won't close
	h, _, _ := getData.Call(format)
	if h == 0 {
		return ""
	}
	size, _, _ := globalSize.Call(h)
	p, _, _ := globalLock.Call(h)
	if p == 0 || size == 0 {
		return ""
	}
	defer globalUnlock.Call(h) //nolint:errcheck // unlocked whatever it says
	data := make([]byte, size)
	moveMemory.Call(uintptr(unsafe.Pointer(&data[0])), p, size) //nolint:errcheck // a copy returns nothing
	for i, b := range data {
		if b == 0 {
			data = data[:i]
			break
		}
	}
	return Fragment(data)
}

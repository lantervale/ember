//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>

// emberSetClipboardHTML puts both on the general pasteboard as one item,
// public.html beside the plain string, as a browser's copy does.
static void emberSetClipboardHTML(const char *html, const char *text) {
	@autoreleasepool {
		NSPasteboard *board = [NSPasteboard generalPasteboard];
		[board clearContents];
		[board setString:[NSString stringWithUTF8String:html] forType:NSPasteboardTypeHTML];
		[board setString:[NSString stringWithUTF8String:text] forType:NSPasteboardTypeString];
	}
}

// emberClipboardHTML is the pasteboard's public.html, to free, NULL when
// it holds none.
static char *emberClipboardHTML(void) {
	@autoreleasepool {
		NSString *html = [[NSPasteboard generalPasteboard] stringForType:NSPasteboardTypeHTML];
		if (!html) {
			return NULL;
		}
		return strdup([html UTF8String]);
	}
}
*/
import "C"

import (
	"strings"
	"unsafe"
)

func setClipboardHTML(html, text string) bool {
	h, t := C.CString(pasteboardHTML(html)), C.CString(strings.ReplaceAll(text, "\x00", ""))
	defer C.free(unsafe.Pointer(h))
	defer C.free(unsafe.Pointer(t))
	C.emberSetClipboardHTML(h, t)
	return true
}

func clipboardHTML() string {
	h := C.emberClipboardHTML()
	if h == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(h))
	return C.GoString(h)
}

// pasteboardHTML is html as Chrome puts it on the pasteboard: marked
// UTF-8, which public.html says nothing of.
func pasteboardHTML(html string) string {
	return "<meta charset='utf-8'>" + strings.ReplaceAll(html, "\x00", "")
}

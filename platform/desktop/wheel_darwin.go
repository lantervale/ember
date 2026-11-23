//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#include <stdint.h>

extern void emberWheelKind(uintptr_t window, int precise);

static void emberWatchWheel(void) {
	@autoreleasepool {
		[NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskScrollWheel handler:^NSEvent *(NSEvent *e) {
			emberWheelKind((uintptr_t)[e window], [e hasPreciseScrollingDeltas] ? 1 : 0);
			return e;
		}];
	}
}
*/
import "C"

import (
	"sync"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// preciseWheel is whether each window's last wheel event, by its
// NSWindow, came from a trackpad or a smooth wheel: AppKit's monitor
// sees it just before GLFW scales its points to tenths.
var (
	preciseWheel = map[uintptr]bool{} //nolint:gochecknoglobals // AppKit calls back with no state of ours
	watchWheel   sync.Once            //nolint:gochecknoglobals // one monitor for the application
)

// wheelPrecise reports whether h's wheel event being delivered scrolls
// by points.
func wheelPrecise(h *glfw.Window) bool {
	watchWheel.Do(func() { C.emberWatchWheel() })
	return preciseWheel[uintptr(h.GetCocoaWindow())]
}

func forgetWheel(h *glfw.Window) { delete(preciseWheel, uintptr(h.GetCocoaWindow())) }

//export emberWheelKind
func emberWheelKind(window C.uintptr_t, precise C.int) {
	preciseWheel[uintptr(window)] = precise != 0
}

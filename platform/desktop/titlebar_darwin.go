//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// The last left press AppKit handed out: what a window drag starts from.
static NSEvent *emberPress;

static void emberWatchPresses(void) {
	static id monitor;
	if (monitor) {
		return;
	}
	monitor = [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskLeftMouseDown
		handler:^NSEvent *(NSEvent *event) {
			[emberPress release];
			emberPress = [event retain];
			return event;
		}];
}

static void emberOwnTitleBar(void *nswindow) {
	NSWindow *w = (NSWindow *)nswindow;
	w.titlebarAppearsTransparent = YES;
	w.titleVisibility = NSWindowTitleHidden;
	w.styleMask |= NSWindowStyleMaskFullSizeContentView;
	// An empty toolbar makes the title bar a header's height, the
	// window's buttons centred in it, as in the system's own apps.
	if (!w.toolbar) {
		NSToolbar *bar = [[NSToolbar alloc] initWithIdentifier:@"ember"];
		w.toolbar = bar;
		[bar release];
		if (@available(macOS 11.0, *)) {
			w.toolbarStyle = NSWindowToolbarStyleUnifiedCompact;
			w.titlebarSeparatorStyle = NSTitlebarSeparatorStyleNone;
		}
	}
}

// emberTitleBarArea is the title bar AppKit draws over the content, zero
// for a window without one or fullscreen. GLFW rebuilds the style mask
// when the window leaves its fullscreen or changes whether it resizes,
// so a mask that lost the full-size content is given it back.
static void emberTitleBarArea(void *nswindow, double *left, double *right, double *height) {
	NSWindow *w = (NSWindow *)nswindow;
	*left = *right = *height = 0;
	NSWindowStyleMask mask = w.styleMask;
	if (!(mask & NSWindowStyleMaskTitled) || (mask & NSWindowStyleMaskFullScreen)) {
		return;
	}
	if (!(mask & NSWindowStyleMaskFullSizeContentView)) {
		emberOwnTitleBar(nswindow);
	}
	*height = NSHeight(w.frame) - NSHeight(w.contentLayoutRect);
	NSButton *close = [w standardWindowButton:NSWindowCloseButton];
	NSButton *zoom = [w standardWindowButton:NSWindowZoomButton];
	if (!close || !zoom) {
		return;
	}
	NSRect c = [close convertRect:close.bounds toView:nil];
	NSRect z = [zoom convertRect:zoom.bounds toView:nil];
	double width = NSWidth(w.frame);
	if (NSMinX(c) < NSMinX(z)) {
		*left = NSMaxX(z) + NSMinX(c);
	} else {
		*right = 2 * width - NSMinX(z) - NSMaxX(c);
	}
}

static void emberStartDrag(void *nswindow) {
	NSWindow *w = (NSWindow *)nswindow;
	if (emberPress && emberPress.window == w && ([NSEvent pressedMouseButtons] & 1)) {
		[w performWindowDragWithEvent:emberPress];
	}
}

static void emberDoubleClickTitleBar(void *nswindow) {
	NSWindow *w = (NSWindow *)nswindow;
	NSString *action = [[NSUserDefaults standardUserDefaults] stringForKey:@"AppleActionOnDoubleClick"];
	if ([action isEqualToString:@"None"]) {
		return;
	}
	if ([action isEqualToString:@"Minimize"]) {
		[w performMiniaturize:nil];
		return;
	}
	[w performZoom:nil];
}

// The C side ends here.
*/
import "C"

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/platform"
)

// ownsTitleBar reports whether this system lets a window's content fill
// its title bar.
const ownsTitleBar = true

// watchPresses keeps the last left press, for startDrag.
func watchPresses() { C.emberWatchPresses() }

// ownTitleBar has h's content fill its title bar, the title hidden and
// the window's buttons kept.
func ownTitleBar(h *glfw.Window) { C.emberOwnTitleBar(h.GetCocoaWindow()) }

func titleBarArea(h *glfw.Window) platform.TitleBarArea {
	var left, right, height C.double
	C.emberTitleBarArea(h.GetCocoaWindow(), &left, &right, &height)
	return platform.TitleBarArea{Left: float32(left), Right: float32(right), Height: float32(height)}
}

func startDrag(h *glfw.Window) { C.emberStartDrag(h.GetCocoaWindow()) }

func doubleClickTitleBar(h *glfw.Window) { C.emberDoubleClickTitleBar(h.GetCocoaWindow()) }

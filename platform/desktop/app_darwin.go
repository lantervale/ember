//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

static int emberQuit, emberReopen;
static NSApplicationTerminateReply (*emberTerminate)(id, SEL, NSApplication *);

// GLFW answers Quit by asking every window to close, which a program
// without a window never hears, and which a window can't tell from its
// own close button: the request is noted before GLFW has it.
static NSApplicationTerminateReply emberShouldTerminate(id self, SEL cmd, NSApplication *app) {
	emberQuit = 1;
	return emberTerminate(self, cmd, app);
}

// A click on the Dock icon of a program showing no window, which GLFW
// 3.3 has no callback for.
static BOOL emberShouldReopen(id self, SEL cmd, NSApplication *app, BOOL visible) {
	if (!visible) {
		emberReopen = 1;
	}
	return YES;
}

static void emberWatchApp(void) {
	@autoreleasepool {
		id delegate = [NSApp delegate];
		if (!delegate || emberTerminate) {
			return;
		}
		Class c = [delegate class];
		Method m = class_getInstanceMethod(c, @selector(applicationShouldTerminate:));
		if (!m) {
			return;
		}
		emberTerminate = (NSApplicationTerminateReply (*)(id, SEL, NSApplication *))
			method_setImplementation(m, (IMP)emberShouldTerminate);
		class_addMethod(c, @selector(applicationShouldHandleReopen:hasVisibleWindows:),
			(IMP)emberShouldReopen, "c@:@c");
		// AppKit asks a delegate what it answers to when it is set.
		[NSApp setDelegate:nil];
		[NSApp setDelegate:delegate];
	}
}

static int emberTake(int *flag) {
	int was = *flag;
	*flag = 0;
	return was;
}

static int emberTakeQuit(void) { return emberTake(&emberQuit); }
static int emberTakeReopen(void) { return emberTake(&emberReopen); }

// The C side ends here.
*/
import "C"

// watchApp notes what the system asks of the program itself, not of a
// window: to quit (the menu's Quit, the Dock's) and to show itself again
// (a click on its Dock icon while it shows no window), and what the user
// changes of their preferences.
func watchApp() {
	C.emberWatchApp()
	watchPreferences()
}

// appQuitRequested reports, once, that the system asked the program to
// quit since it was last read.
func appQuitRequested() bool { return C.emberTakeQuit() != 0 }

// appReopened reports, once, that the user asked for the program again
// while it showed no window.
func appReopened() bool { return C.emberTakeReopen() != 0 }

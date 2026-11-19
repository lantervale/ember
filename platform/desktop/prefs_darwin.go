//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#include <stdlib.h>
#include <string.h>

static int emberPrefsChanged = 1;

static void emberPrefsObserved(id self, SEL cmd, NSString *path, id object, NSDictionary *change, void *context) {
	emberPrefsChanged = 1;
}

// The app's effective appearance is observed by key: it follows the
// system's, Auto's turn at dusk included. The languages have no
// notification of their own: they are read again when the locale changes
// and each time the user comes back to the program, as from the system's
// settings.
static void emberWatchPreferences(void) {
	@autoreleasepool {
		static id observer;
		if (observer || !NSApp) {
			return;
		}
		Class c = objc_allocateClassPair([NSObject class], "EmberPreferencesObserver", 0);
		if (!c) {
			return;
		}
		class_addMethod(c, @selector(observeValueForKeyPath:ofObject:change:context:),
			(IMP)emberPrefsObserved, "v@:@@@^v");
		objc_registerClassPair(c);
		observer = [[c alloc] init];
		[NSApp addObserver:observer forKeyPath:@"effectiveAppearance" options:0 context:NULL];
		void (^changed)(NSNotification *) = ^(NSNotification *note) {
			emberPrefsChanged = 1;
		};
		NSNotificationCenter *center = [NSNotificationCenter defaultCenter];
		[center addObserverForName:NSCurrentLocaleDidChangeNotification object:nil queue:nil usingBlock:changed];
		[center addObserverForName:NSApplicationDidBecomeActiveNotification object:nil queue:nil usingBlock:changed];
	}
}

static int emberTakePrefsChanged(void) {
	int was = emberPrefsChanged;
	emberPrefsChanged = 0;
	return was;
}

// 0 unknown, 1 light, 2 dark, as platform.Appearance.
static int emberAppearance(void) {
	@autoreleasepool {
		NSAppearanceName name = [[NSApp effectiveAppearance]
			bestMatchFromAppearancesWithNames:@[NSAppearanceNameAqua, NSAppearanceNameDarkAqua]];
		if (!name) {
			return 0;
		}
		return [name isEqualToString:NSAppearanceNameDarkAqua] ? 2 : 1;
	}
}

static void emberOverrideAppearance(int appearance) {
	@autoreleasepool {
		NSAppearanceName name = appearance == 2 ? NSAppearanceNameDarkAqua : NSAppearanceNameAqua;
		[NSApp setAppearance:appearance ? [NSAppearance appearanceNamed:name] : nil];
	}
}

static char *emberLanguages(void) {
	@autoreleasepool {
		const char *tags = [[[NSLocale preferredLanguages] componentsJoinedByString:@"\n"] UTF8String];
		return strdup(tags ? tags : "");
	}
}
*/
import "C"

import (
	"strings"
	"time"
	"unsafe"

	"github.com/lantervale/ember/platform"
)

// watchPreferences has AppKit note when the appearance or the languages
// may have changed.
func watchPreferences() { C.emberWatchPreferences() }

// systemChanged reports, once, that the preferences are to read again: at
// the start, then when AppKit said so.
func systemChanged(*time.Time) bool { return C.emberTakePrefsChanged() != 0 }

// systemAppearance is the app's effective appearance, the system's unless
// the program gave the app one of its own.
func systemAppearance() platform.Appearance { return platform.Appearance(C.emberAppearance()) }

// systemLanguages is the user's preferred languages, which macOS names as
// BCP 47 does.
func systemLanguages() []string {
	tags := C.emberLanguages()
	defer C.free(unsafe.Pointer(tags))
	return strings.Fields(C.GoString(tags))
}

// overrideAppearance gives the app an appearance of its own, or the
// system's back with AppearanceUnknown, for tests: the system's setting is
// never touched.
func overrideAppearance(a platform.Appearance) { C.emberOverrideAppearance(C.int(a)) }

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
		[center addObserverForName:NSSystemColorsDidChangeNotification object:nil queue:nil usingBlock:changed];
		[[[NSWorkspace sharedWorkspace] notificationCenter]
			addObserverForName:NSWorkspaceAccessibilityDisplayOptionsDidChangeNotification object:nil queue:nil
			usingBlock:changed];
	}
}

// 1 full, 2 reduced, as platform.Motion: Accessibility's Reduce motion.
static int emberMotion(void) {
	@autoreleasepool {
		return [[NSWorkspace sharedWorkspace] accessibilityDisplayShouldReduceMotion] ? 2 : 1;
	}
}

// 1 no preference, 2 more, as platform.Contrast: Accessibility's Increase
// contrast. macOS forces no colours.
static int emberContrast(void) {
	@autoreleasepool {
		return [[NSWorkspace sharedWorkspace] accessibilityDisplayShouldIncreaseContrast] ? 2 : 1;
	}
}

// emberSelectionColors writes the colours AppKit selects text in, focused
// then not, as sRGB 0x1RRGGBB, 0 for one it can't say, resolved in the
// light appearance as Chrome resolves them whatever the scheme.
static void emberSelectionColors(unsigned int *focused, unsigned int *unfocused) {
	@autoreleasepool {
		unsigned int (^rgb)(NSColor *) = ^unsigned int(NSColor *color) {
			NSColor *c = [color colorUsingColorSpace:[NSColorSpace sRGBColorSpace]];
			if (!c) {
				return 0;
			}
			return 1u << 24 | (unsigned int)lround([c redComponent] * 255) << 16 |
				(unsigned int)lround([c greenComponent] * 255) << 8 | (unsigned int)lround([c blueComponent] * 255);
		};
		[[NSAppearance appearanceNamed:NSAppearanceNameAqua] performAsCurrentDrawingAppearance:^{
			*focused = rgb([NSColor selectedTextBackgroundColor]);
			*unfocused = rgb([NSColor unemphasizedSelectedTextBackgroundColor]);
		}];
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
	"unsafe"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/sysprefs"
	"github.com/lantervale/ember/platform"
)

// watchPreferences has AppKit note when the appearance, the display's
// accessibility options, the system's colours or the languages may have
// changed.
func watchPreferences() { C.emberWatchPreferences() }

// systemChanged reports, once, that the preferences are to read again: at
// the start, then when AppKit said so.
func systemChanged(bool) bool { return C.emberTakePrefsChanged() != 0 }

func systemPreferences() sysprefs.System {
	return sysprefs.System{
		Appearance: systemAppearance(), Motion: systemMotion(), Contrast: systemContrast(),
		ForcedColors: platform.ForcedColorsNone, Highlight: systemHighlight(), Languages: systemLanguages(),
	}
}

// systemContrast is whether Accessibility's Display asks for more
// contrast.
func systemContrast() platform.Contrast { return platform.Contrast(C.emberContrast()) }

// systemHighlight is the colours AppKit selects text in, the text keeping
// its own, as Chrome keeps it on macOS.
func systemHighlight() platform.Highlight {
	var focused, unfocused C.uint
	C.emberSelectionColors(&focused, &unfocused)
	rgb := func(v C.uint) color.Color {
		if v>>24 == 0 {
			return color.Color{}
		}
		return color.RGB8(uint8(v>>16), uint8(v>>8), uint8(v))
	}
	return platform.Highlight{Background: rgb(focused), Inactive: rgb(unfocused)}
}

// systemAppearance is the app's effective appearance, the system's unless
// the program gave the app one of its own.
func systemAppearance() platform.Appearance { return platform.Appearance(C.emberAppearance()) }

// systemMotion is whether Accessibility's Display asks to reduce motion.
func systemMotion() platform.Motion { return platform.Motion(C.emberMotion()) }

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

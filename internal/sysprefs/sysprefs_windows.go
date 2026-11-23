package sysprefs

import (
	"syscall"
	"unsafe"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
)

// languages is the user's display languages, the preferred one first, as
// Windows names them, which is as BCP 47 does.
func languages() []string {
	const byName = 0x8 // MUI_LANGUAGE_NAME
	ask := syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserPreferredUILanguages")
	if ask.Find() != nil {
		return nil
	}
	var count, size uint32
	ok, _, _ := ask.Call(byName, uintptr(unsafe.Pointer(&count)), 0, uintptr(unsafe.Pointer(&size)))
	if ok == 0 || size == 0 {
		return nil
	}
	names := make([]uint16, size)
	ok, _, _ = ask.Call(
		byName, uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&names[0])), uintptr(unsafe.Pointer(&size)),
	)
	if ok == 0 {
		return nil
	}
	var tags []string
	for from, i := 0, 0; i < len(names); i++ {
		if names[i] != 0 {
			continue
		}
		if i > from {
			tags = append(tags, syscall.UTF16ToString(names[from:i]))
		}
		from = i + 1
	}
	return tags
}

// appearance is the look the user chose for apps, which Windows 10 and
// later keep in the registry; unknown before them.
func appearance() platform.Appearance {
	path, err := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`)
	if err != nil {
		return platform.AppearanceUnknown
	}
	var key syscall.Handle
	if syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, path, 0, syscall.KEY_READ, &key) != nil {
		return platform.AppearanceUnknown
	}
	defer syscall.RegCloseKey(key) //nolint:errcheck // nothing to do about a key that won't close
	name, err := syscall.UTF16PtrFromString("AppsUseLightTheme")
	if err != nil {
		return platform.AppearanceUnknown
	}
	var kind, light uint32
	size := uint32(unsafe.Sizeof(light))
	err = syscall.RegQueryValueEx(key, name, nil, &kind, (*byte)(unsafe.Pointer(&light)), &size)
	if err != nil || kind != syscall.REG_DWORD {
		return platform.AppearanceUnknown
	}
	if light == 0 {
		return platform.AppearanceDark
	}
	return platform.AppearanceLight
}

// motion is reduced where the user turned Windows' animation effects
// off, as SPI_GETCLIENTAREAANIMATION reads them; unknown where it can't
// be asked.
func motion() platform.Motion {
	const clientAreaAnimation = 0x1042 // SPI_GETCLIENTAREAANIMATION
	ask := syscall.NewLazyDLL("user32.dll").NewProc("SystemParametersInfoW")
	if ask.Find() != nil {
		return platform.MotionUnknown
	}
	var animate int32
	if ok, _, _ := ask.Call(clientAreaAnimation, 0, uintptr(unsafe.Pointer(&animate)), 0); ok == 0 {
		return platform.MotionUnknown
	}
	if animate == 0 {
		return platform.MotionReduced
	}
	return platform.MotionFull
}

// Read is what Windows says: the registry's appearance, its animation
// effects, its contrast theme and the colours it selects in, and the
// user's display languages.
func Read() System {
	s := System{Appearance: appearance(), Motion: motion(), Languages: languages()}
	s.ForcedColors, s.Contrast = contrast()
	s.Highlight = highlight()
	return s
}

// The system colours GetSysColor reads.
const (
	colorWindow        = 5
	colorWindowText    = 8
	colorHighlight     = 13
	colorHighlightText = 14
)

// contrast is whether a contrast theme forces its colours, as
// SPI_GETHIGHCONTRAST says, and the contrast it then asks for, from its
// window's text and background; unknown where it can't be asked.
func contrast() (platform.ForcedColors, platform.Contrast) {
	const getHighContrast, on = 0x0042, 0x1 // SPI_GETHIGHCONTRAST, HCF_HIGHCONTRASTON
	ask := syscall.NewLazyDLL("user32.dll").NewProc("SystemParametersInfoW")
	if ask.Find() != nil {
		return platform.ForcedColorsUnknown, platform.ContrastUnknown
	}
	var hc struct {
		size, flags uint32
		scheme      uintptr
	}
	hc.size = uint32(unsafe.Sizeof(hc))
	if ok, _, _ := ask.Call(getHighContrast, uintptr(hc.size), uintptr(unsafe.Pointer(&hc)), 0); ok == 0 {
		return platform.ForcedColorsUnknown, platform.ContrastUnknown
	}
	if hc.flags&on == 0 {
		return platform.ForcedColorsNone, platform.ContrastNoPreference
	}
	back, okBack := sysColor(colorWindow)
	text, okText := sysColor(colorWindowText)
	if !okBack || !okText {
		return platform.ForcedColorsActive, platform.ContrastUnknown
	}
	return platform.ForcedColorsActive, ForcedContrast(back, text)
}

// highlight is the colours Windows selects in, Highlight and
// HighlightText; Windows has no colour of its own for an inactive window's
// selection.
func highlight() platform.Highlight {
	back, ok := sysColor(colorHighlight)
	if !ok {
		return platform.Highlight{}
	}
	text, _ := sysColor(colorHighlightText)
	return platform.Highlight{Background: back, Text: text}
}

// sysColor is the system colour index names, from its COLORREF.
func sysColor(index uintptr) (color.Color, bool) {
	ask := syscall.NewLazyDLL("user32.dll").NewProc("GetSysColor")
	if ask.Find() != nil {
		return color.Color{}, false
	}
	ref, _, _ := ask.Call(index)
	return color.RGB8(uint8(ref&0xff), uint8(ref>>8&0xff), uint8(ref>>16&0xff)), true
}

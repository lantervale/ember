package sysprefs

import (
	"syscall"
	"unsafe"

	"github.com/lantervale/ember/platform"
)

// Languages is the user's display languages, the preferred one first, as
// Windows names them, which is as BCP 47 does.
func Languages() []string {
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

// Appearance is the look the user chose for apps, which Windows 10 and
// later keep in the registry; unknown before them.
func Appearance() platform.Appearance {
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

// Motion is reduced where the user turned Windows' animation effects
// off, as SPI_GETCLIENTAREAANIMATION reads them; unknown where it can't
// be asked.
func Motion() platform.Motion {
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

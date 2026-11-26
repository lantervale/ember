package css //nolint:testpackage // the package's own tests

import "testing"

func TestDecodeReadsASheetsBytesAsCSSSyntaxDoes(t *testing.T) {
	for _, c := range []struct {
		why, data, environment, text, encoding string
	}{
		{"UTF-8 by default", ".caf\xc3\xa9 {}", "", ".café {}", "utf-8"},
		{"the referrer's encoding without its own", ".caf\xe9 {}", "windows-1252", ".café {}", "windows-1252"},
		{"its @charset over the referrer's", "@charset \"shift_jis\";.\x95\xbd\x98\x61{}", "windows-1252",
			"@charset \"shift_jis\";.平和{}", "shift_jis"},
		{"a label as WHATWG names it", "@charset \"us-ascii\";.caf\xe9{}", "",
			"@charset \"us-ascii\";.café{}", "windows-1252"},
		{"a UTF-8 BOM over everything, and dropped", "\xef\xbb\xbf@charset \"shift_jis\";.caf\xc3\xa9{}", "windows-1252",
			"@charset \"shift_jis\";.café{}", "utf-8"},
		{"a UTF-16 BOM", "\xff\xfe.\x00a\x00{\x00}\x00", "", ".a{}", "utf-16le"},
		{"@charset utf-16 without a BOM is UTF-8", "@charset \"utf-16le\";.caf\xc3\xa9{}", "windows-1252",
			"@charset \"utf-16le\";.café{}", "utf-8"},
		{"only the exact @charset counts", "@charset 'shift_jis';.caf\xc3\xa9{}", "",
			"@charset 'shift_jis';.café{}", "utf-8"},
		{"nor an unknown label", "@charset \"klingon\";.caf\xe9{}", "windows-1252",
			"@charset \"klingon\";.café{}", "windows-1252"},
	} {
		text, encoding := Decode([]byte(c.data), c.environment)
		if text != c.text || encoding != c.encoding {
			t.Errorf("%s: %q in %s, want %q in %s", c.why, text, encoding, c.text, c.encoding)
		}
	}
}

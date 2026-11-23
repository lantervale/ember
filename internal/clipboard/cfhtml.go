// Package clipboard holds what the system clipboards need of a rich copy
// that GLFW does not give: Windows' HTML Format.
package clipboard

import (
	"fmt"
	"regexp"
	"strconv"
)

// header is CF_HTML's description, its offsets ten digits wide so that
// its length is known before they are.
const header = "Version:0.9\r\nStartHTML:%010d\r\nEndHTML:%010d\r\nStartFragment:%010d\r\nEndFragment:%010d\r\n"

const (
	before = "<html><body>\r\n<!--StartFragment-->"
	after  = "<!--EndFragment-->\r\n</body></html>"
)

// CFHTML is fragment in Windows' HTML Format: a header giving, in bytes,
// where the page and the fragment start and end in it, then a page
// holding the fragment between its comments.
func CFHTML(fragment string) []byte {
	head := len(fmt.Sprintf(header, 0, 0, 0, 0))
	start := head + len(before)
	end := start + len(fragment)
	total := end + len(after)
	return []byte(fmt.Sprintf(header, head, total, start, end) + before + fragment + after)
}

var offset = regexp.MustCompile(`(?m)^(StartFragment|EndFragment):(\d+)\r?$`)

// Fragment is the fragment data in HTML Format holds, empty when its
// header says no place for it.
func Fragment(data []byte) string {
	at := map[string]int{}
	for _, m := range offset.FindAllSubmatch(data, -1) {
		v, err := strconv.Atoi(string(m[2]))
		if err != nil {
			return ""
		}
		at[string(m[1])] = v
	}
	start, okStart := at["StartFragment"]
	end, okEnd := at["EndFragment"]
	if !okStart || !okEnd || start < 0 || end < start || end > len(data) {
		return ""
	}
	return string(data[start:end])
}

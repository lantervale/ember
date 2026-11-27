package reftest

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lantervale/ember/ui/internal/fonts"
)

var (
	doctype         = regexp.MustCompile(`(?is)^\s*(<\?xml[^>]*\?>\s*)?(<!--.*?-->\s*)*<!doctype[^>]*>`)
	headTag         = regexp.MustCompile(`(?i)<head(\s[^>]*)?>`)
	htmlTag         = regexp.MustCompile(`(?i)<html(\s[^>]*)?>`)
	fontDeclaration = regexp.MustCompile(`(?i)(font(?:-family)?\s*:)([^;{}<>]*)`)
)

// Inject puts a style element holding css at the start of page, before
// its own sheets: in an XHTML page (xml) first in its head, or its html
// when it has none; in an HTML page after its doctype, where the parser
// moves it into the head and the page keeps its mode.
func Inject(page []byte, xml bool, css string) []byte {
	style := []byte("<style>" + css + "</style>")
	at := 0
	if xml {
		if loc := headTag.FindIndex(page); loc != nil {
			at = loc[1]
		} else if loc = htmlTag.FindIndex(page); loc != nil {
			at = loc[1]
		}
	} else if loc := doctype.FindIndex(page); loc != nil {
		at = loc[1]
	}
	out := make([]byte, 0, len(page)+len(style))
	out = append(out, page[:at]...)
	out = append(out, style...)
	return append(out, page[at:]...)
}

// engineFonts is where Handler serves ember's own faces.
const engineFonts = "/.ember-fonts/"

// The families ember draws the generic ones with (ui/internal/fonts),
// and those it has none for, unquoted, so a style attribute's quotes
// still hold them.
const (
	defaultFamily = `Go`
	monoFamily    = `Go Mono`
)

// chromeSheet is the sheet Handler puts first in every page: the WPT
// fonts, ember's own faces under their names, and the page's text in
// ember's default family, so Chrome draws the text ember draws in its
// own faces with the same ones; and the faces' files by name.
func chromeSheet() (string, map[string][]byte) {
	var b strings.Builder
	b.WriteString(strings.ReplaceAll(FontsSheet, Base, "/") + "\n")
	files := map[string][]byte{}
	for i, f := range fonts.Builtin().Faces() {
		name := fmt.Sprintf("%d.ttf", i)
		files[name] = f.TTF
		style := "normal"
		if f.Italic {
			style = "italic"
		}
		fmt.Fprintf(&b, "@font-face { font-family: %q; font-weight: %d; font-style: %s; src: url(%s%s); }\n",
			f.Family, f.Weight, style, engineFonts, name)
	}
	b.WriteString(":root { font-family: " + defaultFamily + "; }\n")
	return b.String(), files
}

// Generics names, in the font and font-family declarations of css, the
// generic families by the faces ember draws them with, so Chrome draws
// them alike: those ember has no face for (cursive, fantasy…) in its
// default one, as ember does.
func Generics(css string) string {
	return fontDeclaration.ReplaceAllStringFunc(css, func(decl string) string {
		m := fontDeclaration.FindStringSubmatch(decl)
		parts := strings.Split(m[2], ",")
		for i, part := range parts {
			words := strings.Fields(part)
			if len(words) == 0 {
				continue
			}
			last := len(words) - 1
			switch strings.ToLower(words[last]) {
			case "serif", "sans-serif", "cursive", "fantasy", "system-ui", "ui-serif", "ui-sans-serif",
				"ui-rounded", "emoji", "fangsong":
				words[last] = defaultFamily
			case "monospace", "ui-monospace":
				words[last] = monoFamily
			default:
				continue
			}
			lead := part[:len(part)-len(strings.TrimLeft(part, " \t\n"))]
			trail := part[len(strings.TrimRight(part, " \t\n")):]
			parts[i] = lead + strings.Join(words, " ") + trail
		}
		return m[1] + strings.Join(parts, ",")
	})
}

// Handler serves the checkout at root to Chrome: its pages with the
// sheet of chromeSheet first and their generic families named as ember
// draws them (Generics), as their sheets, XHTML as XML; with the
// Content-Type of a file's .headers, as WPT's server, and HTML that
// declares no encoding as UTF-8, as ember reads it.
func Handler(root string) http.Handler {
	sheet, files := chromeSheet()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if name, engine := strings.CutPrefix(r.URL.Path, engineFonts); engine {
			if data, ok := files[name]; ok {
				w.Header().Set("Content-Type", "font/ttf")
				_, _ = w.Write(data)
				return
			}
		}
		p := path.Clean("/" + r.URL.Path)[1:]
		file := filepath.Join(root, filepath.FromSlash(p))
		data, err := os.ReadFile(file) //nolint:gosec // cleaned, under root
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ext := strings.ToLower(path.Ext(p))
		kind := mime.TypeByExtension(ext)
		switch ext {
		case ".xht", ".xhtml":
			kind = "application/xhtml+xml"
			data = Inject([]byte(Generics(string(data))), true, sheet)
		case ".html", ".htm":
			kind = "text/html"
			if !declaresEncoding(data) {
				kind += "; charset=utf-8"
			}
			data = Inject([]byte(Generics(string(data))), false, sheet)
		case ".css":
			kind = "text/css"
			data = []byte(Generics(string(data)))
		}
		if kind == "" {
			kind = http.DetectContentType(data)
		}
		if served := servedType(file); served != "" {
			kind = served
		}
		w.Header().Set("Content-Type", kind)
		_, _ = w.Write(data) //nolint:gosec // the checkout's own pages, to a local Chrome
	})
}

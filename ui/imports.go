package ui

import (
	"fmt"
	"io/fs"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/lantervale/ember/ui/css"
)

// SheetFetcher gets the stylesheet an @import names, as the importing
// sheet resolves it: in full from a sheet that says where it came from
// (From), as written beside the sheet's name otherwise, when no file
// system of the sheet holds it. It is called off the frame, a few
// sheets at a time, so it may take as long as it needs and must be safe
// to call from several goroutines. sheet is the address of the sheet
// that imports it, "" when it says none: what a request for it gives as
// its referrer.
type SheetFetcher func(url, sheet string) ([]byte, error)

// FetchSheets has fetch get the sheets the Document's @import rules
// name, as a browser follows them: each in its place, as if its rules
// stood where the @import is, under its media query list and, for a
// sheet read as a browser's, in its layer() and only where its
// supports() holds; an imported sheet's own imports from its address;
// an import of a sheet that imports it stopped, with a warning. A sheet
// is fetched once, however many imports name it, and until it lands the
// Document shows without it, then styles again: once a frame for all
// that landed. A Stylesheet's imports are read from its file system,
// at once, when it holds them. A fetch that fails is a warning naming
// the address, as is an @import without a fetcher. ui/web has a
// fetcher over HTTP.
func FetchSheets(fetch SheetFetcher) DocumentOption {
	return func(d *Document) { d.pageSheets.fetch = fetch }
}

// sheetFetchesAtOnce is how many sheets a Document fetches at a time.
const sheetFetchesAtOnce = 6

// pageSheets are the sheets a Document's @import rules name, by
// address: nil while one is on its way, and for one that failed.
type pageSheets struct {
	fetch   SheetFetcher
	sheets  map[string]*css.Sheet
	loading int

	sync.WaitGroup
	mu     sync.Mutex
	landed []importedSheet
	slots  chan struct{}
}

// importedSheet is a sheet back from its fetch, read off the frame, or
// why there is none.
type importedSheet struct {
	url   string
	sheet *css.Sheet
	warns []css.Warning
	err   error
}

// imported is s with the sheets its @import rules name in their
// places, those that have come so far: read from fsys when it holds
// them, fetched otherwise. chain are the addresses s is imported
// through, its own last, and from is its own, "" for none.
func (d *Document) imported(s *css.Sheet, fsys fs.FS, chain []string, from string) *css.Sheet {
	imps := s.Imports()
	if len(imps) == 0 {
		return s
	}
	got := make([]*css.Sheet, len(imps))
	for i, imp := range imps {
		if slices.Contains(chain, imp.URL) {
			d.warn(fmt.Errorf("ui: @import %s: a sheet that imports itself", imp.URL))
			continue
		}
		if in := d.importedSheet(imp.URL, fsys, from); in != nil {
			got[i] = d.imported(in, fsys, append(slices.Clip(chain), imp.URL), imp.URL)
		}
	}
	return s.Imported(got)
}

// importedSheet is the sheet at address, read once: nil while it is
// on its way, and when it can't be had.
func (d *Document) importedSheet(address string, fsys fs.FS, from string) *css.Sheet {
	p := &d.pageSheets
	if s, ok := p.sheets[address]; ok {
		return s
	}
	if p.sheets == nil {
		p.sheets = map[string]*css.Sheet{}
	}
	p.sheets[address] = nil
	data, inline := dataURL(address)
	if !inline && fsys != nil && !strings.Contains(address, ":") {
		var err error
		if data, err = fs.ReadFile(fsys, address); err == nil {
			inline = true
		} else if p.fetch == nil {
			d.warn(fmt.Errorf("ui: @import %s: %w", address, err))
			return nil
		}
	}
	if inline {
		s, warns := css.Parse(address, string(data), d.importParsing(address)...)
		for _, w := range warns {
			d.warn(w)
		}
		d.addFontFaces(s.FontFaces(), fsys)
		p.sheets[address] = s
		return s
	}
	if p.fetch == nil {
		d.warn(fmt.Errorf("ui: @import %s is not followed; fetch it with FetchSheets", address))
		return nil
	}
	d.fetchSheet(address, from)
	return nil
}

// importParsing are the css options a sheet imported from address is
// read with: from its address when it is one.
func (d *Document) importParsing(address string) []css.ParseOption {
	if at, err := url.Parse(address); err == nil && at.Scheme != "" && at.Scheme != "data" {
		return d.parsing(css.From(address))
	}
	return d.parsing()
}

// fetchSheet asks the fetcher for the sheet at address off the frame,
// and reads it there.
func (d *Document) fetchSheet(address, from string) {
	p := &d.pageSheets
	if p.slots == nil {
		p.slots = make(chan struct{}, sheetFetchesAtOnce)
	}
	fetch, opts := p.fetch, d.importParsing(address)
	p.loading++
	p.Add(1)
	go func() {
		defer p.Done()
		p.slots <- struct{}{}
		defer func() { <-p.slots }()
		got := importedSheet{url: address}
		data, err := fetch(address, from)
		if err != nil {
			got.err = err
		} else {
			got.sheet, got.warns = css.Parse(address, string(data), opts...)
		}
		p.mu.Lock()
		defer p.mu.Unlock()
		p.landed = append(p.landed, got)
	}()
}

// landSheets takes the sheets fetched since the last frame: the
// Document's sheets take them in their places, once for them all.
func (d *Document) landSheets() {
	p := &d.pageSheets
	p.mu.Lock()
	landed := p.landed
	p.landed = nil
	p.mu.Unlock()
	for _, got := range landed {
		p.loading--
		if got.err != nil {
			d.warn(fmt.Errorf("ui: @import %s: %w", got.url, got.err))
			continue
		}
		for _, w := range got.warns {
			d.warn(w)
		}
		d.addFontFaces(got.sheet.FontFaces(), nil)
		p.sheets[got.url] = got.sheet
	}
	if len(landed) == 0 {
		return
	}
	for _, s := range d.authored {
		s.read = nil
	}
	d.resheet = true
}

// SheetsLoading is how many of the sheets the Document's @import rules
// name are on their way: asked for and neither landed nor failed.
func (d *Document) SheetsLoading() int { return d.pageSheets.loading }

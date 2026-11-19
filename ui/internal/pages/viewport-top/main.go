// The page of viewport-end before it scrolls: body's picture, placed at
// the bottom right of the page, is far below the window, whose own bottom
// right corner shows none of it.
package main

import "github.com/lantervale/ember/ui/internal/pages"

func main() { pages.Run(pages.TallPage(), 320, 260) }

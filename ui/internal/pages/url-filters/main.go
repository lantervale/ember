// filter: url(#id) against Chrome's: elements run through the filter
// elements of an svg element of the page, blurs, shadows, glows, colour
// matrices, transfers, morphology, offsets, blends and composites, a
// region cut, one in sRGB, CSS functions after a url(), and backdrops.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	_ "github.com/lantervale/ember/ui/svg"
)

const filters = `<svg xmlns="http://www.w3.org/2000/svg" width="0" height="0">
<filter id="blur"><feGaussianBlur stdDeviation="3"/></filter>
<filter id="srgb" color-interpolation-filters="sRGB"><feGaussianBlur stdDeviation="3"/></filter>
<filter id="shadow"><feDropShadow dx="4" dy="4" stdDeviation="2" flood-color="#203040" flood-opacity="0.6"/></filter>
<filter id="glow" x="-30%" y="-30%" width="160%" height="160%">
  <feGaussianBlur in="SourceAlpha" stdDeviation="4" result="b"/>
  <feFlood flood-color="#f0b020"/><feComposite in2="b" operator="in" result="g"/>
  <feMerge><feMergeNode in="g"/><feMergeNode in="SourceGraphic"/></feMerge></filter>
<filter id="matrix"><feColorMatrix type="saturate" values="0.2"/><feColorMatrix type="hueRotate" values="90"/></filter>
<filter id="transfer"><feComponentTransfer>
  <feFuncR type="table" tableValues="0 1 0.2"/><feFuncG type="discrete" tableValues="0 0.5 1"/>
  <feFuncB type="gamma" amplitude="1" exponent="0.5" offset="0"/><feFuncA type="linear" slope="0.8"/>
</feComponentTransfer></filter>
<filter id="dilate"><feMorphology operator="dilate" radius="2"/></filter>
<filter id="erode"><feMorphology operator="erode" radius="1.5 3"/></filter>
<filter id="offset"><feOffset dx="6" dy="3" result="o"/>
  <feComposite in="SourceGraphic" in2="o" operator="xor"/></filter>
<filter id="blend"><feFlood flood-color="#3070d0"/><feBlend mode="multiply" in2="SourceGraphic"/>
  <feComposite in2="SourceGraphic" operator="atop"/></filter>
<filter id="cut" x="10%" y="20%" width="60%" height="50%"><feOffset dx="0"/></filter>
<filter id="out"><feFlood flood-color="#c03030" result="f"/>
  <feComposite in="f" in2="SourceAlpha" operator="out"/></filter>
</svg>`

const card = "card"

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 12px; display: grid; grid-template-columns: repeat(4, 96px); gap: 18px; }
.card { height: 56px; border-radius: 10px; background: linear-gradient(135deg, #d04a3c, #3c6fd0);
  color: #fff; font: 700 15px sans-serif; display: flex; align-items: center; justify-content: center; }
.ring { background: none; border: 6px solid #2a8a4a; border-radius: 50%; width: 44px; height: 44px;
  justify-self: center; }
.pane { height: 56px; box-sizing: border-box; padding: 10px;
  background: linear-gradient(90deg, #d04a3c 50%, #fff 50%) 0 0 / 12px 12px; }
.pane > div { height: 36px; border: 1px solid #333; }
`

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	d.Body().Append(ui.Svg(filters, ui.Style("position: absolute; width: 0; height: 0")))
	for _, c := range []struct{ filter, text, class string }{
		{"url(#blur)", "blur", card}, {"url(#srgb)", "sRGB", card}, {"url(#shadow)", "shadow", card},
		{"url(#glow)", "glow", card}, {"url(#matrix)", "matrix", card}, {"url(#transfer)", "transfer", card},
		{"url(#dilate)", "", "ring"}, {"url(#erode)", "", "ring"}, {"url(#offset)", "offset", card},
		{"url(#blend)", "blend", card}, {"url(#cut)", "cut", card}, {"url(#out)", "out", card},
		{"url(#blur) grayscale(1)", "then", card}, {"sepia(1) url(#shadow)", "before", card},
		{"url(#missing)", "missing", card}, {"url(#cut) drop-shadow(3px 3px 0 #000)", "both", card},
	} {
		d.Body().Append(ui.Div(ui.Class(c.class), ui.Style("filter: "+c.filter), ui.Text(c.text)))
	}
	for _, f := range []string{"url(#blur)", "url(#matrix)", "url(#dilate) invert(1)", "url(#missing)"} {
		d.Body().Append(ui.Div(ui.Class("pane"), ui.Div(ui.Style("backdrop-filter: "+f))))
	}
	pages.Run(d, 500, 410)
}

package css

import (
	"fmt"
	"strings"
)

// property is one longhand of the subset: whether it inherits and its
// initial value as CSS text.
type property struct {
	inherited bool
	initial   string
}

//nolint:gochecknoglobals,goconst // the properties' table
var properties = map[string]property{
	"display":    {initial: "inline"},
	"box-sizing": {initial: "content-box"},
	"width":      {initial: "auto"}, "height": {initial: "auto"},
	"min-width": {initial: "auto"}, "min-height": {initial: "auto"},
	"max-width": {initial: "none"}, "max-height": {initial: "none"},

	"margin-top": {initial: "0"}, "margin-right": {initial: "0"},
	"margin-bottom": {initial: "0"}, "margin-left": {initial: "0"},
	"padding-top": {initial: "0"}, "padding-right": {initial: "0"},
	"padding-bottom": {initial: "0"}, "padding-left": {initial: "0"},

	"border-top-width": {initial: "medium"}, "border-right-width": {initial: "medium"},
	"border-bottom-width": {initial: "medium"}, "border-left-width": {initial: "medium"},
	"border-top-style": {initial: "none"}, "border-right-style": {initial: "none"},
	"border-bottom-style": {initial: "none"}, "border-left-style": {initial: "none"},
	"border-top-color": {initial: "currentcolor"}, "border-right-color": {initial: "currentcolor"},
	"border-bottom-color": {initial: "currentcolor"}, "border-left-color": {initial: "currentcolor"},
	"border-top-left-radius": {initial: "0"}, "border-top-right-radius": {initial: "0"},
	"border-bottom-right-radius": {initial: "0"}, "border-bottom-left-radius": {initial: "0"},
	"border-image-source": {initial: "none"}, "border-image-slice": {initial: "100%"},
	"border-image-width": {initial: "1"}, "border-image-outset": {initial: "0"},
	"border-image-repeat": {initial: "stretch"},

	"row-gap": {initial: "normal"}, "column-gap": {initial: "normal"},
	"flex-grow": {initial: "0"}, "flex-shrink": {initial: "1"}, "flex-basis": {initial: "auto"},
	"flex-wrap": {initial: "nowrap"}, "flex-direction": {initial: "row"},
	"align-items": {initial: "normal"}, "align-content": {initial: "normal"},
	"align-self": {initial: "auto"}, "justify-content": {initial: "normal"},
	"justify-items": {initial: "legacy"}, "justify-self": {initial: "auto"},
	"grid-template-columns": {initial: "none"}, "grid-template-rows": {initial: "none"},
	"grid-template-areas": {initial: "none"},
	"grid-column-start":   {initial: "auto"}, "grid-column-end": {initial: "auto"},
	"grid-row-start": {initial: "auto"}, "grid-row-end": {initial: "auto"},
	"grid-auto-columns": {initial: "auto"}, "grid-auto-rows": {initial: "auto"},
	"grid-auto-flow": {initial: "row"},
	"order":          {initial: "0"},
	"column-count":   {initial: "auto"}, "column-width": {initial: "auto"},
	"column-fill": {initial: "balance"}, "column-span": {initial: "none"},
	"column-rule-width": {initial: "medium"}, "column-rule-style": {initial: "none"},
	"column-rule-color": {initial: "currentcolor"},
	"break-inside":      {initial: "auto"},
	"break-before":      {initial: "auto"}, "break-after": {initial: "auto"},
	"orphans": {inherited: true, initial: "2"}, "widows": {inherited: true, initial: "2"},
	"aspect-ratio": {initial: "auto"},
	"line-clamp":   {initial: "none"}, "-webkit-line-clamp": {initial: "none"},
	"-webkit-box-orient": {initial: "horizontal"}, "-webkit-box-pack": {initial: "start"},
	"-webkit-box-align": {initial: "stretch"}, "-webkit-box-direction": {initial: "normal"},
	"-webkit-box-flex": {initial: "0"}, "-webkit-box-ordinal-group": {initial: "1"},
	"overflow-x": {initial: "visible"}, "overflow-y": {initial: "visible"},
	"position": {initial: "static"},
	"float":    {initial: "none"}, "clear": {initial: "none"}, propInitialLetter: {initial: "normal"},
	"vertical-align": {initial: "baseline"},
	"top":            {initial: "auto"}, "right": {initial: "auto"},
	"bottom": {initial: "auto"}, "left": {initial: "auto"},
	"z-index":        {initial: "auto"},
	"container-type": {initial: "normal"}, "container-name": {initial: "none"},
	"anchor-name": {initial: "none"}, "anchor-scope": {initial: "none"}, "position-anchor": {initial: "normal"},
	"position-area": {initial: "none"}, "position-try-fallbacks": {initial: "none"},
	"position-try-order": {initial: "normal"}, "position-visibility": {initial: "anchors-visible"},

	"font-family":               {inherited: true, initial: "sans-serif"},
	"font-size":                 {inherited: true, initial: "medium"},
	"font-weight":               {inherited: true, initial: "normal"},
	"font-style":                {inherited: true, initial: "normal"},
	"font-palette":              {inherited: true, initial: "normal"},
	"font-stretch":              {inherited: true, initial: "normal"},
	"font-synthesis":            {inherited: true, initial: "weight style small-caps"},
	propFontVariations:          {inherited: true, initial: "normal"},
	"line-height":               {inherited: true, initial: "normal"},
	"letter-spacing":            {inherited: true, initial: "normal"},
	propCaps:                    {inherited: true, initial: "normal"},
	propLigatures:               {inherited: true, initial: "normal"},
	propNumeric:                 {inherited: true, initial: "normal"},
	propEastAsian:               {inherited: true, initial: "normal"},
	propAlternates:              {inherited: true, initial: "normal"},
	propPosition:                {inherited: true, initial: "normal"},
	propFeatures:                {inherited: true, initial: "normal"},
	propKerning:                 {inherited: true, initial: "auto"},
	"word-spacing":              {inherited: true, initial: "normal"},
	"border-spacing":            {inherited: true, initial: "0"},
	"border-collapse":           {inherited: true, initial: "separate"},
	"caption-side":              {inherited: true, initial: "top"},
	"empty-cells":               {inherited: true, initial: "show"},
	"table-layout":              {initial: "auto"},
	"text-align":                {inherited: true, initial: "start"},
	"direction":                 {inherited: true, initial: "ltr"},
	"unicode-bidi":              {initial: "normal"},
	propWritingMode:             {inherited: true, initial: "horizontal-tb"},
	"text-orientation":          {inherited: true, initial: "mixed"},
	"text-combine-upright":      {inherited: true, initial: "none"},
	"text-indent":               {inherited: true, initial: "0"},
	"text-transform":            {inherited: true, initial: "none"},
	"-webkit-font-smoothing":    {inherited: true, initial: "auto"},
	"text-rendering":            {inherited: true, initial: "auto"},
	"-webkit-text-fill-color":   {inherited: true, initial: "currentcolor"},
	"-webkit-text-stroke-width": {inherited: true, initial: "0"},
	"-webkit-text-stroke-color": {inherited: true, initial: "currentcolor"},
	"white-space-collapse":      {inherited: true, initial: "collapse"},
	"text-wrap-mode":            {inherited: true, initial: "wrap"},
	"text-wrap-style":           {inherited: true, initial: "auto"},
	"hyphens":                   {inherited: true, initial: "manual"},
	"hyphenate-character":       {inherited: true, initial: "auto"},
	"text-box-trim":             {initial: "none"},
	"text-box-edge":             {inherited: true, initial: "auto"},
	"text-spacing-trim":         {inherited: true, initial: "normal"},
	"text-autospace":            {inherited: true, initial: "no-autospace"},
	"interpolate-size":          {inherited: true, initial: "numeric-only"},
	"overflow-wrap":             {inherited: true, initial: "normal"},
	"word-break":                {inherited: true, initial: "normal"},
	"line-break":                {inherited: true, initial: "auto"},
	"tab-size":                  {inherited: true, initial: "8"},
	"text-decoration-skip-ink":  {inherited: true, initial: "auto"},
	"text-underline-offset":     {inherited: true, initial: "auto"},
	"text-underline-position":   {inherited: true, initial: "auto"},
	"color":                     {inherited: true, initial: "canvastext"},
	"color-scheme":              {inherited: true, initial: "normal"},
	"cursor":                    {inherited: true, initial: "auto"},
	"list-style-type":           {inherited: true, initial: "disc"},
	"list-style-image":          {inherited: true, initial: "none"},
	"list-style-position":       {inherited: true, initial: "outside"},
	"quotes":                    {inherited: true, initial: "auto"},
	"content":                   {initial: "normal"},
	"counter-reset":             {initial: "none"}, "counter-increment": {initial: "none"},
	"counter-set":     {initial: "none"},
	"text-shadow":     {inherited: true, initial: "none"},
	"image-rendering": {inherited: true, initial: "auto"},
	"accent-color":    {inherited: true, initial: "auto"},
	"caret-color":     {inherited: true, initial: "auto"},
	"visibility":      {inherited: true, initial: "visible"},
	"pointer-events":  {inherited: true, initial: "auto"},
	// inherited, where Chrome cuts a no-drag box out of the drag boxes
	// around it: the same regions, read off the element under the pointer
	"app-region": {inherited: true, initial: "none"},
	// inherited, as Chrome has it: an element with no rule takes its parent's
	"user-select":        {inherited: true, initial: "auto"},
	"text-overflow":      {initial: "clip"},
	"scrollbar-width":    {initial: "auto"},
	"scrollbar-color":    {inherited: true, initial: "auto"},
	"scrollbar-gutter":   {initial: "auto"},
	propSnapType:         {initial: "none"},
	propSnapAlign:        {initial: "none"},
	propSnapStop:         {initial: "normal"},
	"scroll-padding-top": {initial: "auto"}, "scroll-padding-right": {initial: "auto"},
	"scroll-padding-bottom": {initial: "auto"}, "scroll-padding-left": {initial: "auto"},
	"scroll-margin-top": {initial: "0"}, "scroll-margin-right": {initial: "0"},
	"scroll-margin-bottom": {initial: "0"}, "scroll-margin-left": {initial: "0"},
	propScrollBehavior: {initial: "auto"},
	propOverscrollX:    {initial: "auto"},
	propOverscrollY:    {initial: "auto"},

	"background-color":    {initial: "transparent"},
	"background-image":    {initial: "none"}, // layers of url() or gradients, no repeating ones
	"background-repeat":   {initial: "repeat"},
	"background-size":     {initial: "auto"},
	"background-position": {initial: "0% 0%"},
	// fixed is placed in the Document's viewport, an iframe's own
	"background-attachment": {initial: "scroll"},
	"background-origin":     {initial: "padding-box"},
	"background-clip":       {initial: "border-box"},
	"mask-image":            {initial: "none"}, // layers of url() or gradients, as the background's
	"mask-repeat":           {initial: "repeat"},
	"mask-size":             {initial: "auto"},
	propMaskBoxSource:       {initial: "none"},
	propMaskBoxSlice:        {initial: "0 fill"},
	propMaskBoxWidth:        {initial: "auto"},
	propMaskBoxOutset:       {initial: "0"},
	propMaskBoxRepeat:       {initial: "stretch"},
	propMaskPositionX:       {initial: "0%"},
	propMaskPositionY:       {initial: "0%"},
	"mask-origin":           {initial: "border-box"},
	"mask-clip":             {initial: "border-box"},
	"mask-mode":             {initial: "match-source"},
	"mask-composite":        {initial: "add"},
	"clip-path":             {initial: "none"},
	"object-fit":            {initial: "fill"},
	"object-position":       {initial: "50% 50%"},
	"box-shadow":            {initial: "none"},
	"filter":                {initial: "none"},
	"backdrop-filter":       {initial: "none"},
	"mix-blend-mode":        {initial: "normal"},
	"isolation":             {initial: "auto"},
	"background-blend-mode": {initial: "normal"},
	"outline-width":         {initial: "medium"}, "outline-style": {initial: "none"},
	"outline-color": {initial: "currentcolor"}, "outline-offset": {initial: "0"},
	"opacity":          {initial: "1"},
	"transform":        {initial: "none"},
	"transform-origin": {initial: "50% 50%"},
	"appearance":       {initial: "none"},
	"resize":           {initial: "none"},

	"transition-property": {initial: "all"}, "transition-duration": {initial: "0s"},
	"transition-timing-function": {initial: "ease"}, "transition-delay": {initial: "0s"},
	"transition-behavior": {initial: "normal"},
	"animation-name":      {initial: "none"}, "animation-duration": {initial: "0s"},
	"animation-timing-function": {initial: "ease"}, "animation-delay": {initial: "0s"},
	"animation-iteration-count": {initial: "1"}, "animation-direction": {initial: "normal"},
	"animation-fill-mode": {initial: "none"}, "animation-play-state": {initial: "running"},
	"animation-composition": {initial: "replace"},

	"text-decoration-line":      {initial: "none"},
	"text-decoration-style":     {initial: "solid"},
	"text-decoration-color":     {initial: "currentcolor"},
	"text-decoration-thickness": {initial: "auto"},
}

// shorthands lists each shorthand's longhands, in the order CSS gives them.
//
//nolint:gochecknoglobals,goconst // the shorthands' table
var shorthands = map[string][]string{
	"margin":        {"margin-top", "margin-right", "margin-bottom", "margin-left"},
	"padding":       {"padding-top", "padding-right", "padding-bottom", "padding-left"},
	"inset":         {"top", "right", "bottom", "left"},
	"margin-inline": {"margin-left", "margin-right"}, "margin-block": {"margin-top", "margin-bottom"},
	"padding-inline": {"padding-left", "padding-right"}, "padding-block": {"padding-top", "padding-bottom"},
	"inset-inline": {"left", "right"}, "inset-block": {"top", "bottom"},
	"border-width":  {"border-top-width", "border-right-width", "border-bottom-width", "border-left-width"},
	"border-style":  {"border-top-style", "border-right-style", "border-bottom-style", "border-left-style"},
	"border-color":  {"border-top-color", "border-right-color", "border-bottom-color", "border-left-color"},
	"border-top":    {"border-top-width", "border-top-style", "border-top-color"},
	"border-right":  {"border-right-width", "border-right-style", "border-right-color"},
	"border-bottom": {"border-bottom-width", "border-bottom-style", "border-bottom-color"},
	"border-left":   {"border-left-width", "border-left-style", "border-left-color"},
	"border-inline": {"border-left-width", "border-left-style", "border-left-color", "border-right-width",
		"border-right-style", "border-right-color"},
	"border-block": {"border-top-width", "border-top-style", "border-top-color", "border-bottom-width",
		"border-bottom-style", "border-bottom-color"},
	"border-inline-width": {"border-left-width", "border-right-width"},
	"border-inline-style": {"border-left-style", "border-right-style"},
	"border-inline-color": {"border-left-color", "border-right-color"},
	"border-block-width":  {"border-top-width", "border-bottom-width"},
	"border-block-style":  {"border-top-style", "border-bottom-style"},
	"border-block-color":  {"border-top-color", "border-bottom-color"},
	"border": {"border-top-width", "border-right-width", "border-bottom-width", "border-left-width", "border-top-style",
		"border-right-style", "border-bottom-style", "border-left-style", "border-top-color", "border-right-color",
		"border-bottom-color", "border-left-color", "border-image-source", "border-image-slice", "border-image-width",
		"border-image-outset", "border-image-repeat"},
	"border-radius": {"border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius",
		"border-bottom-left-radius"},
	"border-image": {"border-image-source", "border-image-slice", "border-image-width", "border-image-outset",
		"border-image-repeat"},
	shortGap:        {"row-gap", "column-gap"},
	"flex":          {"flex-grow", "flex-shrink", "flex-basis"},
	"flex-flow":     {"flex-direction", "flex-wrap"},
	"place-items":   {"align-items", "justify-items"},
	"place-content": {"align-content", "justify-content"},
	"place-self":    {"align-self", "justify-self"},
	"grid-column":   {"grid-column-start", "grid-column-end"},
	"grid-row":      {"grid-row-start", "grid-row-end"},
	"grid-area":     {"grid-row-start", "grid-column-start", "grid-row-end", "grid-column-end"},
	"grid-template": {"grid-template-rows", "grid-template-columns", "grid-template-areas"},
	"overflow":      {"overflow-x", "overflow-y"},
	"outline":       {"outline-color", "outline-style", "outline-width"},
	"columns":       {"column-width", "column-count"},
	"white-space":   {"white-space-collapse", "text-wrap-mode"},
	"text-wrap":     {"text-wrap-mode", "text-wrap-style"},
	"text-box":      {"text-box-trim", "text-box-edge"},
	"container":     {"container-name", "container-type"},
	"position-try":  {"position-try-order", "position-try-fallbacks"},
	"column-rule":   {"column-rule-width", "column-rule-style", "column-rule-color"},
	shortStroke:     {"-webkit-text-stroke-width", "-webkit-text-stroke-color"},
	"grid": {"grid-template-rows", "grid-template-columns", "grid-template-areas", "grid-auto-rows", "grid-auto-columns",
		"grid-auto-flow"},
	"background": {"background-color", "background-image", "background-repeat", "background-position",
		"background-size", "background-attachment", "background-origin", "background-clip"},
	propMaskPosition: {propMaskPositionX, propMaskPositionY},
	shortMaskBox:     {propMaskBoxSource, propMaskBoxSlice, propMaskBoxWidth, propMaskBoxOutset, propMaskBoxRepeat},
	"mask": {"mask-image", propMaskPositionX, propMaskPositionY, "mask-size", "mask-repeat", "mask-origin", "mask-clip",
		"mask-composite",
		"mask-mode"},
	"list-style": {"list-style-position", "list-style-image", "list-style-type"},
	"scroll-padding": {"scroll-padding-top", "scroll-padding-right", "scroll-padding-bottom",
		"scroll-padding-left"},
	"scroll-margin":          {"scroll-margin-top", "scroll-margin-right", "scroll-margin-bottom", "scroll-margin-left"},
	shortScrollPaddingInline: {"scroll-padding-left", "scroll-padding-right"},
	"scroll-padding-block":   {"scroll-padding-top", "scroll-padding-bottom"},
	shortScrollMarginInline:  {"scroll-margin-left", "scroll-margin-right"},
	"scroll-margin-block":    {"scroll-margin-top", "scroll-margin-bottom"},
	"overscroll-behavior":    {propOverscrollX, propOverscrollY},
	"font": {"font-style", "font-weight", "font-stretch", "font-size", "line-height", "font-family", propCaps,
		propLigatures, propNumeric, propEastAsian, propAlternates, propPosition, propFeatures, propKerning},
	shortVariant: variantLonghands,
	"transition": {"transition-property", "transition-duration", "transition-timing-function", "transition-delay",
		"transition-behavior"},
	"animation": {"animation-duration", "animation-timing-function", "animation-delay", "animation-iteration-count",
		"animation-direction", "animation-fill-mode", "animation-play-state", "animation-name"},

	"text-decoration": {"text-decoration-line", "text-decoration-style", "text-decoration-color",
		"text-decoration-thickness"},
}

// outside names properties a browser knows that the subset leaves out,
// with why.
//
//nolint:gochecknoglobals // unsupported properties' advice
var outside = map[string]string{
	"word-wrap":       "word-wrap is the old name of overflow-wrap",
	"grid-gap":        "grid-gap is the old name of gap",
	"grid-row-gap":    "grid-row-gap is the old name of row-gap",
	"grid-column-gap": "grid-column-gap is the old name of column-gap",
	"box-flex":        "old flexbox is not supported; use flex",
	"zoom":            "zoom is not supported; use transform: scale()",
	"clip":            "clip is not supported; use overflow: clip",
}

// shortGap is gap, the shorthand of row-gap and column-gap.
const shortGap = "gap"

// oldNames are the names of outside a browser's sheet reads as the plain property they
// were renamed to, as Chrome does.
//
//nolint:gochecknoglobals // lookup table
var oldNames = map[string]string{
	"word-wrap":       "overflow-wrap",
	"grid-gap":        shortGap,
	"grid-row-gap":    "row-gap",
	"grid-column-gap": "column-gap",
}

// checkProperty returns why name can't be used, or "" when it can.
func checkProperty(name string) string {
	if strings.HasPrefix(name, "--") {
		return ""
	}
	if _, ok := properties[name]; ok {
		return ""
	}
	if _, ok := shorthands[name]; ok {
		return ""
	}
	if why, ok := outside[name]; ok {
		return why
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Sprintf("vendor-prefixed property %q is not supported", name)
	}
	return fmt.Sprintf("unknown property %q", name)
}

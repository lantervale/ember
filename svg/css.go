package svg

import (
	"sort"
	"strings"
)

// compound is one step of a selector: a tag, an id and classes, any of
// them absent; child says the step before it must be the parent, not any
// ancestor.
type compound struct {
	tag, id string
	classes []string
	child   bool
}

func (c *compound) matches(n *node) bool {
	if c.tag != "" && c.tag != n.name || c.id != "" && c.id != n.attr["id"] {
		return false
	}
	for _, class := range c.classes {
		found := false
		for _, has := range n.classes {
			found = found || has == class
		}
		if !found {
			return false
		}
	}
	return true
}

type rule struct {
	steps       []compound // the last is the element's own
	specificity int
	decl        map[string]string
	// important names the declarations marked !important.
	important map[string]bool
}

func (r *rule) matches(n *node) bool {
	last := len(r.steps) - 1
	if !r.steps[last].matches(n) {
		return false
	}
	at := n.parent
	for i := last - 1; i >= 0; i-- {
		direct := r.steps[i+1].child
		for {
			if at == nil {
				return false
			}
			ok := r.steps[i].matches(at)
			at = at.parent
			if ok {
				break
			}
			if direct {
				return false
			}
		}
	}
	return true
}

// parseSelector reads tags, classes and ids joined by descendant and
// child combinators; ok is false for anything else.
func parseSelector(sel string) (rule, bool) {
	var r rule
	child := false
	for _, word := range strings.Fields(strings.ReplaceAll(sel, ">", " > ")) {
		if word == ">" {
			if child || len(r.steps) == 0 {
				return r, false
			}
			child = true
			continue
		}
		if strings.ContainsAny(word, ":[+~@()\"'=|") {
			return r, false
		}
		c := compound{child: child}
		child = false
		for word != "" {
			kind := word[0]
			end := 1
			for end < len(word) && word[end] != '.' && word[end] != '#' {
				end++
			}
			part := word[:end]
			word = word[end:]
			switch {
			case kind == '#' && len(part) > 1 && c.id == "":
				c.id = part[1:]
				r.specificity += 1 << 16
			case kind == '.' && len(part) > 1:
				c.classes = append(c.classes, part[1:])
				r.specificity += 1 << 8
			case kind == '*' && part == "*":
			case kind != '#' && kind != '.' && c.tag == "" && c.id == "" && c.classes == nil:
				c.tag = part
				r.specificity++
			default:
				return r, false
			}
		}
		r.steps = append(r.steps, c)
	}
	return r, len(r.steps) > 0 && !child
}

func stripComments(css string) string {
	for {
		i := strings.Index(css, "/*")
		if i < 0 {
			return css
		}
		end := strings.Index(css[i+2:], "*/")
		if end < 0 {
			return css[:i]
		}
		css = css[:i] + " " + css[i+end+4:]
	}
}

// parseSheet appends the rules of a style sheet to rules, up to limit;
// what it cannot honour (at-rules, other selectors) goes to dropped.
func parseSheet(css string, rules []rule, limit int, dropped func(what string)) []rule {
	css = stripComments(strings.NewReplacer("<![CDATA[", "", "]]>", "", "<!--", "", "-->", "").Replace(css))
	for len(rules) < limit {
		open := strings.IndexByte(css, '{')
		if open < 0 {
			break
		}
		prelude := strings.TrimSpace(css[:open])
		if semi := strings.LastIndexByte(prelude, ';'); semi >= 0 {
			// An at-rule without a block, such as @import or @charset.
			if strings.HasPrefix(strings.TrimSpace(prelude[:semi]), "@import") {
				dropped("@import")
			}
			prelude = strings.TrimSpace(prelude[semi+1:])
		}
		if strings.HasPrefix(prelude, "@") {
			depth, end := 0, open
			for ; end < len(css); end++ {
				if css[end] == '{' {
					depth++
				} else if css[end] == '}' {
					if depth--; depth == 0 {
						break
					}
				}
			}
			name, _, _ := strings.Cut(prelude, " ")
			dropped(name)
			css = css[min(end+1, len(css)):]
			continue
		}
		end := strings.IndexByte(css[open:], '}')
		if end < 0 {
			break
		}
		decl, important := declarations(css[open+1 : open+end])
		css = css[open+end+1:]
		if len(decl) == 0 {
			continue
		}
		for _, sel := range strings.Split(prelude, ",") {
			r, ok := parseSelector(sel)
			if !ok {
				if strings.TrimSpace(sel) != "" {
					dropped("style selector")
				}
				continue
			}
			r.decl, r.important = decl, important
			rules = append(rules, r)
		}
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].specificity < rules[j].specificity })
	return rules
}

// declarations are the properties a block declares, and those of them
// marked !important.
func declarations(s string) (map[string]string, map[string]bool) {
	if !strings.Contains(s, ":") {
		return nil, nil
	}
	out := map[string]string{}
	var important map[string]bool
	for _, d := range strings.Split(s, ";") {
		if k, v, ok := strings.Cut(d, ":"); ok {
			k = strings.ToLower(strings.TrimSpace(k))
			v = strings.TrimSpace(v)
			if i := strings.LastIndexByte(v, '!'); i >= 0 && strings.EqualFold(strings.TrimSpace(v[i+1:]), "important") {
				v = strings.TrimSpace(v[:i])
				if important == nil {
					important = map[string]bool{}
				}
				important[k] = true
			} else if important[k] {
				continue
			}
			out[k] = v
		}
	}
	return out, important
}

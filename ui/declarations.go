package ui

// splitDeclarations cuts a style attribute at the semicolons outside
// strings, parentheses and brackets.
func splitDeclarations(css string) []string {
	var out []string
	depth, from := 0, 0
	var quote byte
	for i := 0; i < len(css); i++ {
		c := css[i]
		switch {
		case quote != 0 && c == '\\':
			i++
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth = max(depth-1, 0)
		case c == ';' && depth == 0:
			out = append(out, css[from:i])
			from = i + 1
		}
	}
	return append(out, css[from:])
}

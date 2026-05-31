// Package capyhost bridges the capy configuration syntax and the internal
// YAML data model the rest of Wave consumes. A `server.capy` file is a clean,
// indentation-based tree (no `:` separators, no `-`/`{}` punctuation noise):
// `key value` is a scalar field, a bare `key` followed by an indented block is
// a nested map or list, list items begin with `-`, and backtick blocks carry
// multi-line strings (SQL, templates) verbatim.
//
// capyhost is a pure text/data transform with no system knowledge: ToYAML
// parses capy into a generic tree and marshals it with yaml.v3 (which does all
// the escaping), so the existing resolver/composition/loader pipeline — which
// is built around yaml.Node — runs unchanged. FromYAML is the inverse, used to
// migrate legacy `.yaml` configs to `.capy`. ReadConfig is the single entry the
// loaders call: it returns YAML bytes for any supported config file, transpiling
// `.capy` on the way in.
package capyhost

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ReadConfig reads a config file and returns YAML bytes. Files with a
// `.capy` or `.wave` extension are transpiled from capy syntax;
// `.yaml`/`.yml`/`.json` are returned as-is (yaml.v3 reads JSON too). This
// is the choke point every config loader calls so the rest of the
// pipeline only ever sees YAML.
//
// Both `.capy` and `.wave` are accepted because the language is capy but
// the product is Wave, and users naturally reach for either extension.
// They are treated identically — no semantic difference.
func ReadConfig(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if isCapyExt(filepath.Ext(path)) {
		return ToYAML(string(raw))
	}
	return raw, nil
}

// isCapyExt reports whether the given file extension (including the
// leading dot) should be parsed as capy. The check is case-insensitive.
func isCapyExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".capy", ".wave":
		return true
	}
	return false
}

// ToYAML transpiles capy source into equivalent YAML bytes.
func ToYAML(src string) ([]byte, error) {
	tree, err := Parse(src)
	if err != nil {
		return nil, err
	}
	if tree == nil {
		return []byte{}, nil
	}
	return yaml.Marshal(tree)
}

// FromYAML transpiles YAML (or JSON) bytes into capy source, preserving key
// order. Used to migrate legacy configs.
func FromYAML(src []byte) (string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return "", err
	}
	root := &doc
	if doc.Kind == yaml.DocumentNode {
		if len(doc.Content) == 0 {
			return "", nil
		}
		root = doc.Content[0]
	}
	var b strings.Builder
	emitNode(&b, root, 0)
	return b.String(), nil
}

// ── capy → tree ──────────────────────────────────────────────────────────────

type parser struct {
	lines []string
	i     int
}

// Parse turns capy source into a generic tree of map[string]any / []any /
// scalar values, suitable for yaml.Marshal.
func Parse(src string) (any, error) {
	p := &parser{lines: strings.Split(src, "\n")}
	p.skip()
	if p.i >= len(p.lines) {
		return nil, nil
	}
	return p.parseBlock(indentOf(p.lines[p.i]))
}

func indentOf(s string) int {
	n := 0
	for _, c := range s {
		switch c {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}

// skip advances past blank lines and whole-line comments.
func (p *parser) skip() {
	for p.i < len(p.lines) {
		t := strings.TrimSpace(p.lines[p.i])
		if t == "" || strings.HasPrefix(t, "#") {
			p.i++
			continue
		}
		break
	}
}

func (p *parser) parseBlock(indent int) (any, error) {
	p.skip()
	if p.i >= len(p.lines) || indentOf(p.lines[p.i]) != indent {
		return nil, nil
	}
	t := strings.TrimSpace(p.lines[p.i])
	if t == "-" || strings.HasPrefix(t, "- ") {
		return p.parseList(indent)
	}
	return p.parseMap(indent)
}

func (p *parser) parseMap(indent int) (any, error) {
	m := map[string]any{}
	for {
		p.skip()
		if p.i >= len(p.lines) {
			break
		}
		ci := indentOf(p.lines[p.i])
		if ci < indent {
			break
		}
		if ci > indent {
			return nil, fmt.Errorf("line %d: unexpected indentation", p.i+1)
		}
		content := strings.TrimSpace(p.lines[p.i])
		key, rest := splitKV(content)
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", p.i+1)
		}
		p.i++
		if rest == "" {
			child, err := p.parseChild(indent)
			if err != nil {
				return nil, err
			}
			m[key] = child
		} else {
			v, err := p.scalarValue(rest)
			if err != nil {
				return nil, err
			}
			m[key] = v
		}
	}
	return m, nil
}

func (p *parser) parseList(indent int) (any, error) {
	list := []any{}
	for {
		p.skip()
		if p.i >= len(p.lines) {
			break
		}
		ci := indentOf(p.lines[p.i])
		if ci < indent {
			break
		}
		if ci > indent {
			return nil, fmt.Errorf("line %d: unexpected indentation", p.i+1)
		}
		content := strings.TrimSpace(p.lines[p.i])
		if content == "-" {
			p.i++
			child, err := p.parseChild(indent)
			if err != nil {
				return nil, err
			}
			list = append(list, child)
		} else if strings.HasPrefix(content, "- ") {
			p.i++
			v, err := p.scalarValue(strings.TrimSpace(content[2:]))
			if err != nil {
				return nil, err
			}
			list = append(list, v)
		} else {
			break
		}
	}
	return list, nil
}

// parseChild parses an indented block deeper than parentIndent (a nested map or
// list), or returns nil when there is none.
func (p *parser) parseChild(parentIndent int) (any, error) {
	p.skip()
	if p.i >= len(p.lines) || indentOf(p.lines[p.i]) <= parentIndent {
		return nil, nil
	}
	return p.parseBlock(indentOf(p.lines[p.i]))
}

// splitKV splits a line into its first whitespace-delimited token and the
// trimmed remainder.
func splitKV(s string) (key, rest string) {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			return s[:i], strings.TrimSpace(s[i+1:])
		}
	}
	return s, ""
}

var (
	intRE   = regexp.MustCompile(`^-?\d+$`)
	floatRE = regexp.MustCompile(`^-?\d+\.\d+$`)
)

// scalarValue interprets a value token. A leading backtick opens a literal
// string (single-line if closed on the same line, otherwise multi-line until a
// lone closing backtick). A fully double-quoted token is an explicit string.
// Everything else is type-inferred (int/float/bool/null/bare-string).
func (p *parser) scalarValue(rest string) (any, error) {
	switch rest {
	case "{}":
		return map[string]any{}, nil
	case "[]":
		return []any{}, nil
	}
	if strings.HasPrefix(rest, "`") {
		body := rest[1:]
		if idx := strings.Index(body, "`"); idx >= 0 {
			return body[:idx], nil
		}
		var sb strings.Builder
		if body != "" {
			sb.WriteString(body)
			sb.WriteByte('\n')
		}
		for p.i < len(p.lines) {
			if strings.TrimSpace(p.lines[p.i]) == "`" {
				p.i++
				return strings.TrimSuffix(sb.String(), "\n"), nil
			}
			sb.WriteString(p.lines[p.i])
			sb.WriteByte('\n')
			p.i++
		}
		return nil, fmt.Errorf("unterminated backtick block")
	}
	return parseScalar(rest), nil
}

func parseScalar(s string) any {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		if uq, err := strconv.Unquote(s); err == nil {
			return uq
		}
		return s[1 : len(s)-1]
	}
	switch s {
	case "true":
		return true
	case "false":
		return false
	case "null", "~":
		return nil
	}
	if intRE.MatchString(s) {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	if floatRE.MatchString(s) {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return s
}

// ── yaml.Node → capy ─────────────────────────────────────────────────────────

func emitNode(b *strings.Builder, n *yaml.Node, indent int) {
	switch n.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			emitEntry(b, n.Content[i].Value, n.Content[i+1], indent, false)
		}
	case yaml.SequenceNode:
		for _, item := range n.Content {
			emitEntry(b, "", item, indent, true)
		}
	default:
		writeIndent(b, indent)
		b.WriteString(scalarToCapy(n))
		b.WriteByte('\n')
	}
}

// emitEntry writes one map entry (key + value) or one list item.
func emitEntry(b *strings.Builder, key string, val *yaml.Node, indent int, listItem bool) {
	prefix := ""
	if listItem {
		prefix = "-"
	} else {
		prefix = key
	}
	switch val.Kind {
	case yaml.ScalarNode:
		writeIndent(b, indent)
		b.WriteString(prefix)
		b.WriteByte(' ')
		writeScalar(b, val, indent)
		b.WriteByte('\n')
	case yaml.MappingNode, yaml.SequenceNode:
		if len(val.Content) == 0 {
			// empty map/list → emit an inline empty marker
			writeIndent(b, indent)
			b.WriteString(prefix)
			if val.Kind == yaml.SequenceNode {
				b.WriteString(" []\n")
			} else {
				b.WriteString(" {}\n")
			}
			return
		}
		writeIndent(b, indent)
		b.WriteString(prefix)
		b.WriteByte('\n')
		emitNode(b, val, indent+4)
	case yaml.AliasNode:
		writeIndent(b, indent)
		b.WriteString(prefix)
		b.WriteByte(' ')
		b.WriteString(scalarToCapy(val))
		b.WriteByte('\n')
	}
}

func writeIndent(b *strings.Builder, n int) {
	for i := 0; i < n; i++ {
		b.WriteByte(' ')
	}
}

// writeScalar writes a scalar value inline, using a backtick block for
// multi-line strings (indented to align under the key).
func writeScalar(b *strings.Builder, n *yaml.Node, indent int) {
	v := n.Value
	if isStringScalar(n) && strings.Contains(v, "\n") {
		b.WriteString("`\n")
		for _, ln := range strings.Split(strings.TrimRight(v, "\n"), "\n") {
			b.WriteString(ln)
			b.WriteByte('\n')
		}
		b.WriteByte('`')
		return
	}
	b.WriteString(scalarToCapy(n))
}

func scalarToCapy(n *yaml.Node) string {
	v := n.Value
	if !isStringScalar(n) {
		return v // int / float / bool / null — emit verbatim
	}
	if v == "" || needsQuote(v) {
		return strconv.Quote(v)
	}
	return v
}

func isStringScalar(n *yaml.Node) bool {
	switch n.Tag {
	case "!!int", "!!float", "!!bool", "!!null":
		return false
	}
	return n.Kind == yaml.ScalarNode
}

// needsQuote reports whether a bare string would be misread by the capy parser
// (as a number/bool/null, or via a leading sigil) and must be quoted.
func needsQuote(s string) bool {
	if s != strings.TrimSpace(s) {
		return true
	}
	switch s {
	case "true", "false", "null", "~":
		return true
	}
	if intRE.MatchString(s) || floatRE.MatchString(s) {
		return true
	}
	switch s[0] {
	case '`', '"', '-', '#', '&', '*', '!', '|', '>', '\'':
		return true
	}
	return false
}

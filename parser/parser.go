package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseError records a parse error with line number.
type ParseError struct {
	Line    int
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

// Parse parses EML source text and returns a Document.
func Parse(src string) (*Document, error) {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	doc := &Document{}

	root := &Section{Name: "", Line: 0}
	doc.Sections = append(doc.Sections, root)
	current := root

	i := 0
	for i < len(lines) {
		lineNum := i + 1
		line := lines[i]
		i++

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		runes := []rune(trimmed)

		// Hash comment
		if runes[0] == '#' {
			continue
		}

		// Emoji comment: line starting with 💬
		if isEmojiStart(runes[0]) {
			sc := newScanner(trimmed)
			emoji := sc.readEmoji()
			if emoji == "💬" {
				continue
			}
		}

		// Section header: first non-whitespace char is '['
		if runes[0] == '[' {
			sec, err := parseSection(trimmed, lineNum)
			if err != nil {
				return nil, err
			}
			doc.Sections = append(doc.Sections, sec)
			current = sec
			continue
		}

		// Key-value pair
		kv, err := parseKeyValue(trimmed, lineNum)
		if err != nil {
			return nil, err
		}
		current.KVs = append(current.KVs, kv)
	}

	return doc, nil
}

// parseSection parses a section header line.
//
// Forms:
//
//	[name]           plain section
//	[🔧 name]        section with emoji decoration
//	[🔑]             pure emoji section (emoji becomes the name)
func parseSection(line string, lineNum int) (*Section, error) {
	sc := newScanner(line)
	sc.next() // consume '['
	sc.skipWS()

	sec := &Section{Line: lineNum}

	// Optional emoji prefix
	if !sc.done() && isEmojiStart(sc.peek()) {
		sec.Emoji = sc.readEmoji()
		sc.skipWS()
	}

	// Section name
	if !sc.done() && sc.peek() != ']' {
		if isIdentStart(sc.peek()) {
			sec.Name = sc.readIdent()
		} else if sc.peek() == '"' || sc.peek() == '\'' {
			v, err := parseString(sc, lineNum)
			if err != nil {
				return nil, err
			}
			sec.Name = v.Str
		} else {
			return nil, &ParseError{Line: lineNum, Message: fmt.Sprintf("invalid section name character %q", string(sc.peek()))}
		}
		sc.skipWS()
	}

	// Pure emoji section: use emoji as name
	if sec.Name == "" && sec.Emoji != "" {
		sec.Name = sec.Emoji
		sec.Emoji = ""
	}

	if sc.done() || sc.peek() != ']' {
		return nil, &ParseError{Line: lineNum, Message: "expected ']' to close section header"}
	}
	sc.next() // consume ']'

	return sec, nil
}

// parseKeyValue parses a key-value line.
//
// Forms:
//
//	key = value                 plain key
//	🔑 key = value              emoji annotation + key
//	🏠 = value                  pure emoji key
//	"quoted key" = value        quoted key
func parseKeyValue(line string, lineNum int) (*KeyValue, error) {
	sc := newScanner(line)
	sc.skipWS()

	kv := &KeyValue{Line: lineNum}

	if !sc.done() && isEmojiStart(sc.peek()) {
		emoji := sc.readEmoji()
		sc.skipWS()

		if sc.done() {
			return nil, &ParseError{Line: lineNum, Message: "unexpected end of line after emoji"}
		}

		if sc.peek() == '=' {
			// Pure emoji key
			kv.Key = emoji
		} else {
			// Emoji annotation + regular key
			kv.Annotation = emoji
			if sc.peek() == '"' || sc.peek() == '\'' {
				v, err := parseString(sc, lineNum)
				if err != nil {
					return nil, err
				}
				kv.Key = v.Str
			} else {
				kv.Key = sc.readIdent()
				if kv.Key == "" {
					return nil, &ParseError{Line: lineNum, Message: "expected key after emoji annotation"}
				}
			}
		}
	} else if sc.peek() == '"' || sc.peek() == '\'' {
		v, err := parseString(sc, lineNum)
		if err != nil {
			return nil, err
		}
		kv.Key = v.Str
	} else {
		kv.Key = sc.readIdent()
		if kv.Key == "" {
			return nil, &ParseError{Line: lineNum, Message: fmt.Sprintf("expected key, got %q", string(sc.peek()))}
		}
	}

	sc.skipWS()
	if sc.done() || sc.peek() != '=' {
		return nil, &ParseError{Line: lineNum, Message: fmt.Sprintf("expected '=' after key %q", kv.Key)}
	}
	sc.next() // consume '='
	sc.skipWS()

	val, err := parseValue(sc, lineNum)
	if err != nil {
		return nil, err
	}
	kv.Value = val

	// Allow trailing comments
	sc.skipWS()
	if !sc.done() && sc.peek() != '#' {
		// Check for emoji comment
		if !isEmojiStart(sc.peek()) {
			return nil, &ParseError{Line: lineNum, Message: fmt.Sprintf("unexpected trailing content: %q", sc.readIdent())}
		}
		// Silently ignore trailing emoji comment — caller can verify via the 💬 prefix
	}

	return kv, nil
}

// parseValue parses a value expression from the current scanner position.
func parseValue(sc *scanner, lineNum int) (*Value, error) {
	if sc.done() {
		return nil, &ParseError{Line: lineNum, Message: "expected value, got end of line"}
	}

	r := sc.peek()

	// Quoted string
	if r == '"' || r == '\'' {
		return parseString(sc, lineNum)
	}

	// Array
	if r == '[' {
		return parseArray(sc, lineNum)
	}

	// Inline table
	if r == '{' {
		return parseInlineTable(sc, lineNum)
	}

	// Emoji value
	if isEmojiStart(r) {
		emoji := sc.readEmoji()
		if emoji == "✅" {
			return &Value{Kind: KindBool, Bool: true}, nil
		}
		if emoji == "❌" {
			return &Value{Kind: KindBool, Bool: false}, nil
		}
		return &Value{Kind: KindEmoji, Str: emoji}, nil
	}

	// Bare literal (number, bool, null, or bare word)
	word := readBareWord(sc)
	return parseLiteral(word, lineNum)
}

// readBareWord reads a word that ends at whitespace or structural chars.
func readBareWord(sc *scanner) string {
	start := sc.pos
	for !sc.done() {
		r := sc.peek()
		if r == ' ' || r == '\t' || r == ',' || r == ']' || r == '}' || r == '#' {
			break
		}
		if isEmojiStart(r) {
			break
		}
		sc.pos++
	}
	return string(sc.runes[start:sc.pos])
}

// parseLiteral converts a raw word into a typed Value.
func parseLiteral(word string, lineNum int) (*Value, error) {
	if word == "" {
		return nil, &ParseError{Line: lineNum, Message: "expected value"}
	}
	switch word {
	case "true":
		return &Value{Kind: KindBool, Bool: true}, nil
	case "false":
		return &Value{Kind: KindBool, Bool: false}, nil
	case "null", "~":
		return &Value{Kind: KindNull}, nil
	}
	if i, err := strconv.ParseInt(word, 10, 64); err == nil {
		return &Value{Kind: KindInt, Int: i}, nil
	}
	if f, err := strconv.ParseFloat(word, 64); err == nil {
		return &Value{Kind: KindFloat, Float: f}, nil
	}
	// Bare word string
	return &Value{Kind: KindString, Str: word}, nil
}

// parseString parses a single or double quoted string, with escape support.
// Also handles triple-quoted strings (""" or ''') for multi-line content.
func parseString(sc *scanner, lineNum int) (*Value, error) {
	quote := sc.next() // opening quote

	// Triple-quoted?
	if !sc.done() && sc.peek() == quote {
		sc.next()
		if !sc.done() && sc.peek() == quote {
			sc.next()
			return parseTripleString(sc, quote, lineNum)
		}
		// Two quotes in a row = empty string
		return &Value{Kind: KindString, Str: ""}, nil
	}

	// Single-quoted: no escape processing
	if quote == '\'' {
		var sb strings.Builder
		for {
			if sc.done() {
				return nil, &ParseError{Line: lineNum, Message: "unterminated string"}
			}
			r := sc.next()
			if r == '\'' {
				break
			}
			sb.WriteRune(r)
		}
		return &Value{Kind: KindString, Str: sb.String()}, nil
	}

	// Double-quoted: escape processing
	var sb strings.Builder
	for {
		if sc.done() {
			return nil, &ParseError{Line: lineNum, Message: "unterminated string"}
		}
		r := sc.next()
		if r == '\\' {
			if sc.done() {
				return nil, &ParseError{Line: lineNum, Message: "unexpected end of escape sequence"}
			}
			esc := sc.next()
			switch esc {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case 'r':
				sb.WriteRune('\r')
			case '\\':
				sb.WriteRune('\\')
			case '"':
				sb.WriteRune('"')
			case '\'':
				sb.WriteRune('\'')
			default:
				sb.WriteRune('\\')
				sb.WriteRune(esc)
			}
			continue
		}
		if r == '"' {
			break
		}
		sb.WriteRune(r)
	}
	return &Value{Kind: KindString, Str: sb.String()}, nil
}

// parseTripleString parses content until the matching closing triple-quote.
// Opening triple-quote has already been consumed.
func parseTripleString(sc *scanner, quote rune, lineNum int) (*Value, error) {
	var sb strings.Builder
	for {
		if sc.done() {
			return nil, &ParseError{Line: lineNum, Message: "unterminated triple-quoted string"}
		}
		r := sc.next()
		if r == quote {
			if !sc.done() && sc.peek() == quote {
				sc.next()
				if !sc.done() && sc.peek() == quote {
					sc.next()
					// Strip optional leading/trailing newline
					s := sb.String()
					s = strings.TrimPrefix(s, "\n")
					s = strings.TrimSuffix(s, "\n")
					return &Value{Kind: KindString, Str: s}, nil
				}
				sb.WriteRune(r)
				sb.WriteRune(r)
				continue
			}
			sb.WriteRune(r)
			continue
		}
		sb.WriteRune(r)
	}
}

// parseArray parses an array value: [elem, elem, ...]
func parseArray(sc *scanner, lineNum int) (*Value, error) {
	sc.next() // consume '['
	val := &Value{Kind: KindArray}

	for {
		sc.skipWS()
		if sc.done() {
			return nil, &ParseError{Line: lineNum, Message: "unterminated array"}
		}
		if sc.peek() == ']' {
			sc.next()
			break
		}

		elem, err := parseValue(sc, lineNum)
		if err != nil {
			return nil, err
		}
		val.Elems = append(val.Elems, elem)

		sc.skipWS()
		if sc.done() {
			return nil, &ParseError{Line: lineNum, Message: "unterminated array"}
		}
		switch sc.peek() {
		case ',':
			sc.next()
		case ']':
			// trailing comma optional; handled on next iteration
		default:
			return nil, &ParseError{Line: lineNum, Message: fmt.Sprintf("expected ',' or ']' in array, got %q", string(sc.peek()))}
		}
	}
	return val, nil
}

// parseInlineTable parses an inline table: { key = val, key = val }
func parseInlineTable(sc *scanner, lineNum int) (*Value, error) {
	sc.next() // consume '{'
	val := &Value{Kind: KindTable, Fields: make(map[string]*Value)}

	for {
		sc.skipWS()
		if sc.done() {
			return nil, &ParseError{Line: lineNum, Message: "unterminated inline table"}
		}
		if sc.peek() == '}' {
			sc.next()
			break
		}

		// Key
		var key string
		if isEmojiStart(sc.peek()) {
			key = sc.readEmoji()
		} else if sc.peek() == '"' || sc.peek() == '\'' {
			v, err := parseString(sc, lineNum)
			if err != nil {
				return nil, err
			}
			key = v.Str
		} else {
			key = sc.readIdent()
		}
		if key == "" {
			return nil, &ParseError{Line: lineNum, Message: "expected key in inline table"}
		}

		sc.skipWS()
		if sc.done() || sc.peek() != '=' {
			return nil, &ParseError{Line: lineNum, Message: fmt.Sprintf("expected '=' after key %q in inline table", key)}
		}
		sc.next() // consume '='
		sc.skipWS()

		v, err := parseValue(sc, lineNum)
		if err != nil {
			return nil, err
		}
		val.Fields[key] = v

		sc.skipWS()
		if sc.done() {
			return nil, &ParseError{Line: lineNum, Message: "unterminated inline table"}
		}
		switch sc.peek() {
		case ',':
			sc.next()
		case '}':
			// handled on next iteration
		default:
			return nil, &ParseError{Line: lineNum, Message: fmt.Sprintf("expected ',' or '}' in inline table, got %q", string(sc.peek()))}
		}
	}
	return val, nil
}

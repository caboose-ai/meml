package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/caboose-mcp/eml/parser"
)

const usage = `eml — Emoji Markup Language tool

Usage:
  eml validate <file>    Check syntax; exit 0 if valid
  eml dump <file>        Print parsed document as JSON
  eml env <file>         Print KEY=VALUE exports (for shell/dotenv use)
  eml help               Show this message
`

func main() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(usage)
		os.Exit(0)
	}

	cmd := args[0]
	if cmd != "help" && len(args) < 2 {
		fmt.Fprintf(os.Stderr, "error: command %q requires a file argument\n", cmd)
		os.Exit(1)
	}

	switch cmd {
	case "validate":
		runValidate(args[1])
	case "dump":
		runDump(args[1])
	case "env":
		runEnv(args[1])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n%s", cmd, usage)
		os.Exit(1)
	}
}

func readAndParse(path string) *parser.Document {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", path, err)
		os.Exit(1)
	}
	doc, err := parser.Parse(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error in %s: %v\n", path, err)
		os.Exit(1)
	}
	return doc
}

func runValidate(path string) {
	readAndParse(path)
	fmt.Printf("ok: %s\n", path)
}

func runDump(path string) {
	doc := readAndParse(path)
	out := docToJSON(doc)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "json error: %v\n", err)
		os.Exit(1)
	}
}

// runEnv prints key=value lines suitable for shell export or dotenv loading.
// Section-prefixed keys use UPPER_SNAKE_CASE: SECTION_KEY=value.
// Null values are skipped. Arrays are joined with commas.
func runEnv(path string) {
	doc := readAndParse(path)

	type entry struct {
		key string
		val string
	}
	var entries []entry

	for _, sec := range doc.Sections {
		for _, kv := range sec.KVs {
			envKey := toEnvKey(sec.Name, kv.Key)
			envVal := valueToEnvString(kv.Value)
			if envVal == "" && kv.Value.Kind == parser.KindNull {
				continue
			}
			entries = append(entries, entry{envKey, envVal})
		}
	}

	// Sort for deterministic output
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})

	for _, e := range entries {
		// Shell-quote the value
		fmt.Printf("%s=%s\n", e.key, shellQuote(e.val))
	}
}

func toEnvKey(section, key string) string {
	full := key
	if section != "" {
		full = section + "_" + key
	}
	// Replace non-alphanumeric with _
	var sb strings.Builder
	for _, r := range strings.ToUpper(full) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

func valueToEnvString(v *parser.Value) string {
	switch v.Kind {
	case parser.KindString, parser.KindEmoji:
		return v.Str
	case parser.KindInt:
		return fmt.Sprintf("%d", v.Int)
	case parser.KindFloat:
		return fmt.Sprintf("%g", v.Float)
	case parser.KindBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case parser.KindNull:
		return ""
	case parser.KindArray:
		parts := make([]string, len(v.Elems))
		for i, e := range v.Elems {
			parts[i] = valueToEnvString(e)
		}
		return strings.Join(parts, ",")
	case parser.KindTable:
		// Inline tables aren't representable as a single env var; use JSON
		b, _ := json.Marshal(tableToMap(v))
		return string(b)
	}
	return ""
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	// If no special chars, no quoting needed
	safe := true
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '"' || r == '\'' || r == '\\' ||
			r == '$' || r == '`' || r == '!' || r == '\n' {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// docToJSON converts the document to a JSON-serialisable structure.
func docToJSON(doc *parser.Document) any {
	result := map[string]any{}
	for _, sec := range doc.Sections {
		pairs := map[string]any{}
		for _, kv := range sec.KVs {
			pairs[kv.Key] = valueToAny(kv.Value)
		}
		if sec.Name == "" {
			// Merge root keys into top level
			for k, v := range pairs {
				result[k] = v
			}
		} else {
			// Include section emoji metadata if present
			if sec.Emoji != "" {
				entry := map[string]any{
					"_emoji": sec.Emoji,
				}
				for k, v := range pairs {
					entry[k] = v
				}
				result[sec.Name] = entry
			} else {
				result[sec.Name] = pairs
			}
		}
	}
	return result
}

func valueToAny(v *parser.Value) any {
	switch v.Kind {
	case parser.KindString, parser.KindEmoji:
		return v.Str
	case parser.KindInt:
		return v.Int
	case parser.KindFloat:
		return v.Float
	case parser.KindBool:
		return v.Bool
	case parser.KindNull:
		return nil
	case parser.KindArray:
		elems := make([]any, len(v.Elems))
		for i, e := range v.Elems {
			elems[i] = valueToAny(e)
		}
		return elems
	case parser.KindTable:
		return tableToMap(v)
	}
	return nil
}

func tableToMap(v *parser.Value) map[string]any {
	m := make(map[string]any, len(v.Fields))
	for k, fv := range v.Fields {
		m[k] = valueToAny(fv)
	}
	return m
}

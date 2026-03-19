package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/caboose-mcp/meml/parser"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	styleOk      = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	styleErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	styleSpinner = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleSec     = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	styleAnnot   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	styleKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	styleStr     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleNum     = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	styleBoolT   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	styleBoolF   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	styleNull    = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	styleEmoji   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleEnvKey  = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	styleEquals  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleJSON    = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// noAnimate disables spinner and typewriter when true.
var noAnimate bool

// ── terminal detection ────────────────────────────────────────────────────────

func isTTY() bool {
	if noAnimate {
		return false
	}
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ── spinner ───────────────────────────────────────────────────────────────────

// withSpinner shows a braille spinner on stderr while fn runs.
// Spinner is suppressed when stderr is not a terminal (e.g. pipes/CI).
func withSpinner(label string, fn func() (*parser.Document, error)) (*parser.Document, error) {
	type result struct {
		doc *parser.Document
		err error
	}

	if !isTTY() {
		return fn()
	}

	ch := make(chan result, 1)
	go func() {
		doc, err := fn()
		ch <- result{doc, err}
	}()

	tick := time.NewTicker(80 * time.Millisecond)
	defer tick.Stop()

	i := 0
	fmt.Fprintf(os.Stderr, "  %s  %s", styleSpinner.Render(spinnerFrames[0]), label)

	for {
		select {
		case r := <-ch:
			fmt.Fprintf(os.Stderr, "\r\033[2K") // clear spinner line
			return r.doc, r.err
		case <-tick.C:
			i++
			fmt.Fprintf(os.Stderr, "\r  %s  %s", styleSpinner.Render(spinnerFrames[i%len(spinnerFrames)]), label)
		}
	}
}

// ── typewriter ────────────────────────────────────────────────────────────────

// typewrite prints each line with a short delay for a subtle "live" feel.
// Falls back to plain print when stdout is not a terminal.
func typewrite(s string) {
	if !isTTY() {
		fmt.Print(s)
		return
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		fmt.Println(line)
		if i < len(lines)-2 { // skip delay on last blank line
			time.Sleep(6 * time.Millisecond)
		}
	}
}

// ── usage ─────────────────────────────────────────────────────────────────────

const usage = `meml — Meme Markup Language

  meml validate <file>           Check syntax; exits 0 if valid
  meml dump <file>               Pretty-print as JSON
  meml pretty <file>             Colorized MEML view
  meml env <file>                KEY=VALUE exports (for shell / dotenv)
  meml help                      Show this message

Flags (place anywhere in the command):
  --no-animate                   Disable spinner and typewriter effect
`

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	// Strip --no-animate from args wherever it appears.
	raw := os.Args[1:]
	args := raw[:0]
	for _, a := range raw {
		if a == "--no-animate" {
			noAnimate = true
		} else {
			args = append(args, a)
		}
	}

	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(usage)
		return
	}

	cmd := args[0]
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "error: %q requires a file argument\n", cmd)
		os.Exit(1)
	}
	path := args[1]

	switch cmd {
	case "validate":
		runValidate(path)
	case "dump":
		runDump(path)
	case "pretty":
		runPretty(path)
	case "env":
		runEnv(path)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n%s", cmd, usage)
		os.Exit(1)
	}
}

// ── commands ──────────────────────────────────────────────────────────────────

func runValidate(path string) {
	doc, err := withSpinner("parsing "+styleDim.Render(path), func() (*parser.Document, error) {
		data, e := os.ReadFile(path)
		if e != nil {
			return nil, e
		}
		return parser.Parse(string(data))
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n  %s\n",
			styleErr.Render("✗"),
			path,
			styleErr.Render(err.Error()),
		)
		os.Exit(1)
	}

	// Count sections and keys
	keys := 0
	for _, s := range doc.Sections {
		keys += len(s.KVs)
	}
	secs := len(doc.Sections) - 1 // exclude root

	fmt.Printf("%s %s  %s\n",
		styleOk.Render("✓"),
		path,
		styleDim.Render(fmt.Sprintf("(%d sections, %d keys)", secs, keys)),
	)
}

func runDump(path string) {
	doc, err := withSpinner("parsing "+styleDim.Render(path), func() (*parser.Document, error) {
		data, e := os.ReadFile(path)
		if e != nil {
			return nil, e
		}
		return parser.Parse(string(data))
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n", styleErr.Render("✗"), err.Error())
		os.Exit(1)
	}

	out := docToJSONMap(doc)
	b, _ := json.MarshalIndent(out, "", "  ")

	if isTTY() {
		typewrite(colorizeJSON(string(b)))
	} else {
		fmt.Println(string(b))
	}
}

func runPretty(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	doc, err := withSpinner("parsing "+styleDim.Render(path), func() (*parser.Document, error) {
		return parser.Parse(string(data))
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n", styleErr.Render("✗"), err.Error())
		os.Exit(1)
	}

	var sb strings.Builder
	for _, sec := range doc.Sections {
		if sec.Name != "" {
			// Section header
			header := ""
			if sec.Emoji != "" {
				header = styleEmoji.Render(sec.Emoji) + " "
			}
			header += styleSec.Render(sec.Name)
			sb.WriteString(styleJSON.Render("[") + header + styleJSON.Render("]") + "\n")
		}
		for _, kv := range sec.KVs {
			line := "  "
			if kv.Annotation != "" {
				line += styleAnnot.Render(kv.Annotation) + " "
			}
			line += styleKey.Render(kv.Key) + styleEquals.Render(" = ") + prettyValue(kv.Value)
			sb.WriteString(line + "\n")
		}
		if len(sec.KVs) > 0 {
			sb.WriteString("\n")
		}
	}

	typewrite(strings.TrimRight(sb.String(), "\n") + "\n")
}

func runEnv(path string) {
	doc, err := withSpinner("parsing "+styleDim.Render(path), func() (*parser.Document, error) {
		data, e := os.ReadFile(path)
		if e != nil {
			return nil, e
		}
		return parser.Parse(string(data))
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n", styleErr.Render("✗"), err.Error())
		os.Exit(1)
	}

	type entry struct{ k, v string }
	var entries []entry

	for _, sec := range doc.Sections {
		for _, kv := range sec.KVs {
			if kv.Value.Kind == parser.KindNull {
				continue
			}
			k := toEnvKey(sec.Name, kv.Key)
			v := valueToEnvString(kv.Value)
			entries = append(entries, entry{k, v})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].k < entries[j].k })

	var sb strings.Builder
	for _, e := range entries {
		if isTTY() {
			sb.WriteString(styleEnvKey.Render(e.k) + styleEquals.Render("=") + styleStr.Render(shellQuote(e.v)) + "\n")
		} else {
			sb.WriteString(e.k + "=" + shellQuote(e.v) + "\n")
		}
	}
	typewrite(sb.String())
}

// ── JSON colorizer ────────────────────────────────────────────────────────────

// colorizeJSON applies lipgloss colors to a JSON string.
func colorizeJSON(input string) string {
	runes := []rune(input)
	var sb strings.Builder
	i := 0

	for i < len(runes) {
		r := runes[i]

		switch {
		case r == '"':
			// Find closing quote, respecting escapes
			j := i + 1
			for j < len(runes) {
				if runes[j] == '\\' {
					j += 2
					continue
				}
				if runes[j] == '"' {
					break
				}
				j++
			}
			s := string(runes[i : j+1])
			// Look ahead past whitespace for ':' to detect keys
			k := j + 1
			for k < len(runes) && (runes[k] == ' ' || runes[k] == '\t') {
				k++
			}
			if k < len(runes) && runes[k] == ':' {
				sb.WriteString(styleKey.Render(s))
			} else {
				sb.WriteString(styleStr.Render(s))
			}
			i = j + 1

		case (r >= '0' && r <= '9') || (r == '-' && i+1 < len(runes) && runes[i+1] >= '0' && runes[i+1] <= '9'):
			j := i + 1
			for j < len(runes) && (runes[j] >= '0' && runes[j] <= '9' || runes[j] == '.' || runes[j] == 'e' || runes[j] == 'E' || runes[j] == '+' || runes[j] == '-') {
				j++
			}
			sb.WriteString(styleNum.Render(string(runes[i:j])))
			i = j

		default:
			rest := string(runes[i:])
			switch {
			case strings.HasPrefix(rest, "true"):
				sb.WriteString(styleBoolT.Render("true"))
				i += 4
			case strings.HasPrefix(rest, "false"):
				sb.WriteString(styleBoolF.Render("false"))
				i += 5
			case strings.HasPrefix(rest, "null"):
				sb.WriteString(styleNull.Render("null"))
				i += 4
			default:
				sb.WriteRune(r)
				i++
			}
		}
	}
	return sb.String()
}

// ── pretty value renderer ─────────────────────────────────────────────────────

func prettyValue(v *parser.Value) string {
	switch v.Kind {
	case parser.KindString:
		return styleStr.Render(`"` + v.Str + `"`)
	case parser.KindInt:
		return styleNum.Render(fmt.Sprintf("%d", v.Int))
	case parser.KindFloat:
		return styleNum.Render(fmt.Sprintf("%g", v.Float))
	case parser.KindBool:
		if v.Bool {
			return styleBoolT.Render("true")
		}
		return styleBoolF.Render("false")
	case parser.KindNull:
		return styleNull.Render("null")
	case parser.KindEmoji:
		return styleEmoji.Render(v.Str)
	case parser.KindArray:
		if len(v.Elems) == 0 {
			return styleDim.Render("[]")
		}
		parts := make([]string, len(v.Elems))
		for i, e := range v.Elems {
			parts[i] = prettyValue(e)
		}
		return styleJSON.Render("[") + strings.Join(parts, styleJSON.Render(", ")) + styleJSON.Render("]")
	case parser.KindTable:
		pairs := make([]string, 0, len(v.Fields))
		for k, fv := range v.Fields {
			pairs = append(pairs, styleKey.Render(k)+styleEquals.Render(" = ")+prettyValue(fv))
		}
		sort.Strings(pairs)
		return styleJSON.Render("{ ") + strings.Join(pairs, styleJSON.Render(", ")) + styleJSON.Render(" }")
	}
	return ""
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toEnvKey(section, key string) string {
	full := key
	if section != "" {
		full = section + "_" + key
	}
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
		b, _ := json.Marshal(tableToMap(v))
		return string(b)
	}
	return ""
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	safe := true
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '"' || r == '\'' || r == '\\' || r == '$' || r == '`' || r == '\n' {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func docToJSONMap(doc *parser.Document) any {
	result := map[string]any{}
	for _, sec := range doc.Sections {
		pairs := map[string]any{}
		for _, kv := range sec.KVs {
			pairs[kv.Key] = valueToAny(kv.Value)
		}
		if sec.Name == "" {
			for k, v := range pairs {
				result[k] = v
			}
		} else {
			if sec.Emoji != "" {
				entry := map[string]any{"_emoji": sec.Emoji}
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

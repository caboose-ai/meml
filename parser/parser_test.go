package parser

import (
	"testing"
)

func TestParseEmpty(t *testing.T) {
	doc, err := Parse("")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sections) != 1 {
		t.Fatalf("expected 1 section (root), got %d", len(doc.Sections))
	}
}

func TestParseComments(t *testing.T) {
	src := `
# hash comment
💬 emoji comment
key = "value"
`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := doc.Get("", "key")
	if !ok {
		t.Fatal("key not found")
	}
	if v.Str != "value" {
		t.Fatalf("expected 'value', got %q", v.Str)
	}
}

func TestParseScalarValues(t *testing.T) {
	src := `
str     = "hello world"
bare    = hello
integer = 42
neg     = -7
flt     = 3.14
yes     = true
no      = false
empty   = null
tilde   = ~
`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		key  string
		kind ValueKind
	}{
		{"str", KindString},
		{"bare", KindString},
		{"integer", KindInt},
		{"neg", KindInt},
		{"flt", KindFloat},
		{"yes", KindBool},
		{"no", KindBool},
		{"empty", KindNull},
		{"tilde", KindNull},
	}

	for _, c := range cases {
		v, ok := doc.Get("", c.key)
		if !ok {
			t.Errorf("key %q not found", c.key)
			continue
		}
		if v.Kind != c.kind {
			t.Errorf("key %q: expected kind %v, got %v", c.key, c.kind, v.Kind)
		}
	}

	if v, _ := doc.Get("", "integer"); v.Int != 42 {
		t.Errorf("integer: expected 42, got %d", v.Int)
	}
	if v, _ := doc.Get("", "yes"); !v.Bool {
		t.Error("yes: expected true")
	}
	if v, _ := doc.Get("", "no"); v.Bool {
		t.Error("no: expected false")
	}
}

func TestParseEmojiBooleans(t *testing.T) {
	src := `
enabled = ✅
disabled = ❌
`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	v, _ := doc.Get("", "enabled")
	if v.Kind != KindBool || !v.Bool {
		t.Error("enabled: expected KindBool true")
	}
	v, _ = doc.Get("", "disabled")
	if v.Kind != KindBool || v.Bool {
		t.Error("disabled: expected KindBool false")
	}
}

func TestParseEmojiValue(t *testing.T) {
	src := `status = 🟢`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := doc.Get("", "status")
	if !ok {
		t.Fatal("status not found")
	}
	if v.Kind != KindEmoji {
		t.Fatalf("expected KindEmoji, got %v", v.Kind)
	}
	if v.Str != "🟢" {
		t.Fatalf("expected 🟢, got %q", v.Str)
	}
}

func TestParseEmojiAnnotation(t *testing.T) {
	src := `🔑 token = "xoxb-abc123"`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := doc.Get("", "token")
	if !ok {
		t.Fatal("token not found")
	}
	if v.Str != "xoxb-abc123" {
		t.Fatalf("expected 'xoxb-abc123', got %q", v.Str)
	}

	for _, kv := range doc.Sections[0].KVs {
		if kv.Key == "token" && kv.Annotation != "🔑" {
			t.Fatalf("expected annotation 🔑, got %q", kv.Annotation)
		}
	}
}

func TestParsePureEmojiKey(t *testing.T) {
	src := `🏠 = "/home/caboose"`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := doc.Get("", "🏠")
	if !ok {
		t.Fatal("emoji key not found")
	}
	if v.Str != "/home/caboose" {
		t.Fatalf("expected '/home/caboose', got %q", v.Str)
	}
}

func TestParseSections(t *testing.T) {
	src := `
[server]
port = 8080

[🔧 database]
url = "postgres://localhost/mydb"

[🔑]
api_key = "secret"
`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	if v, ok := doc.Get("server", "port"); !ok || v.Int != 8080 {
		t.Error("server.port not found or wrong value")
	}

	if v, ok := doc.Get("database", "url"); !ok || v.Str != "postgres://localhost/mydb" {
		t.Error("database.url not found or wrong value")
	}

	// Pure emoji section: [🔑] uses emoji as name
	if v, ok := doc.Get("🔑", "api_key"); !ok || v.Str != "secret" {
		t.Error("🔑.api_key not found or wrong value")
	}
}

func TestParseSectionEmoji(t *testing.T) {
	src := `
[🔧 server]
host = "localhost"
`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var sec *Section
	for _, s := range doc.Sections {
		if s.Name == "server" {
			sec = s
			break
		}
	}
	if sec == nil {
		t.Fatal("section 'server' not found")
	}
	if sec.Emoji != "🔧" {
		t.Fatalf("expected emoji 🔧, got %q", sec.Emoji)
	}
}

func TestParseArray(t *testing.T) {
	src := `tags = [web, api, v2]`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := doc.Get("", "tags")
	if !ok {
		t.Fatal("tags not found")
	}
	if v.Kind != KindArray {
		t.Fatalf("expected array, got %v", v.Kind)
	}
	if len(v.Elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(v.Elems))
	}
	if v.Elems[0].Str != "web" {
		t.Errorf("expected 'web', got %q", v.Elems[0].Str)
	}
}

func TestParseArrayMixedTypes(t *testing.T) {
	src := `things = ["hello", 42, ✅, 🟢]`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	v, _ := doc.Get("", "things")
	if len(v.Elems) != 4 {
		t.Fatalf("expected 4 elements, got %d", len(v.Elems))
	}
	if v.Elems[1].Int != 42 {
		t.Errorf("expected 42, got %d", v.Elems[1].Int)
	}
	if v.Elems[2].Kind != KindBool || !v.Elems[2].Bool {
		t.Error("expected true")
	}
	if v.Elems[3].Kind != KindEmoji {
		t.Error("expected emoji")
	}
}

func TestParseInlineTable(t *testing.T) {
	src := `db = { host = "localhost", port = 5432 }`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := doc.Get("", "db")
	if !ok {
		t.Fatal("db not found")
	}
	if v.Kind != KindTable {
		t.Fatalf("expected table, got %v", v.Kind)
	}
	if v.Fields["host"].Str != "localhost" {
		t.Errorf("expected 'localhost', got %q", v.Fields["host"].Str)
	}
	if v.Fields["port"].Int != 5432 {
		t.Errorf("expected 5432, got %d", v.Fields["port"].Int)
	}
}

func TestParseQuotedKey(t *testing.T) {
	src := `"my-weird key" = "value"`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := doc.Get("", "my-weird key")
	if !ok {
		t.Fatal("quoted key not found")
	}
	if v.Str != "value" {
		t.Fatalf("expected 'value', got %q", v.Str)
	}
}

func TestParseTripleQuotedString(t *testing.T) {
	src := `bio = """hello "world" it's fine"""`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := doc.Get("", "bio")
	if !ok {
		t.Fatal("bio not found")
	}
	if v.Str != `hello "world" it's fine` {
		t.Fatalf("got %q", v.Str)
	}
}

func TestParseFlat(t *testing.T) {
	src := `
key = "root"

[section]
key = "nested"
`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	flat := doc.Flat()
	if flat["key"].Str != "root" {
		t.Error("expected root key")
	}
	if flat["section.key"].Str != "nested" {
		t.Error("expected section.key")
	}
}

func TestParseErrors(t *testing.T) {
	cases := []string{
		`[unclosed`,
		`= no key`,
		`key "no equals"`,
	}
	for _, src := range cases {
		_, err := Parse(src)
		if err == nil {
			t.Errorf("expected error for input %q", src)
		}
	}
}

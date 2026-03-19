package parser

import "fmt"

// ValueKind identifies the type of a parsed value.
type ValueKind int

const (
	KindString ValueKind = iota
	KindInt
	KindFloat
	KindBool
	KindEmoji // emoji atom used as a value (e.g. status = 🟢)
	KindNull
	KindArray
	KindTable
)

func (k ValueKind) String() string {
	switch k {
	case KindString:
		return "string"
	case KindInt:
		return "int"
	case KindFloat:
		return "float"
	case KindBool:
		return "bool"
	case KindEmoji:
		return "emoji"
	case KindNull:
		return "null"
	case KindArray:
		return "array"
	case KindTable:
		return "table"
	default:
		return "unknown"
	}
}

// Value holds a parsed EML value.
type Value struct {
	Kind   ValueKind
	Str    string            // KindString, KindEmoji
	Int    int64             // KindInt
	Float  float64           // KindFloat
	Bool   bool              // KindBool
	Elems  []*Value          // KindArray
	Fields map[string]*Value // KindTable
}

func (v *Value) String() string {
	switch v.Kind {
	case KindString:
		return v.Str
	case KindInt:
		return fmt.Sprintf("%d", v.Int)
	case KindFloat:
		return fmt.Sprintf("%g", v.Float)
	case KindBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case KindEmoji:
		return v.Str
	case KindNull:
		return ""
	default:
		return ""
	}
}

// KeyValue is a key-value pair inside a section.
type KeyValue struct {
	Annotation string // leading emoji annotation, e.g. "🔑" (may be empty)
	Key        string // key name — identifier, quoted string, or pure emoji
	Value      *Value
	Line       int
}

// Section is a named group of key-value pairs.
type Section struct {
	Emoji string // decorative emoji on the section header, e.g. "🔧" (may be empty)
	Name  string // section name; "" = root/default section
	KVs   []*KeyValue
	Line  int
}

// Document is the top-level parsed result.
type Document struct {
	Sections []*Section
}

// Get returns the value for key in the named section.
// Pass "" for section to look in the root section.
func (d *Document) Get(section, key string) (*Value, bool) {
	for _, s := range d.Sections {
		if s.Name == section {
			for _, kv := range s.KVs {
				if kv.Key == key {
					return kv.Value, true
				}
			}
		}
	}
	return nil, false
}

// Flat returns all key-value pairs as a flat map.
// Keys from named sections are prefixed: "section.key".
// Root section keys have no prefix.
func (d *Document) Flat() map[string]*Value {
	result := make(map[string]*Value)
	for _, s := range d.Sections {
		for _, kv := range s.KVs {
			k := kv.Key
			if s.Name != "" {
				k = s.Name + "." + k
			}
			result[k] = kv.Value
		}
	}
	return result
}

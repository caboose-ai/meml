# EML — Emoji Markup Language Specification

**Version:** 0.1
**File extension:** `.eml`
**Encoding:** UTF-8 (required)
**Line endings:** LF or CRLF

---

## Overview

EML is a configuration language inspired by TOML with first-class emoji support.
Emoji can appear as:

- **Section decorators** — `[🔧 server]`
- **Key annotations** — `🔑 token = "..."` (semantic metadata; doesn't change the value)
- **Pure emoji keys** — `🏠 = "/home/user"`
- **Value atoms** — `status = 🟢`
- **Boolean shorthands** — `enabled = ✅` / `debug = ❌`
- **Comment markers** — `💬 this line is a comment`

---

## Comments

```eml
# hash comment — ignored to end of line
💬 emoji comment — any line whose first token is 💬
```

Inline comments are not supported. Use a `#` on a separate line before the key.

---

## Sections

Sections group key-value pairs, like TOML tables.

```eml
[name]           # plain section
[🔧 name]        # section with emoji decoration (stored as metadata)
[🔑]             # pure emoji section — emoji becomes the section name
["my section"]   # quoted section name (allows spaces/special chars)
```

Keys declared before any section header belong to the **root section** (name `""`).

---

## Keys

```eml
key = value              # plain identifier: [a-zA-Z_][a-zA-Z0-9_.-]*
🔑 key = value           # emoji annotation + identifier key
🏠 = value               # pure emoji key
"my key" = value         # quoted key (allows spaces and special chars)
'my key' = value         # single-quoted key
```

Emoji annotations carry semantic meaning for tooling (e.g. mark secrets, paths, URLs)
but do not change the value type or the key name.

### Recommended annotation conventions

| Emoji | Meaning         |
|-------|-----------------|
| 🔑    | Secret / sensitive |
| 📁    | File path       |
| 🌍    | URL / endpoint  |
| 🔢    | Explicit integer hint |
| 📋    | List / array    |
| ⚠️    | Deprecated key  |
| 💬    | Comment         |

These are conventions only — the parser stores annotations but does not enforce them.

---

## Values

### Strings

```eml
name    = "hello world"      # double-quoted; supports escape sequences
name    = 'literal string'   # single-quoted; no escape processing
name    = bareword           # bare word (no spaces or structural chars)
bio     = """multi "line" 's fine"""   # triple-quoted; no escape needed
```

Supported escape sequences in double-quoted strings: `\n \t \r \\ \"  \'`

### Numbers

```eml
port  = 8080       # integer
count = -5         # negative integer
ratio = 3.14       # float
temp  = -0.5       # negative float
```

### Booleans

```eml
enabled  = true    # keyword
debug    = false   # keyword
active   = ✅      # emoji shorthand for true
inactive = ❌      # emoji shorthand for false
```

### Null

```eml
value = null   # keyword
value = ~      # YAML-style shorthand
```

### Emoji atoms

Any emoji that is not `✅` or `❌` is treated as a string value tagged with
kind `emoji`. This allows semantic enum-like values:

```eml
status = 🟢   # green / ok
level  = 🔴   # red / critical
mood   = 😎
```

### Arrays

```eml
tags  = [web, api, v2]
temps = [55, 60, 65]
mixed = ["hello", 42, ✅, 🟢]   # mixed types allowed
```

Trailing commas are allowed. Arrays must be on a single line.

### Inline tables

```eml
db = { host = "localhost", port = 5432 }
```

Inline tables must be on a single line. Nested inline tables are allowed.

---

## Full example

```eml
# caboose-mcp.eml

claude_dir = ~/.claude

[🔧 server]
host   = "0.0.0.0"
port   = 8080
debug  = false
status = 🟢

[🔑 slack]
🔑 token     = "xoxb-..."
🔑 app_token = "xapp-..."
channels     = [C1234, C5678]

[💾 database]
🌍 postgres_url = "postgres://user:pass@localhost:5432/db"
🌍 mongo_url    = "mongodb://localhost:27017"
```

---

## Grammar (EBNF)

```ebnf
document    = { line } ;
line        = blank | comment | section | keyvalue ;
blank       = { whitespace } newline ;
comment     = ( "#" | "💬" ) { any_char } newline ;
section     = "[" [ emoji ] [ identifier | quoted_string ] "]" newline ;
keyvalue    = [ emoji ] ( emoji | identifier | quoted_string ) "=" value newline ;

value       = string | number | bool | null | emoji_atom | array | inline_table ;
string      = quoted_string | triple_quoted | bare_word ;
quoted_string = '"' { char | escape } '"' | "'" { char } "'" ;
triple_quoted = '"""' { any_char } '"""' | "'''" { any_char } "'''" ;
number      = [ "-" ] digit { digit } [ "." digit { digit } ] ;
bool        = "true" | "false" | "✅" | "❌" ;
null        = "null" | "~" ;
emoji_atom  = emoji_char ;   (* not ✅ or ❌ — those are bool *)
array       = "[" [ value { "," value } [ "," ] ] "]" ;
inline_table = "{" [ kv_pair { "," kv_pair } ] "}" ;
kv_pair     = ( identifier | quoted_string | emoji ) "=" value ;
```

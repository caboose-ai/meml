# eml

**EML — Emoji Markup Language**

A configuration language like TOML, with first-class emoji support.
Emoji can be section decorators, key annotations, pure keys, or value atoms.

---

## Syntax at a glance

```eml
# hash comment
💬 emoji comment

claude_dir = ~/.claude          # bare word value (no quotes needed)

[🔧 server]                     # section with emoji decoration
host   = "0.0.0.0"
port   = 8080
debug  = false
status = 🟢                     # emoji atom value

[🔑 slack]
🔑 token     = "xoxb-..."       # emoji annotation marks it as a secret
🔑 app_token = "xapp-..."
channels     = [general, ops]   # array

[💾 database]
🌍 postgres_url = "postgres://user:pass@localhost/db"   # URL annotation
🌍 mongo_url    = "mongodb://localhost:27017"

[🖨️ bambu]
ip          = "192.168.1.100"
🔑 access_code = "ABCD1234"
bed_temp    = 55
nozzle_temp = 220

[⚡ n8n]
🌍 webhook_url = "http://localhost:5678/webhook/events"
🔑 api_key     = ""
```

---

## Emoji roles

| Position | Example | Meaning |
|---|---|---|
| Section decorator | `[🔧 server]` | Visual label; stored as `section.Emoji` |
| Section name | `[🔑]` | Emoji *is* the section name |
| Key annotation | `🔑 token = "..."` | Semantic tag on the key; stored as `kv.Annotation` |
| Pure emoji key | `🏠 = "/home"` | Emoji is the key name |
| Value atom | `status = 🟢` | Stored as `KindEmoji` string |
| Boolean shorthand | `enabled = ✅` | `true`; `❌` = `false` |
| Comment | `💬 a comment` | Line is ignored |

### Recommended annotation conventions

| Emoji | Meaning |
|-------|---------|
| 🔑 | Secret / sensitive value |
| 📁 | File path |
| 🌍 | URL / endpoint |
| 📋 | List / array |
| ⚠️ | Deprecated key |

These are conventions — the parser stores annotations but does not enforce them.

---

## Value types

| EML | Kind | Example |
|-----|------|---------|
| Double-quoted string | `KindString` | `"hello world"` |
| Single-quoted string | `KindString` | `'no\escape'` |
| Triple-quoted string | `KindString` | `"""has "quotes" inside"""` |
| Bare word | `KindString` | `hello` |
| Integer | `KindInt` | `8080`, `-5` |
| Float | `KindFloat` | `3.14`, `-0.5` |
| Boolean | `KindBool` | `true`, `false`, `✅`, `❌` |
| Null | `KindNull` | `null`, `~` |
| Emoji atom | `KindEmoji` | `🟢`, `🔴`, `😎` |
| Array | `KindArray` | `[web, api, 42, ✅]` |
| Inline table | `KindTable` | `{host = "localhost", port = 5432}` |

---

## CLI

### Install

```sh
go install github.com/caboose-mcp/eml/cmd/eml@latest
```

### validate — syntax check

```sh
eml validate config.eml
# ✓ config.eml  (3 sections, 14 keys)
# exits non-zero on errors with line numbers
```

### dump — JSON output (with syntax highlighting)

```sh
eml dump config.eml
```

```json
{
  "claude_dir": "~/.claude",
  "server": {
    "_emoji": "🔧",
    "debug": false,
    "host": "0.0.0.0",
    "port": 8080,
    "status": "🟢"
  },
  "slack": {
    "_emoji": "🔑",
    "app_token": "xapp-...",
    "channels": ["general", "ops"],
    "token": "xoxb-..."
  }
}
```

### pretty — colorized EML view (with typewriter animation)

```sh
eml pretty config.eml
```

Renders the parsed document back as colorized EML — emoji annotations and
section decorations are highlighted, values are colored by type.

### env — shell exports

```sh
eml env config.eml
```

```sh
BAMBU_ACCESS_CODE='ABCD1234'
BAMBU_BED_TEMP=55
BAMBU_IP=192.168.1.100
DATABASE_MONGO_URL=mongodb://localhost:27017
DATABASE_POSTGRES_URL=postgres://user:pass@localhost/db
SERVER_DEBUG=false
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SLACK_CHANNELS=general,ops
SLACK_TOKEN=xoxb-...
```

Feed to caboose-mcp:

```sh
export $(eml env caboose-mcp.eml | xargs)
./caboose-mcp
```

---

## Go library

```go
import "github.com/caboose-mcp/eml/parser"

doc, err := parser.Parse(src)
if err != nil {
    log.Fatal(err)
}

// Look up by section + key
v, ok := doc.Get("server", "port")
fmt.Println(v.Kind)  // KindInt
fmt.Println(v.Int)   // 8080

// Root section
v, _ = doc.Get("", "claude_dir")
fmt.Println(v.Str)   // ~/.claude

// Flat map: section.key -> *Value
flat := doc.Flat()
fmt.Println(flat["server.host"].Str)   // 0.0.0.0
fmt.Println(flat["slack.token"].Str)   // xoxb-...

// Inspect annotations (e.g. find all secrets)
for _, sec := range doc.Sections {
    for _, kv := range sec.KVs {
        if kv.Annotation == "🔑" {
            fmt.Printf("secret: %s.%s\n", sec.Name, kv.Key)
        }
    }
}
// secret: slack.token
// secret: slack.app_token
// secret: bambu.access_code

// Check section emoji decoration
for _, sec := range doc.Sections {
    if sec.Emoji != "" {
        fmt.Printf("[%s %s] — %d keys\n", sec.Emoji, sec.Name, len(sec.KVs))
    }
}
// [🔧 server] — 4 keys
// [🔑 slack] — 3 keys
// [💾 database] — 2 keys
```

---

## Run tests

```sh
go test ./...
```

---

## See also

- [SPEC.md](SPEC.md) — full language grammar and specification
- [example.eml](example.eml) — caboose-mcp config example

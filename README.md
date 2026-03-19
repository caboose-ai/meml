# eml

**EML — Emoji Markup Language**

A configuration language like TOML, with first-class emoji support.

```eml
[🔧 server]
host   = "0.0.0.0"
port   = 8080
status = 🟢

[🔑 slack]
🔑 token = "xoxb-..."
channels = [general, random]

[💾 database]
🌍 url = "postgres://localhost/mydb"
```

Emoji can be decorators on sections (`[🔧 server]`), annotations on keys (`🔑 token`),
pure keys (`🏠 = "/home"`), or values (`status = 🟢`). Booleans have shorthands: `✅` / `❌`.

See [SPEC.md](SPEC.md) for the full language specification.

---

## Install

```sh
go install github.com/caboose-mcp/eml/cmd/eml@latest
```

## CLI

```sh
eml validate config.eml      # syntax check; exits 0 if valid
eml dump config.eml          # print as JSON
eml env config.eml           # print KEY=VALUE exports
```

The `env` command is useful for feeding EML configs to programs that read environment
variables (like caboose-mcp):

```sh
export $(eml env caboose-mcp.eml | xargs)
```

## Parser library

```go
import "github.com/caboose-mcp/eml/parser"

doc, err := parser.Parse(src)

// Look up a value
v, ok := doc.Get("server", "port")  // section, key
fmt.Println(v.Int)                  // 8080

// Flat map of all keys
flat := doc.Flat()
// flat["server.port"].Int == 8080
// flat["slack.token"].Str == "xoxb-..."

// Emoji annotations are preserved
for _, kv := range doc.Sections[1].KVs {
    if kv.Annotation == "🔑" {
        fmt.Println("secret:", kv.Key)
    }
}
```

## Value types

| EML                | Go kind       |
|--------------------|---------------|
| `"hello"`          | KindString    |
| `42`               | KindInt       |
| `3.14`             | KindFloat     |
| `true` / `✅`      | KindBool      |
| `null` / `~`       | KindNull      |
| `🟢`               | KindEmoji     |
| `[1, 2, 3]`        | KindArray     |
| `{key = val}`      | KindTable     |

## Run tests

```sh
go test ./...
```

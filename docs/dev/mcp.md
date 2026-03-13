# MCP Server

Model Context Protocol (MCP) сервер для интеграции заметок с AI-агентами (Claude, Cursor, etc).

## Endpoint

```
POST /_system/mcp
Content-Type: application/json
```

JSON-RPC 2.0 протокол.

## Tools

### Базовые

| Tool | Описание | Параметры |
|------|----------|-----------|
| `search` | Hybrid search по заметкам | `query: string` |
| `similar` | Похожие заметки (vector search) | `text: string`, `limit?: number` |

### Динамические методы

Заметки с `mcp_method` в frontmatter становятся callable tools.

## Frontmatter формат

```yaml
---
free: true                              # обязательно для публичного доступа
mcp_method: code-review                 # имя метода (tool name)
mcp_description: "Детальный code review"  # описание для tools/list
---

Контент заметки - это то, что вернётся при вызове метода.
Может содержать любой текст, шаблоны, инструкции.
```

## Примеры использования

### tools/list

Request:
```json
{
  "jsonrpc": "2.0",
  "method": "tools/list",
  "id": 1
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "search",
        "description": "Search notes by query",
        "inputSchema": {
          "type": "object",
          "properties": {
            "query": {"type": "string"}
          },
          "required": ["query"]
        }
      },
      {
        "name": "similar",
        "description": "Find similar notes by text",
        "inputSchema": {
          "type": "object",
          "properties": {
            "text": {"type": "string"},
            "limit": {"type": "number"}
          },
          "required": ["text"]
        }
      },
      {
        "name": "code-review",
        "description": "Детальный code review"
      }
    ]
  }
}
```

### tools/call search

Request:
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "search",
    "arguments": {
      "query": "golang error handling"
    }
  },
  "id": 2
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Found 3 notes:\n\n1. Error Handling in Go\n   /programming/go-errors\n\n2. ..."
      }
    ]
  }
}
```

### tools/call {method}

Request:
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "code-review",
    "arguments": {}
  },
  "id": 3
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Контент заметки code-review..."
      }
    ]
  }
}
```

## Фильтрация заметок

- **search/similar**: все заметки с `free: true`
- **dynamic methods**: заметки с `free: true` И `mcp_method` заданным

## План имплементации

### Story 1: Поля в NoteView

Добавить в `internal/model/note.go`:

```go
type NoteView struct {
    // ... existing fields ...

    MCPMethod      string // from frontmatter mcp_method
    MCPDescription string // from frontmatter mcp_description
}
```

Добавить extraction в `ExtractMetaData()`:

```go
func (n *NoteView) extractMCPFields() {
    if method, ok := n.RawMeta["mcp_method"].(string); ok {
        n.MCPMethod = method
    }
    if desc, ok := n.RawMeta["mcp_description"].(string); ok {
        n.MCPDescription = desc
    }
}
```

### Story 2: MCP Endpoint

Создать `internal/case/mcp/`:

```
internal/case/mcp/
├── endpoint.go      # HTTP handler, JSON-RPC routing
├── resolve.go       # main logic
├── tools.go         # tools/list implementation
├── search.go        # search tool
├── similar.go       # similar tool
└── method.go        # dynamic method dispatch
```

endpoint.go:
```go
package mcp

type Endpoint struct{}

func (*Endpoint) Path() string {
    return "/_system/mcp"
}

func (*Endpoint) Method() string {
    return http.MethodPost
}
```

### Story 3: tools/list

```go
func (h *Handler) ListTools(env Env) ListToolsResult {
    tools := []Tool{
        {Name: "search", Description: "Search notes", InputSchema: searchSchema},
        {Name: "similar", Description: "Find similar notes", InputSchema: similarSchema},
    }

    // Add dynamic methods from notes
    for _, note := range env.LiveNoteViews().List {
        if note.Free && note.MCPMethod != "" {
            tools = append(tools, Tool{
                Name:        note.MCPMethod,
                Description: note.MCPDescription,
            })
        }
    }

    return ListToolsResult{Tools: tools}
}
```

### Story 4: tools/call search

Переиспользовать `internal/case/searchnotes/`:

```go
func (h *Handler) CallSearch(env Env, query string) CallToolResult {
    results := searchnotes.Resolve(ctx, env, searchnotes.Input{Query: query})

    // Filter to free notes only
    // Format as text response

    return CallToolResult{Content: [...]}
}
```

### Story 5: tools/call similar

Переиспользовать существующий vector search:

```go
func (h *Handler) CallSimilar(env Env, text string, limit int) CallToolResult {
    // Get embedding for text
    // Find similar notes by vector
    // Filter to free notes
    // Format response
}
```

### Story 6: tools/call {method}

```go
func (h *Handler) CallMethod(env Env, methodName string) CallToolResult {
    for _, note := range env.LiveNoteViews().List {
        if note.Free && note.MCPMethod == methodName {
            return CallToolResult{
                Content: []Content{{Type: "text", Text: string(note.Content)}},
            }
        }
    }

    return ErrorResult{Code: -32601, Message: "Method not found"}
}
```

## Фаза 2 (будущее)

Marketplace для промптов:
- Платные методы (`price: 0.01` в frontmatter)
- Авторизация пользователей
- Трекинг usage для billing
- Discovery: `search_marketplace` tool
- Per-author endpoints: `/_system/mcp/{username}`

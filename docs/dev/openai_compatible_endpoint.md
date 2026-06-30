# OpenAI-compatible endpoint

The fleet and agentruntime use the OpenAI chat completions API contract — any server that implements `/v1/chat/completions` with function-calling works as a drop-in backend.

## What "OpenAI-compatible" means

The fleet sends `POST /v1/chat/completions` with:
- `model` — the model name string (passed through from the role or `--default-model`)
- `messages` — `[{role, content}, ...]` including system prompt, user turn, assistant tool-calls, and tool results
- `tools` — JSON schema array (OpenAI function-calling format)
- `tool_choice` — `"auto"` (let the model decide)
- `max_tokens` — the per-run cap

It expects a response with `choices[0].message.content`, `choices[0].message.tool_calls[]`, and `usage.prompt_tokens` / `usage.completion_tokens`.

## Pointing the fleet at a backend

| Backend | `--llm-base-url` |
|---------|-----------------|
| OpenAI (default) | *(omit; SDK default)* |
| OpenRouter | `https://openrouter.ai/api/v1` |
| Ollama | `http://localhost:11434/v1` |
| Any vendor | their `/v1` base URL |
| E2E llm-mock | `http://localhost:<mock-port>/v1` |

## Fleet flags

```
--llm-base-url   TRIP2G_LLM_BASE_URL   OpenAI-compatible base URL
--llm-api-key    TRIP2G_LLM_API_KEY    API key (falls back to OPENAI_API_KEY)
--default-model  TRIP2G_DEFAULT_MODEL  model name when a role omits model:
```

The fleet's `agentruntime.NewOpenAILLM(apiKey, baseURL)` wraps the official `go-openai` client and sets a custom `BaseURL` when non-empty.

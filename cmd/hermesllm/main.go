// Command hermesllm is a standalone OpenAI-compatible chat-completions service
// that fronts a Hermes agent, so cmd/fleet can drive Hermes as its LLM.
//
// Hermes ignores the OpenAI `tools` array and answers with final text only;
// fleet drives its loop off tool_calls. hermesllm bridges the two by advertising
// fleet's tools in a synthetic system preamble and translating Hermes' JSON
// answer back into real tool_calls (see internal/hermesllm).
//
// Auth: hermesllm's own OpenAI-standard api_key (HERMESLLM_API_KEY,
// Authorization: Bearer) gates /v1/chat/completions. An empty key disables the
// check — unlike codellm there is no second lane to fall through to.
package main

import (
	"log"
	"net/http"
	"time"

	"trip2g/cmd/hermesllm/appconfig"
	"trip2g/cmd/hermesllm/internal/hermesllm"
)

func main() {
	cfg, err := appconfig.Get()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: hermesllm.New(hermesllm.Config{
			HermesURL: cfg.HermesURL,
			HermesKey: cfg.HermesKey,
			Model:     cfg.Model,
			Timeout:   cfg.Timeout,
			Auth:      hermesllm.APIKeyAuth(cfg.APIKey),
		}).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("hermesllm listening on %s (hermes=%s, model=%s, key_auth=%t)", cfg.Addr, cfg.HermesURL, cfg.Model, cfg.APIKey != "")
	if err = srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

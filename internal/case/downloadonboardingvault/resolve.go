package downloadonboardingvault

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/ptr"
)

type Env interface {
	GenerateAPIKey() string
	InsertAPIKey(ctx context.Context, params db.InsertAPIKeyParams) (db.ApiKey, error)
	SetApiKeyMcpAdminTools(ctx context.Context, arg db.SetApiKeyMcpAdminToolsParams) error
	LatestNoteViews() *model.NoteViews
	PublicURL() string
	// OnboardingVaultZip returns the vault template archive, empty when the
	// vault was not built into this binary.
	OnboardingVaultZip() []byte
}

const oldPrefix = "onboarding-vault/"
const dataJSONPath = oldPrefix + ".obsidian/plugins/trip2g/data.json"
const indexMDPath = oldPrefix + "_index.md"
const mcpJSONPath = oldPrefix + ".mcp.json"
const codexJSONPath = oldPrefix + "codex.json"
const antigravityJSONPath = oldPrefix + "antigravity-mcp-config.json"
const agentsMDPath = oldPrefix + "AGENTS.md"

type pluginData struct {
	SyncDirs             []syncDir `json:"syncDirs"`
	SkipPushConfirmation bool      `json:"skipPushConfirmation"`
}

type syncDir struct {
	Path       string `json:"path"`
	APIKey     string `json:"apiKey"`
	APIURL     string `json:"apiUrl"`
	TwoWaySync bool   `json:"twoWaySync"`
}

type mcpServer struct {
	Type    string            `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type mcpConfig struct {
	MCPServers map[string]mcpServer `json:"mcpServers"`
}

func generateMCPJSON(apiKey, publicURL string) ([]byte, error) {
	cfg := mcpConfig{
		MCPServers: map[string]mcpServer{
			"my-trip2g-instance": {
				Type: "http",
				URL:  publicURL + "/_system/mcp",
				Headers: map[string]string{
					"Authorization": "Bearer " + apiKey,
				},
			},
			"trip2g-docs-public-hub": {
				Type: "http",
				URL:  "https://trip2g.com/_system/mcp",
			},
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// antigravityServer uses serverUrl (not url) per Antigravity's MCP config format.
type antigravityServer struct {
	ServerURL string            `json:"serverUrl"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type antigravityConfig struct {
	MCPServers map[string]antigravityServer `json:"mcpServers"`
}

// generateAntigravityJSON produces a template for ~/.gemini/antigravity/mcp_config.json.
func generateAntigravityJSON(apiKey, publicURL string) ([]byte, error) {
	cfg := antigravityConfig{
		MCPServers: map[string]antigravityServer{
			"my-trip2g-instance": {
				ServerURL: publicURL + "/_system/mcp",
				Headers: map[string]string{
					"Authorization": "Bearer " + apiKey,
				},
			},
			"trip2g-docs-public-hub": {
				ServerURL: "https://trip2g.com/_system/mcp",
			},
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// maxVaultNameLen caps the vault name so it stays a usable filename.
const maxVaultNameLen = 64

// vaultNamePattern is an allowlist: the name becomes both a filename and a
// path inside the archive, so anything that could escape either is rejected
// rather than sanitised. Leading dot/dash keeps the name out of hidden-file
// and option-flag territory.
var vaultNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func validateVaultName(name string) error {
	if name == "" {
		return &appreq.Error{Code: http.StatusBadRequest, Message: "name must not be empty"}
	}

	if len(name) > maxVaultNameLen {
		return &appreq.Error{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("name must be at most %d characters", maxVaultNameLen),
		}
	}

	if !vaultNamePattern.MatchString(name) {
		return &appreq.Error{
			Code:    http.StatusBadRequest,
			Message: "name must start with a letter or digit and contain only letters, digits, dot, dash or underscore",
		}
	}

	return nil
}

// Resolve builds the vault ZIP rooted at vaultName, which must already be
// validated by validateVaultName.
func Resolve(ctx context.Context, env Env, userID int, enableAdminGraphQL bool, vaultName string) ([]byte, error) {
	err := validateVaultName(vaultName)
	if err != nil {
		return nil, err
	}

	// Generate new API key
	apiKey := env.GenerateAPIKey()

	// Hash the API key before storing
	hash := sha256.Sum256([]byte(apiKey))
	hashedValue := hex.EncodeToString(hash[:])

	params := db.InsertAPIKeyParams{
		Value:       hashedValue,
		CreatedBy:   int64(userID),
		Description: "Onboarding vault",
	}

	inserted, err := env.InsertAPIKey(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	// Opt the key into MCP admin tools (admin GraphQL) when requested, so the
	// vault's agent can run admin mutations without a separate manual toggle.
	if enableAdminGraphQL {
		err = env.SetApiKeyMcpAdminTools(ctx, db.SetApiKeyMcpAdminToolsParams{
			ID:      inserted.ID,
			Enabled: ptr.To(true),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to enable admin graphql on api key: %w", err)
		}
	}

	// Create new plugin data with real credentials
	newData := pluginData{
		SyncDirs: []syncDir{
			{
				Path:       "/",
				APIKey:     apiKey,
				APIURL:     env.PublicURL(),
				TwoWaySync: true,
			},
		},
		SkipPushConfirmation: false,
	}

	newDataJSON, err := json.MarshalIndent(newData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plugin data: %w", err)
	}

	publicURL := env.PublicURL()

	mcpJSON, err := generateMCPJSON(apiKey, publicURL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate mcp config: %w", err)
	}

	antigravityJSON, err := generateAntigravityJSON(apiKey, publicURL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate antigravity mcp config: %w", err)
	}

	// Prepare file replacements (also used for new files not in the original zip).
	replacements := map[string][]byte{
		dataJSONPath:        newDataJSON,
		mcpJSONPath:         mcpJSON,
		codexJSONPath:       mcpJSON,
		antigravityJSONPath: antigravityJSON,
	}

	// Check if /_index note exists, use its content instead of template.
	notes := env.LatestNoteViews()
	if notes != nil {
		indexNote := notes.PathMap["_index.md"]
		if indexNote != nil && len(indexNote.Content) > 0 {
			replacements[indexMDPath] = indexNote.Content
		}
	}

	newPrefix := vaultName + "/"

	// Read embedded ZIP and modify files, replacing {{publicUrl}} placeholder.
	modifiedZip, err := modifyZipFiles(env.OnboardingVaultZip(), replacements, publicURL, newPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to modify zip: %w", err)
	}

	return modifiedZip, nil
}

// renamePath replaces the old folder prefix with the new one.
func renamePath(name, newPrefix string) string {
	if strings.HasPrefix(name, oldPrefix) {
		return newPrefix + name[len(oldPrefix):]
	}
	return name
}

// domainFromURL extracts the host without port from a URL.
func domainFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return "vault"
	}

	host := parsed.Host
	// Remove port if present.
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	if host == "" {
		return "vault"
	}

	return host
}

func modifyZipFiles(zipData []byte, replacements map[string][]byte, publicURL, newPrefix string) ([]byte, error) { //nolint:gocognit
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip: %w", err)
	}

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	written := make(map[string]bool, len(replacements))

	for _, file := range reader.File {
		outName := renamePath(file.Name, newPrefix)

		if newContent, ok := replacements[file.Name]; ok {
			written[file.Name] = true
			// Replace with new content.
			w, createErr := writer.Create(outName)
			if createErr != nil {
				return nil, fmt.Errorf("failed to create file in zip: %w", createErr)
			}

			_, writeErr := w.Write(newContent)
			if writeErr != nil {
				return nil, fmt.Errorf("failed to write new content: %w", writeErr)
			}

			continue
		}

		// Replace {{publicUrl}} placeholder in select markdown files.
		if file.Name == indexMDPath || file.Name == agentsMDPath {
			content, readErr := readZipFileContent(file)
			if readErr != nil {
				return nil, fmt.Errorf("failed to read %s: %w", file.Name, readErr)
			}

			content = bytes.ReplaceAll(content, []byte("{{publicUrl}}"), []byte(publicURL))

			w, createErr := writer.Create(outName)
			if createErr != nil {
				return nil, fmt.Errorf("failed to create file in zip: %w", createErr)
			}

			_, writeErr := w.Write(content)
			if writeErr != nil {
				return nil, fmt.Errorf("failed to write content: %w", writeErr)
			}

			continue
		}

		// Copy file as-is, with renamed path.
		err = copyZipFileRenamed(writer, file, outName)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file %s: %w", file.Name, err)
		}
	}

	// Inject files not present in the original zip.
	for name, content := range replacements {
		if written[name] {
			continue
		}
		outName := renamePath(name, newPrefix)
		w, createErr := writer.Create(outName)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create file %s in zip: %w", name, createErr)
		}
		_, writeErr := w.Write(content)
		if writeErr != nil {
			return nil, fmt.Errorf("failed to write file %s in zip: %w", name, writeErr)
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}

	return buf.Bytes(), nil
}

func readZipFileContent(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(io.LimitReader(rc, maxFileSize))
}

// maxFileSize is the maximum size of a single file in the ZIP (10MB).
const maxFileSize = 10 * 1024 * 1024

func copyZipFileRenamed(writer *zip.Writer, file *zip.File, name string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	header := file.FileHeader
	header.Name = name

	w, err := writer.CreateHeader(&header)
	if err != nil {
		return err
	}

	// Limit copy size to prevent decompression bomb attacks.
	_, err = io.Copy(w, io.LimitReader(rc, maxFileSize))

	return err
}

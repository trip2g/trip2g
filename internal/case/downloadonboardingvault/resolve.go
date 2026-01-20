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

	onboardingvault "trip2g/onboarding-vault"

	"trip2g/internal/db"
	"trip2g/internal/model"
)

type Env interface {
	GenerateAPIKey() string
	InsertAPIKey(ctx context.Context, params db.InsertAPIKeyParams) (db.ApiKey, error)
	LatestNoteViews() *model.NoteViews
	PublicURL() string
}

const dataJSONPath = "onboarding-vault/.obsidian/plugins/trip2g/data.json"
const indexMDPath = "onboarding-vault/_index.md"

type pluginData struct {
	SyncDirs             []syncDir `json:"syncDirs"`
	SkipPushConfirmation bool      `json:"skipPushConfirmation"`
}

type syncDir struct {
	Path   string `json:"path"`
	APIKey string `json:"apiKey"`
	APIURL string `json:"apiUrl"`
}

func Resolve(ctx context.Context, env Env, userID int) ([]byte, error) {
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

	_, err := env.InsertAPIKey(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	// Create new plugin data with real credentials
	newData := pluginData{
		SyncDirs: []syncDir{
			{
				Path:   "/",
				APIKey: apiKey,
				APIURL: env.PublicURL(),
			},
		},
		SkipPushConfirmation: false,
	}

	newDataJSON, err := json.MarshalIndent(newData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plugin data: %w", err)
	}

	publicURL := env.PublicURL()

	// Prepare file replacements
	replacements := map[string][]byte{
		dataJSONPath: newDataJSON,
	}

	// Check if /_index note exists, use its content instead of template
	notes := env.LatestNoteViews()
	if notes != nil {
		indexNote := notes.GetByPath("/_index")
		if indexNote != nil && len(indexNote.Content) > 0 {
			replacements[indexMDPath] = indexNote.Content
		}
	}

	// Read embedded ZIP and modify files, replacing {{publicUrl}} placeholder
	modifiedZip, err := modifyZipFiles(onboardingvault.ZipData, replacements, publicURL)
	if err != nil {
		return nil, fmt.Errorf("failed to modify zip: %w", err)
	}

	return modifiedZip, nil
}

func modifyZipFiles(zipData []byte, replacements map[string][]byte, publicURL string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip: %w", err)
	}

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	for _, file := range reader.File {
		if newContent, ok := replacements[file.Name]; ok {
			// Replace with new content
			w, createErr := writer.Create(file.Name)
			if createErr != nil {
				return nil, fmt.Errorf("failed to create file in zip: %w", createErr)
			}

			_, writeErr := w.Write(newContent)
			if writeErr != nil {
				return nil, fmt.Errorf("failed to write new content: %w", writeErr)
			}

			continue
		}

		// For _index.md, replace {{publicUrl}} placeholder.
		if file.Name == indexMDPath {
			content, readErr := readZipFileContent(file)
			if readErr != nil {
				return nil, fmt.Errorf("failed to read %s: %w", file.Name, readErr)
			}

			content = bytes.ReplaceAll(content, []byte("{{publicUrl}}"), []byte(publicURL))

			w, createErr := writer.Create(file.Name)
			if createErr != nil {
				return nil, fmt.Errorf("failed to create file in zip: %w", createErr)
			}

			_, writeErr := w.Write(content)
			if writeErr != nil {
				return nil, fmt.Errorf("failed to write content: %w", writeErr)
			}

			continue
		}

		// Copy file as-is.
		err = copyZipFile(writer, file)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file %s: %w", file.Name, err)
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

func copyZipFile(writer *zip.Writer, file *zip.File) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	w, err := writer.CreateHeader(&file.FileHeader)
	if err != nil {
		return err
	}

	// Limit copy size to prevent decompression bomb attacks
	_, err = io.Copy(w, io.LimitReader(rc, maxFileSize))

	return err
}

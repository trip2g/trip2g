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
)

type Env interface {
	GenerateAPIKey() string
	InsertAPIKey(ctx context.Context, params db.InsertAPIKeyParams) (db.ApiKey, error)
}

const dataJSONPath = "onboarding-vault/.obsidian/plugins/trip2g/data.json"

type pluginData struct {
	SyncDirs             []syncDir `json:"syncDirs"`
	SkipPushConfirmation bool      `json:"skipPushConfirmation"`
}

type syncDir struct {
	Path   string `json:"path"`
	APIKey string `json:"apiKey"`
	APIURL string `json:"apiUrl"`
}

func Resolve(ctx context.Context, env Env, userID int, siteURL string) ([]byte, error) {
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
				APIURL: siteURL,
			},
		},
		SkipPushConfirmation: false,
	}

	newDataJSON, err := json.MarshalIndent(newData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plugin data: %w", err)
	}

	// Read embedded ZIP and modify data.json
	modifiedZip, err := modifyZipFile(onboardingvault.ZipData, dataJSONPath, newDataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to modify zip: %w", err)
	}

	return modifiedZip, nil
}

func modifyZipFile(zipData []byte, targetPath string, newContent []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip: %w", err)
	}

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	for _, file := range reader.File {
		if file.Name == targetPath {
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

		// Copy file as-is
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

package gitapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"

	"github.com/99designs/gqlgen/graphql"
	"github.com/valyala/fasthttp"
)

var ErrNoAuth = errors.New("no auth provided")

type handler func(ctx *fasthttp.RequestCtx) error

type Env interface {
	Logger() logger.Logger

	PutPrivateObject(ctx context.Context, reader io.Reader, objectID string) error
	GetPrivateObject(ctx context.Context, objectID string) (io.ReadCloser, error)
	PrivateObjectExists(ctx context.Context, objectID string) (bool, error)

	// auth
	GitTokenByValueSha256(ctx context.Context, sha256Hash string) (db.GitToken, error)

	// process notes
	AllVisibleNotePaths(ctx context.Context) ([]db.NotePath, error)
	PushNotes(ctx context.Context, input model.PushNotesInput) (model.PushNotesOrErrorPayload, error)
	UploadNoteAsset(ctx context.Context, input model.UploadNoteAssetInput) (model.UploadNoteAssetOrErrorPayload, error)
}

type Config struct {
	BasePath string
	RepoPath string

	MasterBranch string
}

type API struct {
	config Config
	ctx    context.Context
	env    Env
	logger logger.Logger

	handlers map[string]map[string]handler

	mu sync.Mutex

	repoCreated           bool
	preReceiveHookSetuped bool
}

func DefaultConfig() Config {
	return Config{
		BasePath: "/_system/git",
		RepoPath: "tmp/git",

		MasterBranch: "master",
	}
}

func New(ctx context.Context, config Config, env Env) (*API, error) {
	err := os.MkdirAll(config.RepoPath, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create repo path: %w", err)
	}

	requiredBins := []string{
		"git",
		"git-upload-pack",
		"git-receive-pack",
		"tar",
	}

	err = checkBins(requiredBins)
	if err != nil {
		return nil, err
	}

	api := API{
		config: config,
		logger: logger.WithPrefix(env.Logger(), "git:"),
		env:    env,
		ctx:    ctx,
	}

	api.handlers = map[string]map[string]handler{
		"GET": map[string]handler{
			"/info/refs": api.handleInfoRefs,
		},
		"POST": map[string]handler{
			"/git-upload-pack":  api.handleGitUploadPack,
			"/git-receive-pack": api.handleGitReceivePack,
		},
	}

	return &api, nil
}

func (api *API) initRepo() error {
	err := api.ensureBareRepo()
	if err != nil {
		return err
	}

	err = api.setupPreReceiveHook()
	if err != nil {
		return err
	}

	return nil
}

func (api *API) setupPreReceiveHook() error {
	if api.preReceiveHookSetuped {
		return nil
	}

	api.preReceiveHookSetuped = true

	hookPath := path.Join(api.config.RepoPath, "hooks", "pre-receive")

	script := []byte(fmt.Sprintf(`#!/bin/sh

while read oldrev newrev refname
do
  if [ "$refname" != "refs/heads/%s" ]; then
    echo "ERROR: Only '%s' branch can be updated"
    exit 1
  fi
done
`, api.config.MasterBranch, api.config.MasterBranch))

	err := os.WriteFile(hookPath, script, 0755)
	if err != nil {
		return fmt.Errorf("failed to write pre-receive hook: %w", err)
	}

	return nil
}

func (api *API) ensureBareRepo() error {
	if api.repoCreated {
		return nil
	}

	headPath := path.Join(api.config.RepoPath, "HEAD")

	_, err := os.Stat(headPath)
	if err == nil {
		api.repoCreated = true
		api.logger.Info("bare repo already exists", "path", api.config.RepoPath)
		return nil // already exists
	}

	ctx, cancel := context.WithTimeout(api.ctx, 5*time.Second)
	defer cancel()

	exists, err := api.env.PrivateObjectExists(ctx, api.repoStorageObjectID())
	if err != nil {
		return fmt.Errorf("failed to check private object: %w", err)
	}

	if exists {
		downloadErr := api.downloadRepo()
		if downloadErr != nil {
			return fmt.Errorf("failed to download repo: %w", downloadErr)
		}

		api.repoCreated = true
		api.logger.Info("bare repo restored from storage", "path", api.config.RepoPath)

		return nil
	}

	api.logger.Info("initializing bare repo", "path", api.config.RepoPath)

	cmd := exec.Command("git", "init", "--bare", "--initial-branch", api.config.MasterBranch, ".")
	cmd.Dir = api.config.RepoPath
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to init bare repo: %w", err)
	}

	api.repoCreated = true

	return nil
}

func (api *API) HandleRequest(ctx *fasthttp.RequestCtx) bool {
	path := string(ctx.Path())
	method := string(ctx.Method())

	if !strings.HasPrefix(path, api.config.BasePath) {
		return false
	}

	err := api.checkAuth(ctx)
	if err != nil {
		api.logger.Warn("auth failed", "error", err)

		ctx.Response.Header.Set("WWW-Authenticate", `Basic realm="Git Repository"`)
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		_, _ = ctx.WriteString(err.Error())

		return true
	}

	handlers, ok := api.handlers[method]
	if !ok {
		api.logger.Warn("unsupported method", "method", method)
		return false
	}

	hdr, ok := handlers[path[len(api.config.BasePath):]]
	if !ok {
		api.logger.Warn("unsupported path", "path", path)
	}

	api.mu.Lock()
	defer api.mu.Unlock()

	err = api.initRepo()
	if err != nil {
		api.logger.Error("failed to init repo", "error", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(err.Error())
		return true
	}

	err = hdr(ctx)
	if err != nil {
		api.logger.Error("handler error", "error", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(err.Error())
	}

	return true
}

func (api *API) checkAuth(ctx *fasthttp.RequestCtx) error {
	auth := strings.TrimSpace(string(ctx.Request.Header.Peek("Authorization")))
	if auth == "" {
		ctx.Request.Header.VisitAll(func(key, value []byte) {
			api.logger.Debug("header", "key", string(key), "value", string(value))
		})

		api.logger.Debug("no auth header")
		return ErrNoAuth
	}

	const prefix = "Basic "
	if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix {
		api.logger.Debug("invalid auth header", "auth", auth)
		return ErrNoAuth
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return fmt.Errorf("failed to decode auth: %w", err)
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		api.logger.Debug("invalid auth format", "decoded", string(decoded))
		return ErrNoAuth
	}

	if parts[0] != "user" {
		api.logger.Debug("invalid username", "username", parts[0])
		return ErrNoAuth
	}

	// TODO: check password
	api.logger.Info("auth success", "token", parts[1])

	return nil
}

func (api *API) handleInfoRefs(ctx *fasthttp.RequestCtx) error {
	service := string(ctx.QueryArgs().Peek("service"))

	if service != "git-upload-pack" && service != "git-receive-pack" {
		return fmt.Errorf("unsupported service: %s", service)
	}

	api.logger.Info("handling git service", "service", service)

	var cmd *exec.Cmd

	args := []string{
		"--stateless-rpc",
		"--advertise-refs",
		api.config.RepoPath,
	}

	if service == "git-upload-pack" {
		cmd = exec.Command("git-upload-pack", args...)
	} else {
		cmd = exec.Command("git-receive-pack", args...)
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = ctx

	contentType := fmt.Sprintf("application/x-%s-advertisement", service)

	ctx.Response.Header.Set("Content-Type", contentType)
	_, _ = ctx.Write(pktLine(fmt.Sprintf("# service=%s\n", service)))
	_, _ = ctx.Write([]byte("0000"))

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run %s: %w", service, err)
	}

	return nil
}

func (api *API) handleGitUploadPack(ctx *fasthttp.RequestCtx) error {
	cmd := exec.Command("git-upload-pack", "--stateless-rpc", api.config.RepoPath)
	cmd.Stdin = bytes.NewReader(ctx.PostBody())
	cmd.Stdout = ctx

	ctx.Response.Header.Set("Content-Type", "application/x-git-upload-pack-result")

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run git-upload-pack: %w", err)
	}

	return nil
}

func (api *API) handleGitReceivePack(ctx *fasthttp.RequestCtx) error {
	cmd := exec.Command("git-receive-pack", "--stateless-rpc", api.config.RepoPath)
	cmd.Stdin = bytes.NewReader(ctx.PostBody())
	cmd.Stdout = ctx
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run git-receive-pack: %w", err)
	}

	err = api.applyChanges()
	if err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	// todo: run in background
	err = api.uploadRepo()
	if err != nil {
		return fmt.Errorf("failed to upload repo: %w", err)
	}

	return nil
}

func (api *API) preparePushNotesInput(changedFiles []string) (*model.PushNotesInput, error) {
	notePaths, err := api.env.AllVisibleNotePaths(api.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get note paths: %w", err)
	}

	hashes := map[string]string{}

	for _, notePath := range notePaths {
		hashes[notePath.Value] = notePath.LatestContentHash
	}

	pushInput := model.PushNotesInput{}

	for _, file := range changedFiles {
		ext := strings.ToLower(filepath.Ext(file))
		if ext != ".md" {
			continue // only process markdown files
		}

		// read content
		readCmd := exec.Command("git", "--git-dir", api.config.RepoPath, "-c", "core.quotePath=false", "show", fmt.Sprintf("HEAD:%s", file))
		readCmd.Stderr = os.Stderr

		content, readErr := readCmd.Output()
		if readErr != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file, readErr)
		}

		sha := sha256.New()
		sha.Write(content)

		contentHash := base64.URLEncoding.EncodeToString(sha.Sum(nil))

		expectedHash, ok := hashes[file]
		if ok && expectedHash == contentHash {
			continue // already processed
		}

		pushInput.Updates = append(pushInput.Updates, model.PushNoteInput{
			Path:    file,
			Content: string(content),
		})
	}

	return &pushInput, nil
}

func (api *API) isRefExists(ref string) bool {
	checkCmd := exec.Command("git", "rev-parse", "--verify", ref)
	checkCmd.Dir = api.config.RepoPath
	checkCmd.Stderr = nil // suppress error output
	return checkCmd.Run() == nil
}

func (api *API) getChangedFiles() ([]string, error) {
	if !api.isRefExists("HEAD") {
		// HEAD doesn't exist, this is the first commit
		return nil, errors.New("first commit detected, HEAD does not exist")
	}

	if !api.isRefExists("HEAD^1") {
		// HEAD^1 doesn't exist, this is the first commit
		return nil, errors.New("first commit detected, HEAD^1 does not exist")
	}

	// TODO: track the last processed commit
	// git diff --name-only HEAD^1 HEAD
	cmd := exec.Command("git", "-c", "core.quotePath=false", "diff", "--name-only", "HEAD^1", "HEAD")
	cmd.Dir = api.config.RepoPath
	cmd.Stderr = os.Stderr

	// Limit output to prevent memory issues (1MB should be enough for file lists)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	limitedReader := io.LimitReader(stdoutPipe, 1<<20) // 1MB limit
	output, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read output: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	return api.filterDotFiles(files), nil
}

func (api *API) applyChanges() error {
	changedFiles, err := api.getChangedFiles()
	if err != nil {
		api.logger.Warn("no changed files", "error", err)

		// list all files
		// TODO: fix logic for first push
		changedFiles, err = api.getAllFiles()
		if err != nil {
			return fmt.Errorf("failed to get all files: %w", err)
		}

		api.logger.Info("all files", "files", changedFiles)
	}

	pushInput, err := api.preparePushNotesInput(changedFiles)
	if err != nil {
		return fmt.Errorf("failed to prepare push notes input: %w", err)
	}

	pushPayload, err := api.env.PushNotes(api.ctx, *pushInput)
	if err != nil {
		return fmt.Errorf("failed to push notes: %w", err)
	}

	switch payload := pushPayload.(type) {
	case *model.ErrorPayload:
		return fmt.Errorf("failed to push notes: %s", payload.Message)
	case *model.PushNotesPayload:
		api.logger.Info("notes pushed", "count", len(payload.Notes))

		for _, note := range payload.Notes {
			uploadErr := api.uploadNoteAssets(note, changedFiles)
			if uploadErr != nil {
				return fmt.Errorf("failed to upload note assets %s: %w", note.Path, uploadErr)
			}
		}

	default:
		return fmt.Errorf("unknown push payload type: %T", payload)
	}

	return nil
}

func (api *API) getAllFiles() ([]string, error) {
	if !api.isRefExists("HEAD") {
		// HEAD doesn't exist, use ls-files to get staged files
		cmd := exec.Command("git", "--git-dir", api.config.RepoPath, "-c", "core.quotePath=false", "ls-files")
		cmd.Stderr = os.Stderr

		// Limit output to prevent memory issues
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
		}

		err = cmd.Start()
		if err != nil {
			return nil, fmt.Errorf("failed to start command: %w", err)
		}

		limitedReader := io.LimitReader(stdoutPipe, 1<<20) // 1MB limit
		output, err := io.ReadAll(limitedReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read output: %w", err)
		}

		err = cmd.Wait()
		if err != nil {
			return nil, fmt.Errorf("failed to list staged files: %w", err)
		}

		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		return api.filterDotFiles(files), nil
	}

	cmd := exec.Command("git", "--git-dir", api.config.RepoPath, "-c", "core.quotePath=false", "ls-tree", "-r", "HEAD", "--name-only")
	cmd.Stderr = os.Stderr

	// Limit output to prevent memory issues
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	limitedReader := io.LimitReader(stdoutPipe, 1<<20) // 1MB limit
	output, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read output: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	return api.filterDotFiles(files), nil
}

func (api *API) uploadNoteAssets(note model.PushedNote, _ []string) error {
	for _, asset := range note.Assets {
		assetPath := api.resolveAssetPath(note.Path, asset.Path)

		api.logger.Info("resolved asset path", "note", note.Path, "relative", asset.Path, "asset", assetPath)

		content, err := api.readContent(assetPath)
		if err != nil {
			return fmt.Errorf("failed to calculate hash for asset %s: %w", assetPath, err)
		}

		// empty or not exists
		if len(content) == 0 {
			continue
		}

		sha := sha256.New()
		sha.Write(content)

		hash := hex.EncodeToString(sha.Sum(nil))

		api.logger.Info("hashed asset", "path", assetPath, "hash", hash, "assets", note.Assets)
		if asset.Sha256Hash != nil {
			api.logger.Info("existing asset hash", "path", assetPath, "hash", *asset.Sha256Hash)
		}

		if asset.Sha256Hash != nil && *asset.Sha256Hash == hash {
			continue // already uploaded
		}

		input := model.UploadNoteAssetInput{
			NoteID:       note.ID,
			Path:         note.Path,
			Sha256Hash:   hash,
			AbsolutePath: "/" + assetPath,
			File: graphql.Upload{
				File:        bytes.NewReader(content),
				Filename:    filepath.Base(note.Path),
				Size:        int64(len(content)),
				ContentType: "text/plain",
			},
		}

		api.logger.Info("upload asset", "input", input)

		payload, err := api.env.UploadNoteAsset(api.ctx, input)
		if err != nil {
			return fmt.Errorf("failed to upload note asset %s: %w", assetPath, err)
		}

		switch p := payload.(type) {
		case *model.ErrorPayload:
			return fmt.Errorf("failed to upload note asset %s: %s", assetPath, p.Message)

		case *model.UploadNoteAssetPayload:
			// success

		default:
			return fmt.Errorf("unknown upload payload type: %T", p)
		}
	}

	return nil
}

func (api *API) readContent(path string) ([]byte, error) {
	// Check if HEAD exists first
	if !api.isRefExists("HEAD") {
		return nil, nil
	}

	// Check if file exists in HEAD before trying to read it
	checkCmd := exec.Command("git", "--git-dir", api.config.RepoPath, "ls-tree", "HEAD", path)
	checkCmd.Stderr = nil // suppress error output
	err := checkCmd.Run()
	if err != nil {
		// File doesn't exist in HEAD, return empty content
		return nil, fmt.Errorf("file does not exist in HEAD: %w", err)
	}

	cmd := exec.Command("git", "--git-dir", api.config.RepoPath, "-c", "core.quotePath=false", "show", fmt.Sprintf("HEAD:%s", path))

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// limit to 10MB per file
	maxSize := 1 << 20 // 1 MB
	maxSize *= 10      // 10 MB

	limited := io.LimitReader(stdout, int64(maxSize+1))

	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read output: %w", err)
	}

	if len(content) > maxSize {
		_ = cmd.Process.Kill() // kill process if still running
		return nil, errors.New("file too large (>10MB)")
	}

	err = cmd.Wait()
	if err != nil {
		errOutput := stderr.String()
		if strings.Contains(errOutput, "does not exist in") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read file %s: %w | %s", path, err, errOutput)
	}

	return content, nil
}

func (api *API) resolveAssetPath(notePath, relativePath string) string {
	if strings.HasPrefix(relativePath, "/") {
		return relativePath[1:]
	}

	noteDirParts := strings.Split(filepath.Dir(notePath), string(filepath.Separator))

	for i := len(noteDirParts) - 1; i >= 0; i-- {
		noteDir := filepath.Join(noteDirParts[:i]...)
		assetPath := filepath.Join(noteDir, relativePath)

		// check exists in git
		cmd := exec.Command("git", "--git-dir", api.config.RepoPath, "-c", "core.quotePath=false", "ls-files", "--error-unmatch", assetPath)
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err == nil {
			return assetPath
		}
	}

	return relativePath
}

func (api *API) repoStorageObjectID() string {
	return "repo.tar.gz"
}

func (api *API) uploadRepo() error {
	objectID := api.repoStorageObjectID()

	api.logger.Info("uploading repo", "objectID", objectID)

	// Use pipe to stream data directly without loading into memory
	pipeReader, pipeWriter := io.Pipe()

	// Start tar command in goroutine
	go func() {
		defer pipeWriter.Close()

		gzipWriter := gzip.NewWriter(pipeWriter)
		defer gzipWriter.Close()

		cmd := exec.Command("tar", "-c", "-C", api.config.RepoPath, ".")
		cmd.Stdout = gzipWriter
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			api.logger.Error("failed to create tar", "error", err)
			pipeWriter.CloseWithError(fmt.Errorf("failed to create tar: %w", err))
			return
		}
	}()

	// Upload the streamed data
	err := api.env.PutPrivateObject(context.Background(), pipeReader, objectID)
	if err != nil {
		_ = pipeReader.Close()
		return fmt.Errorf("failed to put private object: %w", err)
	}

	return nil
}

func (api *API) downloadRepo() error {
	objectID := api.repoStorageObjectID()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reader, err := api.env.GetPrivateObject(ctx, objectID)
	if err != nil {
		return fmt.Errorf("failed to get private object: %w", err)
	}
	defer reader.Close()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	cmd := exec.Command("tar", "-xz", "-C", api.config.RepoPath)
	cmd.Stdin = gzipReader
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to extract tar.gz: %w", err)
	}

	return nil
}

func (api *API) filterDotFiles(files []string) []string {
	var filtered []string
	for _, file := range files {
		// Skip empty filenames
		if file == "" {
			continue
		}

		// Check if any part of the path starts with a dot
		parts := strings.Split(file, "/")
		shouldSkip := false
		for _, part := range parts {
			if strings.HasPrefix(part, ".") {
				shouldSkip = true
				break
			}
		}

		if !shouldSkip {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func pktLine(s string) []byte {
	totalLen := len(s) + 4
	return []byte(fmt.Sprintf("%04x%s", totalLen, s))
}

func checkBins(bins []string) error {
	for _, bin := range bins {
		_, err := exec.LookPath(bin)
		if err != nil {
			return fmt.Errorf("required binary not found: %s", bin)
		}
	}

	return nil
}

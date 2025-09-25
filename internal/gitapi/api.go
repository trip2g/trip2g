package gitapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	"github.com/valyala/fasthttp"
)

var ErrNoAuth = fmt.Errorf("no auth provided")

type handler func(ctx *fasthttp.RequestCtx) error

type Env interface {
	Logger() logger.Logger

	PutPrivateObject(ctx context.Context, reader io.Reader, objectID string) error
	GetPrivateObject(ctx context.Context, objectID string) (io.ReadCloser, error)
	PrivateObjectExists(ctx context.Context, objectID string) (bool, error)

	AllVisibleNotePaths(ctx context.Context) ([]db.NotePath, error)
	// pushnotes
	// upload assets
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

	cmd := exec.Command("git", "init", "--bare", ".")
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
		ctx.WriteString(err.Error())

		return true
	}

	handlers, ok := api.handlers[method]
	if !ok {
		api.logger.Warn("unsupported method", "method", method)
		return false
	}

	handler, ok := handlers[path[len(api.config.BasePath):]]
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

	err = handler(ctx)
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
	ctx.Write(pktLine(fmt.Sprintf("# service=%s\n", service)))
	ctx.Write([]byte("0000"))

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

	// git diff --name-only HEAD^1 HEAD
	cmd = exec.Command("git", "diff", "--name-only", "HEAD^1", "HEAD")
	cmd.Dir = api.config.RepoPath
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get changed files: %w", err)
	}

	changedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")

	api.logger.Info("files changed", "files", changedFiles)

	// todo: run in background
	err = api.uploadRepo()
	if err != nil {
		return fmt.Errorf("failed to upload repo: %w", err)
	}

	return nil
}

func (api *API) repoStorageObjectID() string {
	return "repo.tar.gz"
}

func (api *API) uploadRepo() error {
	objectID := api.repoStorageObjectID()

	api.logger.Info("uploading repo", "objectID", objectID)

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	cmd := exec.Command("tar", "-cz", "-C", api.config.RepoPath, ".")
	cmd.Stdout = gzipWriter

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create tar.gz: %w", err)
	}

	gzipWriter.Close()

	err = api.env.PutPrivateObject(context.Background(), &buf, objectID)
	if err != nil {
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

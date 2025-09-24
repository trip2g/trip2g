package gitapi

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"trip2g/internal/logger"

	"github.com/valyala/fasthttp"
)

type handler func(ctx *fasthttp.RequestCtx) error

type Env interface {
	Logger() logger.Logger
}

type Config struct {
	BasePath string
	RepoPath string
}

type API struct {
	config Config
	env    Env
	logger logger.Logger

	handlers map[string]map[string]handler

	mu sync.Mutex

	repoCreated bool
}

func DefaultConfig() Config {
	return Config{
		BasePath: "/_system/git",
		RepoPath: "tmp/git",
	}
}

func New(config Config, env Env) (*API, error) {
	err := os.MkdirAll(config.RepoPath, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create repo path: %w", err)
	}

	api := API{
		config: config,
		logger: logger.WithPrefix(env.Logger(), "git:"),
		env:    env,
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

	err := api.initRepo()
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

	// set content-type
	ctx.Response.Header.Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))

	// write smart headre
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

	return nil
}

func pktLine(s string) []byte {
	totalLen := len(s) + 4
	return []byte(fmt.Sprintf("%04x%s", totalLen, s))
}

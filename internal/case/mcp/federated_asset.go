package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"trip2g/internal/model"
)

// FederatedAssetArguments are federated_asset's arguments: the note as the
// caller read it, and one name from that note's assets.
type FederatedAssetArguments struct {
	KBID     string `json:"kb_id"`
	NotePath string `json:"note_path"`
	Name     string `json:"name"`
}

// FederatedAssetPayload describes the attachment and where to download it. The
// bytes never travel through the tool result: an attachment is a file for the
// caller's disk, not text for a model's context.
type FederatedAssetPayload struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Sha256    string    `json:"sha256"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NoteHTMLPayload is the structured half of a note read: the note's path and
// its assets, each named exactly as federated_asset accepts them — the path
// too, whichever reference the note was opened by.
type NoteHTMLPayload struct {
	NotePath string                `json:"note_path"`
	Assets   []model.NoteAssetInfo `json:"assets"`
}

// FederatedAssetDownload is a link the caller can fetch without any other
// credential, valid until ExpiresAt.
type FederatedAssetDownload struct {
	URL       string
	ExpiresAt time.Time
}

type federatedAssetEnv interface {
	model.FederationAssetFetcherFactory
	FederatedAssetDownload(kbID string, params model.FederatedAssetParams) (FederatedAssetDownload, error)
}

func handleFederatedAsset(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[FederatedAssetArguments](argsRaw, id, "federated_asset")
	if errResp != nil {
		return *errResp
	}
	if args.KBID == "" || args.NotePath == "" || args.Name == "" {
		return errorResponse(id, ErrCodeInvalidParams, "kb_id, note_path and name are required")
	}
	if segmentCount(args.KBID) > 1 {
		return errorResponse(id, ErrCodeInvalidParams,
			"federated_asset reaches a directly connected base only; a nested kb_id is not supported")
	}

	kbs, err := accessibleKBNotes(ctx, env)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	kb, _ := findFederationKB(kbs, args.KBID)
	if kb == nil {
		return federationNotConfiguredResponse(id, args.KBID, kbs)
	}

	params := model.FederatedAssetParams{NotePath: args.NotePath, Name: args.Name}

	info, err := describeFederatedAsset(ctx, env, kb.ID, params)
	metricsFromContext(ctx).RecordFederatedRequest(kb.ID, federatedStatus(err))
	if err != nil {
		return federatedAssetError(id, args, err)
	}

	download, err := env.FederatedAssetDownload(kb.ID, params)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}

	payload := FederatedAssetPayload{
		Name:      info.Name,
		Size:      info.Size,
		Sha256:    info.Sha256,
		URL:       download.URL,
		ExpiresAt: download.ExpiresAt,
	}

	return successResponse(id, structuredToolResult(federatedAssetText(payload), payload))
}

func describeFederatedAsset(ctx context.Context, env Env, kbID string, params model.FederatedAssetParams) (model.NoteAssetInfo, error) {
	fetcher, err := env.FederationAssetFetcher(ctx, kbID)
	if err != nil {
		return model.NoteAssetInfo{}, err
	}
	return fetcher.DescribeAsset(ctx, params)
}

// federatedAssetError turns the peer's refusals into answers the caller can act
// on. A missing note, a note outside the grant and a name the note does not
// have are one answer on purpose: the peer does not say which.
func federatedAssetError(id any, args *FederatedAssetArguments, err error) Response {
	var httpErr *model.FederationHTTPError
	isStatus := func(status int) bool {
		return errors.As(err, &httpErr) && httpErr.Status == status
	}

	switch {
	case isStatus(http.StatusNotFound):
		return errorResponse(id, ErrCodeInvalidParams, fmt.Sprintf(
			"no asset %q in note %q of %q; copy note_path and a name from that note's assets in federated_note_html",
			args.Name, args.NotePath, args.KBID))
	case isStatus(http.StatusRequestEntityTooLarge), errors.Is(err, model.ErrFederatedAssetTooLarge):
		return errorResponse(id, ErrCodeInvalidParams, fmt.Sprintf(
			"asset %q is larger than federation is allowed to move", args.Name))
	default:
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
}

func federatedAssetText(p FederatedAssetPayload) string {
	return fmt.Sprintf(
		"%s · %d bytes · sha256 %s\n"+
			"Download before %s (the URL is the only credential needed): curl -fsS -o %s %s\n"+
			"Then check the file's sha256 against the one above.",
		p.Name, p.Size, p.Sha256, p.ExpiresAt.UTC().Format(time.RFC3339),
		shellQuote(lastSegment(p.Name)), shellQuote(p.URL))
}

func lastSegment(name string) string {
	return name[strings.LastIndex(name, "/")+1:]
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// noteReadResult is every successful note read: the text the caller asked for
// and the note's assets beside it, in the structured result for programs and in
// the text for a model that only reads text.
func noteReadResult(note *model.NoteView, text string) CallToolResult {
	assets := noteAssets(note)
	if len(assets) > 0 {
		text += "\n\n" + assetsText(assets)
	}
	return structuredToolResult(text, NoteHTMLPayload{NotePath: note.Path, Assets: assets})
}

func noteAssets(note *model.NoteView) []model.NoteAssetInfo {
	assets := make([]model.NoteAssetInfo, 0, len(note.AssetReplaces))
	for name, replace := range note.AssetReplaces {
		if replace == nil {
			continue
		}
		assets = append(assets, model.NoteAssetInfo{Name: name, Size: replace.Size, Sha256: replace.Hash})
	}
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].Name < assets[j].Name
	})
	return assets
}

func assetsText(assets []model.NoteAssetInfo) string {
	var b strings.Builder
	b.WriteString("Assets (name · bytes · sha256):")
	for _, a := range assets {
		b.WriteString("\n- " + a.Name + " · " + strconv.FormatInt(a.Size, 10) + " · " + a.Sha256)
	}
	return b.String()
}

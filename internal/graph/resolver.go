package graph

import (
	"context"
	"database/sql"
	"trip2g/internal/appreq"
	"trip2g/internal/case/admin/banuser"
	"trip2g/internal/case/admin/createapikey"
	"trip2g/internal/case/admin/createoffer"
	"trip2g/internal/case/admin/createredirect"
	"trip2g/internal/case/admin/createrelease"
	"trip2g/internal/case/admin/deleteredirect"
	"trip2g/internal/case/admin/disableapikey"
	"trip2g/internal/case/admin/makereleaselive"
	"trip2g/internal/case/admin/unbanuser"
	"trip2g/internal/case/admin/updatenotegraphpositions"
	"trip2g/internal/case/admin/updateoffer"
	"trip2g/internal/case/admin/updateredirect"
	"trip2g/internal/case/admin/updatesubgraph"
	"trip2g/internal/case/admin/updateusersubgraphaccess"
	"trip2g/internal/case/checkapikey"
	"trip2g/internal/case/createpaymentlink"
	"trip2g/internal/case/hidenotes"
	"trip2g/internal/case/pushnotes"
	"trip2g/internal/case/requestemailsignin"
	"trip2g/internal/case/signinbyemail"
	"trip2g/internal/case/signout"
	"trip2g/internal/case/uploadnoteasset"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

type Resolver struct {
	DefaultEnv Env
}

func (r *Resolver) env(ctx context.Context) Env {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		panic(err)
	}

	reqEnv, ok := req.Env.(Env)
	if !ok {
		panic("request env is not of type Env")
	}

	return reqEnv
}

type Env interface {
	ListAllUsers(ctx context.Context) ([]db.User, error)
	ListAllUserSubgraphAccesses(ctx context.Context) ([]db.UserSubgraphAccess, error)
	ListAllSubgraphs(ctx context.Context) ([]db.Subgraph, error)
	ListAllUserBans(ctx context.Context) ([]db.UserBan, error)
	ListActiveOffersBySubgraphID(ctx context.Context, subgraphID int64) ([]db.Offer, error)
	ListActiveOffersBySubgraphNames(ctx context.Context, subgraphNames []string) ([]db.Offer, error)
	ListSubgraphsByOfferID(ctx context.Context, offerID int64) ([]db.Subgraph, error)
	ListAllApiKeys(ctx context.Context) ([]db.ApiKey, error)
	ListAllReleases(ctx context.Context) ([]db.Release, error)
	ListAllAdmins(ctx context.Context) ([]db.Admin, error)
	ListAllOffers(ctx context.Context) ([]db.Offer, error)
	ListAllPurchases(ctx context.Context) ([]db.Purchase, error)
	ListSubgraphIDsByOfferID(ctx context.Context, offerID int64) ([]int64, error)
	ListAllRedirects(ctx context.Context) ([]db.Redirect, error)

	UserByID(ctx context.Context, id int64) (db.User, error)
	UserBanByUserID(ctx context.Context, userID int64) (*db.UserBan, error)
	AdminByUserID(ctx context.Context, userID int64) (db.Admin, error)
	SubgraphByID(ctx context.Context, id int64) (db.Subgraph, error)
	SubgraphByName(ctx context.Context, name string) (db.Subgraph, error)
	UserSubgraphAccessByID(ctx context.Context, id int64) (db.UserSubgraphAccess, error)
	OfferByID(ctx context.Context, id int64) (db.Offer, error)
	PurchaseByID(ctx context.Context, id string) (db.Purchase, error)
	RedirectByID(ctx context.Context, id int64) (db.Redirect, error)

	ListActiveSubgraphsByUserID(ctx context.Context, userID int64) ([]db.Subgraph, error)
	ListActiveUserSubgraphAccessesByUserID(ctx context.Context, userID int64) ([]db.UserSubgraphAccess, error)
	ListApiKeyLogsByApiKeyID(ctx context.Context, apiKeyID int64) ([]db.ListApiKeyLogsByApiKeyIDRow, error)

	// activePurchases
	ListActivePurchasesByUserID(ctx context.Context, userID sql.NullInt64) ([]db.Purchase, error)
	ListActivePurchasesByIDs(ctx context.Context, ids []string) ([]db.Purchase, error)
	ExtractPurchaseTokenIDs(ctx context.Context) ([]string, error)

	LatestNoteViews() *model.NoteViews
	AllNotePaths(ctx context.Context) ([]db.NotePath, error)
	NoteGraphPositionByPathID(ctx context.Context, id int64) (db.NoteGraphPositionByPathIDRow, error)

	IDHash(entity string, id int64) string

	Logger() logger.Logger

	AcquireTxEnvInRequest(ctx context.Context, label string) error
	ReleaseTxEnvInRequest(ctx context.Context, commit bool) error

	requestemailsignin.Env
	signinbyemail.Env
	signout.Env
	createpaymentlink.Env
	pushnotes.Env
	uploadnoteasset.Env
	checkapikey.Env

	banuser.Env
	updatesubgraph.Env
	updateusersubgraphaccess.Env
	unbanuser.Env
	createapikey.Env
	disableapikey.Env
	createrelease.Env
	makereleaselive.Env
	updatenotegraphpositions.Env
	createoffer.Env
	updateoffer.Env
	createredirect.Env
	updateredirect.Env
	deleteredirect.Env
	hidenotes.Env
}

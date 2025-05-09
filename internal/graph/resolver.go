package graph

import (
	"context"
	"database/sql"
	"trip2g/internal/appreq"
	"trip2g/internal/case/admin/banuser"
	"trip2g/internal/case/admin/unbanuser"
	"trip2g/internal/case/admin/updatesubgraph"
	"trip2g/internal/case/admin/updateusersubgraphaccess"
	"trip2g/internal/case/createpaymentlink"
	"trip2g/internal/case/requestemailsignin"
	"trip2g/internal/case/signinbyemail"
	"trip2g/internal/case/signout"
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

	UserByID(ctx context.Context, id int64) (db.User, error)
	UserBanByUserID(ctx context.Context, userID int64) (*db.UserBan, error)
	AdminByUserID(ctx context.Context, userID int64) (db.Admin, error)
	SubgraphByID(ctx context.Context, id int64) (db.Subgraph, error)
	SubgraphByName(ctx context.Context, name string) (db.Subgraph, error)
	UserSubgraphAccessByID(ctx context.Context, id int64) (db.UserSubgraphAccess, error)
	ListActiveSubgraphsByUserID(ctx context.Context, userID int64) ([]db.Subgraph, error)

	// activePurchases
	ListActivePurchasesByUserID(ctx context.Context, userID sql.NullInt64) ([]db.Purchase, error)
	ListActivePurchasesByIDs(ctx context.Context, ids []string) ([]db.Purchase, error)
	ExtractPurchaseTokenIDs(ctx context.Context) ([]string, error)

	AllNotes() model.NoteViews

	OnPurchaseUpdatedSubscribe(email string, handler func()) func()

	Logger() logger.Logger

	requestemailsignin.Env
	signinbyemail.Env
	signout.Env
	createpaymentlink.Env

	banuser.Env
	updatesubgraph.Env
	updateusersubgraphaccess.Env
	unbanuser.Env
}

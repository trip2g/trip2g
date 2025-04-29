package graph

import (
	"context"
	"trip2g/internal/case/admin/banuser"
	"trip2g/internal/case/admin/unbanuser"
	"trip2g/internal/case/admin/updatesubgraph"
	"trip2g/internal/case/admin/updateusersubgraphaccess"
	"trip2g/internal/case/requestemailsignin"
	"trip2g/internal/case/signinbyemail"
	"trip2g/internal/case/signout"
	"trip2g/internal/db"
	"trip2g/internal/model"
)

type Resolver struct {
	Env Env
}

type Env interface {
	ListAllUsers(ctx context.Context) ([]db.User, error)
	ListAllUserSubgraphAccesses(ctx context.Context) ([]db.UserSubgraphAccess, error)
	ListAllSubgraphs(ctx context.Context) ([]db.Subgraph, error)
	ListAllUserBans(ctx context.Context) ([]db.UserBan, error)

	UserByID(ctx context.Context, id int64) (db.User, error)
	UserBanByUserID(ctx context.Context, userID int64) (*db.UserBan, error)
	AdminByUserID(ctx context.Context, userID int64) (db.Admin, error)
	SubgraphByID(ctx context.Context, id int64) (db.Subgraph, error)
	UserSubgraphAccessByID(ctx context.Context, id int64) (db.UserSubgraphAccess, error)

	AllNotes() model.NoteViews

	requestemailsignin.Env
	signinbyemail.Env
	signout.Env

	banuser.Env
	updatesubgraph.Env
	updateusersubgraphaccess.Env
	unbanuser.Env
}

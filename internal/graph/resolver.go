package graph

import (
	"context"
	"database/sql"

	"trip2g/internal/appreq"
	"trip2g/internal/case/admin/banuser"
	"trip2g/internal/case/admin/createapikey"
	"trip2g/internal/case/admin/createboostycredentials"
	"trip2g/internal/case/admin/createhtmlinjection"
	"trip2g/internal/case/admin/createnotfoundignoredpattern"
	"trip2g/internal/case/admin/createoffer"
	"trip2g/internal/case/admin/createpatreoncredentials"
	"trip2g/internal/case/admin/createredirect"
	"trip2g/internal/case/admin/createrelease"
	"trip2g/internal/case/admin/createtgbot"
	"trip2g/internal/case/admin/deleteboostycredentials"
	"trip2g/internal/case/admin/deletehtmlinjection"
	"trip2g/internal/case/admin/deletenotfoundignoredpattern"
	"trip2g/internal/case/admin/deletepatreoncredentials"
	"trip2g/internal/case/admin/deleteredirect"
	"trip2g/internal/case/admin/disableapikey"
	"trip2g/internal/case/admin/makereleaselive"
	"trip2g/internal/case/admin/resetnotfoundpath"
	"trip2g/internal/case/admin/restoreboostycredentials"
	"trip2g/internal/case/admin/restorepatreoncredentials"
	"trip2g/internal/case/admin/setboostytiersubgraphs"
	"trip2g/internal/case/admin/setpatreontiersubgraphs"
	"trip2g/internal/case/admin/settgchatsubgraphinvites"
	"trip2g/internal/case/admin/settgchatsubgraphs"
	"trip2g/internal/case/admin/unbanuser"
	"trip2g/internal/case/admin/updateboostycredentials"
	"trip2g/internal/case/admin/updatehtmlinjection"
	"trip2g/internal/case/admin/updatenotegraphpositions"
	"trip2g/internal/case/admin/updatenotfoundignoredpattern"
	"trip2g/internal/case/admin/updateoffer"
	"trip2g/internal/case/admin/updateredirect"
	"trip2g/internal/case/admin/updatesubgraph"
	"trip2g/internal/case/admin/updatetgbot"
	"trip2g/internal/case/admin/updateusersubgraphaccess"
	"trip2g/internal/cronjobs"
	"trip2g/internal/case/checkapikey"
	"trip2g/internal/case/createemailwaitlistrequest"
	"trip2g/internal/case/createpaymentlink"
	"trip2g/internal/case/cronjob/removeexpiredtgchatmembers"
	"trip2g/internal/case/generatetgattachcode"
	"trip2g/internal/case/hidenotes"
	"trip2g/internal/case/listactiveusersubgraphs"
	"trip2g/internal/case/pushnotes"
	"trip2g/internal/case/refreshboostydata"
	"trip2g/internal/case/refreshpatreondata"
	"trip2g/internal/case/rendernotepage"
	"trip2g/internal/case/requestemailsignin"
	"trip2g/internal/case/signinbyemail"
	"trip2g/internal/case/signout"
	"trip2g/internal/case/toggleuserfavoritenote"
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
	ListAllAPIKeys(ctx context.Context) ([]db.ApiKey, error)
	ListAllReleases(ctx context.Context) ([]db.Release, error)
	ListAllAdmins(ctx context.Context) ([]db.Admin, error)
	ListAllOffers(ctx context.Context) ([]db.Offer, error)
	ListAllPurchases(ctx context.Context) ([]db.Purchase, error)
	ListSubgraphIDsByOfferID(ctx context.Context, offerID int64) ([]int64, error)
	ListAllRedirects(ctx context.Context) ([]db.Redirect, error)
	ListAllNotFoundIgnoredPatterns(ctx context.Context) ([]db.NotFoundIgnoredPattern, error)
	ListAllNotFoundPaths(ctx context.Context) ([]db.NotFoundPath, error)
	ListEnabledTgBots(ctx context.Context) ([]db.TgBot, error)
	ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error)

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
	ListAPIKeyLogsByAPIKeyID(ctx context.Context, apiKeyID int64) ([]db.ListAPIKeyLogsByAPIKeyIDRow, error)

	// activePurchases
	ListActivePurchasesByUserID(ctx context.Context, userID sql.NullInt64) ([]db.Purchase, error)
	ListActivePurchasesByIDs(ctx context.Context, ids []string) ([]db.Purchase, error)
	ExtractPurchaseTokenIDs(ctx context.Context) ([]string, error)

	LatestNoteViews() *model.NoteViews
	AllVisibleNotePaths(ctx context.Context) ([]db.NotePath, error)
	NoteGraphPositionByPathID(ctx context.Context, id int64) (db.NoteGraphPositionByPathIDRow, error)

	IDHash(entity string, id int64) string

	Logger() logger.Logger

	AcquireTxEnvInRequest(ctx context.Context, label string) error
	ReleaseTxEnvInRequest(ctx context.Context, commit bool) error

	requestemailsignin.Env
	signinbyemail.Env
	signout.Env
	createpaymentlink.Env
	createemailwaitlistrequest.Env
	toggleuserfavoritenote.Env
	pushnotes.Env
	rendernotepage.Env
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
	createnotfoundignoredpattern.Env
	updatenotfoundignoredpattern.Env
	deletenotfoundignoredpattern.Env
	resetnotfoundpath.Env
	createtgbot.Env
	updatetgbot.Env
	settgchatsubgraphs.Env
	settgchatsubgraphinvites.Env
	createpatreoncredentials.Env
	deletepatreoncredentials.Env
	refreshpatreondata.Env
	restorepatreoncredentials.Env
	setpatreontiersubgraphs.Env
	setboostytiersubgraphs.Env
	removeexpiredtgchatmembers.Env
	listactiveusersubgraphs.Env

	// Boosty credentials
	createboostycredentials.Env
	deleteboostycredentials.Env
	restoreboostycredentials.Env
	updateboostycredentials.Env
	refreshboostydata.Env
	generatetgattachcode.Env
	createhtmlinjection.Env
	updatehtmlinjection.Env
	deletehtmlinjection.Env

	ListUserFavoriteNotes(ctx context.Context, userID int64) ([]db.ListUserFavoriteNotesRow, error)

	AllLatestNoteAssets(ctx context.Context) ([]db.AllLatestNoteAssetsRow, error)
	NoteAssetURL(ctx context.Context, asset db.NoteAsset) (string, error)
	NoteAssetByID(ctx context.Context, id int64) (db.NoteAsset, error)

	// Patreon tier queries
	GetSubgraphsByTierID(ctx context.Context, tierID int64) ([]db.Subgraph, error)

	// Telegram bot queries
	AllTgBots(ctx context.Context) ([]db.TgBot, error)
	TgBot(ctx context.Context, id int64) (db.TgBot, error)
	TgBotChat(ctx context.Context, id int64) (db.TgBotChat, error)
	TgBotChatByTelegramID(ctx context.Context, telegramID int64) (db.TgBotChat, error)
	TgBotChatsByBotID(ctx context.Context, arg db.TgBotChatsByBotIDParams) ([]db.TgBotChat, error)
	FilteredTgBotChats(ctx context.Context, arg db.FilteredTgBotChatsParams) ([]db.TgBotChat, error)
	TgChatMembersByChatID(ctx context.Context, chatID int64) ([]db.TgChatMembersByChatIDRow, error)
	TgChatMembersByChatIDCount(ctx context.Context, chatID int64) (int64, error)
	TgChatSubgraphAccessesByChatID(ctx context.Context, chatID int64) ([]db.TgChatSubgraphAccess, error)
	TgBotChatSubgraphInvitesByChatID(ctx context.Context, chatID int64) ([]db.TgBotChatSubgraphInvite, error)
	TgChatSubgraphAccessesBySubgraphID(ctx context.Context, subgraphID int64) ([]db.TgChatSubgraphAccess, error)
	AllTgChatSubgraphAccesses(ctx context.Context) ([]db.TgChatSubgraphAccess, error)
	TgChatSubgraphAccess(ctx context.Context, id int64) (db.TgChatSubgraphAccess, error)

	// Patreon credentials queries
	AllPatreonCredentials(ctx context.Context) ([]db.PatreonCredential, error)
	AllActivePatreonCredentials(ctx context.Context) ([]db.PatreonCredential, error)
	AllDeletedPatreonCredentials(ctx context.Context) ([]db.PatreonCredential, error)
	PatreonCredentials(ctx context.Context, id int64) (db.PatreonCredential, error)
	GetPatreonCampaignsByCredentialsID(ctx context.Context, credentialsID int64) ([]db.PatreonCampaign, error)
	GetPatreonTiersByCampaignID(ctx context.Context, campaignID int64) ([]db.PatreonTier, error)
	GetPatreonMembersByCampaignID(ctx context.Context, campaignID int64) ([]db.PatreonMember, error)

	// Boosty credentials queries
	AllBoostyCredentials(ctx context.Context) ([]db.BoostyCredential, error)
	AllActiveBoostyCredentials(ctx context.Context) ([]db.BoostyCredential, error)
	AllDeletedBoostyCredentials(ctx context.Context) ([]db.BoostyCredential, error)
	BoostyCredentials(ctx context.Context, id int64) (db.BoostyCredential, error)
	GetBoostyTiers(ctx context.Context) ([]db.BoostyTier, error)
	GetBoostyMembers(ctx context.Context) ([]db.BoostyMember, error)
	GetSubgraphsByBoostyTierID(ctx context.Context, tierID int64) ([]db.Subgraph, error)

	// User note views
	LastUserNoteView(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error)

	// HTML Injections
	ListHTMLInjections(ctx context.Context) ([]db.HtmlInjection, error)
	GetHTMLInjection(ctx context.Context, id int64) (db.HtmlInjection, error)

	// Cron Jobs
	ListAllCronJobs(ctx context.Context) ([]db.CronJob, error)
	UpdateCronJob(ctx context.Context, arg db.UpdateCronJobParams) (db.CronJob, error)
	CronJobByID(ctx context.Context, id int64) (db.CronJob, error)
	ListCronJobExecutionsByJobID(ctx context.Context, jobID int64) ([]db.CronJobExecution, error)
	CronJobs() *cronjobs.CronJobs
}

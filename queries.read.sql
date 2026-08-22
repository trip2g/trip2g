-- name: AllNotePaths :many
select * from note_paths order by id;

-- name: AllVisibleNotePaths :many
select * from note_paths
 where hidden_by is null
 order by id;

-- name: AllNoteVersions :many
select * from note_versions order by path_id, version;

-- name: AllNoteVersionsByPathID :many
select * from note_versions
 where path_id = ?
 order by version desc;

-- name: NoteVersionEditor :one
-- Returns who pushed a given note version: the user (email) and/or the api key
-- (description) recorded at commit time. Admin/editor-only data, gate display.
select
  nv.created_by_user_id,
  nv.created_by_api_key_id,
  nv.created_by_client,
  nv.created_at,
  u.email       as user_email,
  k.description as api_key_description
from note_versions nv
left join users u on u.id = nv.created_by_user_id
left join api_keys k on k.id = nv.created_by_api_key_id
where nv.id = ?;

-- name: AllLatestNotes :many
select value as path, p.id as path_id, v.id as version_id, content, v.created_at, e.embedding
  from note_paths p
  join note_versions v on p.id = v.path_id and p.version_count = v.version
  left join note_version_embeddings e on v.id = e.version_id
 where p.hidden_by is null;

-- name: AllLatestNoteAssets :many
with latest_versions as (
  select 
    p.id as path_id,
    v.id as version_id
  from note_paths p
  join note_versions v on p.id = v.path_id and p.version_count = v.version
),
ranked_assets as (
  select
    lv.version_id,
    na.id as asset_id,
    a.path,
    lv.path_id,
    row_number() over (
      partition by lv.path_id, a.path
      order by v.version desc, a.created_at desc
    ) as rn
  from latest_versions lv
  join note_paths p on lv.path_id = p.id
  join note_versions v on p.id = v.path_id
  join note_version_assets a on v.id = a.version_id
  join note_assets na on a.asset_id = na.id
)
select version_id, path, sqlc.embed(note_assets)
from ranked_assets
join note_assets on ranked_assets.asset_id = note_assets.id
where rn = 1;

-- name: NoteAssetsByVersionID :many
with target_version as (
  select v.id as version_id, p.id as path_id, v.version
  from note_versions v
  join note_paths p on v.path_id = p.id
  where v.id = ?
),
ranked_assets as (
  select
    tv.version_id,
    na.id as asset_id,
    a.path,
    tv.path_id,
    row_number() over (
      partition by tv.path_id, a.path
      order by v.version desc, a.created_at desc
    ) as rn
  from target_version tv
  join note_paths p on tv.path_id = p.id
  join note_versions v on p.id = v.path_id and v.version <= tv.version
  join note_version_assets a on v.id = a.version_id
  join note_assets na on a.asset_id = na.id
)
select version_id, path, sqlc.embed(note_assets)
from ranked_assets
join note_assets on ranked_assets.asset_id = note_assets.id
where rn = 1;

-- name: UserByEmail :one
select * from users where email = lower(?);

-- name: UserByID :one
select * from users where id = ?;

-- name: CountActiveSignInCodes :one
select count(*) from sign_in_codes
 where user_id = ?
   and created_at > datetime('now', '-5 minutes');

-- name: VerifySignInCode :one
select user_id
  from sign_in_codes c
  join users u on c.user_id = u.id
  where u.email = ?
    and c.code = ?
    and c.created_at > datetime('now', '-5 minutes')
  limit 1;

-- name: ListAllUsers :many
select * from users order by created_at desc;

-- name: ListActiveSubgraphNamesByUserID :many
select distinct s.name
  from user_subgraph_accesses a
  join subgraphs s on a.subgraph_id = s.id
 where user_id = ?
   and (expires_at > datetime('now') or expires_at is null)
   and revoke_id is null
 order by 1;

-- name: ListActiveTgChatSubgraphNamesByUserID :many
select distinct s.name
  from users u
  join tg_chat_members m on u.tg_user_id = m.user_id
  join tg_bot_chats bc on bc.id = m.chat_id
  join tg_chat_subgraph_accesses a on a.chat_id = bc.id
  join subgraphs s on s.id = a.subgraph_id
 where u.id = ?
   and bc.removed_at is null
 order by s.name;

-- name: ListActiveTgChatSubgraphNamesByChatID :many
select distinct s.name
  from tg_bot_chat_subgraph_invites tbcsi
  join subgraphs s on s.id = tbcsi.subgraph_id
 where tbcsi.chat_id = ?
 order by s.name;

-- name: ListActivePatreonSubgraphNamesByUserID :many
select distinct s.name
  from users u
  join patreon_members pm on u.email = pm.email
  join patreon_tier_subgraphs pts on pm.current_tier_id = pts.tier_id
  join subgraphs s on pts.subgraph_id = s.id
 where u.id = ? -- if we select by user_id, the sqlc will generate a sql.Null64 arg
   and pm.status = 'active_patron'
 order by s.name;

-- name: ListActiveBoostySubgraphNamesByUserID :many
select distinct s.name
  from users u
  join boosty_members bm on u.email = bm.email
  join boosty_tier_subgraphs bts on bm.current_tier_id = bts.tier_id
  join subgraphs s on bts.subgraph_id = s.id
 where u.id = ? -- if we select by user_id, the sqlc will generate a sql.Null64 arg
   and bm.status = 'active'
 order by s.name;

-- name: ListAllUserSubgraphAccesses :many
select * from user_subgraph_accesses order by id desc;

-- name: UserSubgraphAccessByID :one
select *
  from user_subgraph_accesses
 where id = ?;

-- name: SubgraphByID :one
select * from subgraphs where id = ?;

-- name: SubgraphByName :one
select * from subgraphs where name = ?;

-- name: ListAllSubgraphs :many
select * from subgraphs order by id;

-- name: FederationSecretByKBURL :one
select * from federation_secrets
 where kb_url = ?
   and revoked_at is null
 order by created_at desc, id desc
 limit 1;

-- name: FederationSecretByKID :one
select * from federation_secrets
 where kid = ?
   and kb_url is null
   and revoked_at is null
 order by created_at desc, id desc
 limit 1;

-- name: OutboundFederationSecretByKID :one
select * from federation_secrets
 where kid = ?
   and kb_url is not null
   and revoked_at is null
 order by created_at desc, id desc
 limit 1;

-- name: FederationSecretByID :one
select * from federation_secrets
 where id = ?;

-- name: ListFederationSecrets :many
select
  fs.id,
  fs.kid,
  fs.secret_crypt,
  fs.kb_url,
  fs.description,
  fs.created_at,
  fs.created_by,
  fs.revoked_at,
  fs.rotated_at,
  count(fss.subgraph_id) as subgraph_count
from federation_secrets fs
left join federation_secret_subgraphs fss on fss.kid = fs.kid
group by fs.id
order by fs.created_at desc, fs.id desc;

-- name: ListFederationSecretSubgraphsByKID :many
select s.name
  from federation_secret_subgraphs fss
  join subgraphs s on s.id = fss.subgraph_id
 where fss.kid = ?
 order by s.name;

-- name: ListAllFederationSecretScopes :many
select fss.kid, s.id as subgraph_id, s.name as subgraph_name
  from federation_secret_subgraphs fss
  join subgraphs s on s.id = fss.subgraph_id
 order by fss.kid, s.name;

-- name: ListAllUserBans :many
select * from user_bans;

-- name: AdminByUserID :one
select * from admins where user_id = ?;

-- name: ListActiveOffersBySubgraphID :many
select o.*
  from offers o
  join offer_subgraphs os on o.id = os.offer_id
 where os.subgraph_id = ?
   and (o.starts_at < datetime('now') or o.starts_at is null)
   and (o.ends_at > datetime('now') or o.ends_at is null)
   and o.price_usd > 0
 order by price_usd desc;

-- name: ListActiveOffersBySubgraphNames :many
select o.*
  from offers o
  join offer_subgraphs os on o.id = os.offer_id
  join subgraphs s on os.subgraph_id = s.id
 where s.name in (sqlc.slice(subgraphs))
   and (o.starts_at < datetime('now') or o.starts_at is null)
   and (o.ends_at > datetime('now') or o.ends_at is null)
   and o.price_usd > 0
 order by price_usd desc;

-- name: ListSubgraphsByOfferID :many
select s.*
  from subgraphs s
  join offer_subgraphs os on s.id = os.subgraph_id
 where os.offer_id = ?
 order by s.name;

-- name: ActiveOfferByPublicID :one
select o.*
  from offers o
 where o.public_id = ?
   and (o.starts_at < datetime('now') or o.starts_at is null)
   and (o.ends_at > datetime('now') or o.ends_at is null)
   and o.price_usd > 0
 limit 1;

-- name: PurchaseByID :one
select * from purchases where id = ?;

-- name: OfferByID :one
select * from offers where id = ?;

-- name: CountUserSubgraphAccessByPurchaseID :one
select count(*) from user_subgraph_accesses where purchase_id = ?;

-- name: ListActivePurchasesByUserID :many
select * from purchases
 where user_id = ?
    and status in ('pending', 'waiting', 'confirming', 'confirmed')
    and created_at > datetime('now', '-30 minutes')
 order by created_at desc;

-- name: ListActivePurchasesByIDs :many
select * from purchases
 where id in (sqlc.slice(ids))
   and status in ('pending', 'waiting', 'confirming', 'confirmed')
   and created_at > datetime('now', '-30 minutes')
 order by created_at desc;

-- name: ListActiveSubgraphsByUserID :many
select s.*
  from user_subgraph_accesses a
  join subgraphs s on a.subgraph_id = s.id
 where user_id = ?
   and (expires_at > datetime('now') or expires_at is null)
   and revoke_id is null
 order by s.name;

-- name: ListActiveUserSubgraphAccessesByUserID :many
select a.*
  from user_subgraph_accesses a
  join subgraphs s on a.subgraph_id = s.id
 where user_id = ?
   and (expires_at > datetime('now') or expires_at is null)
   and revoke_id is null
 order by a.user_id, s.name;

-- name: NoteAssetByPathAndHash :one
select * from note_assets
 where absolute_path = ?
   and sha256_hash = ?
 limit 1;

-- name: NoteAssetByAbsolutePathAndSha256Hash :one
select * from note_assets
 where absolute_path = ?
   and sha256_hash = ?
 limit 1;

-- name: NoteAssetsBySha256Hash :many
select * from note_assets
 where sha256_hash = ?;

-- name: NoteVersionByID :one
select p.value as path, path_id, v.id as version_id, v.version, content, v.created_at
  from note_versions v
  join note_paths p on v.path_id = p.id
 where v.id = ?
 limit 1;

-- name: AcmeCertByKey :one
select value from acme_certs where key = ?;

-- name: ApiKeyByValue :one
select * from api_keys where value = ? and disabled_at is null limit 1;

-- name: ApiKeyByID :one
select * from api_keys where id = ? limit 1;

-- name: ListAllAPIKeys :many
select * from api_keys order by created_by, created_at desc;

-- name: ListAllGitTokens :many
select * from git_tokens order by admin_id, created_at desc;

-- name: GitTokenByValueSHA256 :one
select * from git_tokens where value_sha256 = ? and disabled_at is null limit 1;

-- name: ListAPIKeyLogsByAPIKeyID :many
select l.created_at, a.name as action_name, i.value as ip
  from api_key_logs l
  join api_key_log_actions a on l.action_id = a.id
  join api_key_log_ips i on l.ip_id = i.id
 where l.api_key_id = ?
 order by l.created_at desc;

-- name: ListAllReleases :many
select *
  from releases
 order by is_live asc, created_at desc;

-- name: ReleaseByID :one
select *
  from releases
 where id = ?;

-- name: AllLiveNotes :many
select value as path, p.id as path_id, v.id as version_id, content, v.created_at, e.embedding
  from note_paths p
  join note_versions v on p.id = v.path_id
  join release_note_versions rnv on v.id = rnv.note_version_id
  join releases r on rnv.release_id = r.id
  left join note_version_embeddings e on v.id = e.version_id
 where r.is_live = true;

-- name: AllLiveNoteAssets :many
with live_versions as (
  select v.id as version_id, p.id as path_id
  from note_paths p
  join note_versions v on p.id = v.path_id
  join release_note_versions rnv on v.id = rnv.note_version_id
  join releases r on rnv.release_id = r.id
  where r.is_live = true
),
ranked_assets as (
  select
    lv.version_id,
    na.id as asset_id,
    a.path,
    lv.path_id,
    row_number() over (
      partition by lv.path_id, a.path
      order by v.version desc, a.created_at desc
    ) as rn
  from live_versions lv
  join note_paths p on lv.path_id = p.id
  join note_versions v on p.id = v.path_id
  join note_version_assets a on v.id = a.version_id
  join note_assets na on a.asset_id = na.id
)
select version_id, path, path_id, sqlc.embed(note_assets)
from ranked_assets
join note_assets on ranked_assets.asset_id = note_assets.id
where rn = 1;

-- name: NoteGraphPositionByPathID :one
select graph_position_x as x, graph_position_y as y
  from note_paths
 where id = ?
 limit 1;

-- name: ListAllAdmins :many
select * from admins a order by user_id desc;

-- name: ListSubgraphIDsByOfferID :many
select subgraph_id
  from offer_subgraphs
 where offer_id = ?
 order by subgraph_id;

-- name: ListAllOffers :many
select * from offers order by id;

-- name: ListAllPurchases :many
select * from purchases order by created_at desc;

-- name: ListAllRedirects :many
select * from redirects order by is_regex;

-- name: RedirectByID :one
select * from redirects where id = ?;

-- name: ListAllNotFoundIgnoredPatterns :many
select * from not_found_ignored_patterns;

-- name: ListAllNotFoundPaths :many
select * from not_found_paths order by total_hits desc;

-- name: ListActiveNotFoundIPHits :many
select * from not_found_ip_hits where last_hit_at > datetime('now', '-7 days');

-- name: NotFoundIgnoredPatternByID :one
select * from not_found_ignored_patterns where id = ?;

-- name: NotFoundPathByID :one
select * from not_found_paths where id = ?;

-- name: ListEnabledTgBots :many
select * from tg_bots where enabled = true;

-- name: TgUserStateByBotIDAndChatID :one
select *
  from tg_user_states
 where bot_id = ?
   and chat_id = ?
 limit 1;

-- name: TgBotChatByTelegramID :one
select * from tg_bot_chats
where telegram_id = ?;

-- name: TgChatMemberByUserIDAndChatID :one
select user_id, chat_id, created_at
from tg_chat_members
where user_id = ? and chat_id = ?;

-- name: AllTgBots :many
select * from tg_bots
order by created_at desc;

-- name: TgBot :one
select * from tg_bots
where id = ?;

-- name: TgBotChatsByBotID :many
select *
  from tg_bot_chats where bot_id = ?
   and (sqlc.arg(include_removed) = true or removed_at is null);

-- name: TgBotChatsByBotIDCount :one
select count(*)
  from tg_bot_chats
 where bot_id = ?
  and (sqlc.arg(include_removed) = true or removed_at is null);

-- name: FilteredTgBotChats :many
select *
  from tg_bot_chats
where 1=1
  and (sqlc.narg(include_removed) = true or removed_at is null)
  and (bot_id = sqlc.narg(bot_id) or sqlc.narg(bot_id) is null)
  and (can_invite = sqlc.narg(can_invite) or sqlc.narg(can_invite) is null)
order by added_at desc;

-- name: TgChatMembersByChatID :many
select m.*, p.*
from tg_chat_members m
left join tg_user_profiles p on p.chat_id = m.user_id
where m.chat_id = ?
order by m.created_at desc;

-- name: TgChatMembersByChatIDCount :one
select count(*)
from tg_chat_members
where chat_id = ?;

-- name: TgChatSubgraphAccessesByChatID :many
select *
  from tg_chat_subgraph_accesses
 where chat_id = ?
 order by created_at desc;

-- name: TgBotChatSubgraphInvitesByChatID :many
select *
  from tg_bot_chat_subgraph_invites
 where chat_id = ?
 order by created_at desc;

-- name: TgBotChatsWithSubgraphInvites :many
select distinct tbc.id as chat_id, tbc.telegram_id, tbc.chat_title, s.id as subgraph_id, s.name as subgraph_name
from tg_bot_chats tbc
join tg_bot_chat_subgraph_invites tbcsi on tbc.id = tbcsi.chat_id
join subgraphs s on tbcsi.subgraph_id = s.id
where tbc.removed_at is null
  and s.name in (sqlc.slice('subgraph_names'))
order by tbc.chat_title;

-- name: TgChatSubgraphAccessesBySubgraphID :many
select * from tg_chat_subgraph_accesses
where subgraph_id = ?
order by created_at desc;

-- name: AllTgChatSubgraphAccesses :many
select * from tg_chat_subgraph_accesses
order by created_at desc;

-- name: TgChatSubgraphAccess :one
select * from tg_chat_subgraph_accesses
where id = ?;

-- name: TgChatSubgraphInvitesByChatID :many
select * from tg_bot_chat_subgraph_invites
where chat_id = ?
order by created_at desc;

-- name: TgChatSubgraphInvitesBySubgraphID :many
select * from tg_bot_chat_subgraph_invites
where subgraph_id = ?
order by created_at desc;

-- name: TgBotChat :one
select * from tg_bot_chats
where id = ?;

-- name: TgBotChatsCanInvite :many
select * from tg_bot_chats
where can_invite = true
  and removed_at is null
order by chat_title;

-- name: TgUserProfileBySha256Hash :one
select * from tg_user_profiles
where sha256_hash = ?;

-- name: TgUserProfileByChatIDAndBotID :one
select *
  from tg_user_profiles
 where chat_id = ? and bot_id = ?
limit 1;

-- name: UserByTgUserID :one
select *
  from users
 where tg_user_id = ?
limit 1;

-- name: AllWaitListEmailRequests :many
select 
    wler.email,
    wler.created_at,
    wler.ip,
    np.value as note_path
from wait_list_email_requests wler
join note_paths np on wler.note_path_id = np.id
order by wler.created_at desc;

-- name: AllWaitListTgBotRequests :many
select 
    wltr.chat_id,
    wltr.created_at,
    wltr.note_path_id,
    np.value as note_path,
    tb.name as bot_name
from wait_list_tg_bot_requests wltr
join note_paths np on wltr.note_path_id = np.id
join tg_bots tb on wltr.bot_id = tb.id
order by wltr.created_at desc;

-- name: AllPatreonCredentials :many
select * from patreon_credentials
order by created_at desc;

-- name: AllActivePatreonCredentials :many
select * from patreon_credentials
where deleted_at is null
order by created_at desc;

-- name: AllDeletedPatreonCredentials :many
select * from patreon_credentials
where deleted_at is not null
order by created_at desc;

-- name: PatreonCredentials :one
select *
  from patreon_credentials
 where id = ?;

-- name: ListActivePatreonCredentials :many
select *
  from patreon_credentials
 where deleted_by is null;

-- name: GetPatreonCampaignsByCredentialsID :many
select * from patreon_campaigns
where credentials_id = ?
order by created_at desc;

-- name: GetPatreonTiersByCampaignID :many
select *
  from patreon_tiers
 where campaign_id = ?
 order by amount_cents desc;

-- name: GetPatreonMembersByCampaignID :many
select *
  from patreon_members
 where campaign_id = ?
 order by id desc;

-- name: GetPatreonMemberByEmail :one
select *
  from patreon_members
 where email = ?
 limit 1;

-- name: AllBoostyCredentials :many
select * from boosty_credentials
order by created_at desc;

-- name: AllActiveBoostyCredentials :many
select * from boosty_credentials
where deleted_at is null
order by created_at desc;

-- name: AllDeletedBoostyCredentials :many
select * from boosty_credentials
where deleted_at is not null
order by created_at desc;

-- name: BoostyCredentials :one
select *
  from boosty_credentials
 where id = ?;

-- name: GetBoostyTiers :many
select * from boosty_tiers
order by created_at;

-- name: GetSubgraphsByBoostyTierID :many
select s.*
from subgraphs s
join boosty_tier_subgraphs bts on s.id = bts.subgraph_id
where bts.tier_id = ?;

-- name: GetBoostyMembers :many
select * from boosty_members
order by created_at;

-- name: GetBoostyMemberByEmail :one
select * from boosty_members
where email = ? and status = 'active'
order by created_at desc
limit 1;

-- name: GetBoostyTierByBoostyID :one
select * from boosty_tiers
where boosty_id = ?
limit 1;

-- name: BoostyTierByID :one
select * from boosty_tiers
where id = ?;

-- name: GetBoostyTierIDByCredentialsAndBoostyID :one
select id from boosty_tiers
where credentials_id = ? and boosty_id = ?
limit 1;

-- name: GetPatreonTierByTierID :one
select * from patreon_tiers
where campaign_id = ? and tier_id = ?
limit 1;

-- name: PatreonTierByID :one
select * from patreon_tiers
where id = ?;

-- name: GetSubgraphsByTierID :many
select s.*
from subgraphs s
join patreon_tier_subgraphs pts on s.id = pts.subgraph_id
where pts.tier_id = ?
order by s.name;

-- name: GetPatreonMemberByPatreonIDAndCampaignID :one
select * from patreon_members
where patreon_id = ? and campaign_id = ?
limit 1;

-- name: ListUserFavoriteNotes :many
select nv.path_id, nv.id as version_id
  from user_favorite_notes ufn
  join note_versions nv on ufn.note_version_id = nv.id
  join note_paths np on nv.path_id = np.id
 where ufn.user_id = ? and np.hidden_by is null
 order by ufn.created_at desc;

-- name: NoteAssetByID :one
select *
  from note_assets
 where id = ?;

-- name: LastUserNoteView :one
select unv.version_id, unv.created_at
  from user_note_views unv
  join note_versions nv on unv.version_id = nv.id
 where unv.user_id = ?
   and nv.path_id = ?
   and unv.created_at < datetime('now', '-10 minutes')
 order by unv.created_at desc
 limit 1;

-- name: TgAttachCodeByCode :one
select 
    tac.user_id,
    tac.bot_id,
    tac.created_at,
    u.tg_user_id as current_tg_user_id
from tg_attach_codes tac
left join users u on tac.user_id = u.id
where tac.code = ?;

-- name: ListTgBots :many
select * from tg_bots order by description;

-- name: ListTgBotChatSubgraphAccesses :many
select sqlc.embed(tg_bot_chat_subgraph_accesses), sqlc.embed(subgraphs), sqlc.embed(tg_bot_chats)
  from tg_bot_chat_subgraph_accesses
  join subgraphs on tg_bot_chat_subgraph_accesses.subgraph_id = subgraphs.id
  join tg_bot_chats on tg_bot_chat_subgraph_accesses.chat_id = tg_bot_chats.id
 where 1 = 1
   and (user_id = sqlc.narg(user_id) or sqlc.narg(user_id) is null)
   and (chat_id = sqlc.narg(chat_id) or sqlc.narg(chat_id) is null);

-- name: ListAuditLogs :many
select id, created_at, level, message, params
from audit_logs
where (created_at >= sqlc.narg(created_at_gte) or sqlc.narg(created_at_gte) is null)
  and (created_at <= sqlc.narg(created_at_lte) or sqlc.narg(created_at_lte) is null)
order by created_at desc
limit sqlc.arg(limit) offset sqlc.arg(offset);

-- name: ActiveHTMLInjections :many
select *
from html_injections
where (active_from <= datetime('now') or active_from is null)
  and (active_to >= datetime('now') or active_to is null)
order by position;

-- name: ListHTMLInjections :many
select * from html_injections
order by position, created_at desc;

-- name: GetHTMLInjection :one
select * from html_injections
where id = ?;

-- name: CronJobByID :one
select * from cron_jobs where id = ?;

-- name: CronJobByName :one
select * from cron_jobs where name = ?;

-- name: ListAllCronJobs :many
select * from cron_jobs
order by name;

-- name: ListCronJobExecutionsByJobID :many
select * from cron_job_executions
where job_id = ?
order by started_at desc
limit 50;

-- name: GitTokenByValueSha256 :one
select *
  from git_tokens
 where value_sha256 = ?
   and disabled_at is null
 limit 1;

-- name: GetLatestConfigString :one
select c.id, c.value_id, c.created_at, c.created_by, v.value
  from config_changes c
  join config_string_values v on v.change_id = c.id
 where c.value_id = ?
 order by c.id desc
 limit 1;

-- name: GetLatestConfigBool :one
select c.id, c.value_id, c.created_at, c.created_by, v.value
  from config_changes c
  join config_bool_values v on v.change_id = c.id
 where c.value_id = ?
 order by c.id desc
 limit 1;

-- name: ListConfigStringHistory :many
select c.id, c.value_id, c.created_at, c.created_by, v.value
  from config_changes c
  join config_string_values v on v.change_id = c.id
 where c.value_id = ?
 order by c.id desc
 limit 50;

-- name: ListConfigBoolHistory :many
select c.id, c.value_id, c.created_at, c.created_by, v.value
  from config_changes c
  join config_bool_values v on v.change_id = c.id
 where c.value_id = ?
 order by c.id desc
 limit 50;

-- name: AllLatestConfigStrings :many
select c.value_id, v.value
  from config_changes c
  join config_string_values v on v.change_id = c.id
 where c.id in (
   select max(c2.id)
     from config_changes c2
     join config_string_values v2 on v2.change_id = c2.id
    group by c2.value_id
 );

-- name: AllLatestConfigBools :many
select c.value_id, v.value
  from config_changes c
  join config_bool_values v on v.change_id = c.id
 where c.id in (
   select max(c2.id)
     from config_changes c2
     join config_bool_values v2 on v2.change_id = c2.id
    group by c2.value_id
 );

-- name: GetLatestConfigInt :one
select c.id, c.value_id, c.created_at, c.created_by, v.value
  from config_changes c
  join config_int_values v on v.change_id = c.id
 where c.value_id = ?
 order by c.id desc
 limit 1;

-- name: ListConfigIntHistory :many
select c.id, c.value_id, c.created_at, c.created_by, v.value
  from config_changes c
  join config_int_values v on v.change_id = c.id
 where c.value_id = ?
 order by c.id desc
 limit 50;

-- name: AllLatestConfigInts :many
select c.value_id, v.value
  from config_changes c
  join config_int_values v on v.change_id = c.id
 where c.id in (
   select max(c2.id)
     from config_changes c2
     join config_int_values v2 on v2.change_id = c2.id
    group by c2.value_id
 );

-- name: ListNotePathsLike :many
select * from note_paths
 where value like ?
 order by id;

-- name: ListNotePathsByValues :many
select * from note_paths
 where value in (sqlc.slice('paths'))
 order by id;

-- name: FilterNotePathIDsByFrontmatterKey :many
select distinct np.id
  from note_paths np
  join note_versions nv on nv.path_id = np.id and nv.version = np.version_count
  join note_version_frontmatters f on f.version_id = nv.id
  join note_version_frontmatter_keys k on k.note_version_id = nv.id
 where k.key_id = sqlc.arg(key);

-- name: FilterNotePathIDsByFrontmatterEquals :many
select distinct np.id
  from note_paths np
  join note_versions nv on nv.path_id = np.id and nv.version = np.version_count
  join note_version_frontmatters f on f.version_id = nv.id
 where json_extract(f.data, '$.' || sqlc.arg(key)) = sqlc.arg(value);

-- name: NotePathByID :one
select * from note_paths
 where id = ?;

-- name: TelegramPublishTagByLabel :one
select * from telegram_publish_tags
 where label = ?
 limit 1;

-- name: ListTelegramPublishTagsByChatID :many
select t.*
  from telegram_publish_tags t
  join telegram_publish_chats c on t.id = c.tag_id
 where c.chat_id = ?;

-- name: ListTelegramPublishInstantTagsByChatID :many
select t.*
  from telegram_publish_tags t
  join telegram_publish_instant_chats c on t.id = c.tag_id
 where c.chat_id = ?;

-- name: ListAllTelegramPublishTags :many
select * from telegram_publish_tags
 order by label;

-- name: ListAllTelegramPublishNotes :many
select n.*
  from telegram_publish_notes n
  join note_paths p on n.note_path_id = p.id
 where p.hidden_by is null
   and ((coalesce(sqlc.arg(show_scheduled), true) = true and published_at is null)
       or (coalesce(sqlc.arg(show_sent), false) = true and published_at is not null)
       or (coalesce(sqlc.arg(show_outdated), false) = true and published_at is null and error_count > 0))
 order by n.publish_at;

-- name: ListTelegramPublishTagsByNoteID :many
select t.*
  from telegram_publish_tags t
  join telegram_publish_note_tags nt on t.id = nt.tag_id
 where nt.note_path_id = ?
 order by t.label;

-- name: ListScheduledTelegramPublishNoteIDs :many
select n.note_path_id
  from telegram_publish_notes n
  join note_paths p on n.note_path_id = p.id
  -- the note must be tagged with at least one bot chat
  join telegram_publish_note_tags nt on n.note_path_id = nt.note_path_id
  join telegram_publish_chats pc on nt.tag_id = pc.tag_id
  where p.hidden_by is null
   and publish_at <= datetime('now')
   and published_at is null
   and last_error is null
 order by n.publish_at, n.note_path_id;

-- name: ListScheduledTelegramAccountPublishNoteIDs :many
select distinct n.note_path_id
  from telegram_publish_notes n
  join note_paths p on n.note_path_id = p.id
  -- the note must be tagged with at least one account chat
  join telegram_publish_note_tags nt on n.note_path_id = nt.note_path_id
  join telegram_publish_account_chats ac on nt.tag_id = ac.tag_id
  join telegram_accounts a on ac.account_id = a.id
  where p.hidden_by is null
   and publish_at <= datetime('now')
   and published_at is null
   and last_error is null
   and a.enabled = 1
 order by n.publish_at, n.note_path_id;

-- name: ListTgBotChatsByTelegramPublishNotePathID :many
select c.*
  from tg_bot_chats c
  join telegram_publish_chats pc on c.id = pc.chat_id
  join telegram_publish_note_tags nt on pc.tag_id = nt.tag_id
 where nt.note_path_id = ?
   and c.removed_at is null;

-- name: ListTgBotInstantChatsByTelegramPublishNotePathID :many
select c.*
  from tg_bot_chats c
  join telegram_publish_instant_chats pc on c.id = pc.chat_id
  join telegram_publish_note_tags nt on pc.tag_id = nt.tag_id
 where nt.note_path_id = ?
   and c.removed_at is null;

-- name: ListTelegramPublishSentMessagesByNotePathID :many
select tsm.chat_id, tsm.message_id, tsm.content_hash, tsm.content, c.telegram_id
  from telegram_publish_sent_messages tsm
  join tg_bot_chats c on tsm.chat_id = c.id
 where tsm.note_path_id = ?
   and tsm.instant = 0;

-- name: ListTelegramPublishSentMessagesByChatID :many
select tsm.chat_id
     , tsm.message_id
     , tsm.note_path_id
     , p.value as note_path
     , c.telegram_id as telegram_chat_id
  from telegram_publish_sent_messages tsm
  join tg_bot_chats c on tsm.chat_id = c.id
  join note_paths p on tsm.note_path_id = p.id
 where tsm.chat_id = ?
   and p.hidden_at is null
 order by tsm.created_at asc;

-- name: ListTelegramPublishSentAccountMessagesByAccountAndChat :many
select tsam.account_id
     , tsam.message_id
     , tsam.note_path_id
     , p.value as note_path
     , tsam.telegram_chat_id
  from telegram_publish_sent_account_messages tsam
  join note_paths p on tsam.note_path_id = p.id
 where tsam.account_id = ?
   and tsam.telegram_chat_id = ?
   and p.hidden_at is null
 order by tsam.created_at asc;

-- name: GetTelegramPostLinksByNoteVersionID :many
-- Returns all Telegram channels where this note version's path was published.
-- Joins through note_versions to bridge version_id to note_path_id.
-- TODO: consider caching results per version_id to avoid per-request DB query.
select c.chat_title
     , c.telegram_id as telegram_chat_id
     , m.message_id
  from telegram_publish_sent_messages m
  join tg_bot_chats c on c.id = m.chat_id
  join note_versions nv on nv.path_id = m.note_path_id
 where nv.id = ?
   and m.instant = 0

 union all

select '' as chat_title
     , am.telegram_chat_id
     , am.message_id
  from telegram_publish_sent_account_messages am
  join note_versions nv on nv.path_id = am.note_path_id
 where nv.id = ?
   and am.instant = 0;

-- name: GetTelegramChatUsernameByChatID :one
select *
  from telegram_chat_usernames
 where telegram_chat_id = ?;

-- name: ListStaleTelegramChatUsernames :many
select *
  from telegram_chat_usernames
 where refresh_requested_at is not null
    or (username != '' and refreshed_at <= sqlc.arg(positive_stale_before))
    or (username = '' and refreshed_at <= sqlc.arg(negative_stale_before))
 order by coalesce(refresh_requested_at, refreshed_at) asc
 limit sqlc.arg(limit);

-- name: GetTelegramPublishNoteByNotePathID :one
select *
  from telegram_publish_notes
 where note_path_id = ?;

-- name: GetTelegramPublishSentMessageContentHash :one
select content_hash
  from telegram_publish_sent_messages
 where note_path_id = ?
   and chat_id = ?
   and message_id = ?;

-- name: GetTelegramPublishSentMessagePostType :one
select post_type
  from telegram_publish_sent_messages
 where note_path_id = ?
   and chat_id = ?
   and message_id = ?;

-- name: CheckTelegramPublishSentMessageExists :one
select exists(
  select 1
    from telegram_publish_sent_messages
   where note_path_id = ?
     and chat_id = ?
) as message_exists;

-- name: ListDistinctChatIDsFromSentMessages :many
select distinct chat_id
  from telegram_publish_sent_messages
 where instant = 0;

-- name: GetGoqiteQueueStats :one
select queue
     , count(*) as total_jobs
     , count(case when received = 0 then 1 end) as pending_count
     , count(case when received > 1 then 1 end) as retry_count
  from goqite
 where queue = ?
 group by queue;

-- name: ListGoqiteAllQueueStats :many
select queue
     , count(*) as total_jobs
     , count(case when received = 0 then 1 end) as pending_count
     , count(case when received > 1 then 1 end) as retry_count
  from goqite
 group by queue
 order by queue;

-- name: ListGoqiteJobsByQueue :many
select id
     , queue
     , body
     , created
     , received
     , timeout
  from goqite
 where queue = ?
 order by priority desc, created desc
 limit ?;

-- ============================================
-- Telegram Accounts
-- ============================================

-- name: ListAllTelegramAccounts :many
select * from telegram_accounts
 order by created_at desc;

-- name: GetTelegramAccountByID :one
select * from telegram_accounts
 where id = ?;

-- name: GetTelegramAccountByPhone :one
select * from telegram_accounts
 where phone = ?;

-- name: ListTelegramPublishTagsByAccountChatID :many
select t.*
  from telegram_publish_tags t
  join telegram_publish_account_chats c on t.id = c.tag_id
 where c.account_id = ?
   and c.telegram_chat_id = ?;

-- name: ListTelegramPublishInstantTagsByAccountChatID :many
select t.*
  from telegram_publish_tags t
  join telegram_publish_account_instant_chats c on t.id = c.tag_id
 where c.account_id = ?
   and c.telegram_chat_id = ?;

-- name: ListTelegramPublishAccountChatsByAccountID :many
select * from telegram_publish_account_chats
 where account_id = ?;

-- name: ListTelegramPublishAccountInstantChatsByAccountID :many
select * from telegram_publish_account_instant_chats
 where account_id = ?;

-- name: ListTelegramAccountChatsByNotePathID :many
select distinct ac.account_id, ac.telegram_chat_id, a.session_data
  from telegram_publish_account_chats ac
  join telegram_accounts a on ac.account_id = a.id
  join telegram_publish_note_tags nt on ac.tag_id = nt.tag_id
 where nt.note_path_id = ?
   and a.enabled = 1;

-- name: ListTelegramAccountInstantChatsByNotePathID :many
select distinct ac.account_id, ac.telegram_chat_id, a.session_data
  from telegram_publish_account_instant_chats ac
  join telegram_accounts a on ac.account_id = a.id
  join telegram_publish_note_tags nt on ac.tag_id = nt.tag_id
 where nt.note_path_id = ?
   and a.enabled = 1;

-- name: ListEnabledTelegramAccountsByChatID :many
select a.*
  from telegram_accounts a
  join (
    select account_id
      from telegram_publish_account_chats ac
     where ac.telegram_chat_id = ?1
    union
    select account_id
      from telegram_publish_account_instant_chats ac
     where ac.telegram_chat_id = ?1
  ) c on c.account_id = a.id
 where a.enabled = 1
 order by a.created_at desc;

-- name: ListTelegramPublishSentAccountMessagesByNotePathID :many
select account_id, telegram_chat_id, message_id, content_hash, content
  from telegram_publish_sent_account_messages
 where note_path_id = ?
   and instant = 0;

-- name: CheckTelegramPublishSentAccountMessageExists :one
select exists(
  select 1
    from telegram_publish_sent_account_messages
   where note_path_id = ?
     and account_id = ?
     and telegram_chat_id = ?
) as message_exists;

-- name: GetTelegramPublishSentAccountMessageContentHash :one
select content_hash
  from telegram_publish_sent_account_messages
 where note_path_id = ?
   and account_id = ?
   and telegram_chat_id = ?
   and message_id = ?;

-- name: GetTelegramPublishSentAccountMessagePostType :one
select post_type
  from telegram_publish_sent_account_messages
 where note_path_id = ?
   and account_id = ?
   and telegram_chat_id = ?
   and message_id = ?;

-- name: ListDistinctAccountIDsFromSentAccountMessages :many
select distinct account_id
  from telegram_publish_sent_account_messages
 where instant = 0;

-- name: ListTelegramPublishSentAccountMessagesByAccountID :many
select note_path_id, telegram_chat_id, message_id, content_hash
  from telegram_publish_sent_account_messages
 where account_id = ?
   and instant = 0;

-- name: RecentlyModifiedNoteVersionIDs :many
select v.id
  from note_versions v
  join note_paths p on v.path_id = p.id
 where p.hidden_by is null
 order by v.created_at desc limit 20;

-- name: ListUncommittedPaths :many
select note_path_id from note_uncommitted_paths;

-- name: GetTelegramPublishAccountChatAccessHash :one
select access_hash
  from telegram_publish_account_chats
 where account_id = ?
   and telegram_chat_id = ?
   and access_hash is not null
 limit 1;

-- name: GetTelegramPublishAccountInstantChatAccessHash :one
select access_hash
  from telegram_publish_account_instant_chats
 where account_id = ?
   and telegram_chat_id = ?
   and access_hash is not null
 limit 1;

-- name: GetNoteVersionEmbedding :one
select * from note_version_embeddings where version_id = ?;

-- name: GetNoteVersionEmbeddingsByVersionIDs :many
select * from note_version_embeddings where version_id in (sqlc.slice('version_ids'));

-- name: GetNoteVersionChunks :many
select * from note_version_chunks where version_id = ? order by chunk_index;

-- name: GetAllLatestNoteChunksWithEmbeddings :many
select nc.version_id, nc.chunk_index, nc.content, nc.embedding, np.value as path
from note_paths np
join note_versions nv on np.id = nv.path_id and np.version_count = nv.version
join note_version_chunks nc on nv.id = nc.version_id
where nc.embedding is not null and np.hidden_by is null;

-- name: GetAllLiveNoteChunksWithEmbeddings :many
select nc.version_id, nc.chunk_index, nc.content, nc.embedding, np.value as path
from note_paths np
join note_versions nv on np.id = nv.path_id
join release_note_versions rnv on nv.id = rnv.note_version_id
join releases r on rnv.release_id = r.id
join note_version_chunks nc on nv.id = nc.version_id
where nc.embedding is not null and r.is_live = true;

-- name: GetActiveGoogleOAuthCredentials :one
select * from google_oauth_credentials where active = true limit 1;

-- name: GetActiveGitHubOAuthCredentials :one
select * from github_oauth_credentials where active = true limit 1;

-- name: ListGoogleOAuthCredentials :many
select * from google_oauth_credentials order by created_at desc;

-- name: ListGitHubOAuthCredentials :many
select * from github_oauth_credentials order by created_at desc;

-- name: GetGoogleOAuthCredentials :one
select * from google_oauth_credentials where id = ?;

-- name: GetGitHubOAuthCredentials :one
select * from github_oauth_credentials where id = ?;

-- name: GetActiveOIDCCredentials :one
select * from oidc_credentials where active = true limit 1;

-- name: ListOIDCCredentials :many
select * from oidc_credentials order by created_at desc;

-- name: GetOIDCCredentials :one
select * from oidc_credentials where id = ?;

-- ============================================
-- Change Webhooks
-- ============================================

-- name: ListWebhooks :many
select * from change_webhooks where disabled_at is null order by created_at;

-- name: ListEnabledWebhooks :many
select * from change_webhooks where enabled = true and disabled_at is null;

-- name: WebhookByID :one
select * from change_webhooks where id = ? and disabled_at is null;

-- name: ListWebhookDeliveries :many
select * from change_webhook_deliveries
where webhook_id = ?
order by created_at desc
limit ?;

-- ============================================
-- Cron Webhooks
-- ============================================

-- name: ListCronWebhooks :many
select * from cron_webhooks where disabled_at is null order by created_at;

-- name: ListEnabledCronWebhooks :many
select * from cron_webhooks where enabled = true and disabled_at is null;

-- name: CronWebhookByID :one
select * from cron_webhooks where id = ? and disabled_at is null;

-- name: WebhookDeliveryTraceByID :one
select trace, depth_reached from change_webhook_deliveries where id = ?;

-- name: CronWebhookDeliveryTraceByID :one
select trace, depth_reached from cron_webhook_deliveries where id = ?;

-- name: ListDeliveryTraces :many
-- Chain overview: one row per trace, rolled up across both delivery kinds, with
-- the number of note versions the chain produced. only_productive drops chains
-- that wrote nothing: a cron role that finds no work still runs, and those runs
-- otherwise bury the ones that did something.
select * from (
select t.trace as trace,
       cast(strftime('%s', min(t.created_at)) as integer) as started_at_unix,
       cast(strftime('%s', max(t.created_at)) as integer) as last_at_unix,
       count(*) as deliveries,
       cast(max(t.depth_reached) as integer) as depth_reached,
       cast(sum(t.writes) as integer) as writes
  from (select c.trace as trace, c.created_at as created_at,
               c.depth_reached as depth_reached,
               (select count(*) from note_version_delivery_attribution a
                 where a.delivery_kind = 'change' and a.delivery_id = c.id) as writes
          from change_webhook_deliveries c where c.trace is not null
        union all
        select r.trace as trace, r.created_at as created_at,
               r.depth_reached as depth_reached,
               (select count(*) from note_version_delivery_attribution a
                 where a.delivery_kind = 'cron' and a.delivery_id = r.id) as writes
          from cron_webhook_deliveries r where r.trace is not null) as t
 group by t.trace) as g
 where cast(sqlc.arg(only_productive) as integer) = 0 or g.writes > 0
 order by g.started_at_unix desc
 limit sqlc.arg(lim);

-- name: ListTraceCosts :many
-- The raw cost objects of one chain. Summing happens in Go: the keys are
-- whatever the agents reported, and sqlc cannot type json_each.
select c.costs as costs
  from change_webhook_deliveries c
 where c.trace = sqlc.arg(trace) and c.costs is not null
 union all
select r.costs as costs
  from cron_webhook_deliveries r
 where r.trace = sqlc.arg(trace) and r.costs is not null;

-- name: ListDeliveriesByTrace :many
-- Every hop of one chain, in causal order, across both delivery kinds.
select 'change' as kind, c.id as id, c.webhook_id as webhook_id, c.status as status,
       c.response_status as response_status, c.attempt as attempt, c.duration_ms as duration_ms,
       c.costs as costs, c.created_at as created_at,
       c.started_at as started_at, c.completed_at as completed_at,
       c.parent_kind as parent_kind, c.parent_id as parent_id,
       c.depth_reached as depth_reached
  from change_webhook_deliveries c where c.trace = sqlc.arg(trace)
 union all
select 'cron' as kind, r.id as id, r.cron_webhook_id as webhook_id, r.status as status,
       r.response_status as response_status, r.attempt as attempt, r.duration_ms as duration_ms,
       r.costs as costs, r.created_at as created_at,
       r.started_at as started_at, r.completed_at as completed_at,
       r.parent_kind as parent_kind, r.parent_id as parent_id,
       r.depth_reached as depth_reached
  from cron_webhook_deliveries r where r.trace = sqlc.arg(trace)
 order by created_at, id;

-- name: DeliveryLogs :many
-- The run log one delivery stored, fetched per hop rather than alongside the
-- chain: it is capped at 64 KB, and a chain listing has no use for it. Returns
-- at most one row; the kind guard picks the table ids are unique within.
select c.logs as logs from change_webhook_deliveries c
 where sqlc.arg(kind) = 'change' and c.id = sqlc.arg(delivery_id)
 union all
select r.logs as logs from cron_webhook_deliveries r
 where sqlc.arg(kind) = 'cron' and r.id = sqlc.arg(delivery_id);

-- name: ListDeliveryWrites :many
-- What a delivery wrote: the note versions attributed to it. This is what makes
-- a chain readable, since the writes of one hop are the trigger of the next.
select np.value as path, nv.version as version, nv.id as version_id
  from note_version_delivery_attribution a
  join note_versions nv on nv.id = a.note_version_id
  join note_paths np on np.id = nv.path_id
 where a.delivery_kind = sqlc.arg(kind)
   and a.delivery_id = sqlc.arg(delivery_id)
 order by np.value;

-- name: ListCronWebhookDeliveries :many
select * from cron_webhook_deliveries
where cron_webhook_id = ?
order by created_at desc
limit ?;

-- name: ListCronWebhooksDueForExecution :many
select * from cron_webhooks
where enabled = true
  and disabled_at is null
  and next_run_at <= datetime('now');

-- ============================================
-- Webhook Delivery Logs
-- ============================================

-- name: WebhookDeliveryLogByDelivery :one
select * from webhook_delivery_logs
where kind = ? and delivery_id = ?
order by created_at desc
limit 1;

-- ============================================
-- Frontmatter Patches
-- ============================================

-- name: ListFrontmatterPatches :many
select * from note_frontmatter_patches order by priority asc, id asc;

-- name: ListEnabledFrontmatterPatches :many
select * from note_frontmatter_patches where enabled = true order by priority asc, id asc;

-- name: FrontmatterPatchByID :one
select * from note_frontmatter_patches where id = ?;

-- name: CountNotePaths :one
select count(*) from note_paths where hidden_by is null;

-- name: CountAllNotePaths :one
select count(*) from note_paths;

-- name: CountVisibleNotePaths :one
select count(*) from note_paths where hidden_by is null;

-- name: CountNoteVersions :one
select count(*) from note_versions;

-- name: SumNoteAssetsSizes :one
select cast(coalesce(sum(size), 0) as integer) from note_assets;

-- name: CountNoteAssets :one
select count(*) from note_assets;

-- ============================================
-- User Tokens
-- ============================================

-- name: UserTokenByHash :one
select * from user_tokens
where token_hash = ?
  and revoked_at is null
  and (expires_at is null or expires_at > datetime('now'));

-- Unfiltered on purpose: the owner-token seeder must tell a revoked row apart
-- from a missing one, because a revoked row is deliberate and stays revoked.
-- name: UserTokenByHashAny :one
select * from user_tokens
where token_hash = ?;

-- name: ListUserTokensByUserID :many
select * from user_tokens
where user_id = ?
order by created_at desc;

-- name: ListUserTokensFiltered :many
select * from user_tokens
where (user_id = sqlc.narg(user_id) or sqlc.narg(user_id) is null)
order by created_at desc;

-- name: UserTokenByID :one
select * from user_tokens
where id = ?;

-- name: CountActiveUserTokensByUserID :one
select count(*) from user_tokens
where user_id = ?
  and revoked_at is null
  and (expires_at is null or expires_at > datetime('now'));

-- name: ListFormSubmits :many
select fs.id, fs.note_version_id, fs.form_id, fs.user_id, fs.ip, fs.status, fs.created_at,
       fs.processed_at, fs.processed_by, fs.comment
from form_submits fs
join note_versions nv on nv.id = fs.note_version_id
where (sqlc.narg(note_path_id) is null or nv.path_id = sqlc.narg(note_path_id))
  and (sqlc.narg(form_id) is null or fs.form_id = sqlc.narg(form_id))
  and (sqlc.narg(status) is null or fs.status = sqlc.narg(status))
  and (sqlc.narg(processed_filter) is null
       or (sqlc.narg(processed_filter) = 1 and fs.processed_at is not null)
       or (sqlc.narg(processed_filter) = 0 and fs.processed_at is null))
  and (sqlc.narg(created_at_gte) is null or fs.created_at >= sqlc.narg(created_at_gte))
  and (sqlc.narg(created_at_lte) is null or fs.created_at <= sqlc.narg(created_at_lte))
order by fs.created_at desc, fs.id desc
limit sqlc.arg(lim) offset sqlc.arg(off);

-- name: CountFormSubmits :one
select count(*)
from form_submits fs
join note_versions nv on nv.id = fs.note_version_id
where (sqlc.narg(note_path_id) is null or nv.path_id = sqlc.narg(note_path_id))
  and (sqlc.narg(form_id) is null or fs.form_id = sqlc.narg(form_id))
  and (sqlc.narg(status) is null or fs.status = sqlc.narg(status))
  and (sqlc.narg(processed_filter) is null
       or (sqlc.narg(processed_filter) = 1 and fs.processed_at is not null)
       or (sqlc.narg(processed_filter) = 0 and fs.processed_at is null))
  and (sqlc.narg(created_at_gte) is null or fs.created_at >= sqlc.narg(created_at_gte))
  and (sqlc.narg(created_at_lte) is null or fs.created_at <= sqlc.narg(created_at_lte));

-- name: GetFormSubmitByID :one
select id, note_version_id, form_id, user_id, ip, status, created_at,
       processed_at, processed_by, comment
from form_submits where id = ?;

-- name: CountUnprocessedFormSubmits :one
select count(*) from form_submits where processed_at is null;

-- name: GetFormStringValuesBySubmitID :many
select field_name, value from form_string_values where submit_id = ?;

-- name: GetFormIntValuesBySubmitID :many
select field_name, value from form_int_values where submit_id = ?;

-- name: GetFormBoolValuesBySubmitID :many
select field_name, value from form_bool_values where submit_id = ?;

-- name: GetNotesWithFormSubmits :many
select np.id as path_id, np.value as path, MAX(fs.created_at) as last_submit_at, COUNT(*) as submit_count
from form_submits fs
join note_versions nv on nv.id = fs.note_version_id
join note_paths np on np.id = nv.path_id
group by np.id
order by last_submit_at desc;

-- name: GetTgBotDefaultHandler :one
select default_handler from tg_bots where id = ?;

-- name: GetTgBotDefaultCanvas :one
select default_canvas from tg_bots where id = ?;

-- name: GetTgUserCurrentHandler :one
select value from tg_user_current_handlers
 where bot_id = ? and business_connection_id = ? and user_id = ?;

-- name: GetTgUserNavigationState :one
select value from tg_user_navigation_states
 where bot_id = ? and business_connection_id = ? and user_id = ?;

-- name: GetTgUserCanvasState :one
select bot_id, business_connection_id, user_id, canvas_path, current_node, stack, last_media, message_id, updated_at
  from tg_user_canvas_states
 where bot_id = ? and business_connection_id = ? and user_id = ?;

-- name: NoteVersionHistoryByPath :many
select nv.id, nv.version, length(nv.content) as content_length, nv.created_at
  from note_versions nv
  join note_paths np on np.id = nv.path_id
 where np.value = ?
 order by nv.version desc
 limit ? offset ?;

-- name: CountNoteVersionsByPath :one
select count(*) from note_versions nv
  join note_paths np on np.id = nv.path_id
 where np.value = ?;

-- name: GetSecret :one
select id, key, value_crypt, created_at, created_by from secrets where key = ?;

-- name: ListSecretKeys :many
select key from secrets where key like ? order by key;

-- name: GetChartData :one
select version_id, chart_hash, data_json, fetched_at, last_error, last_error_at
  from chart_data_cache
 where version_id = ? and chart_hash = ?;

-- name: NotesReloadSignal :one
select
  cast(coalesce((select max(id) from note_versions), 0) as integer) as version_gen,
  cast((select count(*) from note_paths where hidden_by is not null) as integer) as hidden_count;

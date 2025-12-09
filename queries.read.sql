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

-- name: AllLatestNotes :many
select value as path, p.id as path_id, v.id as version_id, content, v.created_at
  from note_paths p
  join note_versions v on p.id = v.path_id and p.version_count = v.version
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
   and bm.status = 'active_patron'
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

-- name: NoteVersionByID :one
select p.value as path, path_id, v.id as version_id, content, v.created_at
  from note_versions v
  join note_paths p on v.path_id = p.id
 where v.id = ?
 limit 1;

-- name: AcmeCertByKey :one
select value from acme_certs where key = ?;

-- name: ApiKeyByValue :one
select * from api_keys where value = ? and disabled_at is null limit 1;

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
select value as path, p.id as path_id, v.id as version_id, content, v.created_at
  from note_paths p
  join note_versions v on p.id = v.path_id
  join release_note_versions rnv on v.id = rnv.note_version_id
  join releases r on rnv.release_id = r.id
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

-- name: NotionIntegration :one
select *
  from notion_integrations
 where id = ?;

-- name: AllNotionIntegrations :many
select * from notion_integrations order by id;

-- name: GetLatestConfig :one
select *
  from config_versions
 order by id desc
 limit 1;

-- name: ListAllConfigVersions :many
select *
  from config_versions
 order by id desc
 limit 50;

-- name: ListNotePathsLike :many
select * from note_paths
 where value like ?
 order by id;

-- name: ListNotePathsByValues :many
select * from note_paths
 where value in (sqlc.slice('paths'))
 order by id;

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

-- name: ListSheduledTelegarmPublishNoteIDs :many
select n.note_path_id
  from telegram_publish_notes n
  join note_paths p on n.note_path_id = p.id
  -- the note must be tagged with at least one bot chat
  join telegram_publish_note_tags nt on n.note_path_id = nt.note_path_id
  join telegram_publish_chats pc on nt.tag_id = pc.tag_id
  where p.hidden_by is null
   and publish_at <= datetime('now')
   and published_at is null
   and last_error is null;

-- name: ListSheduledTelegarmAccountPublishNoteIDs :many
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
   and a.enabled = 1;

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
     , count(case when received > 0 then 1 end) as retry_count
  from goqite
 where queue = ?
 group by queue;

-- name: ListGoqiteAllQueueStats :many
select queue
     , count(*) as total_jobs
     , count(case when received = 0 then 1 end) as pending_count
     , count(case when received > 0 then 1 end) as retry_count
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

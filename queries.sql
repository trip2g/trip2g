-- name: InsertNotePath :one
insert into note_paths (value, value_hash, latest_content_hash)
values (?, ?, ?)
on conflict(value) do update set value = excluded.value
returning id, version_count, latest_content_hash;

-- name: IncrementNoteVersionCount :one
update note_paths
   set version_count = version_count + 1
     , latest_content_hash = ?
 where id = ?
returning version_count;

-- name: InsertNoteVersion :exec
insert into note_versions (path_id, version, content)
values (?, ?, ?);

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
select value as path, p.id as path_id, v.id as version_id, content
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

-- name: UserByEmail :one
select * from users where email = lower(?);

-- name: InsertUserWithEmail :one
insert into users (email) values (lower(?))
returning *;

-- name: InsertUserWithTgUserID :one
insert into users (tg_user_id)
values (?)
returning *;

-- name: UserByID :one
select * from users where id = ?;

-- name: CountActiveSignInCodes :one
select count(*) from sign_in_codes
 where user_id = ?
   and created_at > datetime('now', '-5 minutes');

-- name: InsertSignInCode :exec
insert into sign_in_codes (user_id, code)
values (?, ?);

-- name: VerifySignInCode :one
select user_id
  from sign_in_codes c
  join users u on c.user_id = u.id
  where u.email = ?
    and c.code = ?
    and c.created_at > datetime('now', '-5 minutes')
  limit 1;

-- name: DeleteSignInCodesByUserID :exec
delete from sign_in_codes
 where user_id = ?;

-- name: DeleteOffer :one
update offers
   set ends_at = datetime('now')
 where id = ?
returning *;

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
  from tg_bot_chats bc
  join tg_chat_members m on bc.id = m.chat_id
  join tg_chat_subgraph_accesses a on a.chat_id = bc.id
  join subgraphs s on s.id = a.subgraph_id
 where m.user_id = ?
   and bc.removed_at is null
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

-- name: InsertSubgraph :exec
insert into subgraphs (name)
values (?)
on conflict(name) do update set hidden = false;

-- name: UpdateAdminSubgraph :one
update subgraphs
   set color = ?, hidden = ?
 where id = ?
returning *;

-- name: CreateUserSubgraphAccess :one
insert into user_subgraph_accesses (user_id, subgraph_id, purchase_id, expires_at)
values (?, ?, ?, ?)
returning *;

-- name: ListAllUserSubgraphAccesses :many
select * from user_subgraph_accesses order by id desc;

-- name: UserSubgraphAccessByID :one
select *
  from user_subgraph_accesses
 where id = ?;

-- name: UpdateUserSubgraphAccess :one
update user_subgraph_accesses
   set expires_at = ?
     , subgraph_id = ?
 where id = ?
returning *;

-- name: CreateRevoke :one
insert into revokes (target_type, target_id, by_id, reason)
values (?, ?, ?, ?)
returning id;

-- name: RevokeUserSubgraphAccess :exec
update user_subgraph_accesses
   set revoke_id = ?
 where id = ?;

-- name: SubgraphByID :one
select * from subgraphs where id = ?;

-- name: SubgraphByName :one
select * from subgraphs where name = ?;

-- name: ListAllSubgraphs :many
select * from subgraphs order by id;

-- name: ListAllUserBans :many
select * from user_bans;

-- name: BanUser :exec
insert into user_bans (user_id, banned_by, reason)
values (?, ?, ?);

-- name: UnbanUser :exec
delete from user_bans where user_id = ?;

-- name: AdminByUserID :one
select * from admins where user_id = ?;

-- name: InsertUserNoteView :exec
insert into user_note_views (user_id, version_id, referer_version_id) values (?, ?, ?);

-- name: UpsertUserNoteDailyView :one
-- Unfortunately, sqlc cannot generate a parameter for greatest(count + 1, sqlc.arg(max_count)).
insert into user_note_daily_view_counts (user_id, path_id) values (?, ?)
on conflict(user_id, path_id) do update set count = count + 1
returning count;

-- name: IncreaseUserNoteViewCount :exec
update users
   set note_view_count = note_view_count + 1
 where id = ?;

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

-- name: InsertPurchase :exec
insert into purchases (id, email, offer_id, payment_provider, payment_data, price_usd, status)
values (?, ?, ?, ?, ?, ?, ?);

-- name: PurchaseByID :one
select * from purchases where id = ?;

-- name: UpdatePurchaseStatus :exec
update purchases
   set status = ?
     , payment_data = ?
 where id = ?;

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

-- name: InsertNoteAsset :one
insert into note_assets (absolute_path, file_name, sha256_hash, size)
values (?, ?, ?, ?)
returning *;

-- name: NoteAssetByPathAndHash :one
select * from note_assets
 where absolute_path = ?
   and sha256_hash = ?
 limit 1;

-- name: UpsertNoteVersionAsset :exec
insert into note_version_assets (asset_id, version_id, path)
values (?, ?, ?)
on conflict (asset_id, version_id, path) do update set created_at = datetime('now');

-- name: NoteAssetByAbsolutePathAndSha256Hash :one
select * from note_assets
 where absolute_path = ?
   and sha256_hash = ?
 limit 1;

-- name: NoteVersionByID :one
select p.value as path, path_id, v.id as version_id, content
  from note_versions v
  join note_paths p on v.path_id = p.id
 where v.id = ?
 limit 1;

-- name: AcmeCertByKey :one
select value from acme_certs where key = ?;

-- name: InsertAcmeCert :exec
insert into acme_certs (key, value)
values (?, ?);

-- name: DeleteAcmeCert :exec
delete from acme_certs where key = ?;

-- name: ApiKeyByValue :one
select * from api_keys where value = ? and disabled_at is null limit 1;

-- name: InsertAPIKey :one
insert into api_keys (value, created_by, description)
values (?, ?, ?)
returning *;

-- name: DisableApiKey :one
update api_keys
  set disabled_by = ?, disabled_at = datetime('now')
 where id = ?
returning *;

-- name: ListAllAPIKeys :many
select * from api_keys order by created_by, created_at desc;

-- name: InsertAPIKeyLog :exec
insert into api_key_logs (api_key_id, ip_id, action_id)
values (?,
  (select id from api_key_log_ips where value = sqlc.arg(ip)),
  (select id from api_key_log_actions where name = sqlc.arg(action)));

-- name: UpsertAPIKeyLogAction :exec
insert into api_key_log_actions (name)
values (?)
on conflict(name) do nothing;

-- name: UpsertAPIKeyLogIP :exec
insert into api_key_log_ips (value)
values (?)
on conflict(value) do nothing;

-- name: ListAPIKeyLogsByAPIKeyID :many
select l.created_at, a.name as action_name, i.value as ip
  from api_key_logs l
  join api_key_log_actions a on l.action_id = a.id
  join api_key_log_ips i on l.ip_id = i.id
 where l.api_key_id = ?
 order by l.created_at desc;

-- name: InsertRelease :one
insert into releases (created_by, title, home_note_version_id, is_live)
values (?, ?, ?, ?)
returning *;

-- name: InsertReleaseNoteVersion :exec
insert into release_note_versions (release_id, note_version_id)
values (?, ?);

-- name: ChangeLiveRelease :exec
update releases set is_live = (sqlc.arg(id) = id);

-- name: ListAllReleases :many
select *
  from releases
 order by is_live asc, created_at desc;

-- name: ReleaseByID :one
select *
  from releases
 where id = ?;

-- name: AllLiveNotes :many
select value as path, p.id as path_id, v.id as version_id, content
  from note_paths p
  join note_versions v on p.id = v.path_id
  join release_note_versions rnv on v.id = rnv.note_version_id
  join releases r on rnv.release_id = r.id
 where r.is_live = true;

-- name: AllLiveNoteAssets :many
with ranked_assets as (
  select
    v.id as version_id,
    na.id as asset_id,
    a.path,
    row_number() over (
      partition by v.id, a.path
      order by a.created_at desc
    ) as rn
  from note_paths p
  join note_versions v on p.id = v.path_id
  join note_version_assets a on v.id = a.version_id
  join note_assets na on a.asset_id = na.id
  join release_note_versions rnv on v.id = rnv.note_version_id
  join releases r on rnv.release_id = r.id
 where r.is_live = true
)
select version_id, path, sqlc.embed(note_assets)
from ranked_assets
join note_assets on ranked_assets.asset_id = note_assets.id
where rn = 1;

-- name: NoteGraphPositionByPathID :one
select graph_position_x as x, graph_position_y as y
  from note_paths
 where id = ?
 limit 1;

-- name: UpdateNoteGraphPositionByPathID :exec
update note_paths
   set graph_position_x = ?
     , graph_position_y = ?
 where id = ?;

-- name: ListAllAdmins :many
select * from admins a order by user_id desc;

-- name: ListSubgraphIDsByOfferID :many
select subgraph_id
  from offer_subgraphs
 where offer_id = ?
 order by subgraph_id;

-- name: ListAllOffers :many
select * from offers order by id;

-- name: InsertOffer :one
insert into offers (public_id, lifetime, price_usd, starts_at, ends_at)
values (?, ?, ?, ?, ?)
returning *;

-- name: InsertOfferSubgraph :exec
insert into offer_subgraphs (offer_id, subgraph_id)
values (?, ?);

-- name: UpdateOffer :one
update offers
   set lifetime = coalesce(sqlc.narg(lifetime), lifetime)
     , price_usd = coalesce(sqlc.narg(price_usd), price_usd)
     , starts_at = coalesce(sqlc.narg(starts_at), starts_at)
     , ends_at = coalesce(sqlc.narg(ends_at), ends_at)
 where id = sqlc.arg(id)
returning *;

-- name: DeleteOfferSubgraphs :exec
delete from offer_subgraphs where offer_id = ?;

-- name: ListAllPurchases :many
select * from purchases order by created_at desc;

-- name: InsertAdmin :one
insert into admins (user_id, granted_by)
values (?, ?)
returning *;

-- name: HideNotePath :exec
update note_paths
   set hidden_by = ?
     , hidden_at = datetime('now')
 where value = ?;

-- name: UnhideNotePath :exec
update note_paths
   set hidden_by = null
     , hidden_at = null
 where value = ?;

-- name: ListAllRedirects :many
select * from redirects order by is_regex;

-- name: InsertRedirect :one
insert into redirects (created_by, pattern, ignore_case, is_regex, target)
values (?, ?, ?, ?, ?)
returning *;

-- name: UpdateRedirect :one
update redirects
   set pattern = ?
     , ignore_case = ?
     , is_regex = ?
     , target = ?
 where id = ?
returning *;

-- name: DeleteRedirect :exec
delete from redirects where id = ?;

-- name: RedirectByID :one
select * from redirects where id = ?;

-- name: ListAllNotFoundIgnoredPatterns :many
select * from not_found_ignored_patterns;

-- name: ListAllNotFoundPaths :many
select * from not_found_paths order by total_hits desc;

-- name: ListActiveNotFoundIPHits :many
select * from not_found_ip_hits where last_hit_at > datetime('now', '-7 days');

-- name: UpsertNotFoundHit :exec
insert into not_found_paths (path)
values (?)
on conflict (path) do update set total_hits = total_hits + 1, last_hit_at = datetime('now');

-- name: UpsertNotFoundIPHit :exec
insert into not_found_ip_hits (ip, total_hits)
values (?, ?)
on conflict(ip) do
update set total_hits = excluded.total_hits, last_hit_at = datetime('now');

-- name: InsertNotFoundIgnoredPattern :one
insert into not_found_ignored_patterns (pattern, created_by)
values (?, ?)
returning *;

-- name: UpdateNotFoundIgnoredPattern :one
update not_found_ignored_patterns
set pattern = ?
where id = ?
returning *;

-- name: DeleteNotFoundIgnoredPattern :exec
delete from not_found_ignored_patterns where id = ?;

-- name: NotFoundIgnoredPatternByID :one
select * from not_found_ignored_patterns where id = ?;

-- name: NotFoundPathByID :one
select * from not_found_paths where id = ?;

-- name: ResetNotFoundPathTotalHits :one
update not_found_paths
set total_hits = 1, last_hit_at = datetime('now')
where id = ?
returning *;

-- name: ListEnabledTgBots :many
select * from tg_bots where enabled = true;

-- name: InsertTgUserProfile :exec
insert into tg_user_profiles (chat_id, bot_id, first_name, last_name, username, sha256_hash)
values (?, ?, ?, ?, ?, ?)
on conflict(sha256_hash) do nothing;

-- name: UpsertTgUserState :exec
insert into tg_user_states (chat_id, bot_id, value, data, update_count)
values (?, ?, ?, ?, ?)
on conflict(chat_id, bot_id) do update set
  value = excluded.value,
  data = excluded.data,
  update_count = excluded.update_count,
  updated_at = current_timestamp;

-- name: TgUserStateByBotIDAndChatID :one
select *
  from tg_user_states
 where bot_id = ?
   and chat_id = ?
 limit 1;

-- name: UpsertTgBotChat :exec
insert into tg_bot_chats (id, chat_type, chat_title)
values (?, ?, ?)
on conflict(id) do update set
  chat_type = excluded.chat_type,
  chat_title = excluded.chat_title,
  removed_at = null;

-- name: MarkTgBotChatRemoved :exec
update tg_bot_chats
set removed_at = current_timestamp
where id = ?;

-- name: InsertTgChatMember :exec
insert into tg_chat_members (user_id, chat_id)
values (?, ?)
on conflict(user_id, chat_id) do nothing;

-- name: RemoveTgChatMember :exec
delete from tg_chat_members
where user_id = ? and chat_id = ?;

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

-- name: InsertTgBot :one
insert into tg_bots (token, name, description, created_by)
values (?, ?, ?, ?)
returning *;

-- name: UpdateTgBot :one
update tg_bots
set description = coalesce(sqlc.narg(description), description),
    enabled = coalesce(sqlc.narg(enabled), enabled)
where id = sqlc.arg(id)
returning *;

-- name: TgBotChatsByBotID :many
select * from tg_bot_chats
where id in (
  select distinct chat_id from tg_user_states where bot_id = ?
)
  and (sqlc.arg(include_removed) = true or removed_at is null)
order by added_at desc;

-- name: TgBotChatsByBotIDCount :one
select count(*) from tg_bot_chats
where id in (
  select distinct chat_id from tg_user_states where bot_id = ?
)
  and (sqlc.arg(include_removed) = true or removed_at is null);

-- name: AllTgBotChats :many
select * from tg_bot_chats
where (sqlc.arg(include_removed) = true or removed_at is null)
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
select * from tg_chat_subgraph_accesses
where chat_id = ?
order by created_at desc;

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

-- name: InsertTgChatSubgraphAccess :one
insert into tg_chat_subgraph_accesses (chat_id, subgraph_id)
values (?, ?)
returning *;

-- name: DeleteTgChatSubgraphAccess :exec
delete from tg_chat_subgraph_accesses
where id = ?;

-- name: DeleteTgChatSubgraphAccessesByChatID :exec
delete from tg_chat_subgraph_accesses
where chat_id = ?;

-- name: TgBotChat :one
select * from tg_bot_chats
where id = ?;

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

-- name: InsertWaitListEmailRequest :exec
insert into wait_list_email_requests (email, note_path_id, ip)
values (?, ?, ?);

-- name: InsertWaitListTgBotRequest :exec
insert into wait_list_tg_bot_requests (bot_id, chat_id, note_path_id)
values (?, ?, ?);

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

-- name: InsertPatreonCredentials :one
insert into patreon_credentials (created_by, creator_access_token)
values (?, ?)
returning *;

-- name: SoftDeletePatreonCredentials :one
update patreon_credentials
set deleted_at = current_timestamp, deleted_by = ?
where id = ? and deleted_at is null
returning *;

-- name: RestorePatreonCredentials :one
update patreon_credentials
set deleted_at = null, deleted_by = null
where id = ? and deleted_at is not null
returning *;

-- name: ListActivePatreonCredentials :many
select *
  from patreon_credentials
 where deleted_by is null;

-- name: InsertPatreonCampaign :exec
insert into patreon_campaigns (credentials_id, campaign_id, attributes)
values (?, ?, ?);

-- name: UpsertPatreonCampaign :exec
insert into patreon_campaigns (credentials_id, campaign_id, attributes)
values (?, ?, ?)
on conflict(credentials_id, campaign_id) do update set
  attributes = excluded.attributes,
  missed_at = null;


-- name: GetPatreonCampaignsByCredentialsID :many
select * from patreon_campaigns
where credentials_id = ?
order by created_at desc;

-- name: UpsertPatreonTier :exec
insert into patreon_tiers (campaign_id, tier_id, title, amount_cents, attributes)
values (?, ?, ?, ?, ?)
on conflict(campaign_id, tier_id) do update set
  title = excluded.title,
  amount_cents = excluded.amount_cents,
  attributes = excluded.attributes,
  missed_at = null;


-- name: GetPatreonTiersByCampaignID :many
select *
  from patreon_tiers
 where campaign_id = ?
 order by amount_cents desc;

-- name: UpsertPatreonMember :exec
insert into patreon_members (patreon_id, campaign_id, status, email)
values (?, ?, ?, ?)
on conflict(patreon_id, campaign_id) do update set
  status = excluded.status,
  email = excluded.email;

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

-- name: UpdatePatreonMemberUserID :exec
update patreon_members
   set user_id = ?
 where id = ?;

-- name: UpdatePatreonCredentialsSyncedAt :exec
update patreon_credentials
set synced_at = current_timestamp
where id = ?;

-- name: UpdatePatreonCredentialsWebhookSecret :exec
update patreon_credentials
set webhook_secret = ?
where id = ?;

-- name: ClearPatreonCredentialsWebhookSecret :exec
update patreon_credentials
set webhook_secret = null
where id = ?;

-- Boosty credentials

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

-- name: InsertBoostyCredentials :one
insert into boosty_credentials (created_by, auth_data, device_id, blog_name)
values (?, ?, ?, ?)
returning *;

-- name: SoftDeleteBoostyCredentials :one
update boosty_credentials
set deleted_at = current_timestamp, deleted_by = ?
where id = ? and deleted_at is null
returning *;

-- name: RestoreBoostyCredentials :one
update boosty_credentials
set deleted_at = null, deleted_by = null
where id = ? and deleted_at is not null
returning *;

-- name: UpdateBoostyCredentials :one
update boosty_credentials
set auth_data = ?, device_id = ?, blog_name = ?
where id = ?
returning *;

-- name: UpdateBoostyCredentialsTokens :one
update boosty_credentials
set auth_data = ?, expires_at = ?
where id = ?
returning *;

-- name: UpdateBoostyCredentialsSyncedAt :exec
update boosty_credentials
set synced_at = current_timestamp
where id = ?;

-- Boosty tiers

-- name: GetBoostyTiers :many
select * from boosty_tiers
order by created_at;

-- name: InsertBoostyTier :exec
insert into boosty_tiers (credentials_id, boosty_id, name, data)
values (?, ?, ?, ?);

-- name: UpsertBoostyTier :exec
insert into boosty_tiers (credentials_id, boosty_id, name, data)
values (?, ?, ?, ?)
on conflict(credentials_id, boosty_id) do update set
  name = excluded.name,
  data = excluded.data,
  missed_at = null;

-- Boosty tier subgraphs

-- name: GetSubgraphsByBoostyTierID :many
select s.*
from subgraphs s
join boosty_tier_subgraphs bts on s.id = bts.subgraph_id
where bts.tier_id = ?;

-- name: DeleteBoostyTierSubgraphsByTierID :exec
delete from boosty_tier_subgraphs where tier_id = ?;

-- name: InsertBoostyTierSubgraph :exec
insert into boosty_tier_subgraphs (tier_id, subgraph_id, created_by)
values (?, ?, ?);

-- Boosty members

-- name: GetBoostyMembers :many
select * from boosty_members
order by created_at;

-- name: InsertBoostyMember :exec
insert into boosty_members (boosty_id, email, status, data)
values (?, ?, ?, ?);

-- name: UpsertBoostyMember :exec
insert into boosty_members (credentials_id, boosty_id, email, status, data, current_tier_id)
values (?, ?, ?, ?, ?, ?)
on conflict(credentials_id, boosty_id) do update set
  email = excluded.email,
  status = excluded.status,
  data = excluded.data,
  current_tier_id = excluded.current_tier_id,
  missed_at = null;

-- name: GetBoostyMemberByEmail :one
select * from boosty_members
where email = ? and status = 'active'
order by created_at desc
limit 1;

-- name: UpdateBoostyMemberUserID :exec
update boosty_members
set user_id = ?
where id = ?;

-- name: MarkBoostyMembersAsMissed :exec
update boosty_members
set missed_at = datetime('now')
where missed_at is null
  and boosty_id not in (sqlc.slice('boosty_ids'));

-- name: MarkBoostyTiersAsMissed :exec
update boosty_tiers
set missed_at = datetime('now')
where credentials_id = ?
  and missed_at is null
  and boosty_id not in (sqlc.slice('boosty_ids'));

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

-- name: MarkPatreonMembersAsMissed :exec
update patreon_members
set status = 'missed'
where campaign_id = ?
  and patreon_id not in (
    select json_extract(value, '$') 
    from json_each(?)
  );

-- name: SetPatreonMemberCurrentTier :exec
update patreon_members
set current_tier_id = ?
where id = ?;

-- name: GetPatreonTierByTierID :one
select * from patreon_tiers
where campaign_id = ? and tier_id = ?
limit 1;

-- name: DeletePatreonTierSubgraphsByTierID :exec
delete from patreon_tier_subgraphs
where tier_id = ?;

-- name: InsertPatreonTierSubgraph :exec
insert into patreon_tier_subgraphs (tier_id, subgraph_id, created_by)
values (?, ?, ?);

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

-- name: InsertUserFavoriteNote :exec
insert into user_favorite_notes (user_id, note_version_id)
values (?, ?) on conflict do nothing;

-- name: DeleteUserFavoriteNote :exec
delete from user_favorite_notes
where user_id = ? and note_version_id = ?;

-- name: ListUserFavoriteNotes :many
select nv.path_id, nv.id as version_id
  from user_favorite_notes ufn
  join note_versions nv on ufn.note_version_id = nv.id
  join note_paths np on nv.path_id = np.id
 where ufn.user_id = ? and np.hidden_by is null
 order by ufn.created_at desc;

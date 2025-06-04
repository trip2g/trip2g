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

-- name: AllNoteVersions :many
select * from note_versions order by path_id, version;

-- name: AllNoteVersionsByPathID :many
select * from note_versions
 where path_id = ?
 order by version desc;

-- name: AllLatestNotes :many
select value as path, p.id as path_id, v.id as version_id, content
  from note_paths p
  join note_versions v on p.id = v.path_id and p.version_count = v.version;

-- name: AllLatestNoteAssets :many
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
  join note_versions v on p.id = v.path_id and p.version_count = v.version
  join note_version_assets a on v.id = a.version_id
  join note_assets na on a.asset_id = na.id
)
select version_id, path, sqlc.embed(note_assets)
from ranked_assets
join note_assets on ranked_assets.asset_id = note_assets.id
where rn = 1;

-- name: UserByEmail :one
select * from users where email = lower(?);

-- name: InsertUser :one
insert into users (email) values (lower(?))
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

-- name: InsertSubgraph :exec
insert into subgraphs (name)
values (?)
on conflict(name) do nothing;

-- name: UpdateAdminSubgraph :one
update subgraphs
   set color = ?
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
insert into purchases (id, email, offer_id, payment_provider, payment_data, price_usd)
values (?, ?, ?, ?, ?, ?);

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
insert into note_assets (absolute_path, file_name, sha256_hash, content_type, size)
values (?, ?, ?, ?, ?)
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

-- name: ApiKeyIDByValue :one
select id from api_keys where value = ? and disabled_at is null limit 1;

-- name: InsertApiKey :one
insert into api_keys (value, created_by, description)
values (?, ?, ?)
returning *;

-- name: DisableApiKey :one
update api_keys
  set disabled_by = ?, disabled_at = datetime('now')
 where id = ?
returning *;

-- name: ListAllApiKeys :many
select * from api_keys order by created_by, created_at desc;

-- name: InsertApiKeyLog :exec
insert into api_key_logs (api_key_id, ip_id, action_id)
values (?,
  (select id from api_key_log_ips where value = sqlc.arg(ip)),
  (select id from api_key_log_actions where name = sqlc.arg(action)));

-- name: UpsertApiKeyLogAction :exec
insert into api_key_log_actions (name)
values (?)
on conflict(name) do nothing;

-- name: UpsertApiKeyLogIP :exec
insert into api_key_log_ips (value)
values (?)
on conflict(value) do nothing;

-- name: ListApiKeyLogsByApiKeyID :many
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

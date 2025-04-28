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
select value as path, content
  from note_paths p
  join note_versions v on p.id = v.path_id and p.version_count = v.version;

-- name: GetUserByEmail :one
select * from users where email = lower(?);

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

-- name: AllOffers :many
select * from offers order by id;

-- name: CreateOffer :one
insert into offers (id, names, lifetime, price_usd, price_rub, price_btc, starts_at, ends_at)
values (?, ?, ?, ?, ?, ?, ?, ?)
returning *;

-- name: UpdateOffer :one
update offers
   set names = ?
     , lifetime = ?
     , price_usd = ?
     , price_rub = ?
     , price_btc = ?
     , starts_at = ?
     , ends_at = ?
 where id = ?
returning *;

-- name: DeleteOffer :one
update offers
   set ends_at = datetime('now')
 where id = ?
returning *;

-- name: ListAllUsers :many
select * from users order by created_at desc;

-- name: ListActiveSubgraphsByUserID :many
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

-- name: UserBanByUserID :one
select * from user_bans where user_id = ?;

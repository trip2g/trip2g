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

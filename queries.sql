-- name: InsertNotePath :one
insert into note_paths (path, path_hash)
values (?, ?)
on conflict(path) do update set path = excluded.path
returning id, latest_content_hash;

-- name: IncrementNoteVersionCount :one
update note_paths
   set version_count = version_count + 1
     , latest_content_hash = ?
 where id = ?
returning version_count;

-- name: InsertNoteVersion :exec
insert into note_versions (path_id, version, content)
values (?, ?, ?);

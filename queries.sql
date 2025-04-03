-- name: InsertNotePath :exec
insert into note_paths (path, path_hash)
values (?, ?);

-- name: IncrementNoteVersionCount :one
update note_paths
   set version_count = version_count + 1
 where path = ?
returning version_count, id;

-- name: InsertNoteVersion :exec
insert into note_versions (path_id, version, content, content_hash)
values (?, ?, ?, ?);

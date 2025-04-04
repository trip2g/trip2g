-- name: InsertNotePath :exec
insert into note_paths (path, path_hash, latest_content_hash)
values (?, ?, ?);

-- name: IncrementNoteVersionCount :one
update note_paths
   set version_count = version_count + 1
     , latest_content_hash = ?
 where path = ?
    and latest_content_hash <> ?
returning id, version_count;

-- name: InsertNoteVersion :exec
insert into note_versions (path_id, version, content)
values (?, ?, ?);

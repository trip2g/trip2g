package db

type CreateNoteAssetParams struct {
	Asset InsertNoteAssetParams

	// UpsertNoteVersionAssetParams without AssetID
	VersionID int64
	Path      string
}

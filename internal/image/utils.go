package image

import "path/filepath"

//nolint:gochecknoglobals // readonly constant map
var exts = map[string]struct{}{
	".apng":  {},
	".avif":  {},
	".gif":   {},
	".jpg":   {},
	".jpeg":  {},
	".jfif":  {},
	".pjpeg": {},
	".pjp":   {},
	".png":   {},
	".svg":   {},
	".webp":  {},
}

func GetExtensions() map[string]struct{} {
	return exts
}

func IsRightExtension(target string) bool {
	ext := filepath.Ext(target)
	_, ok := exts[ext]
	return ok
}

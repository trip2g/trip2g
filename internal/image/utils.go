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

//nolint:gochecknoglobals // readonly constant map
var videoExts = map[string]struct{}{
	".mp4":  {},
	".avi":  {},
	".mov":  {},
	".mkv":  {},
	".webm": {},
	".m4v":  {},
}

func GetExtensions() map[string]struct{} {
	return exts
}

func IsRightExtension(target string) bool {
	ext := filepath.Ext(target)
	_, ok := exts[ext]
	return ok
}

func IsVideoExtension(target string) bool {
	ext := filepath.Ext(target)
	_, ok := videoExts[ext]
	return ok
}

func IsMediaExtension(target string) bool {
	return IsRightExtension(target) || IsVideoExtension(target)
}

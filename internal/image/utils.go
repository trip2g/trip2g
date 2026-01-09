package image

import (
	"path/filepath"
	"strings"
)

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

//nolint:gochecknoglobals // readonly constant map
var docExts = map[string]struct{}{
	".pdf":  {},
	".doc":  {},
	".docx": {},
	".xls":  {},
	".xlsx": {},
	".ppt":  {},
	".pptx": {},
	".txt":  {},
	".rtf":  {},
	".odt":  {},
	".ods":  {},
	".odp":  {},
	".csv":  {},
	".zip":  {},
	".rar":  {},
	".7z":   {},
	".mp3":  {},
	".wav":  {},
	".ogg":  {},
	".flac": {},
	".m4a":  {},
	".aac":  {},
}

func GetExtensions() map[string]struct{} {
	return exts
}

func IsRightExtension(target string) bool {
	ext := strings.ToLower(filepath.Ext(target))
	_, ok := exts[ext]
	return ok
}

func IsVideoExtension(target string) bool {
	ext := strings.ToLower(filepath.Ext(target))
	_, ok := videoExts[ext]
	return ok
}

func IsDocExtension(target string) bool {
	ext := strings.ToLower(filepath.Ext(target))
	_, ok := docExts[ext]
	return ok
}

func IsMediaExtension(target string) bool {
	return IsRightExtension(target) || IsVideoExtension(target) || IsDocExtension(target)
}

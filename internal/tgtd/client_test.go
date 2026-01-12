package tgtd

import (
	"testing"
)

func TestFilenameFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "simple URL with extension",
			url:      "https://example.com/image.jpg",
			expected: "image.jpg",
		},
		{
			name:     "URL with query parameters",
			url:      "https://example.com/image.jpg?token=abc&size=large",
			expected: "image.jpg",
		},
		{
			name:     "URL without extension",
			url:      "https://example.com/image",
			expected: "image.jpg",
		},
		{
			name:     "URL with path and query",
			url:      "https://cdn.example.com/uploads/2024/photo.png?v=123",
			expected: "photo.png",
		},
		{
			name:     "video URL with query",
			url:      "https://example.com/video.mp4?quality=hd",
			expected: "video.mp4",
		},
		{
			name:     "minio-style URL with webp (normalized to jpg)",
			url:      "https://minio.local/bucket/file.webp?X-Amz-Signature=abc123",
			expected: "file.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filenameFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("filenameFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsVideoExtension(t *testing.T) {
	tests := []struct {
		ext      string
		expected bool
	}{
		{".mp4", true},
		{".avi", true},
		{".mov", true},
		{".mkv", true},
		{".webm", true},
		{".m4v", true},
		{".jpg", false},
		{".png", false},
		{".gif", false},
		{".webp", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := isVideoExtension(tt.ext)
			if result != tt.expected {
				t.Errorf("isVideoExtension(%q) = %v, want %v", tt.ext, result, tt.expected)
			}
		})
	}
}

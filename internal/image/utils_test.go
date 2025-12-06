package image

import "testing"

func TestIsMediaExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		// Images - lowercase
		{"png lowercase", "image.png", true},
		{"jpg lowercase", "photo.jpg", true},
		{"gif lowercase", "anim.gif", true},
		{"webp lowercase", "image.webp", true},
		// Images - uppercase
		{"PNG uppercase", "image.PNG", true},
		{"JPG uppercase", "photo.JPG", true},
		{"GIF uppercase", "anim.GIF", true},
		{"WEBP uppercase", "image.WEBP", true},
		// Videos - lowercase
		{"mp4 lowercase", "video.mp4", true},
		{"mov lowercase", "video.mov", true},
		{"mkv lowercase", "video.mkv", true},
		{"webm lowercase", "video.webm", true},
		// Videos - uppercase
		{"MP4 uppercase", "video.MP4", true},
		{"MOV uppercase", "video.MOV", true},
		{"MKV uppercase", "video.MKV", true},
		{"WEBM uppercase", "video.WEBM", true},
		// Mixed case
		{"Mp4 mixed", "video.Mp4", true},
		{"Mov mixed", "video.Mov", true},
		// Non-media
		{"txt file", "doc.txt", false},
		{"pdf file", "doc.pdf", false},
		{"no extension", "noext", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMediaExtension(tt.filename)
			if got != tt.want {
				t.Errorf("IsMediaExtension(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

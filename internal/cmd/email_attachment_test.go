package cmd

import (
	"testing"
)

func TestParseAttachmentFlag(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPath string
		wantName string
		wantErr  bool
	}{
		{
			name:     "simple path",
			input:    "/path/to/file.pdf",
			wantPath: "/path/to/file.pdf",
			wantName: "file.pdf",
			wantErr:  false,
		},
		{
			name:     "path with custom name",
			input:    "/path/to/file.pdf:CustomName.pdf",
			wantPath: "/path/to/file.pdf",
			wantName: "CustomName.pdf",
			wantErr:  false,
		},
		{
			name:     "relative path",
			input:    "document.pdf",
			wantPath: "document.pdf",
			wantName: "document.pdf",
			wantErr:  false,
		},
		{
			name:     "path with spaces",
			input:    "/path/to/my file.pdf",
			wantPath: "/path/to/my file.pdf",
			wantName: "my file.pdf",
			wantErr:  false,
		},
		{
			name:     "custom name with colon separator",
			input:    "file.pdf:Report 2024.pdf",
			wantPath: "file.pdf",
			wantName: "Report 2024.pdf",
			wantErr:  false,
		},
		// Note: Windows path behavior is platform-specific
		// On Unix, backslashes are not path separators, so the entire string is the filename
		// On Windows, backslashes are path separators, so only "file.pdf" is extracted
		{
			name:     "windows path with custom name",
			input:    "C:\\Users\\test\\file.pdf:MyFile.pdf",
			wantPath: "C:\\Users\\test\\file.pdf",
			wantName: "MyFile.pdf",
			wantErr:  false,
		},
		{
			name:     "empty custom name defaults to filename",
			input:    "/path/to/file.pdf:",
			wantPath: "/path/to/file.pdf",
			wantName: "file.pdf",
			wantErr:  false,
		},
		{
			name:     "empty input",
			input:    "",
			wantPath: "",
			wantName: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, name, err := parseAttachmentFlag(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if path != tt.wantPath {
				t.Errorf("path = %s, want %s", path, tt.wantPath)
			}
			if name != tt.wantName {
				t.Errorf("name = %s, want %s", name, tt.wantName)
			}
		})
	}
}

func TestGetMimeType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		// Documents
		{"pdf", "document.pdf", "application/pdf"},
		{"word doc", "report.doc", "application/msword"},
		{"word docx", "report.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"excel xls", "data.xls", "application/vnd.ms-excel"},
		{"excel xlsx", "data.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},

		// Images
		{"png", "photo.png", "image/png"},
		{"jpg", "photo.jpg", "image/jpeg"},
		{"jpeg", "photo.jpeg", "image/jpeg"},
		{"gif", "animation.gif", "image/gif"},
		{"svg", "icon.svg", "image/svg+xml"},

		// Text
		{"txt", "readme.txt", "text/plain"},
		{"html", "index.html", "text/html"},
		{"css", "style.css", "text/css"},

		// Archives
		{"zip", "archive.zip", "application/zip"},
		{"tar", "backup.tar", "application/x-tar"},
		{"gzip", "compressed.gz", "application/gzip"},

		// Media
		{"mp3", "song.mp3", "audio/mpeg"},
		{"mp4", "video.mp4", "video/mp4"},
		{"wav", "audio.wav", "audio/wav"},

		// Code
		{"json", "data.json", "application/json"},
		{"xml", "config.xml", "application/xml"},
		{"js", "script.js", "application/javascript"},

		// Case insensitive
		{"uppercase pdf", "FILE.PDF", "application/pdf"},
		{"mixed case", "File.PdF", "application/pdf"},

		// Unknown/default
		{"unknown extension", "file.xyz", "application/octet-stream"},
		{"no extension", "README", "application/octet-stream"},
		{"empty", "", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMimeType(tt.filename)
			if got != tt.want {
				t.Errorf("getMimeType(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

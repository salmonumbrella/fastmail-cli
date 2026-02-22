package format

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ParseAttachmentFlag parses an attachment flag value.
// Format: /path/to/file[:displayname]
// Returns the file path and display name (defaults to basename if not specified).
func ParseAttachmentFlag(value string) (path, name string, err error) {
	if value == "" {
		return "", "", fmt.Errorf("attachment path cannot be empty")
	}

	// Check for custom name separator (last colon that's not part of Windows drive letter)
	// Handle Windows paths like C:\path\file.pdf
	lastColon := strings.LastIndex(value, ":")

	// On Windows, skip the drive letter colon (e.g., C:)
	isWindowsDrive := lastColon == 1 && len(value) > 2 && (value[2] == '\\' || value[2] == '/')

	if lastColon > 1 && !isWindowsDrive {
		// Found a colon for custom name
		path = value[:lastColon]
		name = value[lastColon+1:]
		if name == "" {
			name = filepath.Base(path)
		}
		return path, name, nil
	}

	// No custom name specified (or Windows drive letter)
	path = value
	name = filepath.Base(path)
	return path, name, nil
}

// MimeType returns the MIME type for a file based on extension.
func MimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".txt":  "text/plain",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".zip":  "application/zip",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".mp3":  "audio/mpeg",
		".mp4":  "video/mp4",
		".wav":  "audio/wav",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// SanitizeFilename removes path components and dangerous characters to prevent
// path traversal attacks. Returns only the base filename.
// SECURITY: Handles null bytes, control characters, reserved names (Windows),
// and enforces length limits.
func SanitizeFilename(name string) string {
	// SECURITY: Remove null bytes first (can bypass filesystem checks)
	name = strings.ReplaceAll(name, "\x00", "")

	// SECURITY: Remove control characters (0x00-0x1F and 0x7F)
	var clean strings.Builder
	for _, r := range name {
		if r >= 32 && r != 127 {
			clean.WriteRune(r)
		}
	}
	name = clean.String()

	// Remove any path components (prevents ../../etc/passwd attacks)
	name = filepath.Base(name)

	// Trim whitespace (prevents " .bashrc" becoming valid after dot trim)
	name = strings.TrimSpace(name)

	// Remove leading dots (prevents hidden files)
	name = strings.TrimLeft(name, ".")

	// SECURITY: Check for Windows reserved names (CON, PRN, AUX, NUL, COM1-9, LPT1-9)
	// These can cause issues even on non-Windows systems when files are transferred
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}
	nameUpper := strings.ToUpper(name)
	// Check both exact match and "RESERVED.ext" pattern
	for _, reserved := range reservedNames {
		if nameUpper == reserved || strings.HasPrefix(nameUpper, reserved+".") {
			name = "_" + name
			break
		}
	}

	// SECURITY: Limit filename length (most filesystems max 255 bytes)
	if len(name) > 255 {
		// Preserve extension if possible
		ext := filepath.Ext(name)
		if len(ext) < 20 && len(ext) > 0 {
			name = name[:255-len(ext)] + ext
		} else {
			name = name[:255]
		}
	}

	// Handle empty or dangerous names
	if name == "" || name == "." || name == ".." {
		return "attachment"
	}

	return name
}

// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package chat

import (
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// DecodeDataURL decodes data URL to data and mime type.
func DecodeDataURL(dataURL string) (data []byte, mimeType string, err error) {
	parts := strings.Split(dataURL, ";base64,")
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("invalid data URL: %s", dataURL)
	}
	if !strings.HasPrefix(parts[0], "data:") {
		return nil, "", fmt.Errorf("invalid data URL: %s", dataURL)
	}
	data, err = base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, "", fmt.Errorf("base64 decode failed: %w", err)
	}
	mimeType = strings.TrimPrefix(parts[0], "data:")
	return
}

// EncodeDataURL encodes data to data URL with mime type.
func EncodeDataURL(mimeType string, data []byte) string {
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// EncodeDataURLFromPath encodes data from a file path.
// mime type is determined by the file extension.
func EncodeDataURLFromPath(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		return "", "", fmt.Errorf("unknown file extension: %s", path)
	}
	return EncodeDataURL(mimeType, data), mimeType, nil
}

// IsDataURL checks if the data URL is valid.
func IsDataURL(dataURL string) bool {
	return strings.HasPrefix(dataURL, "data:") && strings.Contains(dataURL, ";base64,")
}

// SplitDataURL splits data URL to mime type and encoded data.
func SplitDataURL(dataURL string) (mimeType string, encodedData string, err error) {
	if !IsDataURL(dataURL) {
		return "", "", fmt.Errorf("not a data URL: %s", dataURL)
	}
	parts := strings.Split(strings.TrimPrefix(dataURL, "data:"), ";base64,")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid data URL: %s", dataURL)
	}
	mimeType = parts[0]
	encodedData = parts[1]
	return
}

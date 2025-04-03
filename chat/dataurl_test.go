// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package chat

import (
	"reflect"
	"testing"
)

func TestIsDataURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"valid data url", "data:image/png;base64,iVBORw0KGgo=", true},
		{"not data url", "https://example.com/image.png", false},
		{"no base64", "data:image/png,iVBORw0KGgo=", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDataURL(tt.url); got != tt.want {
				t.Errorf("IsDataURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitDataURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantMime   string
		wantData   string
		wantErrNil bool
	}{
		{
			name:       "valid data url",
			url:        "data:image/png;base64,iVBORw0KGgo=",
			wantMime:   "image/png",
			wantData:   "iVBORw0KGgo=",
			wantErrNil: true,
		},
		{
			name:       "not data url",
			url:        "https://example.com/image.png",
			wantMime:   "",
			wantData:   "",
			wantErrNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMime, gotData, err := SplitDataURL(tt.url)
			if (err == nil) != tt.wantErrNil {
				t.Fatalf("SplitDataURL() error = %v, wantErrNil %v", err, tt.wantErrNil)
			}
			if gotMime != tt.wantMime {
				t.Fatalf("SplitDataURL() gotMime = %v, want %v", gotMime, tt.wantMime)
			}
			if gotData != tt.wantData {
				t.Fatalf("SplitDataURL() gotData = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestEncodeDecodeDataURL(t *testing.T) {
	testCases := []struct {
		name     string
		mimeType string
		data     []byte
	}{
		{
			name:     "text/plain",
			mimeType: "text/plain",
			data:     []byte("Hello, world!"),
		},
		{
			name:     "image/png",
			mimeType: "image/png",
			data:     []byte{0x89, 0x50, 0x4E, 0x47},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dataURL := EncodeDataURL(tc.mimeType, tc.data)

			gotData, gotMime, err := DecodeDataURL(dataURL)
			if err != nil {
				t.Fatalf("DecodeDataURL() error = %v", err)
			}

			if !reflect.DeepEqual(gotData, tc.data) {
				t.Errorf("DecodeDataURL() gotData = %v, want %v", gotData, tc.data)
			}

			if gotMime != tc.mimeType {
				t.Errorf("DecodeDataURL() gotMime = %v, want %v", gotMime, tc.mimeType)
			}
		})
	}
}

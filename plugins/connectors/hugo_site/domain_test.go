/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package hugo_site

import "testing"

func TestGetFullURL(t *testing.T) {
	tests := []struct {
		name    string
		seedURL string
		rawURL  string
		want    string
		wantErr bool
	}{
		{
			name:    "absolute url passes through unchanged",
			seedURL: "https://example.com/docs/index.json",
			rawURL:  "http://a.com",
			want:    "http://a.com",
		},
		{
			name:    "site relative path resolves from root",
			seedURL: "https://example.com/docs/index.json",
			rawURL:  "/posts/hello",
			want:    "https://example.com/posts/hello",
		},
		{
			name:    "plain relative path resolves from site root instead of feed parent",
			seedURL: "https://example.com/docs/index.json",
			rawURL:  "posts/hello",
			want:    "https://example.com/posts/hello",
		},
		{
			name:    "scheme relative url inherits seed scheme",
			seedURL: "https://example.com/docs/index.json",
			rawURL:  "//cdn.example.com/assets/app.js",
			want:    "https://cdn.example.com/assets/app.js",
		},
		{
			name:    "nested relative path still resolves from site root",
			seedURL: "https://example.com/easysearch/main/index.json",
			rawURL:  "guides/getting-started",
			want:    "https://example.com/guides/getting-started",
		},
		{
			name:    "empty payload url stays empty",
			seedURL: "https://example.com/docs/index.json",
			rawURL:  "   ",
			want:    "",
		},
		{
			name:    "invalid seed url returns error",
			seedURL: "://bad-seed",
			rawURL:  "/posts/hello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFullURL(tt.seedURL, tt.rawURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("getFullURL(%q, %q) = %q, want %q", tt.seedURL, tt.rawURL, got, tt.want)
			}
		})
	}
}

package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLValidator_ValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Test case for validating a standard HTTP URL
		// Verifies that basic HTTP URLs without any path or query parameters are accepted
		{
			name:    "valid http url",
			url:     "http://example.com",
			wantErr: false,
		},
		// Test case for validating a secure HTTPS URL
		// Verifies that HTTPS protocol is accepted as a valid protocol
		{
			name:    "valid https url",
			url:     "https://example.com",
			wantErr: false,
		},
		// Test case for validating a URL with a path component
		// Verifies that URLs containing additional path segments are accepted
		{
			name:    "valid url with path",
			url:     "https://example.com/path",
			wantErr: false,
		},
		// Test case for validating a URL with query parameters
		// Verifies that URLs containing query string parameters are accepted
		{
			name:    "valid url with query",
			url:     "https://example.com?param=value",
			wantErr: false,
		},
		// Test case for validating a URL without protocol
		// Verifies that URLs missing the protocol specification are rejected
		{
			name:    "invalid url - missing protocol",
			url:     "example.com",
			wantErr: true,
		},
		// Test case for validating a URL with unsupported protocol
		// Verifies that URLs using protocols other than HTTP/HTTPS are rejected
		{
			name:    "invalid url - wrong protocol",
			url:     "ftp://example.com",
			wantErr: true,
		},
		// Test case for validating an empty URL
		// Verifies that empty strings are rejected as invalid URLs
		{
			name:    "invalid url - empty",
			url:     "",
			wantErr: true,
		},
	}

	v := NewURLValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestURLValidator_ValidateURLs(t *testing.T) {
	tests := []struct {
		name           string
		urls           []string
		wantInvalid    int
		wantValidCount int
	}{
		// Test case for validating multiple valid URLs
		// Verifies that all URLs in a list of valid URLs are accepted
		{
			name:           "all valid urls",
			urls:           []string{"https://example.com", "http://test.com"},
			wantInvalid:    0,
			wantValidCount: 2,
		},
		// Test case for validating a mix of valid and invalid URLs
		// Verifies that the validator correctly identifies both valid and invalid URLs
		// and maintains the correct count of each
		{
			name:           "mixed valid and invalid urls",
			urls:           []string{"https://example.com", "invalid-url", "http://test.com"},
			wantInvalid:    1,
			wantValidCount: 2,
		},
		// Test case for validating multiple invalid URLs
		// Verifies that all URLs in a list of invalid URLs are rejected
		{
			name:           "all invalid urls",
			urls:           []string{"invalid-url", "also-invalid"},
			wantInvalid:    2,
			wantValidCount: 0,
		},
		// Test case for validating an empty URL list
		// Verifies that an empty list of URLs is handled correctly
		{
			name:           "empty urls",
			urls:           []string{},
			wantInvalid:    0,
			wantValidCount: 0,
		},
	}

	v := NewURLValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := v.ValidateURLs(tt.urls)

			// Check total count
			assert.Equal(t, len(tt.urls), len(results))

			// Check invalid count
			invalidCount := 0
			validCount := 0
			for _, r := range results {
				if !r.IsValid() {
					invalidCount++
				} else {
					validCount++
				}
			}
			assert.Equal(t, tt.wantInvalid, invalidCount)
			assert.Equal(t, tt.wantValidCount, validCount)

			// Check GetInvalidURLs
			invalidURLs := results.GetInvalidURLs()
			assert.Equal(t, tt.wantInvalid, len(invalidURLs))

			// Check HasInvalidURLs
			assert.Equal(t, tt.wantInvalid > 0, HasInvalidURLs(results))
		})
	}
}

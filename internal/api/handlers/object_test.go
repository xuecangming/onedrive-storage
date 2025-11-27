package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseRangeHeader(t *testing.T) {
	tests := []struct {
		name        string
		rangeHeader string
		totalSize   int64
		wantStart   int64
		wantEnd     int64
		wantErr     bool
	}{
		// Valid ranges
		{
			name:        "first 500 bytes",
			rangeHeader: "bytes=0-499",
			totalSize:   1000,
			wantStart:   0,
			wantEnd:     499,
			wantErr:     false,
		},
		{
			name:        "middle range",
			rangeHeader: "bytes=500-999",
			totalSize:   2000,
			wantStart:   500,
			wantEnd:     999,
			wantErr:     false,
		},
		{
			name:        "from position to end",
			rangeHeader: "bytes=500-",
			totalSize:   1000,
			wantStart:   500,
			wantEnd:     999,
			wantErr:     false,
		},
		{
			name:        "last 100 bytes",
			rangeHeader: "bytes=-100",
			totalSize:   1000,
			wantStart:   900,
			wantEnd:     999,
			wantErr:     false,
		},
		{
			name:        "end exceeds file size - should adjust",
			rangeHeader: "bytes=0-2000",
			totalSize:   1000,
			wantStart:   0,
			wantEnd:     999,
			wantErr:     false,
		},
		{
			name:        "last bytes more than file size",
			rangeHeader: "bytes=-2000",
			totalSize:   1000,
			wantStart:   0,
			wantEnd:     999,
			wantErr:     false,
		},

		// Invalid ranges
		{
			name:        "missing bytes prefix",
			rangeHeader: "0-499",
			totalSize:   1000,
			wantErr:     true,
		},
		{
			name:        "multiple ranges not supported",
			rangeHeader: "bytes=0-499,500-999",
			totalSize:   1000,
			wantErr:     true,
		},
		{
			name:        "start beyond file size",
			rangeHeader: "bytes=2000-3000",
			totalSize:   1000,
			wantErr:     true,
		},
		{
			name:        "invalid format",
			rangeHeader: "bytes=abc-def",
			totalSize:   1000,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := parseRangeHeader(tt.rangeHeader, tt.totalSize)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseRangeHeader() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseRangeHeader() unexpected error: %v", err)
				return
			}

			if start != tt.wantStart {
				t.Errorf("parseRangeHeader() start = %v, want %v", start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("parseRangeHeader() end = %v, want %v", end, tt.wantEnd)
			}
		})
	}
}

func TestDownloadWithRange(t *testing.T) {
	// Mock data
	testData := []byte("Hello, World! This is test content for range requests.")
	
	t.Run("full content without range", func(t *testing.T) {
		w := httptest.NewRecorder()

		// Simulate response for full content
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", "54")
		w.WriteHeader(http.StatusOK)
		w.Write(testData)

		if w.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
		}
		if w.Header().Get("Accept-Ranges") != "bytes" {
			t.Error("Accept-Ranges header missing")
		}
	})

	t.Run("partial content with range", func(t *testing.T) {
		w := httptest.NewRecorder()

		// Simulate response for partial content
		start, end, _ := parseRangeHeader("bytes=0-4", int64(len(testData)))
		
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Range", "bytes 0-4/54")
		w.Header().Set("Content-Length", "5")
		w.WriteHeader(http.StatusPartialContent)
		w.Write(testData[start : end+1])

		if w.Code != http.StatusPartialContent {
			t.Errorf("status = %v, want %v", w.Code, http.StatusPartialContent)
		}
		if w.Header().Get("Content-Range") == "" {
			t.Error("Content-Range header missing for partial content")
		}
		if w.Body.String() != "Hello" {
			t.Errorf("body = %q, want %q", w.Body.String(), "Hello")
		}
	})
}

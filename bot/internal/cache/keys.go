package cache

import "fmt"

// VideoKey returns the cache key for a resolved video result.
func VideoKey(normalizedURL string, audio bool, quality string) string {
	if quality == "" {
		quality = "best"
	}
	return fmt.Sprintf("video:%s:audio=%v:q=%s", normalizedURL, audio, quality)
}

// VideoLockKey returns the deduplication lock key for a URL.
func VideoLockKey(normalizedURL string) string {
	return fmt.Sprintf("video:%s:lock", normalizedURL)
}

// VideoFileIDKey returns the key for storing a Telegram file_id after first upload.
func VideoFileIDKey(normalizedURL string, audio bool, quality string) string {
	if quality == "" {
		quality = "best"
	}
	return fmt.Sprintf("video:%s:audio=%v:q=%s:file_id", normalizedURL, audio, quality)
}

package cmd

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

func parseHashFromHeader(etag string, headerHash string) string {
	if headerHash != "" {
		return strings.Trim(headerHash, "\"")
	}
	if etag == "" {
		return ""
	}
	clean := strings.TrimSpace(etag)
	clean = strings.TrimPrefix(clean, "W/")
	clean = strings.Trim(clean, "\"")
	return clean
}

func computeHash(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}

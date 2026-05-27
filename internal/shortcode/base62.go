package shortcode

import (
	"math"
	"strings"
)

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const base = 62

// Encode converts an integer to a base62 string
func Encode(num int64) string {
	if num == 0 {
		return string(alphabet[0])
	}

	var result strings.Builder
	for num > 0 {
		result.WriteByte(alphabet[num%base])
		num /= base
	}

	// Reverse the string
	s := result.String()
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	// Pad to minimum 6 characters for better-looking URLs
	encoded := string(runes)
	for len(encoded) < 6 {
		encoded = string(alphabet[0]) + encoded
	}

	return encoded
}

// Decode converts a base62 string back to an integer
func Decode(s string) int64 {
	var result int64
	s = strings.TrimLeft(s, string(alphabet[0]))
	for i, char := range s {
		p := strings.IndexRune(alphabet, char)
		if p < 0 {
			return -1
		}
		result += int64(p) * int64(math.Pow(float64(base), float64(len(s)-1-i)))
	}
	return result
}

// IsValidAlias checks if a custom alias is valid
func IsValidAlias(alias string) bool {
	if len(alias) < 3 || len(alias) > 20 {
		return false
	}

	// Reserved words
	reserved := map[string]bool{
		"api": true, "admin": true, "health": true,
		"web": true, "static": true, "css": true,
		"js": true, "img": true, "favicon": true,
	}
	if reserved[strings.ToLower(alias)] {
		return false
	}

	for _, c := range alias {
		if !strings.ContainsRune(alphabet+"-_", c) {
			return false
		}
	}
	return true
}

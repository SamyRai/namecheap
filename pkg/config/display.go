package config

// MaskAPIKey masks an API key for display, showing only first 4 and last 4 characters.
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "(not set)"
	}
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "***" + apiKey[len(apiKey)-4:]
}

// Min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

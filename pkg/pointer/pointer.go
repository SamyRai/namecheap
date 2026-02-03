package pointer

// String safely dereferences a string pointer, returning empty string if nil
func String(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Int safely dereferences an int pointer, returning 0 if nil
func Int(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// Bool safely dereferences a bool pointer, returning false if nil
func Bool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

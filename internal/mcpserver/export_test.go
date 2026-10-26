package mcpserver

// CheckForStaleBinaryOnEveryCall drops the stat interval so a test sees a replaced binary at once;
// the returned func restores it.
func CheckForStaleBinaryOnEveryCall() (restore func()) {
	old := staleCheckEvery
	staleCheckEvery = 0
	return func() { staleCheckEvery = old }
}

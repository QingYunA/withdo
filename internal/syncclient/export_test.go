package syncclient

// SetLegacyDBPath makes path the database that owned the old fixed refresh-token key, until the
// returned function restores the default.
func SetLegacyDBPath(path string) (restore func()) {
	saved := legacyDBPath
	legacyDBPath = func() string { return path }
	return func() { legacyDBPath = saved }
}

package syncclient

import "strings"

// OfficialServer is the address of the official sync server.
const OfficialServer = "https://api.withdo.si"

// LegacyServer is the official server's earlier name. It is the same backend, so an account, a
// device id and a token stay valid when a saved config moves from it to OfficialServer.
const LegacyServer = "https://todo.sri-ai.xyz"

// sameServer tells whether a and b are one address: spaces, a trailing slash and case do not matter.
func sameServer(a, b string) bool {
	return strings.EqualFold(strings.TrimRight(strings.TrimSpace(a), "/"), strings.TrimRight(strings.TrimSpace(b), "/"))
}

// IsLegacyServer tells whether server is LegacyServer.
func IsLegacyServer(server string) bool { return sameServer(server, LegacyServer) }

// IsOfficialServer tells whether server is OfficialServer or LegacyServer.
func IsOfficialServer(server string) bool {
	return sameServer(server, OfficialServer) || IsLegacyServer(server)
}

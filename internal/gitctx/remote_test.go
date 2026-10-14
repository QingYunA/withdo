package gitctx

import "testing"

func TestParseRemoteURL(t *testing.T) {
	tests := []struct{ url, want string }{
		{"git@github.com:acme/widgets.git", "acme/widgets"},
		{"git@github.com:acme/widgets", "acme/widgets"},
		{"https://github.com/acme/widgets.git", "acme/widgets"},
		{"https://github.com/acme/widgets", "acme/widgets"},
		{"https://github.com/acme/widgets/", "acme/widgets"},
		{"https://user:tok@github.com/acme/widgets.git", "acme/widgets"},
		{"ssh://git@github.com/acme/widgets.git", "acme/widgets"},
		{"ssh://git@github.com:22/acme/widgets.git", "acme/widgets"},
		{"git://github.com/acme/widgets.git", "acme/widgets"},
		{"https://github.com/acme/wid.gets.git", "acme/wid.gets"},
		{"https://GitHub.com/Acme/Widgets.git", "Acme/Widgets"},
		{"  https://github.com/acme/widgets.git\n", "acme/widgets"},
		{"https://gitlab.com/acme/widgets.git", ""},
		{"git@example.com:acme/widgets.git", ""},
		{"https://github.com/acme", ""},
		{"https://github.com/acme/widgets/extra", ""},
		{"/local/path/repo.git", ""},
		{"https://", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := ParseRemoteURL(tt.url); got != tt.want {
				t.Errorf("ParseRemoteURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

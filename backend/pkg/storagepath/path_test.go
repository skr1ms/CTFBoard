package storagepath

import "testing"

func TestValidateDownloadPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "legacy", path: "0123456789abcdef/archive.zip", want: false},
		{name: "tasks", path: "tasks/0123456789abcdef/archive.zip", want: true},
		{name: "tasks nested", path: "tasks/0123456789abcdef/nested/archive.zip", want: false},
		{name: "empty", path: "", want: false},
		{name: "short hash", path: "0123456789abcde/archive.zip", want: false},
		{name: "non hex hash", path: "0123456789abcdeg/archive.zip", want: false},
		{name: "path traversal", path: "tasks/0123456789abcdef/../secret", want: false},
		{name: "missing filename", path: "tasks/0123456789abcdef/", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ValidateDownloadPath(tt.path)
			if got != tt.want {
				t.Fatalf("ValidateDownloadPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDownloadFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "legacy", path: "0123456789abcdef/archive.zip", want: "download"},
		{name: "tasks", path: "tasks/0123456789abcdef/archive.zip", want: "archive.zip"},
		{name: "nested tasks", path: "tasks/0123456789abcdef/nested/archive.zip", want: "download"},
		{name: "invalid", path: "invalid", want: "download"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DownloadFilename(tt.path)
			if got != tt.want {
				t.Fatalf("DownloadFilename(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

package workspace

import (
	"os/exec"
	"strings"
	"testing"
)

func TestFindUnmountBinary(t *testing.T) {
	tests := []struct {
		name      string
		available []string
		want      string
	}{
		{"only fusermount3", []string{"fusermount3"}, "fusermount3"},
		{"only fusermount", []string{"fusermount"}, "fusermount"},
		{"both", []string{"fusermount", "fusermount3"}, "fusermount3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findUnmountBinary(fakeLookPath(tt.available...))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("none", func(t *testing.T) {
		_, err := findUnmountBinary(fakeLookPath())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		for _, name := range []string{"fusermount3", "fusermount"} {
			if !strings.Contains(err.Error(), name) {
				t.Errorf("error %q does not name %q", err, name)
			}
		}
	})
}

func fakeLookPath(available ...string) func(string) (string, error) {
	return func(file string) (string, error) {
		for _, a := range available {
			if a == file {
				return "/fake/bin/" + file, nil
			}
		}
		return "", exec.ErrNotFound
	}
}

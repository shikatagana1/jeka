package input

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFromArg(t *testing.T) {
	if _, err := FromArg("8.8.8.8"); err != nil {
		t.Fatalf("valid IPv4: %v", err)
	}
	if _, err := FromArg("2001:4860:4860::8888"); err != nil {
		t.Fatalf("valid IPv6: %v", err)
	}
	if _, err := FromArg("not-an-ip"); err == nil {
		t.Fatal("expected error for bad IP")
	}
}

func TestFromFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ips.txt")
	content := "# a comment\n\n8.8.8.8\n  1.1.1.1  \n   # indented comment\n2606:4700:4700::1111\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	addrs, err := FromFile(p)
	if err != nil {
		t.Fatalf("FromFile: %v", err)
	}
	if len(addrs) != 3 {
		t.Fatalf("got %d addrs, want 3: %v", len(addrs), addrs)
	}
}

func TestFromFileBadLine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.txt")
	if err := os.WriteFile(p, []byte("8.8.8.8\nnope\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := FromFile(p); err == nil {
		t.Fatal("expected error for invalid line")
	}
}

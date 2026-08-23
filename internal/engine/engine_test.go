package engine

import (
	"testing"
	"time"
)

func TestRewindRestoresDeletedFile(t *testing.T) {
	fs := New()

	fs.WriteFile("src/main.go", []byte("package main"))
	beforeDelete := time.Now()

	time.Sleep(time.Millisecond)

	if err := fs.DeleteFile("src/main.go"); err != nil {
		t.Fatal(err)
	}

	if _, exists := fs.ReadFile("src/main.go"); exists {
		t.Fatal("expected file to be deleted")
	}

	fs.Rewind(beforeDelete)

	content, exists := fs.ReadFile("src/main.go")
	if !exists {
		t.Fatal("expected rewind to restore the deleted file")
	}

	if string(content) != "package main" {
		t.Fatalf("unexpected restored content: %q", content)
	}
}
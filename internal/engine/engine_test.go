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

func TestRewindStepsRestoresMostRecentState(t *testing.T) {
	fs := New()

	fs.WriteFile("incident.txt", []byte("recover me"))

	if err := fs.DeleteFile("incident.txt"); err != nil {
		t.Fatal(err)
	}

	if !fs.RewindSteps(1) {
		t.Fatal("expected one-step rewind to succeed")
	}

	content, exists := fs.ReadFile("incident.txt")
	if !exists {
		t.Fatal("expected deleted file to be restored")
	}

	if string(content) != "recover me" {
		t.Fatalf("unexpected content: %q", content)
	}
}

func TestRewindRestoresDeletedDirectory(t *testing.T) {
	fs := New()

	if err := fs.MakeDir("/assets"); err != nil {
		t.Fatal(err)
	}

	beforeDelete := time.Now()

	time.Sleep(time.Millisecond)

	if err := fs.RemoveDir("/assets"); err != nil {
		t.Fatal(err)
	}

	fs.Rewind(beforeDelete)

	if _, exists := fs.ListDirectories()["/assets"]; !exists {
		t.Fatal("expected rewind to restore deleted directory")
	}
}

func TestStepForwardRestoresNewerSnapshot(t *testing.T) {
	fs := New()

	fs.WriteFile("status.txt", []byte("before"))
	fs.WriteFile("status.txt", []byte("after"))

	if !fs.RewindSteps(1) {
		t.Fatal("expected rewind to succeed")
	}

	content, _ := fs.ReadFile("status.txt")
	if string(content) != "before" {
		t.Fatalf("expected old content, got %q", content)
	}

	if !fs.StepForward(1) {
		t.Fatal("expected step forward to succeed")
	}

	content, _ = fs.ReadFile("status.txt")
	if string(content) != "after" {
		t.Fatalf("expected newer content, got %q", content)
	}
}

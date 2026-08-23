package engine

import (
	"errors"
	"sync"
	"time"
)

type EventType string

const (
	EventWrite  EventType = "WRITE"
	EventDelete EventType = "DELETE"
	EventRename EventType = "RENAME"
)

type Event struct {
	Type      EventType
	Path      string
	OldPath   string
	Timestamp time.Time
}

type File struct {
	Content   []byte
	UpdatedAt time.Time
}

type snapshot struct {
	Timestamp time.Time
	Files     map[string]File
}

type Engine struct {
	mu        sync.RWMutex
	files     map[string]File
	events    []Event
	snapshots []snapshot
}

func New() *Engine {
	now := time.Now()
	e := &Engine{
		files: make(map[string]File),
	}
	e.saveSnapshot(now)
	return e
}

func (e *Engine) WriteFile(path string, content []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	e.files[path] = File{
		Content:   append([]byte(nil), content...),
		UpdatedAt: now,
	}
	e.events = append(e.events, Event{
		Type:      EventWrite,
		Path:      path,
		Timestamp: now,
	})
	e.saveSnapshot(now)
}

func (e *Engine) DeleteFile(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.files[path]; !exists {
		return errors.New("file does not exist: " + path)
	}

	now := time.Now()
	delete(e.files, path)
	e.events = append(e.events, Event{
		Type:      EventDelete,
		Path:      path,
		Timestamp: now,
	})
	e.saveSnapshot(now)

	return nil
}

func (e *Engine) RenameFile(oldPath, newPath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	file, exists := e.files[oldPath]
	if !exists {
		return errors.New("file does not exist: " + oldPath)
	}

	now := time.Now()
	file.UpdatedAt = now
	e.files[newPath] = file
	delete(e.files, oldPath)

	e.events = append(e.events, Event{
		Type:      EventRename,
		Path:      newPath,
		OldPath:   oldPath,
		Timestamp: now,
	})
	e.saveSnapshot(now)

	return nil
}

func (e *Engine) ReadFile(path string) ([]byte, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	file, exists := e.files[path]
	if !exists {
		return nil, false
	}

	return append([]byte(nil), file.Content...), true
}

func (e *Engine) ListFiles() map[string]File {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return cloneFiles(e.files)
}

func (e *Engine) Events() []Event {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return append([]Event(nil), e.events...)
}

func (e *Engine) Rewind(target time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()

	selected := e.snapshots[0]
	for _, item := range e.snapshots {
		if item.Timestamp.After(target) {
			break
		}
		selected = item
	}

	e.files = cloneFiles(selected.Files)
}
func (e *Engine) saveSnapshot(timestamp time.Time) {
	e.snapshots = append(e.snapshots, snapshot{
		Timestamp: timestamp,
		Files:     cloneFiles(e.files),
	})
}

func cloneFiles(source map[string]File) map[string]File {
	result := make(map[string]File, len(source))

	for path, file := range source {
		result[path] = File{
			Content:   append([]byte(nil), file.Content...),
			UpdatedAt: file.UpdatedAt,
		}
	}

	return result
}
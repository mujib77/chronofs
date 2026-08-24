package engine

import (
	"errors"
	"strings"
	"sync"
	"time"
)

type EventType string

const (
	EventWrite  EventType = "WRITE"
	EventDelete EventType = "DELETE"
	EventRename EventType = "RENAME"
	EventMkdir  EventType = "MKDIR"
	EventRmdir  EventType = "RMDIR"
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
	Dirs      map[string]time.Time
}

type Engine struct {
	mu        sync.RWMutex
	files     map[string]File
	dirs      map[string]time.Time
	events    []Event
	snapshots []snapshot
}

func New() *Engine {
	now := time.Now()

	e := &Engine{
		files: make(map[string]File),
		dirs: map[string]time.Time{
			"/": now,
		},
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

func (e *Engine) MakeDir(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.dirs[path]; exists {
		return errors.New("directory already exists: " + path)
	}

	if _, exists := e.files[path]; exists {
		return errors.New("a file already exists at: " + path)
	}

	now := time.Now()
	e.dirs[path] = now
	e.events = append(e.events, Event{
		Type:      EventMkdir,
		Path:      path,
		Timestamp: now,
	})
	e.saveSnapshot(now)

	return nil
}

func (e *Engine) RemoveDir(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if path == "/" {
		return errors.New("cannot remove root directory")
	}

	if _, exists := e.dirs[path]; !exists {
		return errors.New("directory does not exist: " + path)
	}

	prefix := path + "/"

	for filePath := range e.files {
		if strings.HasPrefix(filePath, prefix) {
			return errors.New("directory is not empty: " + path)
		}
	}

	for dirPath := range e.dirs {
		if dirPath != path && strings.HasPrefix(dirPath, prefix) {
			return errors.New("directory is not empty: " + path)
		}
	}

	now := time.Now()
	delete(e.dirs, path)
	e.events = append(e.events, Event{
		Type:      EventRmdir,
		Path:      path,
		Timestamp: now,
	})
	e.saveSnapshot(now)

	return nil
}

func (e *Engine) ListDirectories() map[string]time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return cloneDirectories(e.dirs)
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
	e.dirs = cloneDirectories(selected.Dirs)
}

func (e *Engine) RewindSteps(steps int) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if steps < 1 || len(e.snapshots) <= steps {
		return false
	}

	selected := e.snapshots[len(e.snapshots)-1-steps]
	e.files = cloneFiles(selected.Files)
	e.dirs = cloneDirectories(selected.Dirs)

	return true
}
func (e *Engine) saveSnapshot(timestamp time.Time) {
	e.snapshots = append(e.snapshots, snapshot{
		Timestamp: timestamp,
		Files:     cloneFiles(e.files),
		Dirs:      cloneDirectories(e.dirs),
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

func cloneDirectories(source map[string]time.Time) map[string]time.Time {
	result := make(map[string]time.Time, len(source))

	for path, createdAt := range source {
		result[path] = createdAt
	}

	return result
}

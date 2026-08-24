package fs

import (
	"path"
	"sort"
	"sync"
	"time"

	chronofsengine "github.com/mujib77/chronofs/internal/engine"
	"github.com/winfsp/cgofuse/fuse"
)

type FileSystem struct {
	fuse.FileSystemBase
	mu     sync.RWMutex
	engine *chronofsengine.Engine
	ready  chan struct{}
}

func New() *FileSystem {
	engine := chronofsengine.New()

	engine.WriteFile("/README.md", []byte("# ChronoFS\n\nThis file exists inside a time-scrubbable filesystem.\n"))
	engine.WriteFile("/status.txt", []byte("Timeline recording is active.\n"))
	engine.WriteFile("/demo.go", []byte("package main\n\nfunc main() {\n\tprintln(\"Time is reversible\")\n}\n"))

	return &FileSystem{
		engine: engine,
		ready:  make(chan struct{}),
	}
}

func (fs *FileSystem) Init() {
	close(fs.ready)
}

func (fs *FileSystem) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if path == "/" {
		stat.Mode = fuse.S_IFDIR | 0777
		stat.Nlink = 1
		return 0
	}

	dirs := fs.engine.ListDirectories()
	if createdAt, exists := dirs[path]; exists {
		stat.Mode = fuse.S_IFDIR | 0777
		stat.Nlink = 1
		stat.Mtim = fuse.NewTimespec(createdAt)
		stat.Ctim = fuse.NewTimespec(createdAt)
		stat.Birthtim = fuse.NewTimespec(createdAt)
		return 0
	}

	files := fs.engine.ListFiles()
	file, exists := files[path]
	if !exists {
		return -fuse.ENOENT
	}

	stat.Mode = fuse.S_IFREG | 0666
	stat.Nlink = 1
	stat.Size = int64(len(file.Content))
	stat.Mtim = fuse.NewTimespec(file.UpdatedAt)
	stat.Ctim = fuse.NewTimespec(file.UpdatedAt)
	stat.Birthtim = fuse.NewTimespec(file.UpdatedAt)

	return 0
}

func (fs *FileSystem) Create(filePath string, flags int, mode uint32) (int, uint64) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dirs := fs.engine.ListDirectories()
	if _, exists := dirs[path.Dir(filePath)]; !exists {
		return -fuse.ENOENT, ^uint64(0)
	}

	if _, exists := fs.engine.ReadFile(filePath); !exists {
		fs.engine.WriteFile(filePath, []byte{})
	}

	return 0, 0
}

func (fs *FileSystem) Mkdir(dirPath string, mode uint32) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dirs := fs.engine.ListDirectories()
	if _, exists := dirs[path.Dir(dirPath)]; !exists {
		return -fuse.ENOENT
	}

	if err := fs.engine.MakeDir(dirPath); err != nil {
		return -fuse.EEXIST
	}

	return 0
}

func (fs *FileSystem) Rmdir(dirPath string) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.engine.RemoveDir(dirPath); err != nil {
		return -fuse.ENOTEMPTY
	}

	return 0
}

func (fs *FileSystem) Open(path string, flags int) (int, uint64) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if _, exists := fs.engine.ReadFile(path); !exists {
		return -fuse.ENOENT, ^uint64(0)
	}

	return 0, 0
}

func (fs *FileSystem) Read(path string, buff []byte, ofst int64, fh uint64) int {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	content, exists := fs.engine.ReadFile(path)
	if !exists {
		return -fuse.ENOENT
	}

	if ofst >= int64(len(content)) {
		return 0
	}

	end := ofst + int64(len(buff))
	if end > int64(len(content)) {
		end = int64(len(content))
	}

	return copy(buff, content[ofst:end])
}

func (fs *FileSystem) Write(path string, buff []byte, ofst int64, fh uint64) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	content, exists := fs.engine.ReadFile(path)
	if !exists {
		return -fuse.ENOENT
	}

	end := int(ofst) + len(buff)
	if end > len(content) {
		expanded := make([]byte, end)
		copy(expanded, content)
		content = expanded
	}

	copy(content[ofst:], buff)
	fs.engine.WriteFile(path, content)

	return len(buff)
}

func (fs *FileSystem) Truncate(path string, size int64, fh uint64) int {
	if size < 0 {
		return -fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	content, exists := fs.engine.ReadFile(path)
	if !exists {
		return -fuse.ENOENT
	}

	resized := make([]byte, size)
	copy(resized, content)
	fs.engine.WriteFile(path, resized)

	return 0
}

func (fs *FileSystem) Unlink(path string) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.engine.DeleteFile(path); err != nil {
		return -fuse.ENOENT
	}

	return 0
}

func (fs *FileSystem) Rename(oldPath, newPath string) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.engine.RenameFile(oldPath, newPath); err != nil {
		return -fuse.ENOENT
	}

	return 0
}

func (fs *FileSystem) Readdir(
	dirPath string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64,
) int {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	dirs := fs.engine.ListDirectories()
	if _, exists := dirs[dirPath]; !exists {
		return -fuse.ENOENT
	}

	fill(".", nil, 0)
	fill("..", nil, 0)

	entries := make(map[string]bool)

	for folderPath := range dirs {
		if folderPath == dirPath {
			continue
		}

		if path.Dir(folderPath) == dirPath {
			entries[path.Base(folderPath)] = true
		}
	}

	for filePath := range fs.engine.ListFiles() {
		if path.Dir(filePath) == dirPath {
			entries[path.Base(filePath)] = true
		}
	}

	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		if !fill(name, nil, 0) {
			break
		}
	}

	return 0
}

func (fs *FileSystem) Rewind(seconds int) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	target := time.Now().Add(-time.Duration(seconds) * time.Second)
	fs.engine.Rewind(target)

	return len(fs.engine.ListFiles())
}

func (fs *FileSystem) Undo() (int, bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !fs.engine.RewindSteps(1) {
		return 0, false
	}

	return len(fs.engine.ListFiles()), true
}

func (fs *FileSystem) Events() []chronofsengine.Event {
	return fs.engine.Events()
}

func Mount(mountPoint string, controls func(*FileSystem)) bool {
	fileSystem := New()

	if controls != nil {
		go func() {
			<-fileSystem.ready
			controls(fileSystem)
		}()
	}

	host := fuse.NewFileSystemHost(fileSystem)
	return host.Mount("", []string{mountPoint})
}

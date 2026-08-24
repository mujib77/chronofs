package fs

import (
	"sort"
	"sync"

	"github.com/winfsp/cgofuse/fuse"
)

type FileSystem struct {
	fuse.FileSystemBase
	mu    sync.RWMutex
	files map[string][]byte
}

func New() *FileSystem {
	return &FileSystem{
		files: map[string][]byte{
			"/README.md":  []byte("# ChronoFS\n\nThis file exists inside a time-scrubbable filesystem.\n"),
			"/status.txt": []byte("Timeline recording is active.\n"),
			"/demo.go":    []byte("package main\n\nfunc main() {\n\tprintln(\"Time is reversible\")\n}\n"),
		},
	}
}

func (fs *FileSystem) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if path == "/" {
		stat.Mode = fuse.S_IFDIR | 0555
		stat.Nlink = 1
		return 0
	}

	content, exists := fs.files[path]
	if !exists {
		return -fuse.ENOENT
	}

	stat.Mode = fuse.S_IFREG | 0444
	stat.Nlink = 1
	stat.Size = int64(len(content))
	return 0
}

func (fs *FileSystem) Open(path string, flags int) (int, uint64) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if _, exists := fs.files[path]; !exists {
		return -fuse.ENOENT, ^uint64(0)
	}

	return 0, 0
}

func (fs *FileSystem) Read(path string, buff []byte, ofst int64, fh uint64) int {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	content, exists := fs.files[path]
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

func (fs *FileSystem) Readdir(
	path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64,
) int {
	if path != "/" {
		return -fuse.ENOENT
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	fill(".", nil, 0)
	fill("..", nil, 0)

	paths := make([]string, 0, len(fs.files))
	for filePath := range fs.files {
		paths = append(paths, filePath)
	}

	sort.Strings(paths)

	for _, filePath := range paths {
		name := filePath[1:]
		if !fill(name, nil, 0) {
			break
		}
	}

	return 0
}

func Mount(mountPoint string) bool {
	host := fuse.NewFileSystemHost(New())
	return host.Mount("", []string{mountPoint})
}

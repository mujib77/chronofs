package fs

import "github.com/winfsp/cgofuse/fuse"

type FileSystem struct {
	fuse.FileSystemBase
}

func New() *FileSystem {
	return &FileSystem{}
}

func (fs *FileSystem) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
	if path != "/" {
		return -fuse.ENOENT
	}

	stat.Mode = fuse.S_IFDIR | 0555
	stat.Nlink = 1
	return 0
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

	fill(".", nil, 0)
	fill("..", nil, 0)
	return 0
}

func Mount(mountPoint string) bool {
	host := fuse.NewFileSystemHost(New())
	return host.Mount("", []string{mountPoint})
}
package fs

import (
	"os"
	"time"
)

type FileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi FileInfo) Name() string {
	return fi.name
}

func (fi FileInfo) Size() int64 {
	return fi.size
}

func (fi FileInfo) Mode() os.FileMode {
	return fi.mode
}

func (fi FileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi FileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

func (fi FileInfo) Sys() interface{} {
	return nil
}

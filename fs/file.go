package fs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileListing struct {
	data []byte
	dir  string
	fi   FileInfo
	fs   *FS
}

type File struct {
	*bytes.Reader
	*FileListing
}

var (
	errNoFS = errors.New("Not connected to an FS")
)

func NewFileListing(name string, size int64, mode os.FileMode, modTime time.Time, data []byte) *FileListing {
	dir, file := filepath.Split(name)

	if filepath.IsAbs(dir) {
		dir = filepath.Clean(dir)
	} else {
		if abs, err := filepath.Abs(dir); err == nil {
			dir = filepath.Clean(abs)
		} else {
			dir = filepath.Clean(string(filepath.Separator) + abs)[1:]
		}
	}

	dir = filepath.ToSlash(dir)

	if strings.HasSuffix(dir, "/") {
		dir = dir[:len(dir)-1]
	}

	if file == "" {
		index := strings.LastIndex(dir, "/")
		if index > 0 {
			file = dir[index+1:]
			dir = dir[:index]
		}
	}

	if file == "" {
		file = "."
	}

	return &FileListing{
		data: data,
		dir:  dir,
		fi: FileInfo{
			name:    file,
			size:    size,
			mode:    mode,
			modTime: modTime,
		},
	}
}

func (fl *FileListing) Open() File {
	return NewFile(fl)
}

func NewFile(fl *FileListing) File {
	r := bytes.NewReader(fl.data)
	return File{
		Reader:      r,
		FileListing: fl,
	}
}

func (f File) Close() error {
	_, err := f.Seek(0, 0)
	return err
}

func (fl *FileListing) Readdir(count int) (fi []os.FileInfo, err error) {
	if fl.fs == nil {
		return []os.FileInfo{}, errNoFS
	}

	if l, ok := fl.fs.getListing(fl.dir); ok {
		i := 0
		for _, listing := range l.files {
			var st os.FileInfo
			if st, err = listing.fileListing.Stat(); err == nil {
				if count <= 0 || i < count {
					fi = append(fi, st)
				}
				i++
			} else {
				break
			}
		}
	} else {
		err = os.ErrNotExist
	}

	return
}

func (fl *FileListing) Stat() (os.FileInfo, error) {
	return fl.fi, nil
}

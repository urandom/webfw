package fs

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"
)

func TestAdd(t *testing.T) {
	data := []byte("foobar")
	f := NewFileListing("/foo/bar/baz", int64(len(data)), 0, time.Now(), data)

	if !DefaultFS.Add(f) {
		t.Fatal("Couldn't add the file to the fs")
	}

	if l, ok := DefaultFS.root.files["foo"]; ok {
		st, _ := l.fileListing.Stat()
		if !st.IsDir() {
			t.Fatal("Listing for 'foo' should be a directory")
		}

		if l, ok := l.files["bar"]; ok {
			st, _ := l.fileListing.Stat()
			if !st.IsDir() {
				t.Fatal("Listing for 'bar' should be a directory")
			}

			if l, ok := l.files["baz"]; ok {
				st, _ := l.fileListing.Stat()
				if st.IsDir() {
					t.Fatal("Listing for 'baz' should be a file")
				}

				if b, err := ioutil.ReadAll(l.fileListing.Open()); err == nil {
					if !bytes.Equal(b, data) {
						t.Fatalf("Expected bytes: %v, got :%v\n", data, b)
					}
				} else {
					t.Fatalf("Error while trying to read file 'baz': %v\v", err)
				}
			} else {
				t.Fatal("Listing for file 'baz' not found")
			}
		} else {
			t.Fatal("Listing for directory 'bar' not found")
		}
	} else {
		t.Fatal("Listing for directory 'foo' not found")
	}

	if l, ok := DefaultFS.getListing("/foo/bar"); ok {
		if st, err := l.fileListing.Stat(); err != nil || !st.IsDir() {
			t.Fatal("Listing for '/foo/bar' is not a directory")
		}
	} else {
		t.Fatal("Can't find listing for '/foo/bar'")
	}

	f = NewFileListing("/foo/bar/baz/alpha", int64(len(data)), 0, time.Now(), data)
	if DefaultFS.Add(f) {
		t.Fatalf("Can't add a file to a file")
	}
}

func TestOpen(t *testing.T) {
	data := []byte("foobar")
	path := "/foo/bar/baz"
	f := NewFileListing(path, int64(len(data)), 0, time.Now(), data)
	fs := NewFS()

	if !fs.Add(f) {
		t.Fatal("Couldn't add the file to the fs")
	}

	if file, err := fs.Open(path); err == nil {
		if b, err := ioutil.ReadAll(file); err == nil {
			if !bytes.Equal(b, data) {
				t.Fatalf("Expected bytes: %v, got :%v\n", data, b)
			}
		} else {
			t.Fatalf("Error while trying to read file '': %v\v", path, err)
		}
	} else {
		t.Fatalf("Error trying to open file '%s': %v\n", path, err)
	}

	if file, err := fs.OpenRoot("/foo", "../bar/baz"); err == nil {
		if b, err := ioutil.ReadAll(file); err == nil {
			if !bytes.Equal(b, data) {
				t.Fatalf("Expected bytes: %v, got :%v\n", data, b)
			}
		} else {
			t.Fatalf("Error while trying to read file '': %v\v", path, err)
		}
	} else {
		t.Fatalf("Error trying to open file '%s': %v\n", path, err)
	}

	path = "fs_test.go"
	if file, err := fs.Open(path); err == nil {
		defer file.Close()
		if b, err := ioutil.ReadAll(file); err == nil {
			if !bytes.Contains(b, []byte("package fs")) {
				t.Fatalf("File '%s' doesn't contain 'package fs'\n", path)
			}
		} else {
			t.Fatalf("Error reading real file '%s': %v\b", path, err)
		}
	} else {
		t.Fatalf("Error trying to open file '%s': %v\n", path, err)
	}
}

package fs

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFile(t *testing.T) {
	data := []byte("foobar")
	f := NewFileListing("/foo/bar/baz", int64(len(data)), 0, time.Now(), data)

	expectedStr := "/foo/bar"
	if f.dir != expectedStr {
		t.Fatalf("Expected dir '%s', got '%s'\n", expectedStr, f.dir)
	}

	expectedStr = "baz"
	if f.fi.name != expectedStr {
		t.Fatalf("Expected name '%s', got '%s'\n", expectedStr, f.fi.name)
	}

	st, err := f.Stat()
	if err != nil {
		t.Fatalf("Expected Stat, got error %v\n", err)
	}

	if st.Name() != expectedStr {
		t.Fatalf("Expected name '%s', got '%s'\n", expectedStr, st.Name())
	}

	expectedInt64 := int64(len(data))
	if st.Size() != expectedInt64 {
		t.Fatalf("Expected size '%d', got '%d'\n", expectedInt64, st.Size())
	}

	if st.IsDir() {
		t.Fatal("Didn't expect a directory")
	}

	b, err := ioutil.ReadAll(f.Open())
	if err != nil {
		t.Fatalf("Error reading the file: %v\n", err)
	}

	if !bytes.Equal(b, data) {
		t.Fatalf("Expected data '%v', got '%v'\n", data, b)
	}

	f = NewFileListing("foo/bar/baz", int64(len(data)), 0, time.Now(), data)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	expectedStr = filepath.Join(cwd, "/foo/bar")
	if f.dir != expectedStr {
		t.Fatalf("Expected dir '%s', got '%s'\n", expectedStr, f.dir)
	}

	expectedStr = "baz"
	if f.fi.name != expectedStr {
		t.Fatalf("Expected name '%s', got '%s'\n", expectedStr, f.fi.name)
	}

	f = NewFileListing("/foo/bar/baz/", int64(len(data)), os.ModeDir, time.Now(), data)
	expectedStr = "/foo/bar"
	if f.dir != expectedStr {
		t.Fatalf("Expected dir '%s', got '%s'\n", expectedStr, f.dir)
	}

	expectedStr = "baz"
	if f.fi.name != expectedStr {
		t.Fatalf("Expected name '%s', got '%s'\n", expectedStr, f.fi.name)
	}

	st, err = f.Stat()
	if err != nil {
		t.Fatalf("Expected Stat, got error %v\n", err)
	}

	if !st.IsDir() {
		t.Fatal("Expected a directory")
	}

}

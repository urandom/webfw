package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	output    string
	pkg       string
	function  string
	fs        string
	buildTags string
	doFormat  bool

	tmpl = template.Must(template.New("go-template").Parse(goTemplate))
)

type templateData struct {
	Pkg      string
	Function string
	FS       string
	Tags     string

	Files []File
}

type File struct {
	Path    string
	Data    string
	Size    int64
	Mode    uint32
	ModTime int64
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "No input files/directories given")
		os.Exit(0)
	}

	if pkg == "" {
		fmt.Fprintln(os.Stderr, "No output file package given")
		os.Exit(0)
	}

	var out *os.File

	if output == "-" {
		out = os.Stdout
	} else {
		var err error
		if out, err = os.OpenFile(output, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error opening '%s' for writing: %v\n", output, err)
			os.Exit(0)
		}

		defer out.Close()
	}

	files := []File{}

	for _, arg := range flag.Args() {
		recursive := false
		if strings.HasSuffix(arg, "/...") {
			recursive = true
			arg = arg[:len(arg)-4]
		}
		func() {
			file, err := os.Open(arg)
			defer file.Close()

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening '%s': %v\n", arg, err)
				return
			}

			stat, err := file.Stat()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error stat-ing '%s': %v\n", arg, err)
				return
			}

			if stat.IsDir() {
				filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
					if info.IsDir() {
						if !recursive {
							return filepath.SkipDir
						}
					} else {
						if b, err := ioutil.ReadFile(path); err == nil {
							files = append(files, File{
								Path:    path,
								Data:    fmt.Sprintf("%q", b),
								Size:    info.Size(),
								ModTime: info.ModTime().Unix(),
								Mode:    uint32(info.Mode()),
							})
						} else {
							fmt.Fprintf(os.Stderr, "Error reading '%s': %v\n", path, err)
						}
					}

					return nil
				})
			} else {
				if b, err := ioutil.ReadAll(file); err == nil {
					files = append(files, File{
						Path:    arg,
						Data:    fmt.Sprintf("%q", b),
						Size:    stat.Size(),
						ModTime: stat.ModTime().Unix(),
						Mode:    uint32(stat.Mode()),
					})
				} else {
					fmt.Fprintf(os.Stderr, "Error reading '%s': %v\n", arg, err)
					return
				}

			}
		}()
	}

	buf := new(bytes.Buffer)
	err := tmpl.Execute(buf, templateData{Pkg: pkg, Function: function, FS: fs, Tags: buildTags, Files: files})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing template: %v\n", err)
		os.Exit(0)
	}

	if doFormat {
		if b, err := format.Source(buf.Bytes()); err == nil {
			buf.Reset()
			if _, err = buf.Write(b); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing formatted code to buffer: %v\n", err)
				os.Exit(0)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error formatting generated code: %v\n", err)
			os.Exit(0)
		}
	}

	if _, err := buf.WriteTo(out); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing template result to output: %v\n", err)
		os.Exit(0)
	}
}

func init() {
	flag.StringVar(&output, "output", "-", "the output file")
	flag.StringVar(&pkg, "package", "main", "the package of the output go file")
	flag.StringVar(&function, "function", "addFiles", "the function that will add the files to the fs")
	flag.StringVar(&fs, "fs", "DefaultFS", "the fs object to be used. Will be created if different from the default")
	flag.StringVar(&buildTags, "build-tags", "", "optional build tags for the output code")
	flag.BoolVar(&doFormat, "format", false, "run the output go code through go/format")
}

const goTemplate = `
{{ if .Tags }}// +build {{ .Tags }}
{{ end }}
package {{ .Pkg }}

// DO NOT EDIT ** This file was generated with the webfw-fs tool ** DO NOT EDIT //

import (
	"fmt"
	"os"
	"time"

	"github.com/urandom/webfw/fs"
)

type ErrNotAdded struct {
	Path string
}

func {{ .Function }}() (*fs.FS, error) {
{{ if eq .FS "DefaultFS" }}
	wfs := fs.DefaultFS
{{ else }}
	wfs := fs.NewFS()
{{ end }}
	var (
		size int64
		mode os.FileMode
		t    time.Time
	)
{{ range $index, $file := .Files }}
	size = int64({{ $file.Size }})
	mode = os.FileMode({{ $file.Mode }})
	t = time.Unix({{ $file.ModTime }}, 0)
	if !wfs.Add(fs.NewFileListing("{{ $file.Path }}", size, mode, t, []byte({{ $file.Data }}))) {
		return nil, &ErrNotAdded{Path: "$path"}
	}
{{ end }}
	return wfs, nil
}

func (e ErrNotAdded) Error() string {
	return fmt.Sprintf("Error adding file '%s'\n", e.Path)
}
`

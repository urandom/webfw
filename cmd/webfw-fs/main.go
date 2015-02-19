package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	input     string
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

	if pkg == "" {
		fmt.Fprintln(os.Stderr, "No output file package given")
		os.Exit(0)
	}

	if function == "" {
		fmt.Fprintln(os.Stderr, "No output file function given")
		os.Exit(0)
	}

	if fs == "" {
		fmt.Fprintln(os.Stderr, "No output file FS given")
		os.Exit(0)
	}

	var args []string
	var err error

	if input == "" {
		args = flag.Args()
	} else {
		var in *os.File
		if input == "-" {
			in = os.Stdin
		} else {
			if in, err = os.Open(input); err != nil {
				fmt.Fprintf(os.Stderr, "Error opening '%s': %v\n", input, err)
				os.Exit(0)
			}
		}

		r := bufio.NewReader(in)

		args, err = parseInput(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input '%s': %v\n", input, err)
			os.Exit(0)
		}
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "No input files/directories given")
		os.Exit(0)
	}

	var out *os.File

	if output == "-" {
		out = os.Stdout
	} else {
		if out, err = os.OpenFile(output, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error opening '%s' for writing: %v\n", output, err)
			os.Exit(0)
		}

		defer out.Close()
	}

	files := []File{}

	for _, arg := range args {
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
	err = tmpl.Execute(buf, templateData{Pkg: pkg, Function: function, FS: fs, Tags: buildTags, Files: files})
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
	flag.StringVar(&input, "input", "", "the input file. If omitted, any additional arguments will be used as input")
	flag.StringVar(&output, "output", "-", "the output file")
	flag.StringVar(&pkg, "package", "main", "the package of the output go file")
	flag.StringVar(&function, "function", "addFiles", "the function that will add the files to the fs")
	flag.StringVar(&fs, "fs", "DefaultFS", "the fs object to be used. Will be created if different from the default")
	flag.StringVar(&buildTags, "build-tags", "", "optional build tags for the output code")
	flag.BoolVar(&doFormat, "format", false, "run the output go code through go/format")
}

func parseInput(r *bufio.Reader) (args []string, err error) {
	for {
		buf, err := r.ReadSlice('\n')

		end := false
		if err == io.EOF {
			end = true
		} else if err != nil {
			return args, errors.New(fmt.Sprintf("Error reading line '%s': %v\n", buf, err))
		}

		index := bytes.IndexByte(buf, '#')
		if index != -1 {
			buf = buf[:index]
		}

		if len(buf) == 0 {
			if end {
				break
			}
			continue
		}

		if buf[len(buf)-1] == '\n' {
			buf = buf[:len(buf)-1]
		}

		if buf[len(buf)-1] == '\r' {
			buf = buf[:len(buf)-1]
		}

		args = append(args, string(buf))

		if end {
			break
		}
	}

	return
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

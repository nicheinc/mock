package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"text/template"

	"github.com/nicheinc/mock/iface"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
)

//go:embed template.tmpl
var tmpl string

const helpMessage = `Usage: %s [options] [interface]

When the positional interface argument is omitted, all interfaces in the search
directory annotated with a "go:mock [output file]" directive will be mocked and
output to stdout or, with the -w option, written to files. If a go:mock
directive in a file called example.go doesn't specify an output file, the
default output file will be the -o flag (if provided) or else example_mock.go.

When an interface name is provided as a positional argument after all other
flags, only that interface will be mocked. The -w option is incompatible with an
interface argument.

Options:
`

type config struct {
	dir        string
	outputFile string
	write      bool
}

func main() {
	var config config
	flag.StringVar(&config.dir, "d", ".", "Directory to search for interfaces in")
	flag.StringVar(&config.outputFile, "o", "", "Output file (default stdout)")
	flag.BoolVar(&config.write, "w", false, "Write mocks to files rather than stdout")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), helpMessage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// Load package info.
	pkgs, packageErr := packages.Load(&packages.Config{Mode: packages.LoadSyntax}, config.dir)
	if packageErr != nil {
		log.Fatalf(`Error loading package information: %s`, packageErr)
	}
	if len(pkgs) < 1 {
		log.Fatalf(`No packages found in %s`, config.dir)
	}

	filesByPath := func() map[string]iface.File {
		// The presence/absence of a positional argument determines whether
		// we're generating mocks for all interfaces annotated with "go:mock" or
		// for a single interface.
		if len(flag.Args()) < 1 {
			// Search all packages in the target directory for interfaces
			// annotated with "go:mock".
			filesByPath, getErr := iface.GetAllInterfaces(pkgs, config.outputFile)
			if getErr != nil {
				log.Fatalf(`Error getting interface information: %s`, getErr)
			}
			return filesByPath
		} else {
			if config.write {
				log.Fatalf("The -w option is only permitted when generating all mocks")
			}
			config.write = config.outputFile != ""

			// The first positional argument is the interface name. In this
			// case, the target directory must contain a single package.
			if len(pkgs) > 1 {
				log.Fatalf(`Found more than one package in %s`, config.dir)
			}
			// Search the package for info about the interface.
			file, getErr := iface.GetInterface(pkgs[0], flag.Args()[0])
			if getErr != nil {
				log.Fatalf("Error getting interface information: %s", getErr)
			}
			return map[string]iface.File{config.outputFile: file}
		}
	}()

	// Parse the template
	tmpl, templateErr := template.New("default").Parse(tmpl)
	if templateErr != nil {
		log.Fatalf("Error parsing template: %s", templateErr)
	}

	for outputPath, file := range filesByPath {
		// Execute/output the template for this interface.
		buf := &bytes.Buffer{}
		if executeErr := tmpl.Execute(buf, file); executeErr != nil {
			log.Fatalf("Error executing template: %s", executeErr)
		}

		// Format it with go imports.
		formatted, importsErr := imports.Process(outputPath, buf.Bytes(), nil)
		if importsErr != nil {
			log.Fatalf("Error formatting output: %s", importsErr)
		}

		// Open the output file, if provided, or use stdout.
		out := os.Stdout
		if config.write {
			var createErr error
			out, createErr = os.Create(outputPath)
			if createErr != nil {
				log.Fatalf("Error creating output file: %s", createErr)
			}
			defer out.Close()
		}

		// Write the formatted output to the file.
		if _, writeErr := out.Write(formatted); writeErr != nil {
			log.Fatalf("Error writing to file: %s", writeErr)
		}
	}
}

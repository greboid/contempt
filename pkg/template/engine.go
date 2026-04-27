package template

import (
	"fmt"
	"github.com/csmith/contempt/pkg/materials"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
)

// Engine is responsible for evaluating templates and producing outputs.
type Engine struct {
	logger     *slog.Logger
	functions  template.FuncMap
	bom        materials.BOM
	includes   fs.FS
	versionMap map[string]string
}

// NewEngine creates a new templating engine that will read template includes
// from the given file system.
func NewEngine(logger *slog.Logger, includes fs.FS) *Engine {
	return &Engine{
		logger:    logger.With("component", "template.Engine"),
		includes:  includes,
		functions: make(template.FuncMap),
	}
}

// Register registers functions for use in templates. The FunctionSource is
// given a BomWriter that it should call whenever functions are called that
// depend on some material.
//
// If Register is called with sources that return functions with conflicting
// names, later instances of the functions replace earlier ones.
func (e *Engine) Register(source FunctionSource) {
	functions := source(&engineBomWriter{engine: e})

	for i := range functions {
		e.logger.Debug("registered template function", "name", i)
		e.functions[i] = functions[i]
	}
}

// DryRun parses and executes the template at the given path, but wraps all
// registered functions with no-ops that simply record their arguments.
func (e *Engine) DryRun(path string) (map[string][][]any, error) {
	e.logger.Debug("dry run of template", "path", path)
	tpl := template.New(filepath.Base(path))
	dryFuncs := template.FuncMap{}
	calls := make(map[string][][]any)

	for f := range e.functions {
		out := reflect.ValueOf(e.functions[f]).Type().Out(0).Kind()
		if out == reflect.String {
			dryFuncs[f] = func(args ...any) string {
				calls[f] = append(calls[f], args)
				return ""
			}
		} else if out == reflect.Slice {
			dryFuncs[f] = func(args ...any) []string {
				calls[f] = append(calls[f], args)
				return nil
			}
		} else if out == reflect.Map {
			dryFuncs[f] = func(args ...any) map[string]string {
				calls[f] = append(calls[f], args)
				return nil
			}
		} else if out == reflect.Int {
			dryFuncs[f] = func(args ...any) int {
				calls[f] = append(calls[f], args)
				return 0
			}
		} else {
			return nil, fmt.Errorf("template function %s has unsupported return type: %v", f, out)
		}
	}

	tpl.Funcs(dryFuncs)

	// Parse includes
	if _, err := tpl.ParseFS(e.includes, "*.gotpl"); err != nil {
		if !strings.Contains(err.Error(), "pattern matches no files") {
			// Urgh.
			e.logger.Error("failed to parse included templates", "err", err, "dry-run", true)
			return nil, err
		}
	}

	// Parse the actual template
	if _, err := tpl.ParseFiles(path); err != nil {
		e.logger.Error("failed to parse template", "path", path, "err", err, "dry-run", true)
		return nil, err
	}

	// Reset the BOM and execute the template
	e.bom = make(materials.BOM)
	err := tpl.ExecuteTemplate(io.Discard, filepath.Base(path), nil)
	if err != nil {
		e.logger.Error("failed to execute template", "path", path, "err", err, "dry-run", true)
		return nil, err
	}

	return calls, err
}

// Execute parses the template at the given path, executes it, and writes it to
// the given writer.
//
// As the template is being executed, functions registered with [Register] may
// call their [BomWriter] to add material to the bill. These materials are
// collated and returned once the template has been executed.
//
// Because of the way materials are gathered, the Execute method is not thread
// safe.
func (e *Engine) Execute(out io.Writer, path string) (materials.BOM, error) {
	e.logger.Debug("executing template", "path", path)
	tpl := template.New(filepath.Base(path))
	tpl.Funcs(e.functions)

	// Parse includes
	if _, err := tpl.ParseFS(e.includes, "*.gotpl"); err != nil {
		if !strings.Contains(err.Error(), "pattern matches no files") {
			// Urgh.
			e.logger.Error("failed to parse included templates", "err", err)
			return nil, err
		}
	}

	// Parse the actual template
	if _, err := tpl.ParseFiles(path); err != nil {
		e.logger.Error("failed to parse template", "path", path, "err", err)
		return nil, err
	}

	// Reset the BOM and execute the template
	e.bom = make(materials.BOM)
	err := tpl.ExecuteTemplate(out, filepath.Base(path), nil)
	if err != nil {
		e.logger.Error("failed to execute template", "path", path, "err", err)
		return nil, err
	}

	return e.bom, err
}

// ExecuteWithBOM parses the template at the given path, executes it, and writes it to
// the given writer. Unlike Execute, it uses the provided BOM instead of resolving
// new versions. Template functions will still be called, but their version lookups
// are ignored in favor of the provided BOM.
func (e *Engine) ExecuteWithBOM(out io.Writer, path string, bom materials.BOM) error {
	e.logger.Debug("executing template with BOM", "path", path)
	tpl := template.New(filepath.Base(path))
	tpl.Funcs(e.functions)

	// Parse includes
	if _, err := tpl.ParseFS(e.includes, "*.gotpl"); err != nil {
		if !strings.Contains(err.Error(), "pattern matches no files") {
			e.logger.Error("failed to parse included templates", "err", err)
			return err
		}
	}

	// Parse the actual template
	if _, err := tpl.ParseFiles(path); err != nil {
		e.logger.Error("failed to parse template", "path", path, "err", err)
		return err
	}

	// Set the version map for approved versions
	e.versionMap = bom
	e.bom = bom
	defer func() { e.versionMap = nil }()

	err := tpl.ExecuteTemplate(out, filepath.Base(path), nil)
	if err != nil {
		e.logger.Error("failed to execute template", "path", path, "err", err)
		return err
	}

	return err
}

type engineBomWriter struct {
	engine *Engine
}

func (e *engineBomWriter) Write(material, version string) string {
	if e.engine.versionMap != nil {
		if approvedVersion, ok := e.engine.versionMap[material]; ok {
			e.engine.logger.Debug("using approved version", "material", material, "version", approvedVersion)
			e.engine.bom[material] = approvedVersion
			return approvedVersion
		}
	}
	e.engine.logger.Debug("gathered material", "material", material, "version", version)
	e.engine.bom[material] = version
	return version
}

type FunctionSource = func(BomWriter) template.FuncMap

type BomWriter interface {
	Write(material, version string) string
}

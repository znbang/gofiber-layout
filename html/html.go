package html

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// Engine struct
type Engine struct {
	// delimiters
	left  string
	right string
	// views folder
	directory string
	// http.FileSystem supports embedded files
	fileSystem http.FileSystem
	// views extension
	extension string
	// layout variable name that incapsulates the template
	layout string
	// determines if the engine parsed all templates
	loaded bool
	// reload on each render
	reload bool
	// debug prints the parsed templates
	debug bool
	// lock for funcmap and templates
	mutex sync.RWMutex
	// template funcmap
	funcmap map[string]interface{}
	// templates
	Templates map[string]*template.Template

	//used for walking fileSystem, not serving
	rawFileSystem  fs.FS
}

// New returns a HTML render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		left:      "{{",
		right:     "}}",
		directory: directory,
		extension: extension,
		layout:    "",
		funcmap:   make(map[string]interface{}),
	}
	return engine
}

//NewFileSystem ...
func NewFileSystem(httpFS http.FileSystem, rawFS fs.FS, ext string) *Engine {
	engine := &Engine{
		left:       "{{",
		right:      "}}",
		directory:  ".",
		fileSystem: httpFS,
		rawFileSystem: rawFS,
		extension:  ext,
		layout:     "",
		funcmap:    make(map[string]interface{}),
	}
	return engine
}

func toFS(hfs http.FileSystem) fs.FS {
	if dir, ok := hfs.(http.Dir); ok {
		return os.DirFS(string(dir))
	}
	panic("unsupported http.FileSystem type")
}


// Layout defines the variable name that will incapsulate the template
func (e *Engine) Layout(key string) *Engine {
	e.layout = key
	return e
}

// Delims sets the action delimiters to the specified strings, to be used in
// templates. An empty delimiter stands for the
// corresponding default: {{ or }}.
func (e *Engine) Delims(left, right string) *Engine {
	e.left, e.right = left, right
	return e
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions
func (e *Engine) AddFunc(name string, fn interface{}) *Engine {
	e.mutex.Lock()
	e.funcmap[name] = fn
	e.mutex.Unlock()
	return e
}

// Reload if set to true the templates are reloading on each render,
// use it when you're in development and you don't want to restart
// the application when you edit a template file.
func (e *Engine) Reload(enabled bool) *Engine {
	e.reload = enabled
	return e
}

// Debug will print the parsed templates when Load is triggered.
func (e *Engine) Debug(enabled bool) *Engine {
	e.debug = enabled
	return e
}

// Parse is deprecated, please use Load() instead
func (e *Engine) Parse() error {
	fmt.Println("Parse() is deprecated, please use Load() instead.")
	return e.Load()
}

// ReadFile function to replace the deprecated utils.ReadFile() call
func readFile(path string, fsys http.FileSystem) ([]byte, error) {
	if fsys != nil {
		f, err := fsys.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(f)
	}
	return os.ReadFile(path)
}

// walkFS replaces utils.Walk for embedded or OS files
func walkFS(fsys fs.FS, root string, walkFn filepath.WalkFunc) error {
	return fs.WalkDir(fsys, root, func(entryPath string, d fs.DirEntry, err error) error {
		if err != nil {
			// If the DirEntry is nil due to an error, call walkFn with nil info
			return walkFn(entryPath, nil, err)
		}
		if d == nil {
			return walkFn(entryPath, nil, fmt.Errorf("nil DirEntry for path: %s", entryPath))
		}
		info, statErr := d.Info()
		if statErr != nil {
			return walkFn(entryPath, nil, statErr)
		}
		return walkFn(entryPath, info, nil)
	})
}




// Wrap fs.FS into http.FileSystem
func ToHTTPFileSystem(fsys fs.FS) http.FileSystem {
	return http.FS(fsys)
}


// Load parses the templates to the engine.
func (e *Engine) Load() error {
	if e.loaded {
		return nil
	}
	// race safe
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.Templates = make(map[string]*template.Template)

	// Load layout using ReadFile function
	var layoutBuf []byte
	if e.layout != "" {
		var err error
		layoutPath := path.Join(e.directory, e.layout+e.extension)
		if layoutBuf, err = readFile(layoutPath, e.fileSystem); err != nil {
			return err
		}
}


	walkFn := func(path string, info os.FileInfo, err error) error {
		// Return error if exist
		if err != nil {
			return err
		}
		// Skip file if it's a directory or has no file info
		if info == nil || info.IsDir() {
			return nil
		}
		// Get file extension of file
		ext := filepath.Ext(path)
		// Skip file if it does not equal the given template extension
		if ext != e.extension {
			return nil
		}
		// Skip layout
		if e.layout != "" && strings.HasSuffix(path, e.layout+e.extension) {
			return nil
		}
		// Get the relative file path
		// ./views/html/index.tmpl -> index.tmpl
		rel, err := filepath.Rel(e.directory, path)
		if err != nil {
			return err
		}
		// Reverse slashes '\' -> '/' and
		// partials\footer.tmpl -> partials/footer.tmpl
		name := filepath.ToSlash(rel)
		// Remove ext from name 'index.tmpl' -> 'index'
		name = strings.TrimSuffix(name, e.extension)
		// name = strings.Replace(name, e.extension, "", -1)
		// Read the file
		// #gosec G304
		//
		//buf, err := utils.ReadFile(path, e.fileSystem) This is deprecated in the latest version of gofiber
		buf, err := readFile(path, e.fileSystem)
		if err != nil {
			return err
		}
		// Create new template
		var tmpl *template.Template
		if e.layout != "" {
			tmpl = template.New(e.layout)
		} else {
			tmpl = template.New(name)
		}
		// Set template settings
		tmpl.Delims(e.left, e.right)
		tmpl.Funcs(e.funcmap)
		// Parse layout
		if e.layout != "" {
			if _, err = tmpl.Parse(string(layoutBuf)); err != nil {
				return err
			}
			if _, err = tmpl.New(name).Parse(string(buf)); err != nil {
				return err
			}
		} else {
			if _, err = tmpl.Parse(string(buf)); err != nil {
				return err
			}
		}
		e.Templates[name] = tmpl
		// Debugging
		if e.debug {
			fmt.Printf("views: parsed template: %s\n", name)
		}
		return err
	}
	// notify engine that we parsed all templates
	e.loaded = true
	if e.fileSystem != nil {
		//return utils.Walk(e.fileSystem, e.directory, walkFn) utils.Walk is deprecated in the latest version of gofiber
		return walkFS(e.rawFileSystem, e.directory, walkFn)
	}
	return filepath.Walk(e.directory, walkFn)
}

// Render will execute the template name along with the given values.
func (e *Engine) Render(out io.Writer, template string, binding interface{}, layout ...string) error {
	if !e.loaded || e.reload {
		if e.reload {
			e.loaded = false
		}
		if err := e.Load(); err != nil {
			return err
		}
	}
	tmpl := e.Templates[template]
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", template)
	}
	if len(layout) > 0 {
		return fmt.Errorf("render: layout argument is not supported")
	}
	return tmpl.Execute(out, binding)
}

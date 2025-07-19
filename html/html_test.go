package html

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

func trim(str string) string {
	trimmed := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
	trimmed = strings.Replace(trimmed, " <", "<", -1)
	trimmed = strings.Replace(trimmed, "> ", ">", -1)
	return trimmed
}

func Test_Render(t *testing.T) {
	engine := New("./views", ".html")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})
	// engine.Layout("layouts/main")
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	// Partials
	var buf bytes.Buffer
	engine.Render(&buf, "home", map[string]interface{}{
		"Title": "Hello, World!",
	})
	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
	// Single
	buf.Reset()
	engine.Render(&buf, "errors/404", map[string]interface{}{
		"Error": "404 Not Found!",
	})
	expect = `<h1>404 Not Found!</h1>`
	result = trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_AddFunc(t *testing.T) {
	engine := New("./views", ".html")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	// Func is admin
	var buf bytes.Buffer
	engine.Render(&buf, "admin", map[string]interface{}{
		"User": "admin",
	})
	expect := `<h1>Hello, Admin!</h1>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}

	// Func is not admin
	buf.Reset()
	engine.Render(&buf, "admin", map[string]interface{}{
		"User": "john",
	})
	expect = `<h1>Access denied!</h1>`
	result = trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_Layout(t *testing.T) {
	engine := New("./views", ".html")
	engine.Layout("layouts/main")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	})
	expect := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

//Test_Layout_Multi checks if the layout can be rendered multiple times
func Test_Layout_Multi(t *testing.T) {
	engine := New("./views", ".html")
	engine.Layout("layouts/main")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	for i := 0; i < 2; i++ {
		var buf bytes.Buffer
		err := engine.Render(&buf, "index", map[string]interface{}{
			"Title": "Hello, World!",
		})
		expect := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
		result := trim(buf.String())
		if expect != result {
			t.Fatalf("\nExpected:\n%s\nResult:\n%s\n\nError: %s", expect, result, err)
		}
	}

}


func Test_FileSystem(t *testing.T) {
	// Step 1: Set up filesystem
	fsys := os.DirFS("./views")       // fs.FS
	httpFS := http.FS(fsys)           // http.FileSystem

	// Step 2: Create engine
	engine := NewFileSystem(httpFS, fsys, ".html")

	// Step 3: Set layout
	engine.Layout("layouts/main")

	// Step 4: Add function
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})

	// Step 5: Load templates
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	// Step 6: Render
	var buf bytes.Buffer
	engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	})

	expect := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())

	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}



func Test_Reload(t *testing.T) {
	// Define both file systems based on disk
	httpFS := http.Dir("./views")
	rawFS := os.DirFS("./views")

	// Now pass both along with extension
	engine := NewFileSystem(httpFS, rawFS, ".html")
	engine.Reload(true) // Enable reloading for test

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	// Write "after reload" to the file
	if err := ioutil.WriteFile("./views/reload.html", []byte("after reload\n"), 0644); err != nil {
		t.Fatalf("write file: %v\n", err)
	}
	defer func() {
		// Restore file content after test
		if err := ioutil.WriteFile("./views/reload.html", []byte("before reload\n"), 0644); err != nil {
			t.Fatalf("restore file: %v\n", err)
		}
	}()

	engine.Load() // Reload templates again

	var buf bytes.Buffer
	engine.Render(&buf, "reload", nil)

	expect := "after reload"
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

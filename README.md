This package modified Fiber's [html](https://github.com/gofiber/template/tree/master/html) template engine to support layout using block and define.
### Installation
```
go get -u github.com/znbang/gofiber-layout
```

### Example
#### views/layouts/main.html
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{block "title" .}}Layout{{end}}</title>
</head>
<body>
{{block "content" .}}
This is layout.html.
{{end}}
</body>
</html>
```
#### views/index.html
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{define "title"}}Index{{end}}</title>
</head>
<body>
{{define "content"}}
This is index.html.
{{end}}
</body>
</html>
```

#### main.go
```go
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/znbang/gofiber-layout/html"
)

//go:embed views/*
var viewsFS embed.FS

func main() {
	subFS, err := fs.Sub(viewsFS, "views")
	if err != nil {
		panic(err)
	}

	engine := html.NewFileSystem(http.FS(subFS), ".html")
	engine.Layout("layouts/main")

	app := fiber.New(fiber.Config{
		Views: engine,
	})
	app.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.Render("index", fiber.Map{})
	})
	log.Fatal(app.Listen(":9000"))
}
```

#### Generated HTML
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Index</title>
</head>
<body>

This is index.html.

</body>
</html>
```
# go-xmlstarlet-validate

[![Go Reference](https://pkg.go.dev/badge/github.com/afansv/go-xmlstarlet-validate.svg)](https://pkg.go.dev/github.com/afansv/go-xmlstarlet-validate)

Go wrapper for [`xmlstarlet`](http://xmlstar.sourceforge.net/) that allows validating XML documents against **XSD**, **DTD**, or **RelaxNG** schemas.
The package supports both file-based and stream-based validation and returns structured errors with precise positions.

### Features

* Validate XML documents against **XSD**, **DTD** or **RelaxNG**
* Validate by file names or `io.Reader` streams.
* Collect structured validation errors
* Option to stop at the first error

### Requirements

* Go 1.20+
* Installed [`xmlstarlet`](http://xmlstar.sourceforge.net/) binary available in `$PATH`.

---

#### Installation of `xmlstarlet`

This package requires the `xmlstarlet` command-line tool to be available in your `$PATH`.
Check if it is already installed:

```bash
xmlstarlet --version
```

If not, install it depending on your system:

##### Linux

Most distributions provide it via their package manager:

* Debian / Ubuntu

  ```bash
  sudo apt-get update
  sudo apt-get install xmlstarlet
  ```
* Fedora

  ```bash
  sudo dnf install xmlstarlet
  ```
* Arch Linux

  ```bash
  sudo pacman -S xmlstarlet
  ```

##### macOS

With [Homebrew](https://brew.sh/):

```bash
brew install xmlstarlet
```

##### Windows

Options:

1. Chocolatey

   ```powershell
   choco install xmlstarlet
   ```

2. Scoop

   ```powershell
   scoop install xmlstarlet
   ```

3. Manual

  * Download from [xmlstar.sourceforge.net](http://xmlstar.sourceforge.net/download.php)
  * Extract and add the folder with `xml.exe` (or `xmlstarlet.exe`) to your system `PATH`.


### Installation

```bash
go get -u github.com/afansv/go-xmlstarlet-validate@latest
```

### Usage

```go
package main

import (
	"fmt"
	"os"
	"strings"

	validate "github.com/afansv/go-xmlstarlet-validate"
)

func main() {
	// Load schema from file
	xsdFile, _ := os.Open("schema.xsd")
	defer xsdFile.Close()

	schema, err := validate.NewSchemaFromReader(xsdFile, validate.SchemaTypeXSD)
	if err != nil {
		panic(err)
	}
	defer schema.Clean()

	// Validate XML string
	xml := `<root><invalid/></root>`
	result, err := validate.ValidateFromReaderAgainstSchema(schema, strings.NewReader(xml), false)
	if err != nil {
		panic(err)
	}

	if result.Valid {
		fmt.Println("Document is valid")
	} else {
		for _, p := range result.Problems {
			fmt.Printf("%s:%d.%d: %s\n", p.Filename, p.Line, p.Col, p.Issue)
		}
	}
}
```
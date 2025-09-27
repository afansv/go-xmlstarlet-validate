# go-xmlstarlet-validate

Go wrapper for [`xmlstarlet`](http://xmlstar.sourceforge.net/) that allows validating XML documents against **XSD**, **DTD**, or **RelaxNG** schemas.
The package supports both file-based and stream-based validation and returns structured errors with precise positions.

[![Go Reference](https://pkg.go.dev/badge/github.com/afansv/go-xmlstarlet-validate.svg)](https://pkg.go.dev/github.com/afansv/go-xmlstarlet-validate)

### Features

* Validate XML documents against:

    * **XSD**
    * **DTD**
    * **RelaxNG**
* Validate by file names or `io.Reader` streams.
* Collect structured validation errors:

    * file name
    * line number
    * column number
    * issue description
* Option to stop at the first error (`-S` flag in `xmlstarlet`).

### Requirements

* Go 1.20+
* Installed [`xmlstarlet`](http://xmlstar.sourceforge.net/) binary available in `$PATH`.

Check installation:

```bash
xmlstarlet --version
```

### Installation

```bash
go get -u github.com/afansv/go-xmlstarlet-validate
```

### Example

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
	result, err := validate.ValidateThroughSchemaFromReader(schema, strings.NewReader(xml), false)
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
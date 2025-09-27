package xmlstarlet_validate

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type ResultProblem struct {
	Filename string
	Line     int
	Col      int
	Issue    string
}

type ValidateResult struct {
	Problems []ResultProblem
	Valid    bool
}

// Err returns combined error with all problems found.
// It returns nil if content is valid
func (r ValidateResult) Err() error {
	if r.Valid {
		return nil
	}
	var errs []error
	for _, p := range r.Problems {
		errs = append(
			errs,
			fmt.Errorf("%s (line: %d, col: %d)", p.Issue, p.Line, p.Col),
		)
	}
	if len(errs) == 0 {
		return fmt.Errorf("content is invalid, but it is not possible to determine why")
	}
	return errors.Join(errs...)
}

type SchemaType int

const (
	SchemaTypeXSD SchemaType = iota + 1
	SchemaTypeDTD
	SchemaTypeRelaxNG
)

const executable = "xmlstarlet"

type Schema struct {
	file       *os.File
	schemaType SchemaType
}

func (s *Schema) Cleanup() error {
	return os.Remove(s.file.Name())
}

// NewSchemaFromReader creates new Schema instance with SchemaType from io.Reader.
// Remember to call Schema.Cleanup after stop working with Schema.
func NewSchemaFromReader(r io.Reader, schemaType SchemaType) (*Schema, error) {
	f, err := os.CreateTemp("", "go-xmlstarlet-validate-schema-sch-*.xml")
	if err != nil {
		return nil, fmt.Errorf("create temp: %w", err)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	if _, err = io.Copy(f, r); err != nil {
		return nil, fmt.Errorf("copy from reader to temp file: %w", err)
	}
	return &Schema{
		file:       f,
		schemaType: schemaType,
	}, nil
}

// ValidateFromReader validates data from io.Reader against Schema
func ValidateFromReader(s *Schema, r io.Reader, stopOnFirstErr bool) (ValidateResult, error) {
	f, err := os.CreateTemp("", "go-xmlstarlet-validate-schema-obj-*.xml")
	if err != nil {
		return ValidateResult{}, fmt.Errorf("create temp: %w", err)
	}
	defer func(f *os.File) {
		_ = os.Remove(f.Name())
	}(f)
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	if _, err = io.Copy(f, r); err != nil {
		return ValidateResult{}, fmt.Errorf("copy from reader to temp file: %w", err)
	}
	return ValidateByFilenames(f.Name(), s.file.Name(), s.schemaType, stopOnFirstErr)
}

// ValidateByFilenames validates file with filenames
func ValidateByFilenames(filename, schemaFilename string, schemaType SchemaType, stopOnFirstErr bool) (ValidateResult, error) {
	args := []string{
		"val",
		"-e",
	}

	switch schemaType {
	case SchemaTypeXSD:
		args = append(args, []string{
			"-s", schemaFilename,
		}...)
	case SchemaTypeDTD:
		args = append(args, []string{
			"-d", schemaFilename,
		}...)
	case SchemaTypeRelaxNG:
		args = append(args, []string{
			"-r", schemaFilename,
		}...)
	default:
		return ValidateResult{}, fmt.Errorf("unsupported schema type (%d)", schemaType)
	}

	if stopOnFirstErr {
		args = append(args, "-S")
	}

	args = append(args, filename)

	cmd := exec.Command(executable, args...)

	outputData, err := cmd.CombinedOutput()
	if err != nil && len(outputData) == 0 {
		return ValidateResult{}, fmt.Errorf("run and access output: %w", err)
	}

	output := string(outputData)

	problems, valid := parseValidateOutputLines(strings.Split(output, "\n"))

	return ValidateResult{
		Problems: problems,
		Valid:    valid,
	}, nil
}

func parseValidateOutputLines(lines []string) (problems []ResultProblem, valid bool) {
	for _, line := range lines {
		if strings.HasSuffix(line, " - valid") {
			return nil, true
		}

		problemParts := strings.SplitN(strings.TrimPrefix(line, "file:///"), ":", 4)

		// windows patch C:\ or c:/
		if len(problemParts) == 4 {
			if strings.HasPrefix(problemParts[1], "\\") || strings.HasPrefix(problemParts[1], "/") {
				problemParts = []string{problemParts[0] + ":" + problemParts[1], problemParts[2], problemParts[3]}
			} else {
				problemParts = []string{problemParts[0], problemParts[1], problemParts[2] + ":" + problemParts[3]}
			}
		}

		if len(problemParts) != 3 {
			continue
		}

		issue := strings.TrimSpace(problemParts[2])
		whereFileName := problemParts[0]

		whereLineParts := strings.Split(problemParts[1], ".")
		if len(whereLineParts) != 2 {
			continue
		}

		whereLine, err := strconv.Atoi(whereLineParts[0])
		if err != nil {
			continue
		}
		whereCol, err := strconv.Atoi(whereLineParts[1])
		if err != nil {
			continue
		}

		problems = append(
			problems,
			ResultProblem{
				Filename: whereFileName,
				Line:     whereLine,
				Col:      whereCol,
				Issue:    issue,
			},
		)
	}

	return problems, valid
}

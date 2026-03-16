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

const defaultExecutable = "xmlstarlet"

type Option func(*options)

type options struct {
	executable string
}

func WithExecutable(path string) Option {
	return func(o *options) {
		o.executable = path
	}
}

func getExecutable(opts []Option) string {
	o := options{executable: defaultExecutable}
	for _, opt := range opts {
		opt(&o)
	}
	return o.executable
}

type Schema struct {
	filename   string
	needClean  bool
	schemaType SchemaType
}

func (s *Schema) Cleanup() error {
	if !s.needClean {
		return nil
	}
	return os.Remove(s.filename)
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
		filename:   f.Name(),
		needClean:  true,
		schemaType: schemaType,
	}, nil
}

// NewSchemaFromFilename creates new Schema instance with SchemaType from filename.
func NewSchemaFromFilename(filename string, schemaType SchemaType) (*Schema, error) {
	if _, err := os.Stat(filename); err != nil {
		return nil, err
	}
	return &Schema{
		filename:   filename,
		schemaType: schemaType,
	}, nil
}

// ValidateFromReader validates data from io.Reader against Schema
func ValidateFromReader(s *Schema, r io.Reader, stopOnFirstErr bool, opts ...Option) (ValidateResult, error) {
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
	return ValidateByFilenames(f.Name(), s.filename, s.schemaType, stopOnFirstErr, opts...)
}

// ValidateByFilenames validates file with filenames
func ValidateByFilenames(filename, schemaFilename string, schemaType SchemaType, stopOnFirstErr bool, opts ...Option) (ValidateResult, error) {
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

	execPath := getExecutable(opts)
	cmd := exec.Command(execPath, args...)

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
		line = strings.TrimSpace(line)

		if strings.HasSuffix(line, " - valid") {
			return nil, true
		}

		// Find pattern: filename:line.col: message
		// We need to find where line.col ends to get the message

		// Remove file:/// prefix if present
		line = strings.TrimPrefix(line, "file:///")

		// Find position of : that separates filename:line.col from message
		// Look for : followed by space and then message
		sepIdx := strings.Index(line, ": ")
		if sepIdx < 0 {
			continue
		}

		message := strings.TrimSpace(line[sepIdx+2:])
		beforeMessage := line[:sepIdx]

		// Now find last : in beforeMessage to separate filename from line.col
		lastColonIdx := strings.LastIndex(beforeMessage, ":")
		if lastColonIdx < 0 {
			continue
		}

		lineColPart := beforeMessage[lastColonIdx+1:]
		whereLineParts := strings.Split(lineColPart, ".")
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

		whereFileName := beforeMessage[:lastColonIdx]

		problems = append(
			problems,
			ResultProblem{
				Filename: whereFileName,
				Line:     whereLine,
				Col:      whereCol,
				Issue:    message,
			},
		)
	}

	return problems, valid
}

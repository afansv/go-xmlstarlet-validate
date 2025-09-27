package xmlstarlet_validate

import (
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

func (s *Schema) Clean() error {
	return os.Remove(s.file.Name())
}

// NewSchemaFromReader creates new Schema instance with SchemaType from io.Reader.
// Remember to call Schema.Clean after stop working with Schema.
func NewSchemaFromReader(r io.Reader, schemaType SchemaType) (*Schema, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("readall: %w", err)
	}

	f, err := os.CreateTemp("", "go-xmlstarlet-validate-schema-sch-*.xml")
	if err != nil {
		return nil, fmt.Errorf("create temp: %w", err)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	_, err = f.Write(data)
	if err != nil {
		return nil, fmt.Errorf("write to temp file: %w", err)
	}

	return &Schema{
		file:       f,
		schemaType: schemaType,
	}, nil
}

// ValidateThroughSchemaFromReader validates data from io.Reader through Schema
func ValidateThroughSchemaFromReader(s *Schema, r io.Reader, stopOnFirstErr bool) (ValidateResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return ValidateResult{}, fmt.Errorf("readall: %w", err)
	}

	f, err := os.CreateTemp("", "go-xmlstarlet-validate-schema-obj-*.xml")
	if err != nil {
		return ValidateResult{}, fmt.Errorf("create temp: %w", err)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	defer func(f *os.File) {
		_ = os.Remove(f.Name())
	}(f)

	_, err = f.Write(data)
	if err != nil {
		return ValidateResult{}, fmt.Errorf("write to temp file: %w", err)
	}

	return ValidateFile(f.Name(), s.file.Name(), s.schemaType, stopOnFirstErr)
}

// ValidateFile validates file by filename and schemaFilename
func ValidateFile(filename, schemaFilename string, schemaType SchemaType, stopOnFirstErr bool) (ValidateResult, error) {
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

	problems, valid, err := parseValidateOutputLines(strings.Split(output, "\n"))
	if err != nil {
		return ValidateResult{}, err
	}

	return ValidateResult{
		Problems: problems,
		Valid:    valid,
	}, nil
}

func parseValidateOutputLines(lines []string) (problems []ResultProblem, valid bool, err error) {
	for _, line := range lines {
		if line == "" {
			continue
		}
		isValid := strings.HasSuffix(line, " - valid")
		isInvalid := strings.HasSuffix(line, " - invalid")

		if isValid {
			return nil, true, nil
		}

		if isInvalid {
			valid = false
			continue
		}

		line = strings.TrimPrefix(line, "file:///")

		problemParts := strings.SplitN(line, ":", 4)

		// windows patch C:\ or c:/
		if len(problemParts) == 4 {
			if strings.HasPrefix(problemParts[1], "\\") || strings.HasPrefix(problemParts[1], "/") {
				problemParts = []string{problemParts[0] + ":" + problemParts[1], problemParts[2], problemParts[3]}
			} else {
				problemParts = []string{problemParts[0], problemParts[1], problemParts[2] + ":" + problemParts[3]}
			}
		}

		if len(problemParts) != 3 {
			return nil, false, fmt.Errorf("unexpeted line format")
		}

		issue := strings.TrimSpace(problemParts[2])
		whereFileName := problemParts[0]

		whereLineParts := strings.Split(problemParts[1], ".")
		if len(whereLineParts) != 2 {
			continue
		}

		whereLine, err := strconv.Atoi(whereLineParts[0])
		if err != nil {
			return nil, false, fmt.Errorf("unexpected line format - bad where line part - line: %w", err)
		}
		whereCol, err := strconv.Atoi(whereLineParts[1])
		if err != nil {
			return nil, false, fmt.Errorf("unexpected line format - bad where line part - col: %w", err)
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

	return problems, valid, nil
}

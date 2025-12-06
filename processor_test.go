package fileprep

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/parquet-go/parquet-go"
)

// TestRecord is a test struct for processing
type TestRecord struct {
	Name  string `prep:"trim" validate:"required"`
	Email string `prep:"trim"`
	Age   string
}

func TestProcessor_Process_CSV(t *testing.T) {
	t.Parallel()

	csvData := `name,email,age
  John Doe  ,john@example.com,30
Jane Smith,jane@example.com,25
  ,invalid,
Bob Wilson,bob@example.com,35
`

	tests := []struct {
		name           string
		input          string
		wantRowCount   int
		wantValidCount int
		wantErrorCount int
	}{
		{
			name:           "CSV with trim and required validation",
			input:          csvData,
			wantRowCount:   4,
			wantValidCount: 3, // One row has empty name after trim
			wantErrorCount: 1, // One required validation error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			processor := NewProcessor(FileTypeCSV)
			var records []TestRecord

			reader, result, err := processor.Process(strings.NewReader(tt.input), &records)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if reader == nil {
				t.Fatal("Process() returned nil reader")
			}

			if result.RowCount != tt.wantRowCount {
				t.Errorf("RowCount = %d, want %d", result.RowCount, tt.wantRowCount)
			}

			if result.ValidRowCount != tt.wantValidCount {
				t.Errorf("ValidRowCount = %d, want %d", result.ValidRowCount, tt.wantValidCount)
			}

			if len(result.Errors) != tt.wantErrorCount {
				t.Errorf("len(Errors) = %d, want %d", len(result.Errors), tt.wantErrorCount)
			}

			// Verify struct population
			if len(records) != tt.wantRowCount {
				t.Errorf("len(records) = %d, want %d", len(records), tt.wantRowCount)
			}

			// Verify trim was applied
			if len(records) > 0 && records[0].Name != "John Doe" {
				t.Errorf("Name not trimmed: got %q, want %q", records[0].Name, "John Doe")
			}
		})
	}
}

func TestProcessor_Process_TSV(t *testing.T) {
	t.Parallel()

	tsvData := "name\temail\tage\n  Alice  \talice@example.com\t28\nBob\tbob@example.com\t32\n"

	processor := NewProcessor(FileTypeTSV)
	var records []TestRecord

	reader, result, err := processor.Process(strings.NewReader(tsvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if reader == nil {
		t.Fatal("Process() returned nil reader")
	}

	if result.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", result.RowCount)
	}

	if len(records) != 2 {
		t.Errorf("len(records) = %d, want 2", len(records))
	}

	// Verify trim was applied
	if records[0].Name != "Alice" {
		t.Errorf("Name not trimmed: got %q, want %q", records[0].Name, "Alice")
	}
}

func TestProcessor_Process_LTSV(t *testing.T) {
	t.Parallel()

	ltsvData := "name:Charlie\temail:charlie@example.com\tage:40\nname:Diana\temail:diana@example.com\tage:35\n"

	processor := NewProcessor(FileTypeLTSV)
	var records []TestRecord

	reader, result, err := processor.Process(strings.NewReader(ltsvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if reader == nil {
		t.Fatal("Process() returned nil reader")
	}

	if result.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", result.RowCount)
	}
}

func TestProcessor_OutputReader(t *testing.T) {
	t.Parallel()

	csvData := `name,email,age
  John  ,john@example.com,30
`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	reader, _, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Read the output
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	outputStr := string(output)

	// Verify the output contains trimmed data
	if !strings.Contains(outputStr, "John") {
		t.Errorf("Output should contain trimmed name 'John', got: %s", outputStr)
	}

	// Verify output is valid CSV (contains header and data)
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	if len(lines) != 2 {
		t.Errorf("Output should have 2 lines (header + 1 data row), got %d", len(lines))
	}
}

func TestProcessor_ValidationError(t *testing.T) {
	t.Parallel()

	csvData := `name,email,age
,john@example.com,30
`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected validation errors for empty required field")
	}

	validationErrors := result.ValidationErrors()
	if len(validationErrors) != 1 {
		t.Errorf("Expected 1 validation error, got %d", len(validationErrors))
	}

	if len(validationErrors) > 0 {
		ve := validationErrors[0]
		if ve.Row != 1 {
			t.Errorf("ValidationError.Row = %d, want 1", ve.Row)
		}
		if ve.Field != "Name" {
			t.Errorf("ValidationError.Field = %q, want %q", ve.Field, "Name")
		}
		if ve.Tag != "required" {
			t.Errorf("ValidationError.Tag = %q, want %q", ve.Tag, "required")
		}
	}
}

func TestProcessor_EmptyFile(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, _, err := processor.Process(strings.NewReader(""), &records)
	if err == nil {
		t.Error("Expected error for empty file")
	}
}

func TestProcessor_InvalidStructSlicePointer(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(FileTypeCSV)

	// Test with non-pointer
	var records []TestRecord
	_, _, err := processor.Process(strings.NewReader("a,b,c\n1,2,3"), records)
	if err == nil {
		t.Error("Expected error for non-pointer")
	}

	// Test with pointer to non-slice
	var record TestRecord
	_, _, err = processor.Process(strings.NewReader("a,b,c\n1,2,3"), &record)
	if err == nil {
		t.Error("Expected error for pointer to non-slice")
	}
}

func TestProcessor_Process_Parquet(t *testing.T) {
	t.Parallel()

	// Create a parquet file in memory
	type ParquetRow struct {
		Name  string `parquet:"name"`
		Email string `parquet:"email"`
		Age   string `parquet:"age"`
	}

	rows := []ParquetRow{
		{Name: "  John Doe  ", Email: "john@example.com", Age: "30"},
		{Name: "Jane Smith", Email: "jane@example.com", Age: "25"},
		{Name: "", Email: "invalid@example.com", Age: "40"},
	}

	var buf bytes.Buffer
	writer := parquet.NewGenericWriter[ParquetRow](&buf)
	_, err := writer.Write(rows)
	if err != nil {
		t.Fatalf("failed to write parquet data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close parquet writer: %v", err)
	}

	processor := NewProcessor(FileTypeParquet)
	var records []TestRecord

	reader, result, err := processor.Process(bytes.NewReader(buf.Bytes()), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if reader == nil {
		t.Fatal("Process() returned nil reader")
	}

	if result.RowCount != 3 {
		t.Errorf("RowCount = %d, want 3", result.RowCount)
	}

	// One row has empty name after trim, which fails required validation
	if result.ValidRowCount != 2 {
		t.Errorf("ValidRowCount = %d, want 2", result.ValidRowCount)
	}

	if len(result.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(result.Errors))
	}

	// Verify struct population
	if len(records) != 3 {
		t.Errorf("len(records) = %d, want 3", len(records))
	}

	// Verify trim was applied
	if len(records) > 0 && records[0].Name != "John Doe" {
		t.Errorf("Name not trimmed: got %q, want %q", records[0].Name, "John Doe")
	}

	// Verify columns were captured
	expectedColumns := []string{"name", "email", "age"}
	if len(result.Columns) != len(expectedColumns) {
		t.Errorf("len(Columns) = %d, want %d", len(result.Columns), len(expectedColumns))
	}
	for i, col := range result.Columns {
		if col != expectedColumns[i] {
			t.Errorf("Columns[%d] = %q, want %q", i, col, expectedColumns[i])
		}
	}

	// Verify original format
	if result.OriginalFormat != FileTypeParquet {
		t.Errorf("OriginalFormat = %v, want %v", result.OriginalFormat, FileTypeParquet)
	}
}

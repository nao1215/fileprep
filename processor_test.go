package fileprep

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nao1215/fileparser"
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

			processor := NewProcessor(fileparser.CSV)
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

	processor := NewProcessor(fileparser.TSV)
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

	processor := NewProcessor(fileparser.LTSV)
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

	processor := NewProcessor(fileparser.CSV)
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

	processor := NewProcessor(fileparser.CSV)
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

	processor := NewProcessor(fileparser.CSV)
	var records []TestRecord

	_, _, err := processor.Process(strings.NewReader(""), &records)
	if err == nil {
		t.Error("Expected error for empty file")
	}
}

func TestProcessor_InvalidStructSlicePointer(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(fileparser.CSV)

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

	processor := NewProcessor(fileparser.Parquet)
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
	if result.OriginalFormat != fileparser.Parquet {
		t.Errorf("OriginalFormat = %v, want %v", result.OriginalFormat, fileparser.Parquet)
	}
}

// EdgeCaseRecord is a struct for edge case testing
type EdgeCaseRecord struct {
	Col1 string `name:"col1" prep:"trim"`
	Col2 string `name:"col2" prep:"trim"`
	Col3 string `name:"col3" prep:"trim"`
}

func TestProcessor_CSV_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		wantRowCount int
		wantColCount int
		wantErr      bool
		wantCol1     string // expected value for first record's Col1 (empty string means skip check)
		wantCol1Len  int    // expected length of Col1 (0 means skip check)
		checkRow     int    // which row to check for wantCol3 (-1 means no check)
		wantCol3     string // expected value for Col3 at checkRow
	}{
		{
			name:         "very long line",
			input:        "col1,col2,col3\n" + strings.Repeat("a", 10000) + ",b,c\n",
			wantRowCount: 1,
			wantColCount: 3,
			wantErr:      false,
			wantCol1Len:  10000,
			checkRow:     -1,
		},
		{
			name:         "many columns (50)",
			input:        strings.Join(makeHeaders(50), ",") + "\n" + strings.Join(makeValues(50), ",") + "\n",
			wantRowCount: 1,
			wantColCount: 50,
			wantErr:      false,
			checkRow:     -1,
		},
		{
			name:         "uneven rows - short row",
			input:        "col1,col2,col3\na,b,c\nd,e\nf,g,h\n",
			wantRowCount: 0,
			wantColCount: 0,
			wantErr:      true, // fileparser returns error for mismatched column count
			checkRow:     -1,
		},
		{
			name:         "empty file",
			input:        "",
			wantRowCount: 0,
			wantColCount: 0,
			wantErr:      true, // ErrEmptyFile
			checkRow:     -1,
		},
		{
			name:         "header only",
			input:        "col1,col2,col3\n",
			wantRowCount: 0,
			wantColCount: 3,
			wantErr:      false,
			checkRow:     -1,
		},
		{
			name:         "quoted fields with commas",
			input:        "col1,col2,col3\n\"a,b\",c,d\n",
			wantRowCount: 1,
			wantColCount: 3,
			wantErr:      false,
			wantCol1:     "a,b",
			checkRow:     -1,
		},
		{
			name:         "quoted fields with newlines",
			input:        "col1,col2,col3\n\"line1\nline2\",b,c\n",
			wantRowCount: 1,
			wantColCount: 3,
			wantErr:      false,
			wantCol1:     "line1\nline2",
			checkRow:     -1,
		},
		{
			name:         "unicode content",
			input:        "col1,col2,col3\n日本語,한국어,中文\n",
			wantRowCount: 1,
			wantColCount: 3,
			wantErr:      false,
			wantCol1:     "日本語",
			checkRow:     -1,
		},
		{
			name:         "whitespace-only values",
			input:        "col1,col2,col3\n   ,\t\t,  \n",
			wantRowCount: 1,
			wantColCount: 3,
			wantErr:      false,
			checkRow:     -1,
			// trim preprocessor removes whitespace - checked separately
		},
		{
			name:         "empty values between commas",
			input:        "col1,col2,col3\n,,\na,,c\n,b,\n",
			wantRowCount: 3,
			wantColCount: 3,
			wantErr:      false,
			checkRow:     -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			processor := NewProcessor(fileparser.CSV)
			var records []EdgeCaseRecord

			reader, result, err := processor.Process(strings.NewReader(tt.input), &records)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if reader == nil {
				t.Fatal("Process() returned nil reader")
			}

			if result.RowCount != tt.wantRowCount {
				t.Errorf("RowCount = %d, want %d", result.RowCount, tt.wantRowCount)
			}

			if len(result.Columns) != tt.wantColCount {
				t.Errorf("Column count = %d, want %d", len(result.Columns), tt.wantColCount)
			}

			if len(records) > 0 {
				if tt.wantCol1 != "" && records[0].Col1 != tt.wantCol1 {
					t.Errorf("Col1 = %q, want %q", records[0].Col1, tt.wantCol1)
				}
				if tt.wantCol1Len > 0 && len(records[0].Col1) != tt.wantCol1Len {
					t.Errorf("Col1 length = %d, want %d", len(records[0].Col1), tt.wantCol1Len)
				}
			}

			if tt.checkRow >= 0 && tt.checkRow < len(records) {
				if records[tt.checkRow].Col3 != tt.wantCol3 {
					t.Errorf("Row %d Col3 = %q, want %q", tt.checkRow, records[tt.checkRow].Col3, tt.wantCol3)
				}
			}
		})
	}
}

func TestProcessor_CSV_WhitespaceValues(t *testing.T) {
	t.Parallel()

	input := "col1,col2,col3\n   ,\t\t,  \n"
	processor := NewProcessor(fileparser.CSV)
	var records []EdgeCaseRecord

	_, _, err := processor.Process(strings.NewReader(input), &records)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	// trim preprocessor should remove whitespace
	if records[0].Col1 != "" {
		t.Errorf("Whitespace-only Col1 should be trimmed to empty: got %q", records[0].Col1)
	}
}

func TestProcessor_CSV_EmptyValues(t *testing.T) {
	t.Parallel()

	input := "col1,col2,col3\n,,\na,,c\n,b,\n"
	processor := NewProcessor(fileparser.CSV)
	var records []EdgeCaseRecord

	_, _, err := processor.Process(strings.NewReader(input), &records)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("Expected 3 records, got %d", len(records))
	}

	// First row: all empty
	if records[0].Col1 != "" || records[0].Col2 != "" || records[0].Col3 != "" {
		t.Errorf("First row should have all empty: got %q, %q, %q", records[0].Col1, records[0].Col2, records[0].Col3)
	}
}

// makeHeaders creates n unique header names
func makeHeaders(n int) []string {
	headers := make([]string, n)
	// Ensure first 3 are col1, col2, col3 for struct mapping
	for i := range n {
		if i < 3 {
			headers[i] = "col" + string(rune('1'+i))
		} else {
			headers[i] = "column_" + strconv.Itoa(i)
		}
	}
	return headers
}

// makeValues creates n test values
func makeValues(n int) []string {
	values := make([]string, n)
	for i := range n {
		values[i] = "val" + string(rune('a'+i%26))
	}
	return values
}

func TestProcessor_CSV_LargeColumnCount(t *testing.T) {
	t.Parallel()

	// Test with 100 columns to ensure no issues with many columns
	colCount := 100
	headers := make([]string, colCount)
	values := make([]string, colCount)
	for i := range colCount {
		headers[i] = "c" + string(rune('a'+i%26)) + string(rune('0'+i%10))
		values[i] = "v" + string(rune('0'+i%10))
	}

	// Map first 3 columns to struct
	headers[0], headers[1], headers[2] = "col1", "col2", "col3"

	input := strings.Join(headers, ",") + "\n" + strings.Join(values, ",") + "\n"

	processor := NewProcessor(fileparser.CSV)
	var records []EdgeCaseRecord

	_, result, err := processor.Process(strings.NewReader(input), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.RowCount != 1 {
		t.Errorf("RowCount = %d, want 1", result.RowCount)
	}

	if len(result.Columns) != colCount {
		t.Errorf("Column count = %d, want %d", len(result.Columns), colCount)
	}
}

func TestProcessor_CSV_ManyRows(t *testing.T) {
	t.Parallel()

	// Test with 1000 rows to ensure no issues with many rows
	rowCount := 1000
	var buf strings.Builder
	buf.WriteString("col1,col2,col3\n")
	for i := range rowCount {
		buf.WriteString("a" + string(rune('0'+i%10)) + ",b,c\n")
	}

	processor := NewProcessor(fileparser.CSV)
	var records []EdgeCaseRecord

	_, result, err := processor.Process(strings.NewReader(buf.String()), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.RowCount != rowCount {
		t.Errorf("RowCount = %d, want %d", result.RowCount, rowCount)
	}

	if len(records) != rowCount {
		t.Errorf("len(records) = %d, want %d", len(records), rowCount)
	}
}

// JSONRecord is a test struct for JSON/JSONL processing.
// fileparser stores JSON data in a single "data" column containing raw JSON strings.
type JSONRecord struct {
	Data string `name:"data" prep:"trim" validate:"required"`
}

func TestProcessor_Process_JSON(t *testing.T) {
	t.Parallel()

	jsonData := `[{"key":"value1"},{"key":"value2"},{"key":"value3"}]`

	processor := NewProcessor(fileparser.JSON)
	var records []JSONRecord

	reader, result, err := processor.Process(strings.NewReader(jsonData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if reader == nil {
		t.Fatal("Process() returned nil reader")
	}

	if result.RowCount != 3 {
		t.Errorf("RowCount = %d, want 3", result.RowCount)
	}

	if result.ValidRowCount != 3 {
		t.Errorf("ValidRowCount = %d, want 3", result.ValidRowCount)
	}

	if len(records) != 3 {
		t.Errorf("len(records) = %d, want 3", len(records))
	}

	// Verify struct population — each record contains raw JSON
	if len(records) > 0 && records[0].Data != `{"key":"value1"}` {
		t.Errorf("records[0].Data = %q, want %q", records[0].Data, `{"key":"value1"}`)
	}

	// Verify output format is JSONL
	s, ok := reader.(Stream)
	if !ok {
		t.Fatal("reader does not implement Stream")
	}
	if s.Format() != fileparser.JSONL {
		t.Errorf("Format() = %v, want JSONL", s.Format())
	}
	if s.OriginalFormat() != fileparser.JSON {
		t.Errorf("OriginalFormat() = %v, want JSON", s.OriginalFormat())
	}
}

func TestProcessor_Process_JSONL(t *testing.T) {
	t.Parallel()

	jsonlData := "{\"name\":\"Alice\"}\n{\"name\":\"Bob\"}\n{\"name\":\"Charlie\"}\n"

	processor := NewProcessor(fileparser.JSONL)
	var records []JSONRecord

	reader, result, err := processor.Process(strings.NewReader(jsonlData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if reader == nil {
		t.Fatal("Process() returned nil reader")
	}

	if result.RowCount != 3 {
		t.Errorf("RowCount = %d, want 3", result.RowCount)
	}

	if len(records) != 3 {
		t.Errorf("len(records) = %d, want 3", len(records))
	}

	if len(records) > 0 && records[0].Data != `{"name":"Alice"}` {
		t.Errorf("records[0].Data = %q, want %q", records[0].Data, `{"name":"Alice"}`)
	}

	// Verify output format is JSONL
	s, ok := reader.(Stream)
	if !ok {
		t.Fatal("reader does not implement Stream")
	}
	if s.Format() != fileparser.JSONL {
		t.Errorf("Format() = %v, want JSONL", s.Format())
	}
}

func TestProcessor_JSON_OutputReader(t *testing.T) {
	t.Parallel()

	jsonData := `[{"a":1},{"a":2}]`

	processor := NewProcessor(fileparser.JSON)
	var records []JSONRecord

	reader, _, err := processor.Process(strings.NewReader(jsonData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	outputStr := string(output)
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	if len(lines) != 2 {
		t.Errorf("Output should have 2 lines, got %d: %q", len(lines), outputStr)
	}

	if lines[0] != `{"a":1}` {
		t.Errorf("line[0] = %q, want %q", lines[0], `{"a":1}`)
	}
	if lines[1] != `{"a":2}` {
		t.Errorf("line[1] = %q, want %q", lines[1], `{"a":2}`)
	}
}

func TestProcessor_JSON_Validation(t *testing.T) {
	t.Parallel()

	// Use a struct that requires numeric validation on the raw JSON string.
	// A JSON object like {"key":"value"} is not numeric, so validation fails.
	type NumericRecord struct {
		Data string `name:"data" validate:"numeric"`
	}

	jsonData := `[{"key":"value"}]`

	processor := NewProcessor(fileparser.JSON)
	var records []NumericRecord

	_, result, err := processor.Process(strings.NewReader(jsonData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected validation errors for non-numeric JSON value")
	}

	if result.ValidRowCount != 0 {
		t.Errorf("ValidRowCount = %d, want 0", result.ValidRowCount)
	}
}

func TestProcessor_JSONL_EmptyFile(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(fileparser.JSONL)
	var records []JSONRecord

	_, _, err := processor.Process(strings.NewReader(""), &records)
	if err == nil {
		t.Error("Expected error for empty JSONL file")
	}
}

func TestProcessor_JSON_SingleObject(t *testing.T) {
	t.Parallel()

	jsonData := `{"key":"value"}`

	processor := NewProcessor(fileparser.JSON)
	var records []JSONRecord

	_, result, err := processor.Process(strings.NewReader(jsonData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.RowCount != 1 {
		t.Errorf("RowCount = %d, want 1", result.RowCount)
	}

	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}

	if records[0].Data != `{"key":"value"}` {
		t.Errorf("records[0].Data = %q, want %q", records[0].Data, `{"key":"value"}`)
	}
}

func TestProcessor_JSON_InvalidJSONAfterPrep(t *testing.T) {
	t.Parallel()

	// nullify={} turns {} into "", emptying the JSON data.
	// This produces a PrepError("empty_json_data"). Process succeeds because
	// the second row still produces valid JSONL output.
	type NullifyRecord struct {
		Data string `name:"data" prep:"nullify={}"`
	}

	jsonData := `[{}, {"key":"value"}]`

	processor := NewProcessor(fileparser.JSON)
	var records []NullifyRecord

	reader, result, err := processor.Process(strings.NewReader(jsonData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// First row: {} → nullified to "" → PrepError("empty_json_data") → not counted as valid
	// Second row: {"key":"value"} → untouched → valid JSON → counted as valid
	if result.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", result.RowCount)
	}

	if result.ValidRowCount != 1 {
		t.Errorf("ValidRowCount = %d, want 1", result.ValidRowCount)
	}

	// Verify the PrepError was recorded for the emptied row
	prepErrors := result.PrepErrors()
	if len(prepErrors) != 1 {
		t.Fatalf("Expected 1 PrepError, got %d", len(prepErrors))
	}
	if prepErrors[0].Tag != "empty_json_data" {
		t.Errorf("PrepError.Tag = %q, want %q", prepErrors[0].Tag, "empty_json_data")
	}

	// Verify JSONL output has 1 line (the empty row is skipped, matching ValidRowCount)
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 1 {
		t.Errorf("JSONL output should have 1 line, got %d: %q", len(lines), string(output))
	}
	if lines[0] != `{"key":"value"}` {
		t.Errorf("JSONL output line = %q, want %q", lines[0], `{"key":"value"}`)
	}
}

func TestProcessor_JSON_TruncateDestroysJSON(t *testing.T) {
	t.Parallel()

	// truncate=5 on {"key":"value"} produces {"key → invalid JSON.
	// Process returns ErrInvalidJSONAfterPrep because invalid JSON lines
	// would cause downstream JSONL parsers to fail.
	type TruncateRecord struct {
		Data string `name:"data" prep:"truncate=5"`
	}

	jsonData := `[{"key":"value"}]`

	processor := NewProcessor(fileparser.JSON)
	var records []TruncateRecord

	_, _, err := processor.Process(strings.NewReader(jsonData), &records)
	if err == nil {
		t.Fatal("Expected error for invalid JSON after truncate, got nil")
	}

	if !errors.Is(err, ErrInvalidJSONAfterPrep) {
		t.Errorf("err = %v, want ErrInvalidJSONAfterPrep", err)
	}

	// Verify the error message includes the row number and truncated value
	errMsg := err.Error()
	if !strings.Contains(errMsg, "row 1") {
		t.Errorf("error should contain row number, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, `{"key`) {
		t.Errorf("error should contain truncated value, got: %s", errMsg)
	}
}

func TestProcessor_JSON_AllRowsEmptied(t *testing.T) {
	t.Parallel()

	// When all JSON rows become empty after preprocessing, Process returns
	// ErrEmptyJSONOutput because an empty JSONL stream is unparseable.
	type NullifyAllRecord struct {
		Data string `name:"data" prep:"nullify={}"`
	}

	jsonData := `[{}]`

	processor := NewProcessor(fileparser.JSON)
	var records []NullifyAllRecord

	_, _, err := processor.Process(strings.NewReader(jsonData), &records)
	if err == nil {
		t.Fatal("Expected error for all-empty JSON output, got nil")
	}

	if !errors.Is(err, ErrEmptyJSONOutput) {
		t.Errorf("err = %v, want ErrEmptyJSONOutput", err)
	}
}

func TestProcessor_JSON_PrettyPrinted(t *testing.T) {
	t.Parallel()

	// Pretty-printed JSON contains newlines within each element.
	// writeJSONL must compact each element to a single line, otherwise
	// downstream JSONL parsers see partial JSON on each line and fail.
	jsonData := `[
  {
    "name": "Alice",
    "age": 30
  },
  {
    "name": "Bob",
    "age": 25
  }
]`

	processor := NewProcessor(fileparser.JSON)
	var records []JSONRecord

	reader, result, err := processor.Process(strings.NewReader(jsonData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", result.RowCount)
	}

	// Read JSONL output and verify each line is exactly one compact JSON value
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 {
		t.Fatalf("Output should have 2 lines, got %d: %q", len(lines), string(output))
	}

	// Each line must be valid JSON and contain no newlines (compact)
	want := []string{
		`{"name":"Alice","age":30}`,
		`{"name":"Bob","age":25}`,
	}
	for i, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Errorf("line %d is not valid JSON: %q", i+1, line)
		}
		if line != want[i] {
			t.Errorf("line %d = %q, want %q", i+1, line, want[i])
		}
	}
}

func TestProcessor_JSON_PrettyPrintedGzip(t *testing.T) {
	t.Parallel()

	// Verify that compressed pretty-printed JSON also produces compact JSONL.
	// Decompression is handled by fileparser, but the full pipeline
	// (decompress → parse → prep → compact → JSONL) should be exercised.
	prettyJSON := `[
  {"name": "Alice", "age": 30},
  {"name": "Bob", "age": 25}
]`

	// Gzip compress in memory
	var compressed bytes.Buffer
	gw := gzip.NewWriter(&compressed)
	if _, err := gw.Write([]byte(prettyJSON)); err != nil {
		t.Fatalf("gzip write error: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close error: %v", err)
	}

	processor := NewProcessor(fileparser.JSONGZ)
	var records []JSONRecord

	reader, result, err := processor.Process(bytes.NewReader(compressed.Bytes()), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", result.RowCount)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 {
		t.Fatalf("Output should have 2 lines, got %d: %q", len(lines), string(output))
	}

	for i, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Errorf("line %d is not valid JSON: %q", i+1, line)
		}
		if line != strings.TrimSpace(line) {
			t.Errorf("line %d has leading/trailing whitespace: %q", i+1, line)
		}
		if strings.Contains(line, "\t") {
			t.Errorf("line %d contains tab character: %q", i+1, line)
		}
	}
}

// TestSetFieldValue_IntTypes tests type conversion for int, int8, int16, int32, int64 fields
// via Process(), comparing results with go-cmp.
func TestSetFieldValue_IntTypes(t *testing.T) {
	t.Parallel()

	type IntRecord struct {
		ValInt   int   `name:"val_int"`
		ValInt8  int8  `name:"val_int8"`
		ValInt16 int16 `name:"val_int16"`
		ValInt32 int32 `name:"val_int32"`
		ValInt64 int64 `name:"val_int64"`
	}

	t.Run("valid int values are converted correctly", func(t *testing.T) {
		t.Parallel()
		csvData := "val_int,val_int8,val_int16,val_int32,val_int64\n42,127,32767,2147483647,9223372036854775807\n-100,-128,-32768,-2147483648,-9223372036854775808\n"
		var records []IntRecord

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		want := []IntRecord{
			{ValInt: 42, ValInt8: 127, ValInt16: 32767, ValInt32: 2147483647, ValInt64: 9223372036854775807},
			{ValInt: -100, ValInt8: -128, ValInt16: -32768, ValInt32: -2147483648, ValInt64: -9223372036854775808},
		}
		if diff := cmp.Diff(want, records); diff != "" {
			t.Errorf("records mismatch (-want +got):\n%s", diff)
		}
		if result.RowCount != 2 {
			t.Errorf("RowCount = %d, want 2", result.RowCount)
		}
	})

	t.Run("empty int values default to zero", func(t *testing.T) {
		t.Parallel()
		csvData := "val_int,val_int8,val_int16,val_int32,val_int64\n,,,,\n"
		var records []IntRecord

		processor := NewProcessor(FileTypeCSV)
		_, _, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		want := []IntRecord{{}}
		if diff := cmp.Diff(want, records); diff != "" {
			t.Errorf("records mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid int value produces type_conversion error", func(t *testing.T) {
		t.Parallel()
		csvData := "val_int,val_int8,val_int16,val_int32,val_int64\nnot-a-number,0,0,0,0\n"
		var records []IntRecord

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if len(result.Errors) == 0 {
			t.Fatal("expected at least 1 error for invalid int, got 0")
		}
		var pe *PrepError
		if !errors.As(result.Errors[0], &pe) {
			t.Fatalf("expected PrepError, got %T", result.Errors[0])
		}
		if pe.Row != 1 {
			t.Errorf("Row = %d, want 1", pe.Row)
		}
		if pe.Column != "val_int" {
			t.Errorf("Column = %q, want %q", pe.Column, "val_int")
		}
		if pe.Tag != "type_conversion" {
			t.Errorf("Tag = %q, want %q", pe.Tag, "type_conversion")
		}
	})

	t.Run("int8 overflow produces type_conversion error", func(t *testing.T) {
		t.Parallel()
		csvData := "val_int,val_int8,val_int16,val_int32,val_int64\n0,128,0,0,0\n"
		var records []IntRecord

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if len(result.Errors) == 0 {
			t.Fatal("expected at least 1 error for int8 overflow, got 0")
		}
		var pe *PrepError
		if !errors.As(result.Errors[0], &pe) {
			t.Fatalf("expected PrepError, got %T", result.Errors[0])
		}
		if pe.Row != 1 {
			t.Errorf("Row = %d, want 1", pe.Row)
		}
		if pe.Column != "val_int8" {
			t.Errorf("Column = %q, want %q", pe.Column, "val_int8")
		}
		if pe.Tag != "type_conversion" {
			t.Errorf("Tag = %q, want %q", pe.Tag, "type_conversion")
		}
	})

	t.Run("platform-specific int max values are converted correctly", func(t *testing.T) {
		t.Parallel()
		// Use math.MaxInt/MinInt which are platform-dependent (32-bit or 64-bit)
		maxIntStr := strconv.FormatInt(int64(int(^uint(0)>>1)), 10)
		minIntStr := strconv.FormatInt(int64(-int(^uint(0)>>1)-1), 10)
		csvData := "val_int,val_int8,val_int16,val_int32,val_int64\n" +
			maxIntStr + ",0,0,0,0\n" +
			minIntStr + ",0,0,0,0\n"
		var records []IntRecord

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if len(result.Errors) != 0 {
			t.Errorf("expected 0 errors for platform max int, got %d: %v", len(result.Errors), result.Errors)
		}
		maxInt := int(^uint(0) >> 1)
		minInt := -maxInt - 1
		if records[0].ValInt != maxInt {
			t.Errorf("ValInt = %d, want %d (platform max)", records[0].ValInt, maxInt)
		}
		if records[1].ValInt != minInt {
			t.Errorf("ValInt = %d, want %d (platform min)", records[1].ValInt, minInt)
		}
	})
}

// TestSetFieldValue_UintPlatformMax tests platform-dependent uint max value conversion
func TestSetFieldValue_UintPlatformMax(t *testing.T) {
	t.Parallel()

	type UintRecord struct {
		ValUint   uint   `name:"val_uint"`
		ValUint8  uint8  `name:"val_uint8"`
		ValUint16 uint16 `name:"val_uint16"`
		ValUint32 uint32 `name:"val_uint32"`
		ValUint64 uint64 `name:"val_uint64"`
	}

	t.Run("platform-specific uint max value is converted correctly", func(t *testing.T) {
		t.Parallel()
		maxUintStr := strconv.FormatUint(uint64(^uint(0)), 10)
		csvData := "val_uint,val_uint8,val_uint16,val_uint32,val_uint64\n" +
			maxUintStr + ",0,0,0,0\n"
		var records []UintRecord

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if len(result.Errors) != 0 {
			t.Errorf("expected 0 errors for platform max uint, got %d: %v", len(result.Errors), result.Errors)
		}
		maxUint := ^uint(0)
		if records[0].ValUint != maxUint {
			t.Errorf("ValUint = %d, want %d (platform max)", records[0].ValUint, maxUint)
		}
	})
}

// TestSetFieldValue_UintTypes tests type conversion for uint, uint8, uint16, uint32, uint64 fields
// via Process(), comparing results with go-cmp.
func TestSetFieldValue_UintTypes(t *testing.T) {
	t.Parallel()

	type UintRecord struct {
		ValUint   uint   `name:"val_uint"`
		ValUint8  uint8  `name:"val_uint8"`
		ValUint16 uint16 `name:"val_uint16"`
		ValUint32 uint32 `name:"val_uint32"`
		ValUint64 uint64 `name:"val_uint64"`
	}

	t.Run("valid uint values are converted correctly", func(t *testing.T) {
		t.Parallel()
		csvData := "val_uint,val_uint8,val_uint16,val_uint32,val_uint64\n42,255,65535,4294967295,18446744073709551615\n"
		var records []UintRecord

		processor := NewProcessor(FileTypeCSV)
		_, _, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		want := []UintRecord{
			{ValUint: 42, ValUint8: 255, ValUint16: 65535, ValUint32: 4294967295, ValUint64: 18446744073709551615},
		}
		if diff := cmp.Diff(want, records); diff != "" {
			t.Errorf("records mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("empty uint values default to zero", func(t *testing.T) {
		t.Parallel()
		csvData := "val_uint,val_uint8,val_uint16,val_uint32,val_uint64\n,,,,\n"
		var records []UintRecord

		processor := NewProcessor(FileTypeCSV)
		_, _, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		want := []UintRecord{{}}
		if diff := cmp.Diff(want, records); diff != "" {
			t.Errorf("records mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("negative value for uint produces type_conversion error", func(t *testing.T) {
		t.Parallel()
		csvData := "val_uint,val_uint8,val_uint16,val_uint32,val_uint64\n-1,0,0,0,0\n"
		var records []UintRecord

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if len(result.Errors) == 0 {
			t.Fatal("expected at least 1 error for negative uint, got 0")
		}
		var pe *PrepError
		if !errors.As(result.Errors[0], &pe) {
			t.Fatalf("expected PrepError, got %T", result.Errors[0])
		}
		if pe.Row != 1 {
			t.Errorf("Row = %d, want 1", pe.Row)
		}
		if pe.Column != "val_uint" {
			t.Errorf("Column = %q, want %q", pe.Column, "val_uint")
		}
		if pe.Tag != "type_conversion" {
			t.Errorf("Tag = %q, want %q", pe.Tag, "type_conversion")
		}
	})

	t.Run("non-numeric value for uint produces type_conversion error", func(t *testing.T) {
		t.Parallel()
		csvData := "val_uint,val_uint8,val_uint16,val_uint32,val_uint64\nabc,0,0,0,0\n"
		var records []UintRecord

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if len(result.Errors) == 0 {
			t.Fatal("expected at least 1 error for non-numeric uint, got 0")
		}
		var pe *PrepError
		if !errors.As(result.Errors[0], &pe) {
			t.Fatalf("expected PrepError, got %T", result.Errors[0])
		}
		if pe.Row != 1 {
			t.Errorf("Row = %d, want 1", pe.Row)
		}
		if pe.Column != "val_uint" {
			t.Errorf("Column = %q, want %q", pe.Column, "val_uint")
		}
		if pe.Tag != "type_conversion" {
			t.Errorf("Tag = %q, want %q", pe.Tag, "type_conversion")
		}
	})
}

// TestSetFieldValue_FloatTypes tests type conversion for float32 and float64 fields
// via Process(), comparing results with go-cmp.
func TestSetFieldValue_FloatTypes(t *testing.T) {
	t.Parallel()

	type FloatRecord struct {
		ValFloat32 float32 `name:"val_float32"`
		ValFloat64 float64 `name:"val_float64"`
	}

	t.Run("valid float values are converted correctly", func(t *testing.T) {
		t.Parallel()
		csvData := "val_float32,val_float64\n1.5,3.14\n"
		var records []FloatRecord

		processor := NewProcessor(FileTypeCSV)
		_, _, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		want := []FloatRecord{{ValFloat32: 1.5, ValFloat64: 3.14}}
		if diff := cmp.Diff(want, records); diff != "" {
			t.Errorf("records mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("empty float values default to zero", func(t *testing.T) {
		t.Parallel()
		csvData := "val_float32,val_float64\n,\n"
		var records []FloatRecord

		processor := NewProcessor(FileTypeCSV)
		_, _, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		want := []FloatRecord{{}}
		if diff := cmp.Diff(want, records); diff != "" {
			t.Errorf("records mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid float value produces type_conversion error", func(t *testing.T) {
		t.Parallel()
		csvData := "val_float32,val_float64\nnot-float,0\n"
		var records []FloatRecord

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if len(result.Errors) == 0 {
			t.Fatal("expected at least 1 error for invalid float, got 0")
		}
		var pe *PrepError
		if !errors.As(result.Errors[0], &pe) {
			t.Fatalf("expected PrepError, got %T", result.Errors[0])
		}
		if pe.Row != 1 {
			t.Errorf("Row = %d, want 1", pe.Row)
		}
		if pe.Column != "val_float32" {
			t.Errorf("Column = %q, want %q", pe.Column, "val_float32")
		}
		if pe.Tag != "type_conversion" {
			t.Errorf("Tag = %q, want %q", pe.Tag, "type_conversion")
		}
	})
}

// TestSetFieldValue_BoolType tests type conversion for bool fields
// via Process(), comparing results with go-cmp.
func TestSetFieldValue_BoolType(t *testing.T) {
	t.Parallel()

	type BoolRecord struct {
		ValBool bool   `name:"val_bool"`
		Dummy   string `name:"dummy"`
	}

	t.Run("true/false/1/0 are converted correctly", func(t *testing.T) {
		t.Parallel()
		csvData := "val_bool,dummy\ntrue,a\nfalse,b\n1,c\n0,d\n"
		var records []BoolRecord

		processor := NewProcessor(FileTypeCSV)
		_, _, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		want := []BoolRecord{
			{ValBool: true, Dummy: "a"},
			{ValBool: false, Dummy: "b"},
			{ValBool: true, Dummy: "c"},
			{ValBool: false, Dummy: "d"},
		}
		if diff := cmp.Diff(want, records); diff != "" {
			t.Errorf("records mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("empty bool value defaults to false", func(t *testing.T) {
		t.Parallel()
		csvData := "val_bool,dummy\n,x\n"
		var records []BoolRecord

		processor := NewProcessor(FileTypeCSV)
		_, _, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		want := []BoolRecord{{ValBool: false, Dummy: "x"}}
		if diff := cmp.Diff(want, records); diff != "" {
			t.Errorf("records mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid bool value produces type_conversion error", func(t *testing.T) {
		t.Parallel()
		csvData := "val_bool,dummy\nnot-bool,x\n"
		var records []BoolRecord

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if len(result.Errors) == 0 {
			t.Fatal("expected at least 1 error for invalid bool, got 0")
		}
		var pe *PrepError
		if !errors.As(result.Errors[0], &pe) {
			t.Fatalf("expected PrepError, got %T", result.Errors[0])
		}
		if pe.Row != 1 {
			t.Errorf("Row = %d, want 1", pe.Row)
		}
		if pe.Column != "val_bool" {
			t.Errorf("Column = %q, want %q", pe.Column, "val_bool")
		}
		if pe.Tag != "type_conversion" {
			t.Errorf("Tag = %q, want %q", pe.Tag, "type_conversion")
		}
	})
}

// TestSetFieldValue_StringType tests string field handling via Process().
func TestSetFieldValue_StringType(t *testing.T) {
	t.Parallel()

	type StringRecord struct {
		Name  string
		Email string
	}

	t.Run("string values are set correctly", func(t *testing.T) {
		t.Parallel()
		csvData := "name,email\nhello,world@example.com\n"
		var records []StringRecord

		processor := NewProcessor(FileTypeCSV)
		_, _, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		want := []StringRecord{{Name: "hello", Email: "world@example.com"}}
		if diff := cmp.Diff(want, records); diff != "" {
			t.Errorf("records mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("empty string values are set as empty", func(t *testing.T) {
		t.Parallel()
		csvData := "name,email\n,\n"
		var records []StringRecord

		processor := NewProcessor(FileTypeCSV)
		_, _, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		want := []StringRecord{{Name: "", Email: ""}}
		if diff := cmp.Diff(want, records); diff != "" {
			t.Errorf("records mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestSetFieldValue_MixedTypes tests a struct with multiple type fields in one pass.
func TestSetFieldValue_MixedTypes(t *testing.T) {
	t.Parallel()

	type MixedRecord struct {
		Name   string
		Age    int
		Score  float64
		Active bool
		Level  uint8
	}

	csvData := "name,age,score,active,level\nAlice,30,95.5,true,5\nBob,,,,\n"
	var records []MixedRecord

	processor := NewProcessor(FileTypeCSV)
	_, _, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	want := []MixedRecord{
		{Name: "Alice", Age: 30, Score: 95.5, Active: true, Level: 5},
		{Name: "Bob", Age: 0, Score: 0, Active: false, Level: 0},
	}
	if diff := cmp.Diff(want, records); diff != "" {
		t.Errorf("records mismatch (-want +got):\n%s", diff)
	}
}

func TestWithValidRowsOnly(t *testing.T) {
	t.Parallel()

	type Record struct {
		Name  string `validate:"required"`
		Email string `validate:"email"`
	}

	t.Run("only valid rows in output and struct slice", func(t *testing.T) {
		t.Parallel()
		csvData := "name,email\nAlice,alice@example.com\n,invalid\nBob,bob@example.com\n"
		var records []Record
		processor := NewProcessor(FileTypeCSV, WithValidRowsOnly())
		reader, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		// Total rows should be 3, valid rows should be 2
		if result.RowCount != 3 {
			t.Errorf("RowCount = %d, want 3", result.RowCount)
		}
		if result.ValidRowCount != 2 {
			t.Errorf("ValidRowCount = %d, want 2", result.ValidRowCount)
		}

		// Struct slice should contain only 2 valid records
		if len(records) != 2 {
			t.Fatalf("len(records) = %d, want 2", len(records))
		}
		want := []Record{
			{Name: "Alice", Email: "alice@example.com"},
			{Name: "Bob", Email: "bob@example.com"},
		}
		if diff := cmp.Diff(want, records); diff != "" {
			t.Errorf("records mismatch (-want +got):\n%s", diff)
		}

		// Output should contain only valid rows
		output, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		outputStr := string(output)
		if strings.Contains(outputStr, "invalid") {
			t.Errorf("output should not contain invalid rows, got:\n%s", outputStr)
		}
		lines := strings.Split(strings.TrimSpace(outputStr), "\n")
		// Header + 2 valid rows = 3 lines
		if len(lines) != 3 {
			t.Errorf("output lines = %d, want 3 (header + 2 valid rows), got:\n%s", len(lines), outputStr)
		}
	})

	t.Run("all rows valid produces same output as default", func(t *testing.T) {
		t.Parallel()
		csvData := "name,email\nAlice,alice@example.com\nBob,bob@example.com\n"

		var records1 []Record
		proc1 := NewProcessor(FileTypeCSV)
		reader1, result1, err := proc1.Process(strings.NewReader(csvData), &records1)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		out1, err := io.ReadAll(reader1)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		var records2 []Record
		proc2 := NewProcessor(FileTypeCSV, WithValidRowsOnly())
		reader2, result2, err := proc2.Process(strings.NewReader(csvData), &records2)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		out2, err := io.ReadAll(reader2)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if result1.RowCount != result2.RowCount || result1.ValidRowCount != result2.ValidRowCount {
			t.Errorf("result counts differ: default=%d/%d, validOnly=%d/%d",
				result1.RowCount, result1.ValidRowCount, result2.RowCount, result2.ValidRowCount)
		}
		if string(out1) != string(out2) {
			t.Errorf("output differs:\ndefault:\n%s\nvalidOnly:\n%s", out1, out2)
		}
	})

	t.Run("all rows invalid produces empty output", func(t *testing.T) {
		t.Parallel()
		csvData := "name,email\n,invalid1\n,invalid2\n"
		var records []Record
		processor := NewProcessor(FileTypeCSV, WithValidRowsOnly())
		reader, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if result.ValidRowCount != 0 {
			t.Errorf("ValidRowCount = %d, want 0", result.ValidRowCount)
		}
		if len(records) != 0 {
			t.Errorf("len(records) = %d, want 0", len(records))
		}
		output, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		// Only header line should remain
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) != 1 {
			t.Errorf("output lines = %d, want 1 (header only), got:\n%s", len(lines), output)
		}
	})
}

// errWriter is a writer that always returns an error, used for testing write error paths.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

func TestWriteCSV_ErrorPath(t *testing.T) {
	t.Parallel()

	p := NewProcessor(FileTypeCSV)

	t.Run("header write error", func(t *testing.T) {
		t.Parallel()
		err := p.writeCSV(errWriter{}, []string{"a", "b"}, [][]string{{"1", "2"}})
		if err == nil {
			t.Error("expected error from errWriter, got nil")
		}
	})
}

func TestWriteTSV_ErrorPath(t *testing.T) {
	t.Parallel()

	p := &Processor{fileType: fileparser.TSV}

	t.Run("header write error", func(t *testing.T) {
		t.Parallel()
		err := p.writeTSV(errWriter{}, []string{"a", "b"}, [][]string{{"1", "2"}})
		if err == nil {
			t.Error("expected error from errWriter, got nil")
		}
	})
}

func TestWriteLTSV_ErrorPath(t *testing.T) {
	t.Parallel()

	p := &Processor{fileType: fileparser.LTSV}

	t.Run("write error returns error", func(t *testing.T) {
		t.Parallel()
		err := p.writeLTSV(errWriter{}, []string{"key"}, [][]string{{"value"}})
		if err == nil {
			t.Error("expected error from errWriter, got nil")
		}
	})
}

func TestWriteJSONL_ErrorPath(t *testing.T) {
	t.Parallel()

	p := &Processor{fileType: fileparser.JSONL}

	t.Run("write error returns error", func(t *testing.T) {
		t.Parallel()
		err := p.writeJSONL(errWriter{}, [][]string{{`{"key":"value"}`}})
		if err == nil {
			t.Error("expected error from errWriter, got nil")
		}
	})

	t.Run("empty record is skipped", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := p.writeJSONL(&buf, [][]string{{""}, {}, {`{"a":1}`}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `{"a":1}`) {
			t.Errorf("expected valid JSON in output, got: %s", buf.String())
		}
		// Empty records should be skipped, so only 1 line
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if len(lines) != 1 {
			t.Errorf("expected 1 line (empty records skipped), got %d: %s", len(lines), buf.String())
		}
	})
}

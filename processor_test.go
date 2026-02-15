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
		if strings.Contains(line, "\n") {
			t.Errorf("line %d contains newline (not compacted): %q", i+1, line)
		}
	}
}

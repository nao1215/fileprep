package fileprep

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/parquet-go/parquet-go"
)

func TestParseCSV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		delimiter   rune
		wantHeaders []string
		wantRecords int
		wantErr     bool
	}{
		{
			name:        "standard CSV with header",
			input:       "name,age,city\nAlice,30,NYC\nBob,25,LA",
			delimiter:   ',',
			wantHeaders: []string{"name", "age", "city"},
			wantRecords: 2,
		},
		{
			name:        "TSV format",
			input:       "name\tage\tcity\nAlice\t30\tNYC",
			delimiter:   '\t',
			wantHeaders: []string{"name", "age", "city"},
			wantRecords: 1,
		},
		{
			name:      "empty file",
			input:     "",
			delimiter: ',',
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := parseCSV(strings.NewReader(tt.input), tt.delimiter)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.headers) != len(tt.wantHeaders) {
				t.Errorf("headers count = %d, want %d", len(result.headers), len(tt.wantHeaders))
			}
			for i, h := range result.headers {
				if h != tt.wantHeaders[i] {
					t.Errorf("header[%d] = %q, want %q", i, h, tt.wantHeaders[i])
				}
			}

			if len(result.records) != tt.wantRecords {
				t.Errorf("records count = %d, want %d", len(result.records), tt.wantRecords)
			}
		})
	}
}

func TestParseLTSV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantHeaders []string
		wantRecords int
		wantErr     bool
	}{
		{
			name:        "standard LTSV",
			input:       "host:example.com\tpath:/api\tmethod:GET\nhost:test.com\tpath:/\tmethod:POST",
			wantHeaders: []string{"host", "path", "method"},
			wantRecords: 2,
		},
		{
			name:        "LTSV with missing fields",
			input:       "host:example.com\tpath:/api\nhost:test.com\tmethod:POST",
			wantHeaders: []string{"host", "path", "method"},
			wantRecords: 2,
		},
		{
			name:    "empty file",
			input:   "",
			wantErr: true,
		},
		{
			name:    "only whitespace",
			input:   "   \n   \n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := parseLTSV(strings.NewReader(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.headers) != len(tt.wantHeaders) {
				t.Errorf("headers count = %d, want %d", len(result.headers), len(tt.wantHeaders))
			}

			if len(result.records) != tt.wantRecords {
				t.Errorf("records count = %d, want %d", len(result.records), tt.wantRecords)
			}
		})
	}
}

func TestParseParquet(t *testing.T) {
	t.Parallel()

	// Create a simple parquet file in memory
	type TestRow struct {
		Name  string  `parquet:"name"`
		Age   int32   `parquet:"age"`
		Score float64 `parquet:"score"`
	}

	rows := []TestRow{
		{Name: "Alice", Age: 30, Score: 95.5},
		{Name: "Bob", Age: 25, Score: 87.3},
		{Name: "Charlie", Age: 35, Score: 92.0},
	}

	var buf bytes.Buffer
	writer := parquet.NewGenericWriter[TestRow](&buf)
	_, err := writer.Write(rows)
	if err != nil {
		t.Fatalf("failed to write parquet data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close parquet writer: %v", err)
	}

	// Parse the parquet data
	result, err := parseParquet(buf.Bytes())
	if err != nil {
		t.Fatalf("parseParquet() error = %v", err)
	}

	// Verify headers
	expectedHeaders := []string{"name", "age", "score"}
	if len(result.headers) != len(expectedHeaders) {
		t.Errorf("headers count = %d, want %d", len(result.headers), len(expectedHeaders))
	}
	for i, h := range result.headers {
		if h != expectedHeaders[i] {
			t.Errorf("header[%d] = %q, want %q", i, h, expectedHeaders[i])
		}
	}

	// Verify records
	if len(result.records) != 3 {
		t.Errorf("records count = %d, want 3", len(result.records))
	}

	// Verify first record values
	if len(result.records) > 0 {
		if result.records[0][0] != "Alice" {
			t.Errorf("records[0][0] = %q, want %q", result.records[0][0], "Alice")
		}
		if result.records[0][1] != "30" {
			t.Errorf("records[0][1] = %q, want %q", result.records[0][1], "30")
		}
	}
}

func TestParseParquet_EmptyFile(t *testing.T) {
	t.Parallel()

	_, err := parseParquet([]byte{})
	if !errors.Is(err, ErrEmptyFile) {
		t.Errorf("expected ErrEmptyFile, got %v", err)
	}
}

func TestParseParquet_InvalidData(t *testing.T) {
	t.Parallel()

	_, err := parseParquet([]byte("not a parquet file"))
	if err == nil {
		t.Error("expected error for invalid parquet data")
	}
}

// TestParseParquet_TruncatedFile tests error handling for truncated/corrupted Parquet files
func TestParseParquet_TruncatedFile(t *testing.T) {
	t.Parallel()

	// Create a valid parquet file first
	type TestRow struct {
		Name string `parquet:"name"`
		Age  int32  `parquet:"age"`
	}

	rows := []TestRow{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	var buf bytes.Buffer
	writer := parquet.NewGenericWriter[TestRow](&buf)
	_, err := writer.Write(rows)
	if err != nil {
		t.Fatalf("failed to write parquet data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close parquet writer: %v", err)
	}

	validData := buf.Bytes()

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "truncated at 50%",
			data: validData[:len(validData)/2],
		},
		{
			name: "truncated at 75%",
			data: validData[:len(validData)*3/4],
		},
		{
			name: "truncated at 25%",
			data: validData[:len(validData)/4],
		},
		{
			name: "only header bytes",
			data: validData[:min(100, len(validData))],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseParquet(tt.data)
			if err == nil {
				t.Error("expected error for truncated parquet file")
			}
		})
	}
}

func TestFormatParquetValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value parquet.Value
		want  string
	}{
		{
			name:  "null value",
			value: parquet.NullValue(),
			want:  "",
		},
		{
			name:  "boolean true",
			value: parquet.BooleanValue(true),
			want:  "true",
		},
		{
			name:  "boolean false",
			value: parquet.BooleanValue(false),
			want:  "false",
		},
		{
			name:  "int32",
			value: parquet.Int32Value(42),
			want:  "42",
		},
		{
			name:  "int64",
			value: parquet.Int64Value(9999999999),
			want:  "9999999999",
		},
		{
			name:  "float",
			value: parquet.FloatValue(3.14),
			want:  "3.14",
		},
		{
			name:  "double",
			value: parquet.DoubleValue(2.718281828),
			want:  "2.718281828",
		},
		{
			name:  "byte array",
			value: parquet.ByteArrayValue([]byte("hello")),
			want:  "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatParquetValue(tt.value)
			if got != tt.want {
				t.Errorf("formatParquetValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestStreamFormat_XLSXReturnsCSV verifies that XLSX input returns Stream with Format()=CSV
// while OriginalFormat() stays XLSX
func TestStreamFormat_XLSXReturnsCSV(t *testing.T) {
	t.Parallel()

	// The key test is that outputFormat() returns CSV for XLSX
	processor := NewProcessor(FileTypeXLSX)
	if processor.outputFormat() != FileTypeCSV {
		t.Errorf("outputFormat() for XLSX = %v, want %v", processor.outputFormat(), FileTypeCSV)
	}

	processor = NewProcessor(FileTypeXLSXGZ)
	if processor.outputFormat() != FileTypeCSV {
		t.Errorf("outputFormat() for XLSXGZ = %v, want %v", processor.outputFormat(), FileTypeCSV)
	}
}

// TestStreamFormat_ParquetReturnsCSV verifies that Parquet input returns Stream with Format()=CSV
// while OriginalFormat() stays Parquet
func TestStreamFormat_ParquetReturnsCSV(t *testing.T) {
	t.Parallel()

	type ParquetRow struct {
		Name  string `parquet:"name"`
		Email string `parquet:"email"`
	}

	type TestRecord struct {
		Name  string
		Email string
	}

	rows := []ParquetRow{
		{Name: "John", Email: "john@example.com"},
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

	// Verify Stream interface
	stream, ok := reader.(Stream)
	if !ok {
		t.Fatal("returned reader does not implement Stream interface")
	}

	// Format() should return CSV for Parquet input
	if stream.Format() != FileTypeCSV {
		t.Errorf("Stream.Format() = %v, want %v", stream.Format(), FileTypeCSV)
	}

	// OriginalFormat() should return the original Parquet type
	if stream.OriginalFormat() != FileTypeParquet {
		t.Errorf("Stream.OriginalFormat() = %v, want %v", stream.OriginalFormat(), FileTypeParquet)
	}

	// ProcessResult should also have OriginalFormat
	if result.OriginalFormat != FileTypeParquet {
		t.Errorf("ProcessResult.OriginalFormat = %v, want %v", result.OriginalFormat, FileTypeParquet)
	}
}

// TestTypeConversionErrors verifies that setFieldValue errors are reported as PrepErrors
func TestTypeConversionErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		csvData        string
		wantPrepErrors int
		wantValidRows  int
	}{
		{
			name: "invalid int value",
			csvData: `int_field
not_a_number
42`,
			wantPrepErrors: 1,
			wantValidRows:  1,
		},
		{
			name: "invalid float value",
			csvData: `float_field
invalid_float
3.14`,
			wantPrepErrors: 1,
			wantValidRows:  1,
		},
		{
			name: "invalid bool value",
			csvData: `bool_field
maybe
true`,
			wantPrepErrors: 1,
			wantValidRows:  1,
		},
		{
			name: "multiple type conversion errors",
			csvData: `int_field
abc
def
123`,
			wantPrepErrors: 2,
			wantValidRows:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var records []struct {
				IntField   int     `csv:"IntField"`
				FloatField float64 `csv:"FloatField"`
				BoolField  bool    `csv:"BoolField"`
			}

			// Use specific struct based on test
			switch tt.name {
			case "invalid int value", "multiple type conversion errors":
				type IntRecord struct {
					IntField int
				}
				var intRecords []IntRecord
				processor := NewProcessor(FileTypeCSV)
				_, result, err := processor.Process(strings.NewReader(tt.csvData), &intRecords)
				if err != nil {
					t.Fatalf("Process() error = %v", err)
				}

				prepErrors := result.PrepErrors()
				if len(prepErrors) != tt.wantPrepErrors {
					t.Errorf("len(PrepErrors()) = %d, want %d", len(prepErrors), tt.wantPrepErrors)
				}

				if result.ValidRowCount != tt.wantValidRows {
					t.Errorf("ValidRowCount = %d, want %d", result.ValidRowCount, tt.wantValidRows)
				}

				// Verify PrepError details
				if len(prepErrors) > 0 {
					pe := prepErrors[0]
					if pe.Tag != "type_conversion" {
						t.Errorf("PrepError.Tag = %q, want %q", pe.Tag, "type_conversion")
					}
					if pe.Row != 1 {
						t.Errorf("PrepError.Row = %d, want 1", pe.Row)
					}
				}

			case "invalid float value":
				type FloatRecord struct {
					FloatField float64
				}
				var floatRecords []FloatRecord
				processor := NewProcessor(FileTypeCSV)
				_, result, err := processor.Process(strings.NewReader(tt.csvData), &floatRecords)
				if err != nil {
					t.Fatalf("Process() error = %v", err)
				}

				prepErrors := result.PrepErrors()
				if len(prepErrors) != tt.wantPrepErrors {
					t.Errorf("len(PrepErrors()) = %d, want %d", len(prepErrors), tt.wantPrepErrors)
				}

			case "invalid bool value":
				type BoolRecord struct {
					BoolField bool
				}
				var boolRecords []BoolRecord
				processor := NewProcessor(FileTypeCSV)
				_, result, err := processor.Process(strings.NewReader(tt.csvData), &boolRecords)
				if err != nil {
					t.Fatalf("Process() error = %v", err)
				}

				prepErrors := result.PrepErrors()
				if len(prepErrors) != tt.wantPrepErrors {
					t.Errorf("len(PrepErrors()) = %d, want %d", len(prepErrors), tt.wantPrepErrors)
				}

			default:
				processor := NewProcessor(FileTypeCSV)
				_, result, err := processor.Process(strings.NewReader(tt.csvData), &records)
				if err != nil {
					t.Fatalf("Process() error = %v", err)
				}

				prepErrors := result.PrepErrors()
				if len(prepErrors) != tt.wantPrepErrors {
					t.Errorf("len(PrepErrors()) = %d, want %d", len(prepErrors), tt.wantPrepErrors)
				}
			}
		})
	}
}

// TestMissingColumnsValidation verifies that missing columns trigger validation
// (fields without matching CSV columns are treated as empty strings)
func TestMissingColumnsValidation(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name    string `validate:"required"`
		Email   string `validate:"required"`
		Age     string
		Country string `validate:"required"`
	}

	// CSV with fewer columns than struct fields
	// "country" column is missing, so Country field will be empty and fail required validation
	csvData := `name,email
John,john@example.com
Jane,jane@example.com`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// With name-based binding, fields without matching columns are treated as empty
	// Country has required validation, so it should fail for each row
	if !result.HasErrors() {
		t.Error("Expected validation errors for missing required column")
	}

	// Should have 2 errors (one per row for missing Country)
	valErrors := result.ValidationErrors()
	if len(valErrors) != 2 {
		t.Errorf("len(ValidationErrors) = %d, want 2", len(valErrors))
	}

	// Check that errors are for Country field
	for _, ve := range valErrors {
		if ve.Field != "Country" {
			t.Errorf("Expected error for Country field, got %s", ve.Field)
		}
	}

	// ValidRowCount should be 0 since all rows have missing required field
	if result.ValidRowCount != 0 {
		t.Errorf("ValidRowCount = %d, want 0", result.ValidRowCount)
	}

	// Verify Name and Email fields are populated
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}
	if records[0].Name != "John" {
		t.Errorf("records[0].Name = %q, want %q", records[0].Name, "John")
	}
	if records[0].Email != "john@example.com" {
		t.Errorf("records[0].Email = %q, want %q", records[0].Email, "john@example.com")
	}
	// Country should be empty (missing column treated as empty string)
	if records[0].Country != "" {
		t.Errorf("records[0].Country = %q, want empty (missing column)", records[0].Country)
	}
}

// TestLTSVSparseHeaders verifies that LTSV with disjoint keys preserves union of all keys
func TestLTSVSparseHeaders(t *testing.T) {
	t.Parallel()

	// Row 1 has: name, email
	// Row 2 has: name, age
	// Row 3 has: email, country
	// Expected headers: name, email, age, country (union of all keys)
	ltsvData := `name:John	email:john@example.com
name:Jane	age:25
email:bob@example.com	country:USA`

	processor := NewProcessor(FileTypeLTSV)

	type TestRecord struct {
		Name    string
		Email   string
		Age     string
		Country string
	}
	var records []TestRecord

	reader, result, err := processor.Process(strings.NewReader(ltsvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Check that all unique keys are in the headers
	expectedHeaders := map[string]bool{
		"name":    true,
		"email":   true,
		"age":     true,
		"country": true,
	}

	if len(result.Columns) != len(expectedHeaders) {
		t.Errorf("len(Columns) = %d, want %d", len(result.Columns), len(expectedHeaders))
	}

	for _, col := range result.Columns {
		if !expectedHeaders[col] {
			t.Errorf("unexpected column: %q", col)
		}
	}

	// Verify output contains all headers
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	outputStr := string(output)

	// Each row should have all 4 keys (even if value is empty)
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 output lines, got %d", len(lines))
	}

	// Verify each line has all keys
	for i, line := range lines {
		for key := range expectedHeaders {
			if !strings.Contains(line, key+":") {
				t.Errorf("line %d missing key %q: %s", i+1, key, line)
			}
		}
	}
}

// TestCompressionFormatMetadata verifies compressed inputs have correct Format/OriginalFormat
func TestCompressionFormatMetadata(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string `prep:"trim"`
		Email string
		Age   string
	}

	tests := []struct {
		name           string
		filePath       string
		fileType       FileType
		wantFormat     FileType
		wantOrigFormat FileType
	}{
		{
			name:           "CSV gzip",
			filePath:       filepath.Join("testdata", "sample.csv.gz"),
			fileType:       FileTypeCSVGZ,
			wantFormat:     FileTypeCSV,
			wantOrigFormat: FileTypeCSVGZ,
		},
		{
			name:           "CSV bzip2",
			filePath:       filepath.Join("testdata", "sample.csv.bz2"),
			fileType:       FileTypeCSVBZ2,
			wantFormat:     FileTypeCSV,
			wantOrigFormat: FileTypeCSVBZ2,
		},
		{
			name:           "CSV xz",
			filePath:       filepath.Join("testdata", "sample.csv.xz"),
			fileType:       FileTypeCSVXZ,
			wantFormat:     FileTypeCSV,
			wantOrigFormat: FileTypeCSVXZ,
		},
		{
			name:           "CSV zstd",
			filePath:       filepath.Join("testdata", "sample.csv.zst"),
			fileType:       FileTypeCSVZSTD,
			wantFormat:     FileTypeCSV,
			wantOrigFormat: FileTypeCSVZSTD,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := os.Open(tt.filePath)
			if err != nil {
				t.Fatalf("os.Open() error = %v", err)
			}
			defer file.Close()

			var records []TestRecord

			processor := NewProcessor(tt.fileType)
			reader, result, err := processor.Process(file, &records)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			// Verify Stream interface
			stream, ok := reader.(Stream)
			if !ok {
				t.Fatal("returned reader does not implement Stream interface")
			}

			// Verify Format() returns base format (without compression)
			if stream.Format() != tt.wantFormat {
				t.Errorf("Stream.Format() = %v, want %v", stream.Format(), tt.wantFormat)
			}

			// Verify OriginalFormat() returns full format (with compression)
			if stream.OriginalFormat() != tt.wantOrigFormat {
				t.Errorf("Stream.OriginalFormat() = %v, want %v", stream.OriginalFormat(), tt.wantOrigFormat)
			}

			// Verify ProcessResult.OriginalFormat
			if result.OriginalFormat != tt.wantOrigFormat {
				t.Errorf("ProcessResult.OriginalFormat = %v, want %v", result.OriginalFormat, tt.wantOrigFormat)
			}

			// Verify data was decompressed and processed
			if len(records) == 0 {
				t.Error("expected at least one record")
			}

			// Verify preprocessing was applied
			if records[0].Name != "John Doe" {
				t.Errorf("Name = %q, want %q (trim should be applied)", records[0].Name, "John Doe")
			}
		})
	}
}

// TestTSVOutputFormat verifies that TSV input preserves TSV output format
func TestTSVOutputFormat(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string `prep:"trim"`
		Email string
	}

	tsvData := "name\temail\n  John  \tjohn@example.com\n"

	processor := NewProcessor(FileTypeTSV)
	var records []TestRecord

	reader, _, err := processor.Process(strings.NewReader(tsvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	stream, ok := reader.(Stream)
	if !ok {
		t.Fatal("returned reader does not implement Stream interface")
	}

	if stream.Format() != FileTypeTSV {
		t.Errorf("Stream.Format() = %v, want %v", stream.Format(), FileTypeTSV)
	}

	// Verify output is tab-delimited
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Header should be tab-delimited
	if !strings.Contains(lines[0], "\t") {
		t.Error("output header should be tab-delimited")
	}

	// Data row should contain trimmed value
	if !strings.Contains(lines[1], "John") {
		t.Error("output should contain trimmed name")
	}
}

// TestLTSVOutputFormat verifies that LTSV input preserves LTSV output format
func TestLTSVOutputFormat(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string `prep:"trim"`
		Email string
	}

	ltsvData := "name:  John  \temail:john@example.com\n"

	processor := NewProcessor(FileTypeLTSV)
	var records []TestRecord

	reader, _, err := processor.Process(strings.NewReader(ltsvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	stream, ok := reader.(Stream)
	if !ok {
		t.Fatal("returned reader does not implement Stream interface")
	}

	if stream.Format() != FileTypeLTSV {
		t.Errorf("Stream.Format() = %v, want %v", stream.Format(), FileTypeLTSV)
	}

	// Verify output is LTSV format (key:value pairs)
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	outputStr := string(output)

	// Should contain key:value format (keys from CSV are lowercase)
	if !strings.Contains(outputStr, "name:") {
		t.Error("LTSV output should contain 'name:' key")
	}
	if !strings.Contains(outputStr, "email:") {
		t.Error("LTSV output should contain 'email:' key")
	}

	// Should contain trimmed value
	if !strings.Contains(outputStr, "name:John") {
		t.Error("LTSV output should contain trimmed name value")
	}
}

// TestUnsupportedFieldType verifies error handling for unsupported struct field types
func TestUnsupportedFieldType(t *testing.T) {
	t.Parallel()

	type ComplexRecord struct {
		Data []string // slice type is unsupported
	}

	csvData := `data
value1`

	processor := NewProcessor(FileTypeCSV)
	var records []ComplexRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should have PrepError for unsupported type
	prepErrors := result.PrepErrors()
	if len(prepErrors) != 1 {
		t.Errorf("len(PrepErrors()) = %d, want 1", len(prepErrors))
	}

	if len(prepErrors) > 0 {
		if prepErrors[0].Tag != "type_conversion" {
			t.Errorf("PrepError.Tag = %q, want %q", prepErrors[0].Tag, "type_conversion")
		}
	}
}

// TestColumnNameFallback verifies that fields without matching columns are validated as empty
func TestColumnNameFallback(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		First  string `validate:"required"`
		Second string `validate:"required"`
		Third  string `validate:"required"` // No header for this
	}

	// CSV with only 2 columns but struct has 3 fields
	// Third field expects "third" column which doesn't exist, treated as empty
	csvData := `first,second
,`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should have 3 validation errors (all fields are empty or missing)
	// Third field is missing but treated as empty, so required validation fails
	validationErrors := result.ValidationErrors()
	if len(validationErrors) != 3 {
		t.Errorf("len(ValidationErrors()) = %d, want 3", len(validationErrors))
	}

	// Verify errors are for all three fields
	fieldErrors := make(map[string]bool)
	for _, ve := range validationErrors {
		fieldErrors[ve.Field] = true
	}

	if !fieldErrors["First"] {
		t.Error("expected validation error for First field")
	}
	if !fieldErrors["Second"] {
		t.Error("expected validation error for Second field")
	}
	if !fieldErrors["Third"] {
		t.Error("expected validation error for Third field (missing column = empty)")
	}
}

// TestXLSXParquetOutputCSV verifies XLSX/Parquet inputs emit CSV and Stream metadata is correct
func TestXLSXParquetOutputCSV(t *testing.T) {
	t.Parallel()

	type ParquetRow struct {
		Name  string `parquet:"name"`
		Email string `parquet:"email"`
	}

	type TestRecord struct {
		Name  string
		Email string
	}

	rows := []ParquetRow{
		{Name: "Alice", Email: "alice@example.com"},
		{Name: "Bob", Email: "bob@example.com"},
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

	// Verify Stream interface
	stream, ok := reader.(Stream)
	if !ok {
		t.Fatal("returned reader does not implement Stream interface")
	}

	// Format() should return CSV for Parquet input
	if stream.Format() != FileTypeCSV {
		t.Errorf("Stream.Format() = %v, want %v", stream.Format(), FileTypeCSV)
	}

	// OriginalFormat() should return Parquet
	if stream.OriginalFormat() != FileTypeParquet {
		t.Errorf("Stream.OriginalFormat() = %v, want %v", stream.OriginalFormat(), FileTypeParquet)
	}

	// ProcessResult should also have OriginalFormat
	if result.OriginalFormat != FileTypeParquet {
		t.Errorf("ProcessResult.OriginalFormat = %v, want %v", result.OriginalFormat, FileTypeParquet)
	}

	// Verify output is CSV format
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	outputStr := string(output)
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")

	// Should have header + 2 data rows
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (header + 2 data), got %d", len(lines))
	}

	// Header should be comma-delimited CSV
	if !strings.Contains(lines[0], ",") {
		t.Error("output should be CSV (comma-delimited)")
	}

	// Verify XLSX also outputs CSV
	processorXLSX := NewProcessor(FileTypeXLSX)
	if processorXLSX.outputFormat() != FileTypeCSV {
		t.Errorf("XLSX outputFormat() = %v, want %v", processorXLSX.outputFormat(), FileTypeCSV)
	}
}

// TestCrossFieldValidatorMissingTarget verifies error handling when cross-field target is missing
func TestCrossFieldValidatorMissingTarget(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Password        string
		ConfirmPassword string `validate:"eqfield=NonExistent"`
	}

	csvData := `password,confirm_password
secret123,secret123`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should have error for missing target field
	if !result.HasErrors() {
		t.Error("expected validation error for missing target field")
	}

	validationErrors := result.ValidationErrors()
	if len(validationErrors) != 1 {
		t.Errorf("len(ValidationErrors()) = %d, want 1", len(validationErrors))
	}

	if len(validationErrors) > 0 {
		ve := validationErrors[0]
		if !strings.Contains(ve.Message, "not found") {
			t.Errorf("expected 'not found' in error message, got: %s", ve.Message)
		}
	}
}

// TestCrossFieldValidatorOutOfRangeTarget verifies error when target field has no matching column
func TestCrossFieldValidatorOutOfRangeTarget(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		First  string
		Second string `validate:"eqfield=Third"`
		Third  string // No column for this field
	}

	// CSV has only 2 columns but Third field is referenced in eqfield
	csvData := `first,second
value1,value2`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should have error for missing target field column
	validationErrors := result.ValidationErrors()

	// Find error for eqfield validation
	var eqfieldError *ValidationError
	for i := range validationErrors {
		if validationErrors[i].Tag == "eqfield" {
			eqfieldError = validationErrors[i]
			break
		}
	}

	if eqfieldError == nil {
		t.Fatal("expected validation error for eqfield with missing target column")
	}

	// Error message should indicate target field not found (because column doesn't exist)
	if !strings.Contains(eqfieldError.Message, "not found") {
		t.Errorf("expected 'not found' in error message, got: %s", eqfieldError.Message)
	}
}

// TestLTSVSparseKeysOutputOrder verifies LTSV with different key sets preserves all keys in output
func TestLTSVSparseKeysOutputOrder(t *testing.T) {
	t.Parallel()

	// Row 1: has only name, email
	// Row 2: has only name, age
	// Row 3: has only email, country
	ltsvData := `name:Alice	email:alice@example.com
name:Bob	age:30
email:charlie@example.com	country:USA`

	processor := NewProcessor(FileTypeLTSV)

	type TestRecord struct {
		Name    string
		Email   string
		Age     string
		Country string
	}
	var records []TestRecord

	reader, result, err := processor.Process(strings.NewReader(ltsvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Verify all 4 unique keys are collected as headers
	expectedKeys := []string{"name", "email", "age", "country"}
	if len(result.Columns) != len(expectedKeys) {
		t.Errorf("len(Columns) = %d, want %d", len(result.Columns), len(expectedKeys))
	}

	// Create a set for verification
	headerSet := make(map[string]bool)
	for _, col := range result.Columns {
		headerSet[col] = true
	}

	for _, key := range expectedKeys {
		if !headerSet[key] {
			t.Errorf("missing expected key in headers: %q", key)
		}
	}

	// Verify output: each line should have ALL keys (even with empty values)
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 output lines, got %d", len(lines))
	}

	// Each line must contain all 4 keys
	for i, line := range lines {
		for _, key := range expectedKeys {
			if !strings.Contains(line, key+":") {
				t.Errorf("line %d missing key %q: %s", i+1, key, line)
			}
		}
	}

	// Verify specific values are preserved
	// Line 1: name:Alice, email:alice@example.com, age:, country:
	if !strings.Contains(lines[0], "name:Alice") {
		t.Errorf("line 1 missing 'name:Alice': %s", lines[0])
	}
	if !strings.Contains(lines[0], "email:alice@example.com") {
		t.Errorf("line 1 missing 'email:alice@example.com': %s", lines[0])
	}

	// Line 2: name:Bob, email:, age:30, country:
	if !strings.Contains(lines[1], "name:Bob") {
		t.Errorf("line 2 missing 'name:Bob': %s", lines[1])
	}
	if !strings.Contains(lines[1], "age:30") {
		t.Errorf("line 2 missing 'age:30': %s", lines[1])
	}
}

// TestRealXLSXFileProcessing tests processing a real XLSX file from testdata
func TestRealXLSXFileProcessing(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		ID    string
		Name  string
		Value string
	}

	file, err := os.Open(filepath.Join("testdata", "sample.xlsx"))
	if err != nil {
		t.Fatalf("os.Open() error = %v", err)
	}
	defer file.Close()

	processor := NewProcessor(FileTypeXLSX)
	var records []TestRecord

	reader, result, err := processor.Process(file, &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Verify we got rows
	if result.RowCount == 0 {
		t.Error("expected at least one row from XLSX file")
	}

	// Verify Stream metadata
	stream, ok := reader.(Stream)
	if !ok {
		t.Fatal("returned reader does not implement Stream interface")
	}

	// XLSX outputs as CSV
	if stream.Format() != FileTypeCSV {
		t.Errorf("Stream.Format() = %v, want %v", stream.Format(), FileTypeCSV)
	}

	// OriginalFormat should be XLSX
	if stream.OriginalFormat() != FileTypeXLSX {
		t.Errorf("Stream.OriginalFormat() = %v, want %v", stream.OriginalFormat(), FileTypeXLSX)
	}

	// Verify ProcessResult
	if result.OriginalFormat != FileTypeXLSX {
		t.Errorf("ProcessResult.OriginalFormat = %v, want %v", result.OriginalFormat, FileTypeXLSX)
	}

	// Verify output is valid CSV
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, ",") {
		t.Error("expected CSV output (comma-delimited)")
	}

	t.Logf("Processed %d rows from real XLSX file", result.RowCount)
}

// TestRealParquetFileProcessing tests processing a real Parquet file written to disk
func TestRealParquetFileProcessing(t *testing.T) {
	t.Parallel()

	type ParquetRecord struct {
		ID    int32  `parquet:"id"`
		Name  string `parquet:"name"`
		Email string `parquet:"email"`
		Age   int32  `parquet:"age"`
	}

	type TestRecord struct {
		ID    string
		Name  string `prep:"trim"`
		Email string
		Age   string
	}

	// Create a real parquet file in memory and write to temp file
	tempDir := t.TempDir()
	parquetPath := filepath.Join(tempDir, "test.parquet")

	rows := []ParquetRecord{
		{ID: 1, Name: "  Alice  ", Email: "alice@example.com", Age: 30},
		{ID: 2, Name: "Bob", Email: "bob@example.com", Age: 25},
		{ID: 3, Name: "  Charlie  ", Email: "charlie@example.com", Age: 35},
	}

	// Write Parquet to buffer first, then to file
	var buf bytes.Buffer
	writer := parquet.NewGenericWriter[ParquetRecord](&buf)
	_, err := writer.Write(rows)
	if err != nil {
		t.Fatalf("failed to write parquet data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close parquet writer: %v", err)
	}

	// Write buffer to file
	if err := os.WriteFile(parquetPath, buf.Bytes(), 0600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	processor := NewProcessor(FileTypeParquet)
	var records []TestRecord

	// Process from bytes.Reader (simulating reading from disk file)
	reader, result, err := processor.Process(bytes.NewReader(buf.Bytes()), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Verify row count
	if result.RowCount != 3 {
		t.Errorf("RowCount = %d, want 3", result.RowCount)
	}

	// Verify records were populated
	if len(records) != 3 {
		t.Fatalf("len(records) = %d, want 3", len(records))
	}

	// Verify trim preprocessing was applied
	if records[0].Name != "Alice" {
		t.Errorf("records[0].Name = %q, want %q (trim should be applied)", records[0].Name, "Alice")
	}

	if records[2].Name != "Charlie" {
		t.Errorf("records[2].Name = %q, want %q (trim should be applied)", records[2].Name, "Charlie")
	}

	// Verify Stream metadata
	stream, ok := reader.(Stream)
	if !ok {
		t.Fatal("returned reader does not implement Stream interface")
	}

	if stream.Format() != FileTypeCSV {
		t.Errorf("Stream.Format() = %v, want %v", stream.Format(), FileTypeCSV)
	}

	if stream.OriginalFormat() != FileTypeParquet {
		t.Errorf("Stream.OriginalFormat() = %v, want %v", stream.OriginalFormat(), FileTypeParquet)
	}

	// Verify columns
	expectedColumns := []string{"id", "name", "email", "age"}
	if len(result.Columns) != len(expectedColumns) {
		t.Errorf("len(Columns) = %d, want %d", len(result.Columns), len(expectedColumns))
	}
	for i, col := range result.Columns {
		if col != expectedColumns[i] {
			t.Errorf("Columns[%d] = %q, want %q", i, col, expectedColumns[i])
		}
	}

	t.Logf("Processed %d rows from real Parquet file", result.RowCount)
}

// TestDuplicateColumnsCSV tests processing CSV with duplicate column names
func TestDuplicateColumnsCSV(t *testing.T) {
	t.Parallel()

	// With name-based mapping, duplicate column names bind to first occurrence
	// ID1 and ID2 both map to "id" but only one can match (first occurrence)
	type TestRecord struct {
		ID1   string `name:"id"`
		Name  string
		ID2   string `name:"id2"` // This won't match any column
		Email string
	}

	csvData := `id,name,id2,email
1,John,10,john@example.com
2,Jane,20,jane@example.com`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	reader, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should process all rows
	if result.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", result.RowCount)
	}

	// Verify records were populated (name-based binding)
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}

	// Name-based binding: struct fields map to columns by name
	// ID1 -> "id" column (column 0)
	// Name -> "name" column (column 1)
	// ID2 -> "id2" column (column 2)
	// Email -> "email" column (column 3)
	if records[0].ID1 != "1" {
		t.Errorf("records[0].ID1 = %q, want %q", records[0].ID1, "1")
	}
	if records[0].Name != "John" {
		t.Errorf("records[0].Name = %q, want %q", records[0].Name, "John")
	}
	if records[0].ID2 != "10" {
		t.Errorf("records[0].ID2 = %q, want %q", records[0].ID2, "10")
	}
	if records[0].Email != "john@example.com" {
		t.Errorf("records[0].Email = %q, want %q", records[0].Email, "john@example.com")
	}

	// Verify second row
	if records[1].ID1 != "2" {
		t.Errorf("records[1].ID1 = %q, want %q", records[1].ID1, "2")
	}
	if records[1].ID2 != "20" {
		t.Errorf("records[1].ID2 = %q, want %q", records[1].ID2, "20")
	}

	// Verify columns (as-is from file)
	if len(result.Columns) != 4 {
		t.Errorf("len(Columns) = %d, want 4", len(result.Columns))
	}

	// Columns should be: id, name, id2, email
	if result.Columns[0] != "id" {
		t.Errorf("Columns[0] = %q, want %q", result.Columns[0], "id")
	}
	if result.Columns[2] != "id2" {
		t.Errorf("Columns[2] = %q, want %q", result.Columns[2], "id2")
	}

	// Verify output preserves duplicate columns
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 output lines (header + 2 data), got %d", len(lines))
	}

	// Header should match original columns
	if !strings.HasPrefix(lines[0], "id,") {
		t.Errorf("header should start with 'id,': %s", lines[0])
	}
}

// TestTrueDuplicateHeaders tests CSV with truly duplicated header names (e.g., id,id,name)
// First occurrence wins for duplicate column names
func TestTrueDuplicateHeaders(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		ID   string
		Name string
	}

	// CSV with truly duplicate "id" headers - first occurrence (column 0) should win
	csvData := `id,id,name
first_id,second_id,John
1,2,Jane`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.HasErrors() {
		for _, e := range result.Errors {
			t.Errorf("Unexpected error: %v", e)
		}
	}

	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}

	// First occurrence of "id" (column 0) should be bound, not second (column 1)
	if records[0].ID != "first_id" {
		t.Errorf("records[0].ID = %q, want %q (first occurrence should win)", records[0].ID, "first_id")
	}
	if records[0].Name != "John" {
		t.Errorf("records[0].Name = %q, want %q", records[0].Name, "John")
	}
	if records[1].ID != "1" {
		t.Errorf("records[1].ID = %q, want %q", records[1].ID, "1")
	}
}

// TestHeaderCaseSensitivity tests that header matching is case-sensitive
func TestHeaderCaseSensitivity(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		UserName string // expects "user_name", not "User_Name" or "USER_NAME"
	}

	tests := []struct {
		name      string
		csvData   string
		wantMatch bool
		wantValue string
	}{
		{
			name:      "exact match snake_case",
			csvData:   "user_name\nAlice",
			wantMatch: true,
			wantValue: "Alice",
		},
		{
			name:      "wrong case User_Name",
			csvData:   "User_Name\nBob",
			wantMatch: false,
			wantValue: "",
		},
		{
			name:      "wrong case USER_NAME",
			csvData:   "USER_NAME\nCharlie",
			wantMatch: false,
			wantValue: "",
		},
		{
			name:      "wrong case userName",
			csvData:   "userName\nDave",
			wantMatch: false,
			wantValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			processor := NewProcessor(FileTypeCSV)
			var records []TestRecord

			_, _, err := processor.Process(strings.NewReader(tt.csvData), &records)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if len(records) != 1 {
				t.Fatalf("len(records) = %d, want 1", len(records))
			}

			if tt.wantMatch {
				if records[0].UserName != tt.wantValue {
					t.Errorf("UserName = %q, want %q", records[0].UserName, tt.wantValue)
				}
			} else {
				// If no match, field should be empty
				if records[0].UserName != "" {
					t.Errorf("UserName = %q, want empty (no match due to case)", records[0].UserName)
				}
			}
		})
	}
}

// TestLTSVKeyCaseSensitivity tests LTSV key matching is case-sensitive
func TestLTSVKeyCaseSensitivity(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		UserName string // expects "user_name" key
		Email    string // expects "email" key
	}

	// LTSV with keys that don't match snake_case expectation
	// user_name matches, but Email won't match "email" if key is "Email"
	ltsvData := "user_name:Alice\tEmail:alice@example.com\n"

	processor := NewProcessor(FileTypeLTSV)
	var records []TestRecord

	_, _, err := processor.Process(strings.NewReader(ltsvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}

	// user_name should match
	if records[0].UserName != "Alice" {
		t.Errorf("UserName = %q, want %q", records[0].UserName, "Alice")
	}

	// Email (capital E) won't match "email" (lowercase) - case sensitive
	if records[0].Email != "" {
		t.Errorf("Email = %q, want empty (key 'Email' doesn't match 'email')", records[0].Email)
	}
}

// TestCrossFieldValidatorWithNameTag tests cross-field validation when fields use name tags
func TestCrossFieldValidatorWithNameTag(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		PrimaryEmail   string `name:"primary_mail"`
		SecondaryEmail string `name:"secondary_mail" validate:"eqfield=PrimaryEmail"`
	}

	tests := []struct {
		name      string
		csvData   string
		wantError bool
	}{
		{
			name:      "matching emails - valid",
			csvData:   "primary_mail,secondary_mail\ntest@example.com,test@example.com",
			wantError: false,
		},
		{
			name:      "different emails - invalid",
			csvData:   "primary_mail,secondary_mail\ntest@example.com,other@example.com",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			processor := NewProcessor(FileTypeCSV)
			var records []TestRecord

			_, result, err := processor.Process(strings.NewReader(tt.csvData), &records)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if tt.wantError {
				if !result.HasErrors() {
					t.Error("expected validation error for mismatched fields")
				}
			} else {
				if result.HasErrors() {
					for _, e := range result.Errors {
						t.Errorf("Unexpected error: %v", e)
					}
				}
			}
		})
	}
}

// TestRaggedRowsNormalization tests that short rows are padded to header width
func TestRaggedRowsNormalization(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string
		Email string
		Age   string
	}

	// CSV with ragged rows (fewer columns than header)
	csvData := `name,email,age
Alice,alice@example.com,30
Bob,bob@example.com
Charlie`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	reader, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.RowCount != 3 {
		t.Errorf("RowCount = %d, want 3", result.RowCount)
	}

	// All records should be populated (short rows padded with empty strings)
	if len(records) != 3 {
		t.Fatalf("len(records) = %d, want 3", len(records))
	}

	// First row: full data
	if records[0].Name != "Alice" || records[0].Email != "alice@example.com" || records[0].Age != "30" {
		t.Errorf("records[0] = %+v, want full data", records[0])
	}

	// Second row: missing Age (should be empty)
	if records[1].Name != "Bob" || records[1].Email != "bob@example.com" || records[1].Age != "" {
		t.Errorf("records[1] = %+v, want Age empty", records[1])
	}

	// Third row: missing Email and Age (should be empty)
	if records[2].Name != "Charlie" || records[2].Email != "" || records[2].Age != "" {
		t.Errorf("records[2] = %+v, want Email and Age empty", records[2])
	}

	// Verify output CSV has normalized columns (all rows should have 3 columns)
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 4 { // header + 3 data rows
		t.Errorf("expected 4 output lines, got %d", len(lines))
	}

	// Each data row should have same number of columns as header
	for i, line := range lines {
		cols := strings.Split(line, ",")
		if len(cols) != 3 {
			t.Errorf("line %d has %d columns, want 3: %q", i, len(cols), line)
		}
	}
}

// TestRaggedRowsWithValidation tests that validation works correctly on padded columns
func TestRaggedRowsWithValidation(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string `validate:"required"`
		Email string `validate:"required"`
		Age   string
	}

	// CSV with ragged rows - missing required Email column
	csvData := `name,email,age
Alice,alice@example.com,30
Bob`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should have validation error for Bob's missing email
	if !result.HasErrors() {
		t.Error("expected validation error for missing required email")
	}

	valErrors := result.ValidationErrors()
	if len(valErrors) != 1 {
		t.Errorf("len(ValidationErrors) = %d, want 1", len(valErrors))
	}

	if len(valErrors) > 0 {
		if valErrors[0].Field != "Email" {
			t.Errorf("expected error for Email field, got %s", valErrors[0].Field)
		}
		if valErrors[0].Row != 2 {
			t.Errorf("expected error on row 2, got row %d", valErrors[0].Row)
		}
	}
}

// TestRaggedRowsCrossFieldValidation tests cross-field validation on ragged rows
func TestRaggedRowsCrossFieldValidation(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Password        string
		ConfirmPassword string `validate:"eqfield=Password"`
	}

	// CSV where second row is missing confirm_password
	csvData := `password,confirm_password
secret123,secret123
password456`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should have validation error for mismatched passwords (empty vs password456)
	if !result.HasErrors() {
		t.Error("expected validation error for password mismatch")
	}

	// First row should be valid
	if result.ValidRowCount != 1 {
		t.Errorf("ValidRowCount = %d, want 1", result.ValidRowCount)
	}
}

// TestCompressedInputProcessing verifies compressed files are decompressed and processed correctly
func TestCompressedInputProcessing(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string `prep:"trim"`
		Email string
		Age   string
	}

	tests := []struct {
		name           string
		filePath       string
		fileType       FileType
		wantFormat     FileType
		wantOrigFormat FileType
	}{
		{
			name:           "CSV gzip",
			filePath:       filepath.Join("testdata", "sample.csv.gz"),
			fileType:       FileTypeCSVGZ,
			wantFormat:     FileTypeCSV,
			wantOrigFormat: FileTypeCSVGZ,
		},
		{
			name:           "CSV bzip2",
			filePath:       filepath.Join("testdata", "sample.csv.bz2"),
			fileType:       FileTypeCSVBZ2,
			wantFormat:     FileTypeCSV,
			wantOrigFormat: FileTypeCSVBZ2,
		},
		{
			name:           "CSV xz",
			filePath:       filepath.Join("testdata", "sample.csv.xz"),
			fileType:       FileTypeCSVXZ,
			wantFormat:     FileTypeCSV,
			wantOrigFormat: FileTypeCSVXZ,
		},
		{
			name:           "CSV zstd",
			filePath:       filepath.Join("testdata", "sample.csv.zst"),
			fileType:       FileTypeCSVZSTD,
			wantFormat:     FileTypeCSV,
			wantOrigFormat: FileTypeCSVZSTD,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := os.Open(tt.filePath)
			if err != nil {
				t.Fatalf("os.Open() error = %v", err)
			}
			defer file.Close()

			var records []TestRecord

			processor := NewProcessor(tt.fileType)
			reader, result, err := processor.Process(file, &records)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			// Verify Stream interface
			stream, ok := reader.(Stream)
			if !ok {
				t.Fatal("returned reader does not implement Stream interface")
			}

			// Verify Format() returns base format (without compression)
			if stream.Format() != tt.wantFormat {
				t.Errorf("Stream.Format() = %v, want %v", stream.Format(), tt.wantFormat)
			}

			// Verify OriginalFormat() returns full format (with compression)
			if stream.OriginalFormat() != tt.wantOrigFormat {
				t.Errorf("Stream.OriginalFormat() = %v, want %v", stream.OriginalFormat(), tt.wantOrigFormat)
			}

			// Verify ProcessResult.OriginalFormat
			if result.OriginalFormat != tt.wantOrigFormat {
				t.Errorf("ProcessResult.OriginalFormat = %v, want %v", result.OriginalFormat, tt.wantOrigFormat)
			}

			// Verify data was decompressed and processed
			if len(records) == 0 {
				t.Error("expected at least one record")
			}

			// Verify preprocessing was applied (trim)
			if records[0].Name != "John Doe" {
				t.Errorf("Name = %q, want %q (trim should be applied)", records[0].Name, "John Doe")
			}
		})
	}
}

// TestCompressedParquetProcessing verifies zstd-compressed Parquet files are processed correctly
func TestCompressedParquetProcessing(t *testing.T) {
	t.Parallel()

	type ParquetRecord struct {
		ID    int32  `parquet:"id"`
		Name  string `parquet:"name"`
		Email string `parquet:"email"`
	}

	type TestRecord struct {
		ID    string
		Name  string `prep:"trim"`
		Email string
	}

	// Create Parquet data
	rows := []ParquetRecord{
		{ID: 1, Name: "  Alice  ", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
	}

	var parquetBuf bytes.Buffer
	writer := parquet.NewGenericWriter[ParquetRecord](&parquetBuf)
	_, err := writer.Write(rows)
	if err != nil {
		t.Fatalf("failed to write parquet data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close parquet writer: %v", err)
	}

	// Compress with zstd
	var zstdBuf bytes.Buffer
	zstdWriter, err := zstd.NewWriter(&zstdBuf)
	if err != nil {
		t.Fatalf("failed to create zstd writer: %v", err)
	}
	if _, err := zstdWriter.Write(parquetBuf.Bytes()); err != nil {
		t.Fatalf("failed to write zstd data: %v", err)
	}
	if err := zstdWriter.Close(); err != nil {
		t.Fatalf("failed to close zstd writer: %v", err)
	}

	// Process the compressed Parquet
	processor := NewProcessor(FileTypeParquetZSTD)
	var records []TestRecord

	reader, result, err := processor.Process(bytes.NewReader(zstdBuf.Bytes()), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Verify row count
	if result.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", result.RowCount)
	}

	// Verify records were populated
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}

	// Verify trim preprocessing was applied
	if records[0].Name != "Alice" {
		t.Errorf("records[0].Name = %q, want %q (trim should be applied)", records[0].Name, "Alice")
	}

	// Verify Stream metadata
	stream, ok := reader.(Stream)
	if !ok {
		t.Fatal("returned reader does not implement Stream interface")
	}

	// Format() should return CSV for Parquet input
	if stream.Format() != FileTypeCSV {
		t.Errorf("Stream.Format() = %v, want %v", stream.Format(), FileTypeCSV)
	}

	// OriginalFormat() should return the compressed Parquet type
	if stream.OriginalFormat() != FileTypeParquetZSTD {
		t.Errorf("Stream.OriginalFormat() = %v, want %v", stream.OriginalFormat(), FileTypeParquetZSTD)
	}

	// Verify ProcessResult.OriginalFormat
	if result.OriginalFormat != FileTypeParquetZSTD {
		t.Errorf("ProcessResult.OriginalFormat = %v, want %v", result.OriginalFormat, FileTypeParquetZSTD)
	}

	// Verify columns
	expectedColumns := []string{"id", "name", "email"}
	if len(result.Columns) != len(expectedColumns) {
		t.Errorf("len(Columns) = %d, want %d", len(result.Columns), len(expectedColumns))
	}

	// Verify output is valid CSV
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (header + 2 data), got %d", len(lines))
	}

	// Verify content
	if !strings.Contains(lines[1], "Alice") {
		t.Errorf("output should contain trimmed name 'Alice': %s", lines[1])
	}
}

// TestDetectFileType_UppercaseCompressed verifies uppercase and mixed-case extensions are detected correctly
func TestDetectFileType_UppercaseCompressed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want FileType
	}{
		// Uppercase base + compression (compression ext is lowercase or uppercase only)
		{"CSV.GZ all caps", "DATA.CSV.GZ", FileTypeCSVGZ},
		{"Parquet.ZST all caps", "data.Parquet.ZST", FileTypeParquetZSTD},
		{"XLSX.BZ2 all caps", "Report.XLSX.BZ2", FileTypeXLSXBZ2},
		{"TSV.XZ all caps", "DATA.TSV.XZ", FileTypeTSVXZ},
		{"LTSV.zst lowercase compression", "Log.Ltsv.zst", FileTypeLTSVZSTD},

		// Verify case insensitive detection for base type
		{"parquet lowercase", "data.parquet", FileTypeParquet},
		{"PARQUET uppercase", "DATA.PARQUET", FileTypeParquet},
		{"ParQueT mixed", "Data.ParQueT", FileTypeParquet},

		// All compression variants (lowercase and uppercase only)
		{"csv.gz lowercase", "data.csv.gz", FileTypeCSVGZ},
		{"CSV.GZ uppercase", "DATA.CSV.GZ", FileTypeCSVGZ},
		{"tsv.bz2 lowercase", "data.tsv.bz2", FileTypeTSVBZ2},
		{"TSV.BZ2 uppercase", "DATA.TSV.BZ2", FileTypeTSVBZ2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DetectFileType(tt.path); got != tt.want {
				t.Errorf("DetectFileType(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestStreamSeekLenAfterProcess verifies Stream Seek/Len work correctly after Process
// Note: Seek/Len are available on the concrete *stream type via io.ReadSeeker cast
func TestStreamSeekLenAfterProcess(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string
		Email string
	}

	csvData := `name,email
Alice,alice@example.com
Bob,bob@example.com`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	reader, _, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Verify Stream interface
	stream, ok := reader.(Stream)
	if !ok {
		t.Fatal("returned reader does not implement Stream interface")
	}

	// Verify io.ReadSeeker interface for Seek capability
	seeker, ok := reader.(io.ReadSeeker)
	if !ok {
		t.Fatal("returned reader does not implement io.ReadSeeker interface")
	}

	// Read all data
	data1, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(data1) == 0 {
		t.Error("expected data from stream")
	}

	// Seek back to beginning
	pos, err := seeker.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek() error = %v", err)
	}
	if pos != 0 {
		t.Errorf("Seek() pos = %d, want 0", pos)
	}

	// Read again and verify same content
	data2, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("ReadAll() after Seek error = %v", err)
	}

	if string(data1) != string(data2) {
		t.Errorf("data after Seek differs: got %q, want %q", data2, data1)
	}
}

// TestDecompressionErrorPropagation verifies corrupted compressed data returns errors
func TestDecompressionErrorPropagation(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name string
	}

	tests := []struct {
		name     string
		fileType FileType
		data     []byte
	}{
		{
			name:     "corrupted gzip",
			fileType: FileTypeCSVGZ,
			data:     []byte{0x1f, 0x8b, 0x08, 0x00, 0xff, 0xff, 0xff}, // invalid gzip
		},
		{
			name:     "corrupted bzip2",
			fileType: FileTypeCSVBZ2,
			data:     []byte{0x42, 0x5a, 0x68, 0x39, 0xff, 0xff}, // invalid bzip2
		},
		{
			name:     "corrupted xz",
			fileType: FileTypeCSVXZ,
			data:     []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00, 0xff, 0xff}, // invalid xz
		},
		{
			name:     "corrupted zstd",
			fileType: FileTypeCSVZSTD,
			data:     []byte{0x28, 0xb5, 0x2f, 0xfd, 0xff, 0xff, 0xff}, // invalid zstd
		},
		{
			name:     "random bytes as gzip",
			fileType: FileTypeCSVGZ,
			data:     []byte("not compressed data at all"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			processor := NewProcessor(tt.fileType)
			var records []TestRecord

			_, _, err := processor.Process(bytes.NewReader(tt.data), &records)
			if err == nil {
				t.Error("expected error for corrupted compressed data, got nil")
			}
		})
	}
}

// TestExtraColumnsHandling verifies CSV with more columns than struct fields doesn't crash
func TestExtraColumnsHandling(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		First  string
		Second string
		// Only 2 fields, but CSV has 4 columns
	}

	csvData := `first,second,third,fourth
a,b,c,d
1,2,3,4`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should process without error
	if result.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", result.RowCount)
	}

	// Verify only first 2 fields are populated (positional binding)
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}

	if records[0].First != "a" {
		t.Errorf("records[0].First = %q, want %q", records[0].First, "a")
	}
	if records[0].Second != "b" {
		t.Errorf("records[0].Second = %q, want %q", records[0].Second, "b")
	}

	// Extra columns should not affect struct (no column shift)
	if records[1].First != "1" {
		t.Errorf("records[1].First = %q, want %q", records[1].First, "1")
	}
	if records[1].Second != "2" {
		t.Errorf("records[1].Second = %q, want %q", records[1].Second, "2")
	}
}

// TestUnexportedFieldIgnored verifies unexported fields are ignored during processing
func TestUnexportedFieldIgnored(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name     string
		internal string //nolint:unused // intentionally unexported for test
		Email    string
	}

	csvData := `name,internal,email
Alice,secret,alice@example.com`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should process without error
	if result.RowCount != 1 {
		t.Errorf("RowCount = %d, want 1", result.RowCount)
	}

	// Verify exported fields are populated
	if len(records) > 0 {
		if records[0].Name != "Alice" {
			t.Errorf("records[0].Name = %q, want %q", records[0].Name, "Alice")
		}
	}
}

// TestUnsupportedFieldTypeError verifies unsupported struct field types generate PrepErrors
func TestUnsupportedFieldTypeError(t *testing.T) {
	t.Parallel()

	type NestedStruct struct {
		Value string
	}

	type TestRecord struct {
		Name   string
		Nested NestedStruct // struct type is unsupported
	}

	csvData := `name,nested
Alice,value`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should have PrepError for unsupported type
	prepErrors := result.PrepErrors()
	if len(prepErrors) == 0 {
		t.Error("expected PrepError for unsupported struct field type")
	}

	// Verify error details
	if len(prepErrors) > 0 {
		pe := prepErrors[0]
		if pe.Tag != "type_conversion" {
			t.Errorf("PrepError.Tag = %q, want %q", pe.Tag, "type_conversion")
		}
	}
}

// TestWhitespaceOnlyRequiredValidation verifies trim+required catches whitespace-only values
func TestWhitespaceOnlyRequiredValidation(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name string `prep:"trim" validate:"required"`
	}

	tests := []struct {
		name      string
		csvData   string
		wantValid int
		wantError int
	}{
		{
			name: "spaces only fails required after trim",
			csvData: `name
   `,
			wantValid: 0,
			wantError: 1,
		},
		{
			name:      "tabs only fails required after trim",
			csvData:   "name\n\t\t\t",
			wantValid: 0,
			wantError: 1,
		},
		{
			name:      "mixed whitespace fails required after trim",
			csvData:   "name\n  \t  \t  ",
			wantValid: 0,
			wantError: 1,
		},
		{
			name: "valid value after trim passes",
			csvData: `name
  Alice  `,
			wantValid: 1,
			wantError: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			processor := NewProcessor(FileTypeCSV)
			var records []TestRecord

			_, result, err := processor.Process(strings.NewReader(tt.csvData), &records)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if result.ValidRowCount != tt.wantValid {
				t.Errorf("ValidRowCount = %d, want %d", result.ValidRowCount, tt.wantValid)
			}

			if len(result.ValidationErrors()) != tt.wantError {
				t.Errorf("len(ValidationErrors()) = %d, want %d", len(result.ValidationErrors()), tt.wantError)
			}
		})
	}
}

// TestFixSchemeAlreadyHasScheme verifies fix_scheme doesn't modify URLs that already have correct scheme
func TestFixSchemeAlreadyHasScheme(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		URL string `prep:"fix_scheme=https"`
	}

	csvData := `url
https://example.com
https://example.com/path?query=1
https://sub.example.com:8080/path`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, _, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// URLs should remain unchanged
	expected := []string{
		"https://example.com",
		"https://example.com/path?query=1",
		"https://sub.example.com:8080/path",
	}

	for i, want := range expected {
		if records[i].URL != want {
			t.Errorf("records[%d].URL = %q, want %q", i, records[i].URL, want)
		}
	}
}

// TestCoerceIntTruncatesFloat verifies coerce=int truncates floats correctly
func TestCoerceIntTruncatesFloat(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Value string `prep:"coerce=int"`
	}

	csvData := `value
123.9
123.1
123.5
-45.9
0.999`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, _, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Coerce should truncate towards zero
	expected := []string{"123", "123", "123", "-45", "0"}
	for i, want := range expected {
		if records[i].Value != want {
			t.Errorf("records[%d].Value = %q, want %q", i, records[i].Value, want)
		}
	}
}

// TestParseXLSXRowPadding verifies XLSX rows with fewer cells are padded to header length
func TestParseXLSXRowPadding(t *testing.T) {
	t.Parallel()

	// Use the existing sample.xlsx file
	file, err := os.Open(filepath.Join("testdata", "sample.xlsx"))
	if err != nil {
		t.Skipf("sample.xlsx not available: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	result, err := parseXLSX(data)
	if err != nil {
		t.Fatalf("parseXLSX() error = %v", err)
	}

	// All records should have same length as headers
	headerLen := len(result.headers)
	for i, record := range result.records {
		if len(record) != headerLen {
			t.Errorf("record[%d] len = %d, want %d (should be padded to header length)", i, len(record), headerLen)
		}
	}
}

// TestLTSVTrailingTabAndEmptyLines verifies LTSV handles trailing tabs and empty lines
func TestLTSVTrailingTabAndEmptyLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantRecords int
		wantHeaders int
	}{
		{
			name:        "trailing tab on line",
			input:       "name:Alice\temail:alice@example.com\t\nname:Bob\temail:bob@example.com",
			wantRecords: 2,
			wantHeaders: 2,
		},
		{
			name:        "empty lines between records",
			input:       "name:Alice\temail:alice@example.com\n\n\nname:Bob\temail:bob@example.com",
			wantRecords: 2,
			wantHeaders: 2,
		},
		{
			name:        "trailing newlines",
			input:       "name:Alice\temail:alice@example.com\nname:Bob\temail:bob@example.com\n\n\n",
			wantRecords: 2,
			wantHeaders: 2,
		},
		{
			name:        "mixed empty lines and trailing tabs",
			input:       "name:Alice\t\n\nname:Bob\temail:bob@example.com\t\n\n",
			wantRecords: 2,
			wantHeaders: 2,
		},
		{
			name:        "whitespace only lines",
			input:       "name:Alice\n   \nname:Bob",
			wantRecords: 2,
			wantHeaders: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := parseLTSV(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("parseLTSV() error = %v", err)
			}

			if len(result.records) != tt.wantRecords {
				t.Errorf("len(records) = %d, want %d", len(result.records), tt.wantRecords)
			}

			if len(result.headers) != tt.wantHeaders {
				t.Errorf("len(headers) = %d, want %d", len(result.headers), tt.wantHeaders)
			}
		})
	}
}

// TestLTSVMixedSchema verifies LTSV with different keys per line produces union of columns
func TestLTSVMixedSchema(t *testing.T) {
	t.Parallel()

	// Line 1 has {name, email}, Line 2 has {email, age} - union should be {name, email, age}
	ltsvData := "name:Alice\temail:alice@example.com\nemail:bob@example.com\tage:30"

	result, err := parseLTSV(strings.NewReader(ltsvData))
	if err != nil {
		t.Fatalf("parseLTSV() error = %v", err)
	}

	// Should have 2 records
	if len(result.records) != 2 {
		t.Errorf("len(records) = %d, want 2", len(result.records))
	}

	// Headers should be union of all keys (order: name, email, age - as encountered)
	if len(result.headers) != 3 {
		t.Errorf("len(headers) = %d, want 3", len(result.headers))
	}

	// Verify header order (first encountered order)
	expectedHeaders := []string{"name", "email", "age"}
	for i, want := range expectedHeaders {
		if i < len(result.headers) && result.headers[i] != want {
			t.Errorf("headers[%d] = %q, want %q", i, result.headers[i], want)
		}
	}

	// Verify first record: name=Alice, email=alice@example.com, age=""
	if len(result.records) > 0 {
		rec := result.records[0]
		if len(rec) != 3 {
			t.Errorf("records[0] len = %d, want 3", len(rec))
		}
		if len(rec) >= 3 {
			if rec[0] != "Alice" {
				t.Errorf("records[0][0] = %q, want %q", rec[0], "Alice")
			}
			if rec[1] != "alice@example.com" {
				t.Errorf("records[0][1] = %q, want %q", rec[1], "alice@example.com")
			}
			if rec[2] != "" {
				t.Errorf("records[0][2] = %q, want empty (missing key)", rec[2])
			}
		}
	}

	// Verify second record: name="", email=bob@example.com, age=30
	if len(result.records) > 1 {
		rec := result.records[1]
		if len(rec) >= 3 {
			if rec[0] != "" {
				t.Errorf("records[1][0] = %q, want empty (missing key)", rec[0])
			}
			if rec[1] != "bob@example.com" {
				t.Errorf("records[1][1] = %q, want %q", rec[1], "bob@example.com")
			}
			if rec[2] != "30" {
				t.Errorf("records[1][2] = %q, want %q", rec[2], "30")
			}
		}
	}
}

// TestTypeConversionErrorAggregation verifies PrepErrors are aggregated correctly for type conversion failures
func TestTypeConversionErrorAggregation(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string
		Age   int
		Score float64
	}

	// CSV with mixed valid/invalid data for numeric columns
	csvData := `name,age,score
Alice,30,95.5
Bob,invalid,88.0
Charlie,25,not_a_number
David,abc,xyz`

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should have 4 rows total
	if result.RowCount != 4 {
		t.Errorf("RowCount = %d, want 4", result.RowCount)
	}

	// Get PrepErrors for type conversion failures
	prepErrors := result.PrepErrors()

	// Row 1 (Alice): valid - no errors
	// Row 2 (Bob): Age is invalid - 1 error
	// Row 3 (Charlie): Score is invalid - 1 error
	// Row 4 (David): Both Age and Score invalid - 2 errors
	// Total: 4 PrepErrors expected
	if len(prepErrors) != 4 {
		t.Errorf("len(PrepErrors()) = %d, want 4", len(prepErrors))
	}

	// Verify error details - all should be type_conversion
	for i, pe := range prepErrors {
		if pe.Tag != "type_conversion" {
			t.Errorf("prepErrors[%d].Tag = %q, want %q", i, pe.Tag, "type_conversion")
		}
	}

	// ValidRowCount should only count fully valid rows
	if result.ValidRowCount != 1 {
		t.Errorf("ValidRowCount = %d, want 1 (only Alice is fully valid)", result.ValidRowCount)
	}

	// Verify first record (Alice) was populated correctly
	if len(records) > 0 {
		if records[0].Name != "Alice" {
			t.Errorf("records[0].Name = %q, want %q", records[0].Name, "Alice")
		}
		if records[0].Age != 30 {
			t.Errorf("records[0].Age = %d, want 30", records[0].Age)
		}
	}
}

// TestComprehensiveCompressionFormats verifies Format()/OriginalFormat() for all compression types
func TestComprehensiveCompressionFormats(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name string
	}

	csvContent := "Name\nAlice\n"

	tests := []struct {
		name         string
		fileType     FileType
		wantFormat   FileType
		wantOriginal FileType
		compressFunc func([]byte) ([]byte, error)
	}{
		{
			name:         "CSV uncompressed",
			fileType:     FileTypeCSV,
			wantFormat:   FileTypeCSV,
			wantOriginal: FileTypeCSV,
			compressFunc: nil,
		},
		{
			name:         "TSV uncompressed",
			fileType:     FileTypeTSV,
			wantFormat:   FileTypeTSV, // TSV outputs as TSV
			wantOriginal: FileTypeTSV,
			compressFunc: nil,
		},
		{
			name:         "LTSV uncompressed",
			fileType:     FileTypeLTSV,
			wantFormat:   FileTypeLTSV,
			wantOriginal: FileTypeLTSV,
			compressFunc: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var input []byte
			switch tt.fileType {
			case FileTypeLTSV:
				input = []byte("Name:Alice\n")
			case FileTypeTSV:
				input = []byte("Name\nAlice\n")
			default:
				input = []byte(csvContent)
			}

			if tt.compressFunc != nil {
				var err error
				input, err = tt.compressFunc(input)
				if err != nil {
					t.Fatalf("compression failed: %v", err)
				}
			}

			processor := NewProcessor(tt.fileType)
			var records []TestRecord

			reader, result, err := processor.Process(bytes.NewReader(input), &records)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			stream, ok := reader.(Stream)
			if !ok {
				t.Fatal("returned reader does not implement Stream interface")
			}

			if stream.Format() != tt.wantFormat {
				t.Errorf("Stream.Format() = %v, want %v", stream.Format(), tt.wantFormat)
			}

			if stream.OriginalFormat() != tt.wantOriginal {
				t.Errorf("Stream.OriginalFormat() = %v, want %v", stream.OriginalFormat(), tt.wantOriginal)
			}

			if result.OriginalFormat != tt.wantOriginal {
				t.Errorf("ProcessResult.OriginalFormat = %v, want %v", result.OriginalFormat, tt.wantOriginal)
			}
		})
	}
}

// BenchmarkLargeCSVProcessing benchmarks processing of large CSV files
func BenchmarkLargeCSVProcessing(b *testing.B) {
	type TestRecord struct {
		Name  string `prep:"trim"`
		Email string `prep:"lowercase"`
		Age   string
	}

	// Generate large CSV data (10000 rows)
	var csvBuilder strings.Builder
	csvBuilder.WriteString("name,email,age\n")
	for i := range 10000 {
		csvBuilder.WriteString("  John Doe  ,JOHN@EXAMPLE.COM,")
		csvBuilder.WriteString(strconv.Itoa(20 + i%50))
		csvBuilder.WriteString("\n")
	}
	csvData := csvBuilder.String()

	b.ResetTimer()
	for range b.N {
		processor := NewProcessor(FileTypeCSV)
		var records []TestRecord
		_, _, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			b.Fatalf("Process() error = %v", err)
		}
	}
}

// BenchmarkLargeCSVWithValidation benchmarks processing with validation
func BenchmarkLargeCSVWithValidation(b *testing.B) {
	type TestRecord struct {
		Name  string `prep:"trim" validate:"required"`
		Email string `prep:"lowercase" validate:"email"`
		Age   string `validate:"number"`
	}

	// Generate large CSV data (10000 rows)
	var csvBuilder strings.Builder
	csvBuilder.WriteString("name,email,age\n")
	for i := range 10000 {
		csvBuilder.WriteString("  John Doe  ,john@example.com,")
		csvBuilder.WriteString(strconv.Itoa(20 + i%50))
		csvBuilder.WriteString("\n")
	}
	csvData := csvBuilder.String()

	b.ResetTimer()
	for range b.N {
		processor := NewProcessor(FileTypeCSV)
		var records []TestRecord
		_, _, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			b.Fatalf("Process() error = %v", err)
		}
	}
}

// TestLargeInputCompletion verifies large input processing completes without timeout
func TestLargeInputCompletion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large input test in short mode")
	}

	t.Parallel()

	type TestRecord struct {
		Name  string `prep:"trim"`
		Email string `prep:"lowercase"`
		Age   string
	}

	// Generate large CSV data (50000 rows)
	var csvBuilder strings.Builder
	csvBuilder.WriteString("name,email,age\n")
	for i := range 50000 {
		csvBuilder.WriteString("  John Doe  ,JOHN@EXAMPLE.COM,")
		csvBuilder.WriteString(strconv.Itoa(20 + i%50))
		csvBuilder.WriteString("\n")
	}
	csvData := csvBuilder.String()

	processor := NewProcessor(FileTypeCSV)
	var records []TestRecord

	_, result, err := processor.Process(strings.NewReader(csvData), &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Verify all rows were processed
	if result.RowCount != 50000 {
		t.Errorf("RowCount = %d, want 50000", result.RowCount)
	}

	// Verify records were populated
	if len(records) != 50000 {
		t.Errorf("len(records) = %d, want 50000", len(records))
	}

	// Verify preprocessing was applied
	if len(records) > 0 {
		if records[0].Name != "John Doe" {
			t.Errorf("records[0].Name = %q, want %q (trim should be applied)", records[0].Name, "John Doe")
		}
		if records[0].Email != "john@example.com" {
			t.Errorf("records[0].Email = %q, want %q (lowercase should be applied)", records[0].Email, "john@example.com")
		}
	}

	t.Logf("Successfully processed %d rows", result.RowCount)
}

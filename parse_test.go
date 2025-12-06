package fileprep

import (
	"bytes"
	"errors"
	"strings"
	"testing"

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

package fileprep

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegration_AllPrepTags tests all prep tags in an integrated manner
func TestIntegration_AllPrepTags(t *testing.T) {
	t.Parallel()

	// Test struct using all prep tags
	type TestRecord struct {
		// Basic preprocessors
		TrimField      string `prep:"trim"`
		LtrimField     string `prep:"ltrim"`
		RtrimField     string `prep:"rtrim"`
		LowercaseField string `prep:"lowercase"`
		UppercaseField string `prep:"uppercase"`
		DefaultField   string `prep:"default=default_value"`

		// String transformation preprocessors
		ReplaceField       string `prep:"replace=foo:bar"`
		PrefixField        string `prep:"prefix=pre_"`
		SuffixField        string `prep:"suffix=_suf"`
		TruncateField      string `prep:"truncate=5"`
		StripHTMLField     string `prep:"strip_html"`
		StripNewlineField  string `prep:"strip_newline"`
		CollapseSpaceField string `prep:"collapse_space"`

		// Character filtering preprocessors
		RemoveDigitsField string `prep:"remove_digits"`
		RemoveAlphaField  string `prep:"remove_alpha"`
		KeepDigitsField   string `prep:"keep_digits"`
		KeepAlphaField    string `prep:"keep_alpha"`
		TrimSetField      string `prep:"trim_set=[]"`

		// Padding preprocessors
		PadLeftField  string `prep:"pad_left=5:0"`
		PadRightField string `prep:"pad_right=5:0"`

		// Advanced preprocessors
		NormalizeUnicodeField string `prep:"normalize_unicode"`
		NullifyField          string `prep:"nullify=NA"`
		CoerceIntField        string `prep:"coerce=int"`
		CoerceBoolField       string `prep:"coerce=bool"`
		FixSchemeField        string `prep:"fix_scheme=https"`
		RegexReplaceField     string `prep:"regex_replace=\\d+:X"`
	}

	// Create test CSV data
	// Note: StripNewlineField uses quoted string with embedded newline
	csvData := "TrimField,LtrimField,RtrimField,LowercaseField,UppercaseField,DefaultField,ReplaceField,PrefixField,SuffixField,TruncateField,StripHTMLField,StripNewlineField,CollapseSpaceField,RemoveDigitsField,RemoveAlphaField,KeepDigitsField,KeepAlphaField,TrimSetField,PadLeftField,PadRightField,NormalizeUnicodeField,NullifyField,CoerceIntField,CoerceBoolField,FixSchemeField,RegexReplaceField\n" +
		"  hello  ,  hello  ,  hello  ,HELLO,hello,,foo test,value,value,hello world,<p>text</p>,\"line1\nline2\",a  b  c,abc123,abc123,abc123,abc123,[value],42,42,hello,NA,123.5,yes,example.com,abc123"

	reader := strings.NewReader(csvData)
	var records []TestRecord

	processor := NewProcessor(FileTypeCSV)
	pipeReader, result, err := processor.Process(reader, &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Consume the pipe reader
	go func() {
		_, _ = io.Copy(io.Discard, pipeReader) //nolint:errcheck // discarding output in test
	}()

	if result.HasErrors() {
		for _, e := range result.Errors {
			t.Logf("Error: %v", e)
		}
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	r := records[0]

	// Verify basic preprocessors
	if r.TrimField != "hello" {
		t.Errorf("TrimField = %q, want %q", r.TrimField, "hello")
	}
	if r.LtrimField != "hello  " {
		t.Errorf("LtrimField = %q, want %q", r.LtrimField, "hello  ")
	}
	if r.RtrimField != "  hello" {
		t.Errorf("RtrimField = %q, want %q", r.RtrimField, "  hello")
	}
	if r.LowercaseField != "hello" {
		t.Errorf("LowercaseField = %q, want %q", r.LowercaseField, "hello")
	}
	if r.UppercaseField != "HELLO" {
		t.Errorf("UppercaseField = %q, want %q", r.UppercaseField, "HELLO")
	}
	if r.DefaultField != "default_value" {
		t.Errorf("DefaultField = %q, want %q", r.DefaultField, "default_value")
	}

	// Verify string transformation preprocessors
	if r.ReplaceField != "bar test" {
		t.Errorf("ReplaceField = %q, want %q", r.ReplaceField, "bar test")
	}
	if r.PrefixField != "pre_value" {
		t.Errorf("PrefixField = %q, want %q", r.PrefixField, "pre_value")
	}
	if r.SuffixField != "value_suf" {
		t.Errorf("SuffixField = %q, want %q", r.SuffixField, "value_suf")
	}
	if r.TruncateField != "hello" {
		t.Errorf("TruncateField = %q, want %q", r.TruncateField, "hello")
	}
	if r.StripHTMLField != "text" {
		t.Errorf("StripHTMLField = %q, want %q", r.StripHTMLField, "text")
	}
	if r.StripNewlineField != "line1line2" {
		t.Errorf("StripNewlineField = %q, want %q", r.StripNewlineField, "line1line2")
	}
	if r.CollapseSpaceField != "a b c" {
		t.Errorf("CollapseSpaceField = %q, want %q", r.CollapseSpaceField, "a b c")
	}

	// Verify character filtering preprocessors
	if r.RemoveDigitsField != "abc" {
		t.Errorf("RemoveDigitsField = %q, want %q", r.RemoveDigitsField, "abc")
	}
	if r.RemoveAlphaField != "123" {
		t.Errorf("RemoveAlphaField = %q, want %q", r.RemoveAlphaField, "123")
	}
	if r.KeepDigitsField != "123" {
		t.Errorf("KeepDigitsField = %q, want %q", r.KeepDigitsField, "123")
	}
	if r.KeepAlphaField != "abc" {
		t.Errorf("KeepAlphaField = %q, want %q", r.KeepAlphaField, "abc")
	}
	if r.TrimSetField != "value" {
		t.Errorf("TrimSetField = %q, want %q", r.TrimSetField, "value")
	}

	// Verify padding preprocessors
	if r.PadLeftField != "00042" {
		t.Errorf("PadLeftField = %q, want %q", r.PadLeftField, "00042")
	}
	if r.PadRightField != "42000" {
		t.Errorf("PadRightField = %q, want %q", r.PadRightField, "42000")
	}

	// Verify advanced preprocessors
	if r.NormalizeUnicodeField != "hello" {
		t.Errorf("NormalizeUnicodeField = %q, want %q", r.NormalizeUnicodeField, "hello")
	}
	if r.NullifyField != "" {
		t.Errorf("NullifyField = %q, want %q", r.NullifyField, "")
	}
	if r.CoerceIntField != "123" {
		t.Errorf("CoerceIntField = %q, want %q", r.CoerceIntField, "123")
	}
	if r.CoerceBoolField != "true" {
		t.Errorf("CoerceBoolField = %q, want %q", r.CoerceBoolField, "true")
	}
	if r.FixSchemeField != "https://example.com" {
		t.Errorf("FixSchemeField = %q, want %q", r.FixSchemeField, "https://example.com")
	}
	if r.RegexReplaceField != "abcX" {
		t.Errorf("RegexReplaceField = %q, want %q", r.RegexReplaceField, "abcX")
	}
}

// TestIntegration_CombinedPrepTags tests multiple prep tags on a single field
func TestIntegration_CombinedPrepTags(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Combined string `prep:"trim,lowercase,prefix=pre_"`
	}

	csvData := `Combined
  HELLO WORLD  `

	reader := strings.NewReader(csvData)
	var records []TestRecord

	processor := NewProcessor(FileTypeCSV)
	pipeReader, result, err := processor.Process(reader, &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	go func() {
		_, _ = io.Copy(io.Discard, pipeReader) //nolint:errcheck // discarding output in test
	}()

	if result.HasErrors() {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	expected := "pre_hello world"
	if records[0].Combined != expected {
		t.Errorf("Combined = %q, want %q", records[0].Combined, expected)
	}
}

// TestIntegration_PrepWithValidation tests prep tags working together with validation
func TestIntegration_PrepWithValidation(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Email string `prep:"trim,lowercase" validate:"email"`
	}

	csvData := `Email
  JOHN@EXAMPLE.COM
invalid_email`

	reader := strings.NewReader(csvData)
	var records []TestRecord

	processor := NewProcessor(FileTypeCSV)
	pipeReader, result, err := processor.Process(reader, &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	go func() {
		_, _ = io.Copy(io.Discard, pipeReader) //nolint:errcheck // discarding output in test
	}()

	// Should have validation error for second row
	if !result.HasErrors() {
		t.Error("expected validation errors")
	}

	if len(result.ValidationErrors()) != 1 {
		t.Errorf("expected 1 validation error, got %d", len(result.ValidationErrors()))
	}

	// First record should be processed correctly
	if len(records) < 1 {
		t.Fatal("expected at least 1 record")
	}

	expected := "john@example.com"
	if records[0].Email != expected {
		t.Errorf("Email = %q, want %q", records[0].Email, expected)
	}
}

// TestIntegration_CompressedCSV tests processing compressed CSV files
func TestIntegration_CompressedCSV(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string `prep:"trim"`
		Email string `prep:"trim,lowercase"`
		Age   string
	}

	tests := []struct {
		name     string
		filePath string
		fileType FileType
	}{
		{"gzip CSV", filepath.Join("testdata", "sample.csv.gz"), FileTypeCSVGZ},
		{"bzip2 CSV", filepath.Join("testdata", "sample.csv.bz2"), FileTypeCSVBZ2},
		{"xz CSV", filepath.Join("testdata", "sample.csv.xz"), FileTypeCSVXZ},
		{"zstd CSV", filepath.Join("testdata", "sample.csv.zst"), FileTypeCSVZSTD},
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
			pipeReader, result, err := processor.Process(file, &records)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			go func() {
				_, _ = io.Copy(io.Discard, pipeReader) //nolint:errcheck // discarding output in test
			}()

			if result.OriginalFormat != tt.fileType {
				t.Errorf("OriginalFormat = %v, want %v", result.OriginalFormat, tt.fileType)
			}

			// Verify at least one record was processed
			if len(records) == 0 {
				t.Error("expected at least one record")
			}

			// Verify first record has trimmed name
			if records[0].Name != "John Doe" {
				t.Errorf("Name = %q, want %q", records[0].Name, "John Doe")
			}
		})
	}
}

// TestIntegration_TSVProcessing tests TSV file processing with prep tags
func TestIntegration_TSVProcessing(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string `prep:"trim"`
		Email string `prep:"trim,lowercase"`
		Age   string
	}

	file, err := os.Open(filepath.Join("testdata", "sample.tsv"))
	if err != nil {
		t.Fatalf("os.Open() error = %v", err)
	}
	defer file.Close()

	var records []TestRecord

	processor := NewProcessor(FileTypeTSV)
	pipeReader, result, err := processor.Process(file, &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	go func() {
		_, _ = io.Copy(io.Discard, pipeReader) //nolint:errcheck // discarding output in test
	}()

	if result.OriginalFormat != FileTypeTSV {
		t.Errorf("OriginalFormat = %v, want %v", result.OriginalFormat, FileTypeTSV)
	}

	if len(records) == 0 {
		t.Error("expected at least one record")
	}
}

// TestIntegration_LTSVProcessing tests LTSV file processing with prep tags
func TestIntegration_LTSVProcessing(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string `prep:"trim"`
		Email string `prep:"trim,lowercase"`
		Age   string
	}

	file, err := os.Open(filepath.Join("testdata", "sample.ltsv"))
	if err != nil {
		t.Fatalf("os.Open() error = %v", err)
	}
	defer file.Close()

	var records []TestRecord

	processor := NewProcessor(FileTypeLTSV)
	pipeReader, result, err := processor.Process(file, &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	go func() {
		_, _ = io.Copy(io.Discard, pipeReader) //nolint:errcheck // discarding output in test
	}()

	if result.OriginalFormat != FileTypeLTSV {
		t.Errorf("OriginalFormat = %v, want %v", result.OriginalFormat, FileTypeLTSV)
	}

	if len(records) == 0 {
		t.Error("expected at least one record")
	}
}

// TestIntegration_ErrorReporting tests detailed error reporting
func TestIntegration_ErrorReporting(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string `validate:"required"`
		Email string `validate:"email"`
		Age   string `validate:"numeric"`
	}

	csvData := `Name,Email,Age
,invalid-email,abc
John,john@example.com,25`

	reader := strings.NewReader(csvData)
	var records []TestRecord

	processor := NewProcessor(FileTypeCSV)
	pipeReader, result, err := processor.Process(reader, &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	go func() {
		_, _ = io.Copy(io.Discard, pipeReader) //nolint:errcheck // discarding output in test
	}()

	// Should have 3 validation errors for the first row
	if !result.HasErrors() {
		t.Error("expected validation errors")
	}

	errors := result.ValidationErrors()
	if len(errors) != 3 {
		t.Errorf("expected 3 validation errors, got %d", len(errors))
	}

	// Check that error details are correct
	for _, e := range errors {
		if e.Row != 1 {
			t.Errorf("expected row 1, got %d", e.Row)
		}
	}

	// Verify row counts
	if result.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", result.RowCount)
	}
	if result.ValidRowCount != 1 {
		t.Errorf("ValidRowCount = %d, want 1", result.ValidRowCount)
	}
}

// TestIntegration_CrossFieldValidation tests cross-field validation
func TestIntegration_CrossFieldValidation(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Password        string
		ConfirmPassword string `validate:"eqfield=Password"`
	}

	csvData := `Password,ConfirmPassword
secret123,secret123
password,different`

	reader := strings.NewReader(csvData)
	var records []TestRecord

	processor := NewProcessor(FileTypeCSV)
	pipeReader, result, err := processor.Process(reader, &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	go func() {
		_, _ = io.Copy(io.Discard, pipeReader) //nolint:errcheck // discarding output in test
	}()

	// Should have 1 validation error for the second row
	if !result.HasErrors() {
		t.Error("expected validation errors")
	}

	errors := result.ValidationErrors()
	if len(errors) != 1 {
		t.Errorf("expected 1 validation error, got %d", len(errors))
	}

	if len(errors) > 0 && errors[0].Row != 2 {
		t.Errorf("expected error on row 2, got row %d", errors[0].Row)
	}
}

// TestIntegration_StreamOutput tests that the output reader contains preprocessed data
func TestIntegration_StreamOutput(t *testing.T) {
	t.Parallel()

	type TestRecord struct {
		Name  string `prep:"uppercase"`
		Email string `prep:"lowercase"`
	}

	csvData := `Name,Email
john,JOHN@EXAMPLE.COM`

	reader := strings.NewReader(csvData)
	var records []TestRecord

	processor := NewProcessor(FileTypeCSV)
	pipeReader, _, err := processor.Process(reader, &records)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Read the output stream
	output, err := io.ReadAll(pipeReader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	outputStr := string(output)

	// Verify the output contains preprocessed values
	if !strings.Contains(outputStr, "JOHN") {
		t.Error("expected uppercase JOHN in output")
	}
	if !strings.Contains(outputStr, "john@example.com") {
		t.Error("expected lowercase email in output")
	}
}

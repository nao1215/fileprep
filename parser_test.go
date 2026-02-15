package fileprep

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParsePrepTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tag     string
		wantLen int
		wantErr bool
	}{
		{"empty tag", "", 0, false},
		{"single trim", "trim", 1, false},
		{"multiple preps", "trim,lowercase", 2, false},
		{"with default", "trim,default=N/A", 2, false},
		{"unknown tag", "unknown", 0, true},
		{"spaces in tag", " trim , lowercase ", 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			preps, err := parsePrepTag(tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePrepTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(preps) != tt.wantLen {
				t.Errorf("parsePrepTag() len = %d, want %d", len(preps), tt.wantLen)
			}
		})
	}
}

func TestParseValidateTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tag     string
		wantLen int
		wantErr bool
	}{
		{"empty tag", "", 0, false},
		{"required", "required", 1, false},
		{"unknown tag returns error", "unknown", 0, true},
		{"spaces in tag", " required ", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vals, _, err := parseValidateTag(tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseValidateTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(vals) != tt.wantLen {
				t.Errorf("parseValidateTag() len = %d, want %d", len(vals), tt.wantLen)
			}
		})
	}
}

func TestGetStructType(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		Name string
	}

	var nilSlicePtr *[]TestStruct

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{"valid slice pointer", &[]TestStruct{}, false},
		{"non-pointer", []TestStruct{}, true},
		{"pointer to non-slice", &TestStruct{}, true},
		{"pointer to slice of non-struct", &[]string{}, true},
		{"nil interface", nil, true},
		{"nil typed pointer", nilSlicePtr, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := getStructType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("getStructType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParsePrepTagInvalidFormats tests that invalid tag formats are handled gracefully
func TestParsePrepTagInvalidFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tag        string
		wantLen    int
		wantErr    bool
		errContain string
	}{
		// Invalid pad_left format - non-numeric value should be skipped (length=0)
		{"pad_left non-numeric", "pad_left=abc", 0, false, ""},
		{"pad_left negative", "pad_left=-5", 0, false, ""},
		{"pad_left empty", "pad_left=", 0, false, ""},
		{"pad_left valid", "pad_left=5:0", 1, false, ""},

		// Invalid regex_replace format - bad regex should be skipped
		{"regex_replace bad pattern", "regex_replace=bad[:X", 0, false, ""},
		{"regex_replace no colon", "regex_replace=pattern", 0, false, ""},
		{"regex_replace valid", "regex_replace=\\d+:X", 1, false, ""},

		// Invalid coerce format - wrong type should be skipped
		{"coerce invalid type", "coerce=string", 0, false, ""},
		{"coerce valid int", "coerce=int", 1, false, ""},
		{"coerce valid float", "coerce=float", 1, false, ""},
		{"coerce valid bool", "coerce=bool", 1, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			preps, err := parsePrepTag(tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePrepTag(%q) error = %v, wantErr %v", tt.tag, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContain != "" {
				if err == nil || !containsString(err.Error(), tt.errContain) {
					t.Errorf("parsePrepTag(%q) error should contain %q, got %v", tt.tag, tt.errContain, err)
				}
			}
			if !tt.wantErr && len(preps) != tt.wantLen {
				t.Errorf("parsePrepTag(%q) len = %d, want %d", tt.tag, len(preps), tt.wantLen)
			}
		})
	}
}

// containsString is a helper for checking error messages
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestParseStructTypeWithEmbeddedStruct tests how embedded (anonymous) struct fields are handled
func TestParseStructTypeWithEmbeddedStruct(t *testing.T) {
	t.Parallel()

	type Embedded struct {
		EmbeddedField string `prep:"trim"`
	}

	type TestStruct struct {
		Embedded            // Anonymous embedded field
		RegularField string `prep:"lowercase"`
	}

	structType := reflect.TypeOf(TestStruct{})
	info, err := parseStructType(structType)
	if err != nil {
		t.Fatalf("parseStructType() error = %v", err)
	}

	// Embedded struct should be treated as a single field (Embedded type)
	// Regular exported fields should also be parsed
	if len(info.Fields) < 1 {
		t.Errorf("parseStructType() fields = %d, want at least 1", len(info.Fields))
	}

	// Check that RegularField is parsed correctly
	foundRegular := false
	for _, field := range info.Fields {
		if field.Name == "RegularField" {
			foundRegular = true
			if len(field.Preprocessors) != 1 {
				t.Errorf("RegularField.Preprocessors len = %d, want 1", len(field.Preprocessors))
			}
		}
	}
	if !foundRegular {
		t.Error("RegularField not found in parsed fields")
	}
}

func TestParseStructType(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		Name    string `prep:"trim" validate:"required"`
		Email   string `prep:"trim,lowercase"`
		Age     int
		private string //nolint:unused // intentionally unexported for testing
	}

	structType := reflect.TypeOf(TestStruct{})
	info, err := parseStructType(structType)
	if err != nil {
		t.Fatalf("parseStructType() error = %v", err)
	}

	// Should have 3 fields (private is skipped)
	if len(info.Fields) != 3 {
		t.Errorf("parseStructType() fields = %d, want 3", len(info.Fields))
	}

	// Check first field
	if len(info.Fields) > 0 {
		field := info.Fields[0]
		if field.Name != "Name" {
			t.Errorf("Field[0].Name = %q, want %q", field.Name, "Name")
		}
		if len(field.Preprocessors) != 1 {
			t.Errorf("Field[0].Preprocessors len = %d, want 1", len(field.Preprocessors))
		}
		if len(field.Validators) != 1 {
			t.Errorf("Field[0].Validators len = %d, want 1", len(field.Validators))
		}
	}

	// Check second field
	if len(info.Fields) > 1 {
		field := info.Fields[1]
		if field.Name != "Email" {
			t.Errorf("Field[1].Name = %q, want %q", field.Name, "Email")
		}
		if len(field.Preprocessors) != 2 {
			t.Errorf("Field[1].Preprocessors len = %d, want 2", len(field.Preprocessors))
		}
	}
}

// TestParseStructTypeUnknownValidateTag tests that unknown validate tags propagate
// through parseStructType with the field name included in the error message.
func TestParseStructTypeUnknownValidateTag(t *testing.T) {
	t.Parallel()

	type BadValidate struct {
		Email string `validate:"unknown_tag"`
	}

	structType := reflect.TypeOf(BadValidate{})
	_, err := parseStructType(structType)
	if err == nil {
		t.Fatal("parseStructType() expected error for unknown validate tag, got nil")
	}
	if !errors.Is(err, ErrInvalidTagFormat) {
		t.Errorf("parseStructType() error should wrap ErrInvalidTagFormat, got %v", err)
	}
	if !strings.Contains(err.Error(), "Email") {
		t.Errorf("parseStructType() error should contain field name \"Email\", got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "unknown_tag") {
		t.Errorf("parseStructType() error should contain tag name \"unknown_tag\", got %q", err.Error())
	}
}

// TestParseStructTypeUnknownPrepTag tests that unknown prep tags propagate
// through parseStructType with the field name included in the error message.
func TestParseStructTypeUnknownPrepTag(t *testing.T) {
	t.Parallel()

	type BadPrep struct {
		Name string `prep:"bad_preprocessor"`
	}

	structType := reflect.TypeOf(BadPrep{})
	_, err := parseStructType(structType)
	if err == nil {
		t.Fatal("parseStructType() expected error for unknown prep tag, got nil")
	}
	if !errors.Is(err, ErrInvalidTagFormat) {
		t.Errorf("parseStructType() error should wrap ErrInvalidTagFormat, got %v", err)
	}
	if !strings.Contains(err.Error(), "Name") {
		t.Errorf("parseStructType() error should contain field name \"Name\", got %q", err.Error())
	}
}

// TestToSnakeCase tests the snake_case conversion function
func TestToSnakeCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"name", "name"},
		{"Name", "name"},
		{"UserName", "user_name"},
		{"ID", "id"},
		{"UserID", "user_id"},
		{"HTTPServer", "http_server"},
		{"XMLParser", "xml_parser"},
		{"getHTTPResponse", "get_http_response"},
		{"already_snake_case", "already_snake_case"},
		{"A", "a"},
		{"ABC", "abc"},
		{"ABCdef", "ab_cdef"},
		{"abcDEF", "abc_def"},
		{"IOReader", "io_reader"},
		{"myURLParser", "my_url_parser"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := toSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestColumnNameFromNameTag tests that name tag overrides auto-generated column name
func TestColumnNameFromNameTag(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		UserName    string `name:"user_name"`
		EmailAddr   string `name:"email_address"`
		Age         int    // No name tag - should use "age" (snake_case of "Age")
		HTTPStatus  string // No name tag - should use "http_status"
		CustomField string `name:"custom_col"`
	}

	structType := reflect.TypeOf(TestStruct{})
	info, err := parseStructType(structType)
	if err != nil {
		t.Fatalf("parseStructType() error = %v", err)
	}

	expected := map[string]string{
		"UserName":    "user_name",
		"EmailAddr":   "email_address",
		"Age":         "age",
		"HTTPStatus":  "http_status",
		"CustomField": "custom_col",
	}

	for _, field := range info.Fields {
		want, ok := expected[field.Name]
		if !ok {
			continue
		}
		if field.ColumnName != want {
			t.Errorf("Field %q.ColumnName = %q, want %q", field.Name, field.ColumnName, want)
		}
	}
}

// TestAutoSnakeCaseColumnNames tests automatic snake_case conversion for column names
func TestAutoSnakeCaseColumnNames(t *testing.T) {
	t.Parallel()

	type TestStruct struct {
		FirstName   string
		LastName    string
		EmailAddr   string
		PhoneNumber string
		ID          string
		UserID      string
		HTTPCode    string
		XMLData     string
	}

	structType := reflect.TypeOf(TestStruct{})
	info, err := parseStructType(structType)
	if err != nil {
		t.Fatalf("parseStructType() error = %v", err)
	}

	expected := map[string]string{
		"FirstName":   "first_name",
		"LastName":    "last_name",
		"EmailAddr":   "email_addr",
		"PhoneNumber": "phone_number",
		"ID":          "id",
		"UserID":      "user_id",
		"HTTPCode":    "http_code",
		"XMLData":     "xml_data",
	}

	for _, field := range info.Fields {
		want, ok := expected[field.Name]
		if !ok {
			continue
		}
		if field.ColumnName != want {
			t.Errorf("Field %q.ColumnName = %q, want %q", field.Name, field.ColumnName, want)
		}
	}
}

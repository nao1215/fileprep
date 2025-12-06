package fileprep

import (
	"reflect"
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
	}{
		{"empty tag", "", 0},
		{"required", "required", 1},
		{"unknown tag ignored", "unknown", 0}, // Unknown tags are ignored for now
		{"spaces in tag", " required ", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vals, _ := parseValidateTag(tt.tag)
			if len(vals) != tt.wantLen {
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

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{"valid slice pointer", &[]TestStruct{}, false},
		{"non-pointer", []TestStruct{}, true},
		{"pointer to non-slice", &TestStruct{}, true},
		{"pointer to slice of non-struct", &[]string{}, true},
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

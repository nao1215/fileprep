package fileprep

import (
	"strings"
	"testing"
)

func TestEqFieldValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		srcValue    string
		targetValue string
		targetField string
		wantErr     bool
	}{
		{
			name:        "equal strings pass",
			srcValue:    "hello",
			targetValue: "hello",
			targetField: "Other",
			wantErr:     false,
		},
		{
			name:        "different strings fail",
			srcValue:    "hello",
			targetValue: "world",
			targetField: "Other",
			wantErr:     true,
		},
		{
			name:        "empty strings pass",
			srcValue:    "",
			targetValue: "",
			targetField: "Other",
			wantErr:     false,
		},
		{
			name:        "equal numbers as strings pass",
			srcValue:    "123",
			targetValue: "123",
			targetField: "Other",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := newEqFieldValidator(tt.targetField)
			got := v.Validate(tt.srcValue, tt.targetValue)
			if (got != "") != tt.wantErr {
				t.Errorf("eqFieldValidator.Validate() = %q, wantErr %v", got, tt.wantErr)
			}
			if v.Name() != eqFieldTagValue {
				t.Errorf("eqFieldValidator.Name() = %q, want %q", v.Name(), eqFieldTagValue)
			}
			if v.TargetField() != tt.targetField {
				t.Errorf("eqFieldValidator.TargetField() = %q, want %q", v.TargetField(), tt.targetField)
			}
		})
	}
}

func TestNeFieldValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		srcValue    string
		targetValue string
		targetField string
		wantErr     bool
	}{
		{
			name:        "different strings pass",
			srcValue:    "hello",
			targetValue: "world",
			targetField: "Other",
			wantErr:     false,
		},
		{
			name:        "equal strings fail",
			srcValue:    "hello",
			targetValue: "hello",
			targetField: "Other",
			wantErr:     true,
		},
		{
			name:        "empty vs non-empty pass",
			srcValue:    "",
			targetValue: "something",
			targetField: "Other",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := newNeFieldValidator(tt.targetField)
			got := v.Validate(tt.srcValue, tt.targetValue)
			if (got != "") != tt.wantErr {
				t.Errorf("neFieldValidator.Validate() = %q, wantErr %v", got, tt.wantErr)
			}
			if v.Name() != neFieldTagValue {
				t.Errorf("neFieldValidator.Name() = %q, want %q", v.Name(), neFieldTagValue)
			}
		})
	}
}

func TestGtFieldValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		srcValue    string
		targetValue string
		targetField string
		wantErr     bool
	}{
		{
			name:        "numeric greater than passes",
			srcValue:    "100",
			targetValue: "50",
			targetField: "Min",
			wantErr:     false,
		},
		{
			name:        "numeric equal fails",
			srcValue:    "50",
			targetValue: "50",
			targetField: "Min",
			wantErr:     true,
		},
		{
			name:        "numeric less than fails",
			srcValue:    "25",
			targetValue: "50",
			targetField: "Min",
			wantErr:     true,
		},
		{
			name:        "string comparison greater passes",
			srcValue:    "b",
			targetValue: "a",
			targetField: "Min",
			wantErr:     false,
		},
		{
			name:        "string comparison equal fails",
			srcValue:    "a",
			targetValue: "a",
			targetField: "Min",
			wantErr:     true,
		},
		{
			name:        "float greater than passes",
			srcValue:    "10.5",
			targetValue: "10.4",
			targetField: "Min",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := newGtFieldValidator(tt.targetField)
			got := v.Validate(tt.srcValue, tt.targetValue)
			if (got != "") != tt.wantErr {
				t.Errorf("gtFieldValidator.Validate() = %q, wantErr %v", got, tt.wantErr)
			}
			if v.Name() != gtFieldTagValue {
				t.Errorf("gtFieldValidator.Name() = %q, want %q", v.Name(), gtFieldTagValue)
			}
		})
	}
}

func TestGteFieldValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		srcValue    string
		targetValue string
		targetField string
		wantErr     bool
	}{
		{
			name:        "numeric greater than passes",
			srcValue:    "100",
			targetValue: "50",
			targetField: "Min",
			wantErr:     false,
		},
		{
			name:        "numeric equal passes",
			srcValue:    "50",
			targetValue: "50",
			targetField: "Min",
			wantErr:     false,
		},
		{
			name:        "numeric less than fails",
			srcValue:    "25",
			targetValue: "50",
			targetField: "Min",
			wantErr:     true,
		},
		{
			name:        "float equal passes",
			srcValue:    "10.5",
			targetValue: "10.5",
			targetField: "Min",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := newGteFieldValidator(tt.targetField)
			got := v.Validate(tt.srcValue, tt.targetValue)
			if (got != "") != tt.wantErr {
				t.Errorf("gteFieldValidator.Validate() = %q, wantErr %v", got, tt.wantErr)
			}
			if v.Name() != gteFieldTagValue {
				t.Errorf("gteFieldValidator.Name() = %q, want %q", v.Name(), gteFieldTagValue)
			}
		})
	}
}

func TestLtFieldValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		srcValue    string
		targetValue string
		targetField string
		wantErr     bool
	}{
		{
			name:        "numeric less than passes",
			srcValue:    "25",
			targetValue: "50",
			targetField: "Max",
			wantErr:     false,
		},
		{
			name:        "numeric equal fails",
			srcValue:    "50",
			targetValue: "50",
			targetField: "Max",
			wantErr:     true,
		},
		{
			name:        "numeric greater than fails",
			srcValue:    "100",
			targetValue: "50",
			targetField: "Max",
			wantErr:     true,
		},
		{
			name:        "string comparison less passes",
			srcValue:    "a",
			targetValue: "b",
			targetField: "Max",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := newLtFieldValidator(tt.targetField)
			got := v.Validate(tt.srcValue, tt.targetValue)
			if (got != "") != tt.wantErr {
				t.Errorf("ltFieldValidator.Validate() = %q, wantErr %v", got, tt.wantErr)
			}
			if v.Name() != ltFieldTagValue {
				t.Errorf("ltFieldValidator.Name() = %q, want %q", v.Name(), ltFieldTagValue)
			}
		})
	}
}

func TestLteFieldValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		srcValue    string
		targetValue string
		targetField string
		wantErr     bool
	}{
		{
			name:        "numeric less than passes",
			srcValue:    "25",
			targetValue: "50",
			targetField: "Max",
			wantErr:     false,
		},
		{
			name:        "numeric equal passes",
			srcValue:    "50",
			targetValue: "50",
			targetField: "Max",
			wantErr:     false,
		},
		{
			name:        "numeric greater than fails",
			srcValue:    "100",
			targetValue: "50",
			targetField: "Max",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := newLteFieldValidator(tt.targetField)
			got := v.Validate(tt.srcValue, tt.targetValue)
			if (got != "") != tt.wantErr {
				t.Errorf("lteFieldValidator.Validate() = %q, wantErr %v", got, tt.wantErr)
			}
			if v.Name() != lteFieldTagValue {
				t.Errorf("lteFieldValidator.Name() = %q, want %q", v.Name(), lteFieldTagValue)
			}
		})
	}
}

func TestFieldContainsValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		srcValue    string
		targetValue string
		targetField string
		wantErr     bool
	}{
		{
			name:        "contains substring passes",
			srcValue:    "hello world",
			targetValue: "world",
			targetField: "Substr",
			wantErr:     false,
		},
		{
			name:        "does not contain fails",
			srcValue:    "hello world",
			targetValue: "foo",
			targetField: "Substr",
			wantErr:     true,
		},
		{
			name:        "empty target always passes",
			srcValue:    "hello",
			targetValue: "",
			targetField: "Substr",
			wantErr:     false,
		},
		{
			name:        "exact match passes",
			srcValue:    "hello",
			targetValue: "hello",
			targetField: "Substr",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := newFieldContainsValidator(tt.targetField)
			got := v.Validate(tt.srcValue, tt.targetValue)
			if (got != "") != tt.wantErr {
				t.Errorf("fieldContainsValidator.Validate() = %q, wantErr %v", got, tt.wantErr)
			}
			if v.Name() != fieldContainsTagValue {
				t.Errorf("fieldContainsValidator.Name() = %q, want %q", v.Name(), fieldContainsTagValue)
			}
		})
	}
}

func TestFieldExcludesValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		srcValue    string
		targetValue string
		targetField string
		wantErr     bool
	}{
		{
			name:        "does not contain passes",
			srcValue:    "hello world",
			targetValue: "foo",
			targetField: "Forbidden",
			wantErr:     false,
		},
		{
			name:        "contains substring fails",
			srcValue:    "hello world",
			targetValue: "world",
			targetField: "Forbidden",
			wantErr:     true,
		},
		{
			name:        "empty target always fails (empty string is contained)",
			srcValue:    "hello",
			targetValue: "",
			targetField: "Forbidden",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := newFieldExcludesValidator(tt.targetField)
			got := v.Validate(tt.srcValue, tt.targetValue)
			if (got != "") != tt.wantErr {
				t.Errorf("fieldExcludesValidator.Validate() = %q, wantErr %v", got, tt.wantErr)
			}
			if v.Name() != fieldExcludesTagValue {
				t.Errorf("fieldExcludesValidator.Name() = %q, want %q", v.Name(), fieldExcludesTagValue)
			}
		})
	}
}

func TestCrossFieldValidation_Integration(t *testing.T) {
	t.Parallel()

	// Test parsing cross-field validators
	t.Run("parse cross-field validators", func(t *testing.T) {
		t.Parallel()
		vals, crossVals := parseValidateTag("gtfield=MaxPrice")
		if len(vals) != 0 {
			t.Errorf("expected 0 validators, got %d", len(vals))
		}
		if len(crossVals) != 1 {
			t.Errorf("expected 1 cross-field validator, got %d", len(crossVals))
		}
		if len(crossVals) > 0 {
			if crossVals[0].Name() != gtFieldTagValue {
				t.Errorf("expected validator name %q, got %q", gtFieldTagValue, crossVals[0].Name())
			}
			if crossVals[0].TargetField() != "MaxPrice" {
				t.Errorf("expected target field %q, got %q", "MaxPrice", crossVals[0].TargetField())
			}
		}
	})

	// Test multiple cross-field validators
	t.Run("parse multiple cross-field validators", func(t *testing.T) {
		t.Parallel()
		vals, crossVals := parseValidateTag("required,eqfield=Other,nefield=Another")
		if len(vals) != 1 {
			t.Errorf("expected 1 validator, got %d", len(vals))
		}
		if len(crossVals) != 2 {
			t.Errorf("expected 2 cross-field validators, got %d", len(crossVals))
		}
	})

	// Test all cross-field validator types are parsed
	t.Run("parse all cross-field validator types", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			tag      string
			expected string
		}{
			{"eqfield=X", eqFieldTagValue},
			{"nefield=X", neFieldTagValue},
			{"gtfield=X", gtFieldTagValue},
			{"gtefield=X", gteFieldTagValue},
			{"ltfield=X", ltFieldTagValue},
			{"ltefield=X", lteFieldTagValue},
			{"fieldcontains=X", fieldContainsTagValue},
			{"fieldexcludes=X", fieldExcludesTagValue},
		}

		for _, tc := range testCases {
			_, crossVals := parseValidateTag(tc.tag)
			if len(crossVals) != 1 {
				t.Errorf("tag %q: expected 1 cross-field validator, got %d", tc.tag, len(crossVals))
				continue
			}
			if crossVals[0].Name() != tc.expected {
				t.Errorf("tag %q: expected validator name %q, got %q", tc.tag, tc.expected, crossVals[0].Name())
			}
		}
	})
}

func TestCrossFieldValidation_Processor(t *testing.T) {
	t.Parallel()

	type DateRange struct {
		StartDate string `validate:"ltfield=EndDate"`
		EndDate   string
	}

	t.Run("cross-field validation passes when condition met", func(t *testing.T) {
		t.Parallel()
		csvData := "start_date,end_date\n2024-01-01,2024-12-31\n"
		var records []DateRange

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		if len(result.Errors) != 0 {
			t.Errorf("expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
		}
		if result.ValidRowCount != 1 {
			t.Errorf("expected 1 valid row, got %d", result.ValidRowCount)
		}
	})

	t.Run("cross-field validation fails when condition not met", func(t *testing.T) {
		t.Parallel()
		// StartDate should be less than EndDate, but here StartDate > EndDate
		csvData := "start_date,end_date\n2024-12-31,2024-01-01\n"
		var records []DateRange

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		if len(result.Errors) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(result.Errors), result.Errors)
		}
		if result.ValidRowCount != 0 {
			t.Errorf("expected 0 valid rows, got %d", result.ValidRowCount)
		}
	})

	type Password struct {
		Password        string `validate:"eqfield=ConfirmPassword"`
		ConfirmPassword string
	}

	t.Run("password confirmation validation", func(t *testing.T) {
		t.Parallel()
		csvData := "password,confirm_password\nsecret123,secret123\n"
		var records []Password

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		if len(result.Errors) != 0 {
			t.Errorf("expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
		}
	})

	t.Run("password confirmation mismatch", func(t *testing.T) {
		t.Parallel()
		csvData := "password,confirm_password\nsecret123,different\n"
		var records []Password

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		if len(result.Errors) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(result.Errors), result.Errors)
		}
	})

	type NumericRange struct {
		Min string `validate:"ltefield=Max"`
		Max string `validate:"gtefield=Min"`
	}

	t.Run("numeric range validation with both directions", func(t *testing.T) {
		t.Parallel()
		csvData := "min,max\n10,100\n"
		var records []NumericRange

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		if len(result.Errors) != 0 {
			t.Errorf("expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
		}
	})

	type InvalidTarget struct {
		Value string `validate:"eqfield=NonExistent"`
	}

	t.Run("cross-field validation with non-existent target field", func(t *testing.T) {
		t.Parallel()
		csvData := "value\ntest\n"
		var records []InvalidTarget

		processor := NewProcessor(FileTypeCSV)
		_, result, err := processor.Process(strings.NewReader(csvData), &records)
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}

		if len(result.Errors) != 1 {
			t.Errorf("expected 1 error for non-existent field, got %d: %v", len(result.Errors), result.Errors)
		}
	})
}

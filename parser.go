package fileprep

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// fieldInfo contains parsed information about a struct field
type fieldInfo struct {
	Name                 string               // Struct field name
	ColumnName           string               // Expected CSV column name (from name tag or auto-converted)
	Index                int                  // Field index in struct
	ColumnIndex          int                  // Column index in CSV (resolved at runtime, -1 if not found)
	Preprocessors        preprocessors        // Preprocessing rules
	Validators           validators           // Validation rules
	CrossFieldValidators crossFieldValidators // Cross-field validation rules
}

// structInfo contains parsed information about a struct type
type structInfo struct {
	Fields []fieldInfo
}

// parseStructType parses struct tags from a struct type and returns field information
func parseStructType(structType reflect.Type) (*structInfo, error) {
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: expected struct, got %s", ErrStructSlicePointer, structType.Kind())
	}

	fieldCount := structType.NumField()
	fields := make([]fieldInfo, 0, fieldCount)

	for i := range fieldCount {
		field := structType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Determine column name: use name tag if present, otherwise convert field name to snake_case
		columnName := field.Tag.Get(nameTagName)
		if columnName == "" {
			columnName = toSnakeCase(field.Name)
		}

		info := fieldInfo{
			Name:        field.Name,
			ColumnName:  columnName,
			Index:       i,
			ColumnIndex: -1, // Will be resolved at runtime
		}

		// Parse prep tag
		if prepTag := field.Tag.Get(prepTagName); prepTag != "" {
			preps, err := parsePrepTag(prepTag)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", field.Name, err)
			}
			info.Preprocessors = preps
		}

		// Parse validate tag
		if validateTag := field.Tag.Get(validateTagName); validateTag != "" {
			info.Validators, info.CrossFieldValidators = parseValidateTag(validateTag)
		}

		fields = append(fields, info)
	}

	return &structInfo{Fields: fields}, nil
}

// parsePrepTag parses the prep tag string and returns preprocessors
func parsePrepTag(tag string) (preprocessors, error) {
	if tag == "" {
		return nil, nil
	}

	parts := strings.Split(tag, ",")
	preps := make(preprocessors, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle preprocessors with parameters (key=value format)
		key, value := splitTagKeyValue(part)

		switch key {
		// Basic preprocessors
		case trimTagValue:
			preps = append(preps, newTrimPreprocessor())
		case ltrimTagValue:
			preps = append(preps, newLtrimPreprocessor())
		case rtrimTagValue:
			preps = append(preps, newRtrimPreprocessor())
		case lowercaseTagValue:
			preps = append(preps, newLowercasePreprocessor())
		case uppercaseTagValue:
			preps = append(preps, newUppercasePreprocessor())
		case defaultTagValue:
			preps = append(preps, newDefaultPreprocessor(value))

		// String transformation preprocessors
		case replaceTagValue:
			// replace=old:new format
			oldStr, newStr, found := parseColonSeparatedValue(value)
			if found {
				preps = append(preps, newReplacePreprocessor(oldStr, newStr))
			}
		case prefixTagValue:
			if value != "" {
				preps = append(preps, newPrefixPreprocessor(value))
			}
		case suffixTagValue:
			if value != "" {
				preps = append(preps, newSuffixPreprocessor(value))
			}
		case truncateTagValue:
			if n, err := strconv.Atoi(value); err == nil && n > 0 {
				preps = append(preps, newTruncatePreprocessor(n))
			}
		case stripHTMLTagValue:
			preps = append(preps, newStripHTMLPreprocessor())
		case stripNewlineTagValue:
			preps = append(preps, newStripNewlinePreprocessor())
		case collapseSpaceTagValue:
			preps = append(preps, newCollapseSpacePreprocessor())

		// Character filtering preprocessors
		case removeDigitsTagValue:
			preps = append(preps, newRemoveDigitsPreprocessor())
		case removeAlphaTagValue:
			preps = append(preps, newRemoveAlphaPreprocessor())
		case keepDigitsTagValue:
			preps = append(preps, newKeepDigitsPreprocessor())
		case keepAlphaTagValue:
			preps = append(preps, newKeepAlphaPreprocessor())
		case trimSetTagValue:
			if value != "" {
				preps = append(preps, newTrimSetPreprocessor(value))
			}

		// Padding preprocessors
		case padLeftTagValue:
			// pad_left=N,char format
			length, padChar := parsePadParams(value)
			if length > 0 {
				preps = append(preps, newPadLeftPreprocessor(length, padChar))
			}
		case padRightTagValue:
			// pad_right=N,char format
			length, padChar := parsePadParams(value)
			if length > 0 {
				preps = append(preps, newPadRightPreprocessor(length, padChar))
			}

		// Advanced preprocessors
		case normalizeUnicodeTagValue:
			preps = append(preps, newNormalizeUnicodePreprocessor())
		case nullifyTagValue:
			if value != "" {
				preps = append(preps, newNullifyPreprocessor(value))
			}
		case coerceTagValue:
			if value == "int" || value == "float" || value == "bool" {
				preps = append(preps, newCoercePreprocessor(value))
			}
		case fixSchemeTagValue:
			if value != "" {
				preps = append(preps, newFixSchemePreprocessor(value))
			}
		case regexReplaceTagValue:
			// regex_replace=pattern:replacement format
			pattern, replacement, found := parseColonSeparatedValue(value)
			if found {
				if p := newRegexReplacePreprocessor(pattern, replacement); p != nil {
					preps = append(preps, p)
				}
			}

		default:
			return nil, fmt.Errorf("%w: unknown prep tag %q", ErrInvalidTagFormat, part)
		}
	}

	return preps, nil
}

// parseColonSeparatedValue parses "old:new" format values
// Returns old, new, and true if the format is valid
func parseColonSeparatedValue(value string) (string, string, bool) {
	idx := strings.Index(value, ":")
	if idx < 0 {
		return "", "", false
	}
	return value[:idx], value[idx+1:], true
}

// parsePadParams parses "N:char" format for padding preprocessors
// Returns length and pad character (defaults to space if not specified)
func parsePadParams(value string) (int, rune) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) == 0 {
		return 0, ' '
	}

	length, err := strconv.Atoi(parts[0])
	if err != nil || length <= 0 {
		return 0, ' '
	}

	padChar := ' '
	if len(parts) == 2 && len(parts[1]) > 0 {
		runes := []rune(parts[1])
		padChar = runes[0]
	}

	return length, padChar
}

// parseRequiredIfParams parses "FieldName value" format for required_if/required_unless
// Returns field name and expected/except value
func parseRequiredIfParams(value string) (string, string) {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) == 0 {
		return "", ""
	}
	field := parts[0]
	expectedVal := ""
	if len(parts) == 2 {
		expectedVal = parts[1]
	}
	return field, expectedVal
}

// parseValidateTag parses the validate tag string and returns validators and cross-field validators
func parseValidateTag(tag string) (validators, crossFieldValidators) {
	if tag == "" {
		return nil, nil
	}

	parts := strings.Split(tag, ",")
	vals := make(validators, 0, len(parts))
	crossVals := make(crossFieldValidators, 0)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle validators with parameters (key=value format)
		key, value := splitTagKeyValue(part)

		switch key {
		// Basic validators
		case requiredTagValue:
			vals = append(vals, newRequiredValidator())
		case booleanTagValue:
			vals = append(vals, newBooleanValidator())
		case alphaTagValue:
			vals = append(vals, newAlphaValidator())
		case alphaSpaceTagValue:
			vals = append(vals, newAlphaSpaceValidator())
		case alphaUnicodeTagValue:
			vals = append(vals, newAlphaUnicodeValidator())
		case numericTagValue:
			vals = append(vals, newNumericValidator())
		case numberTagValue:
			vals = append(vals, newNumberValidator())
		case alphanumericTagValue:
			vals = append(vals, newAlphanumericValidator())
		case alphanumericUnicodeTagValue:
			vals = append(vals, newAlphanumericUnicodeValidator())

		// Comparison validators (with threshold)
		case equalTagValue:
			if threshold, err := strconv.ParseFloat(value, 64); err == nil {
				vals = append(vals, newEqualValidator(threshold))
			}
		case notEqualTagValue:
			if threshold, err := strconv.ParseFloat(value, 64); err == nil {
				vals = append(vals, newNotEqualValidator(threshold))
			}
		case greaterThanTagValue:
			if threshold, err := strconv.ParseFloat(value, 64); err == nil {
				vals = append(vals, newGreaterThanValidator(threshold))
			}
		case greaterThanEqualTagValue:
			if threshold, err := strconv.ParseFloat(value, 64); err == nil {
				vals = append(vals, newGreaterThanEqualValidator(threshold))
			}
		case lessThanTagValue:
			if threshold, err := strconv.ParseFloat(value, 64); err == nil {
				vals = append(vals, newLessThanValidator(threshold))
			}
		case lessThanEqualTagValue:
			if threshold, err := strconv.ParseFloat(value, 64); err == nil {
				vals = append(vals, newLessThanEqualValidator(threshold))
			}
		case minTagValue:
			if threshold, err := strconv.ParseFloat(value, 64); err == nil {
				vals = append(vals, newMinValidator(threshold))
			}
		case maxTagValue:
			if threshold, err := strconv.ParseFloat(value, 64); err == nil {
				vals = append(vals, newMaxValidator(threshold))
			}
		case lengthTagValue:
			if length, err := strconv.Atoi(value); err == nil {
				vals = append(vals, newLengthValidator(length))
			}

		// String validators
		case oneOfTagValue:
			// oneof values are space-separated
			if value != "" {
				allowed := strings.Fields(value)
				vals = append(vals, newOneOfValidator(allowed))
			}
		case lowercaseValidatorTagValue:
			vals = append(vals, newLowercaseValidator())
		case uppercaseValidatorTagValue:
			vals = append(vals, newUppercaseValidator())
		case asciiTagValue:
			vals = append(vals, newASCIIValidator())
		case printASCIITagValue:
			vals = append(vals, newPrintASCIIValidator())

		// Format validators
		case emailTagValue:
			vals = append(vals, newEmailValidator())
		case uriTagValue:
			vals = append(vals, newURIValidator())
		case urlTagValue:
			vals = append(vals, newURLValidator())
		case httpURLTagValue:
			vals = append(vals, newHTTPURLValidator())
		case httpsURLTagValue:
			vals = append(vals, newHTTPSURLValidator())
		case urlEncodedTagValue:
			vals = append(vals, newURLEncodedValidator())
		case dataURITagValue:
			vals = append(vals, newDataURIValidator())

		// Network validators
		case ipAddrTagValue:
			vals = append(vals, newIPAddrValidator())
		case ip4AddrTagValue:
			vals = append(vals, newIP4AddrValidator())
		case ip6AddrTagValue:
			vals = append(vals, newIP6AddrValidator())
		case cidrTagValue:
			vals = append(vals, newCIDRValidator())
		case cidrv4TagValue:
			vals = append(vals, newCIDRv4Validator())
		case cidrv6TagValue:
			vals = append(vals, newCIDRv6Validator())

		// Identifier validators
		case uuidTagValue:
			vals = append(vals, newUUIDValidator())
		case fqdnTagValue:
			vals = append(vals, newFQDNValidator())
		case hostnameTagValue:
			vals = append(vals, newHostnameValidator())
		case hostnameRFC1123TagValue:
			vals = append(vals, newHostnameRFC1123Validator())
		case hostnamePortTagValue:
			vals = append(vals, newHostnamePortValidator())

		// String content validators (with parameter)
		case startsWithTagValue:
			if value != "" {
				vals = append(vals, newStartsWithValidator(value))
			}
		case startsNotWithTagValue:
			if value != "" {
				vals = append(vals, newStartsNotWithValidator(value))
			}
		case endsWithTagValue:
			if value != "" {
				vals = append(vals, newEndsWithValidator(value))
			}
		case endsNotWithTagValue:
			if value != "" {
				vals = append(vals, newEndsNotWithValidator(value))
			}
		case containsTagValue:
			if value != "" {
				vals = append(vals, newContainsValidator(value))
			}
		case containsAnyTagValue:
			// containsany values are space-separated
			if value != "" {
				substrs := strings.Fields(value)
				vals = append(vals, newContainsAnyValidator(substrs))
			}
		case containsRuneTagValue:
			if value != "" {
				runes := []rune(value)
				if len(runes) > 0 {
					vals = append(vals, newContainsRuneValidator(runes[0]))
				}
			}

		// Exclusion validators (with parameter)
		case excludesTagValue:
			if value != "" {
				vals = append(vals, newExcludesValidator(value))
			}
		case excludesAllTagValue:
			if value != "" {
				vals = append(vals, newExcludesAllValidator(value))
			}
		case excludesRuneTagValue:
			if value != "" {
				runes := []rune(value)
				if len(runes) > 0 {
					vals = append(vals, newExcludesRuneValidator(runes[0]))
				}
			}

		// Misc validators
		case multibyteTagValue:
			vals = append(vals, newMultibyteValidator())
		case equalIgnoreCaseTagValue:
			if value != "" {
				vals = append(vals, newEqualIgnoreCaseValidator(value))
			}
		case notEqualIgnoreCaseTagValue:
			if value != "" {
				vals = append(vals, newNotEqualIgnoreCaseValidator(value))
			}

		// Datetime validator
		case datetimeTagValue:
			if value != "" {
				vals = append(vals, newDatetimeValidator(value))
			}

		// Phone number validator
		case e164TagValue:
			vals = append(vals, newE164Validator())

		// Geolocation validators
		case latitudeTagValue:
			vals = append(vals, newLatitudeValidator())
		case longitudeTagValue:
			vals = append(vals, newLongitudeValidator())

		// UUID variant validators
		case uuid3TagValue:
			vals = append(vals, newUUID3Validator())
		case uuid4TagValue:
			vals = append(vals, newUUID4Validator())
		case uuid5TagValue:
			vals = append(vals, newUUID5Validator())
		case ulidTagValue:
			vals = append(vals, newULIDValidator())

		// Hexadecimal and color validators
		case hexadecimalTagValue:
			vals = append(vals, newHexadecimalValidator())
		case hexColorTagValue:
			vals = append(vals, newHexColorValidator())
		case rgbTagValue:
			vals = append(vals, newRGBValidator())
		case rgbaTagValue:
			vals = append(vals, newRGBAValidator())
		case hslTagValue:
			vals = append(vals, newHSLValidator())
		case hslaTagValue:
			vals = append(vals, newHSLAValidator())

		// Network validators
		case macTagValue:
			vals = append(vals, newMACValidator())

		// Cross-field validators
		case eqFieldTagValue:
			if value != "" {
				crossVals = append(crossVals, newEqFieldValidator(value))
			}
		case neFieldTagValue:
			if value != "" {
				crossVals = append(crossVals, newNeFieldValidator(value))
			}
		case gtFieldTagValue:
			if value != "" {
				crossVals = append(crossVals, newGtFieldValidator(value))
			}
		case gteFieldTagValue:
			if value != "" {
				crossVals = append(crossVals, newGteFieldValidator(value))
			}
		case ltFieldTagValue:
			if value != "" {
				crossVals = append(crossVals, newLtFieldValidator(value))
			}
		case lteFieldTagValue:
			if value != "" {
				crossVals = append(crossVals, newLteFieldValidator(value))
			}
		case fieldContainsTagValue:
			if value != "" {
				crossVals = append(crossVals, newFieldContainsValidator(value))
			}
		case fieldExcludesTagValue:
			if value != "" {
				crossVals = append(crossVals, newFieldExcludesValidator(value))
			}

		// Conditional required validators (cross-field)
		case requiredIfTagValue:
			// Format: required_if=FieldName value
			if value != "" {
				field, expectedVal := parseRequiredIfParams(value)
				if field != "" {
					crossVals = append(crossVals, newRequiredIfValidator(field, expectedVal))
				}
			}
		case requiredUnlessTagValue:
			// Format: required_unless=FieldName value
			if value != "" {
				field, exceptVal := parseRequiredIfParams(value)
				if field != "" {
					crossVals = append(crossVals, newRequiredUnlessValidator(field, exceptVal))
				}
			}
		case requiredWithTagValue:
			// Format: required_with=FieldName
			if value != "" {
				crossVals = append(crossVals, newRequiredWithValidator(value))
			}
		case requiredWithoutTagValue:
			// Format: required_without=FieldName
			if value != "" {
				crossVals = append(crossVals, newRequiredWithoutValidator(value))
			}

		default:
			// Ignore unknown validators to allow gradual implementation
		}
	}

	return vals, crossVals
}

// splitTagKeyValue splits a tag part into key and value
// For "key=value" returns ("key", "value")
// For "key" returns ("key", "")
func splitTagKeyValue(part string) (string, string) {
	if idx := strings.Index(part, "="); idx > 0 {
		return part[:idx], part[idx+1:]
	}
	return part, ""
}

// toSnakeCase converts a CamelCase or PascalCase string to snake_case.
// Examples: "UserName" -> "user_name", "ID" -> "id", "HTTPServer" -> "http_server"
func toSnakeCase(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	result.Grow(len(s) + 5) // Pre-allocate with some extra space for underscores

	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			// Insert underscore before uppercase letter (except at the beginning)
			if i > 0 {
				// Check if previous char was lowercase or if next char is lowercase (for acronyms)
				prev := s[i-1]
				isAfterLower := prev >= 'a' && prev <= 'z'
				isBeforeLower := i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z'
				if isAfterLower || isBeforeLower {
					result.WriteByte('_')
				}
			}
			result.WriteByte(byte(r - 'A' + 'a')) // Convert to lowercase
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// getStructType extracts the struct type from a pointer to a slice of structs
func getStructType(structSlicePointer any) (reflect.Type, error) {
	if structSlicePointer == nil {
		return nil, fmt.Errorf("%w: nil pointer provided", ErrStructSlicePointer)
	}

	rv := reflect.ValueOf(structSlicePointer)
	if rv.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("%w: expected pointer, got %s", ErrStructSlicePointer, rv.Kind())
	}

	if rv.IsNil() {
		return nil, fmt.Errorf("%w: nil pointer provided", ErrStructSlicePointer)
	}

	elem := rv.Elem()
	switch elem.Kind() {
	case reflect.Slice, reflect.Array:
		elemType := elem.Type().Elem()
		if elemType.Kind() != reflect.Struct {
			return nil, fmt.Errorf("%w: expected slice of structs, got slice of %s", ErrStructSlicePointer, elemType.Kind())
		}
		return elemType, nil
	default:
		return nil, fmt.Errorf("%w: expected slice or array, got %s", ErrStructSlicePointer, elem.Kind())
	}
}

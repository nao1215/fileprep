package fileprep

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/nao1215/fileparser"
)

// Processor handles preprocessing and validation of file data
type Processor struct {
	fileType         fileparser.FileType
	strictTagParsing bool
	validRowsOnly    bool
}

// Option configures a Processor.
type Option func(*Processor)

// WithStrictTagParsing enables strict tag parsing mode.
// When enabled, invalid tag arguments (e.g., "eq=abc" where a number is expected)
// return an error during Process() instead of being silently ignored.
//
// Example:
//
//	processor := fileprep.NewProcessor(fileparser.CSV, fileprep.WithStrictTagParsing())
func WithStrictTagParsing() Option {
	return func(p *Processor) {
		p.strictTagParsing = true
	}
}

// WithValidRowsOnly configures the Processor to include only valid rows
// in the output io.Reader and struct slice. Rows that fail validation are
// excluded from the output but still counted in ProcessResult.RowCount
// and reported in ProcessResult.Errors.
//
// Example:
//
//	processor := fileprep.NewProcessor(fileparser.CSV, fileprep.WithValidRowsOnly())
//	reader, result, err := processor.Process(input, &records)
//	// reader contains only rows that passed all validations
//	// result.RowCount includes all rows, result.ValidRowCount has valid count
func WithValidRowsOnly() Option {
	return func(p *Processor) {
		p.validRowsOnly = true
	}
}

// NewProcessor creates a new Processor for the specified file type.
// Options can be provided to configure behavior such as strict tag parsing
// and output filtering.
//
// Example:
//
//	processor := fileprep.NewProcessor(fileparser.CSV)
//	var records []MyRecord
//	reader, result, err := processor.Process(input, &records)
//
//	// With options:
//	processor := fileprep.NewProcessor(fileparser.CSV,
//	    fileprep.WithStrictTagParsing(),
//	    fileprep.WithValidRowsOnly(),
//	)
func NewProcessor(fileType fileparser.FileType, opts ...Option) *Processor {
	p := &Processor{
		fileType: fileType,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Process reads from the input reader, applies preprocessing and validation,
// populates the struct slice, and returns an io.Reader with preprocessed data.
//
// The returned io.Reader preserves the original file format:
//   - CSV input → CSV output
//   - TSV input → TSV output (tab-delimited)
//   - LTSV input → LTSV output (label:value format)
//   - JSON input → JSONL output (one JSON value per line)
//   - JSONL input → JSONL output (one JSON value per line)
//   - XLSX input → CSV output (tabular data)
//   - Parquet input → CSV output (tabular data)
//
// The returned io.Reader can be passed directly to filesql.AddReader:
//
//	reader, result, err := processor.Process(input, &records)
//	db.AddReader(reader, "table", parser.CSV)
//
// For format information, use ProcessResult.OriginalFormat or cast to Stream:
//
//	stream := reader.(fileprep.Stream)
//	fmt.Println(stream.Format()) // CSV, TSV, etc.
//
// Example:
//
//	type User struct {
//	    Name  string `prep:"trim" validate:"required"`
//	    Email string `prep:"trim,lowercase" validate:"email"`
//	    Age   string `validate:"numeric,min=0,max=150"`
//	}
//
//	csvData := "name,email,age\n  John  ,JOHN@EXAMPLE.COM,30\n"
//	processor := fileprep.NewProcessor(parser.CSV)
//	var users []User
//	reader, result, err := processor.Process(strings.NewReader(csvData), &users)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Processed %d rows, %d valid\n", result.RowCount, result.ValidRowCount)
func (p *Processor) Process(input io.Reader, structSlicePointer any) (io.Reader, *ProcessResult, error) {
	// Get struct type and parse tags
	structType, err := getStructType(structSlicePointer)
	if err != nil {
		return nil, nil, err
	}

	structInfo, err := parseStructType(structType, p.strictTagParsing)
	if err != nil {
		return nil, nil, err
	}

	// Parse the file using fileparser
	tableData, err := fileparser.Parse(input, p.fileType)
	if err != nil {
		return nil, nil, err
	}

	headers := tableData.Headers
	records := tableData.Records

	// Build header name to column index map (first occurrence wins for duplicates)
	headerToColIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		if _, exists := headerToColIdx[h]; !exists {
			headerToColIdx[h] = i
		}
	}

	// Resolve column indices for each field based on column name
	for i := range structInfo.Fields {
		fi := &structInfo.Fields[i]
		if colIdx, ok := headerToColIdx[fi.ColumnName]; ok {
			fi.ColumnIndex = colIdx
		}
		// If not found, ColumnIndex remains -1
	}

	// Process records: apply preprocessing and validation
	// Pre-allocate errors slice with estimated capacity (assume ~10% error rate)
	estimatedErrors := max(len(records)/10, 16)
	result := &ProcessResult{
		Columns:        headers,
		OriginalFormat: p.fileType,
		Errors:         make([]error, 0, estimatedErrors),
	}
	structSliceValue := reflect.ValueOf(structSlicePointer).Elem()

	// Pre-allocate the struct slice to avoid repeated growth
	if structSliceValue.Cap() < len(records) {
		newSlice := reflect.MakeSlice(structSliceValue.Type(), 0, len(records))
		structSliceValue.Set(newSlice)
	}

	// Build field name to column index map for cross-field validation
	fieldNameToColIdx := make(map[string]int)
	for _, fi := range structInfo.Fields {
		fieldNameToColIdx[fi.Name] = fi.ColumnIndex
	}

	headerLen := len(headers)
	baseType := fileparser.BaseFileType(p.fileType)
	isJSONFormat := baseType == fileparser.JSON || baseType == fileparser.JSONL

	// jsonDataColumn is the column name used by fileparser for JSON/JSONL data.
	// Each JSON element is stored as a raw JSON string in this single column.
	const jsonDataColumn = "data"

	// When validRowsOnly is enabled, collect only valid records for output
	var validRecords [][]string
	if p.validRowsOnly {
		validRecords = make([][]string, 0, len(records))
	}

	// Process records in-place to avoid unnecessary allocations
	for rowIdx := range records {
		record := records[rowIdx]
		rowNum := rowIdx + 1 // 1-based row number (excluding header)
		result.RowCount++

		// Pad short rows with empty strings only if needed
		if len(record) < headerLen {
			padded := make([]string, headerLen)
			copy(padded, record)
			records[rowIdx] = padded
			record = padded
		}

		structValue := reflect.New(structType).Elem()

		// First pass: preprocessing and single-field validation
		rowHasError, err := p.processRow(record, rowNum, structInfo, structValue, result, isJSONFormat, jsonDataColumn)
		if err != nil {
			return nil, nil, err
		}

		// Second pass: cross-field validation
		if p.applyCrossFieldValidation(record, rowNum, structInfo, fieldNameToColIdx, result) {
			rowHasError = true
		}

		if !rowHasError {
			result.ValidRowCount++
			if p.validRowsOnly {
				validRecords = append(validRecords, record)
			}
			structSliceValue.Set(reflect.Append(structSliceValue, structValue))
		} else if !p.validRowsOnly {
			structSliceValue.Set(reflect.Append(structSliceValue, structValue))
		}
	}

	// Build output from the processed records
	reader, err := p.buildOutput(headers, records, validRecords, isJSONFormat)
	if err != nil {
		return nil, nil, err
	}

	return reader, result, nil
}

// processRow applies preprocessing and single-field validation to one row.
// It returns true if the row has any errors, and a non-nil error for fatal
// conditions (e.g., JSON corruption after preprocessing).
func (p *Processor) processRow(
	record []string,
	rowNum int,
	structInfo *structInfo,
	structValue reflect.Value,
	result *ProcessResult,
	isJSONFormat bool,
	jsonDataColumn string,
) (bool, error) {
	rowHasError := false

	for _, fieldInfo := range structInfo.Fields {
		colIdx := fieldInfo.ColumnIndex

		// Get value: empty string if column not found or out of range
		value := ""
		if colIdx >= 0 && colIdx < len(record) {
			value = record[colIdx]
		}

		colName := fieldInfo.ColumnName

		// Apply preprocessing and update record in-place
		processedValue := fieldInfo.Preprocessors.Process(value)
		if colIdx >= 0 && colIdx < len(record) {
			record[colIdx] = processedValue
		}

		// For JSON/JSONL formats, verify the "data" column integrity after preprocessing.
		// Only the "data" column contains JSON values; other struct fields may map to
		// non-existent columns and receive default/preprocessed non-JSON values, so
		// checking all fields would cause false positives.
		if isJSONFormat && colName == jsonDataColumn {
			if processedValue != "" && !json.Valid([]byte(processedValue)) {
				// Prep tags (e.g. truncate, replace) destroyed the JSON structure.
				// This is a hard error: invalid JSON lines in JSONL output cause
				// downstream parsers to fail entirely.
				return false, fmt.Errorf("row %d, column %q: %w: %s",
					rowNum, colName, ErrInvalidJSONAfterPrep, truncateForError(processedValue, 100))
			} else if value != "" && processedValue == "" {
				// Preprocessing emptied the JSON data (e.g. nullify).
				// The row will be skipped in JSONL output, so record a PrepError
				// to keep ValidRowCount consistent with actual output line count.
				result.Errors = append(result.Errors, newPrepError(
					rowNum, colName, fieldInfo.Name, "empty_json_data",
					"JSON data is empty after preprocessing (original: "+truncateForError(value, 100)+")",
				))
				rowHasError = true
			}
		}

		// Apply validation
		if tag, msg := fieldInfo.Validators.Validate(processedValue); msg != "" {
			result.Errors = append(result.Errors, newValidationError(
				rowNum, colName, fieldInfo.Name, processedValue, tag, msg,
			))
			rowHasError = true
		}

		// Set struct field value (use field index, not column index)
		if err := setFieldValue(structValue.Field(fieldInfo.Index), processedValue); err != nil {
			result.Errors = append(result.Errors, newPrepError(
				rowNum, colName, fieldInfo.Name, "type_conversion",
				fmt.Sprintf("failed to convert value %q: %v", processedValue, err),
			))
			rowHasError = true
		}
	}

	return rowHasError, nil
}

// applyCrossFieldValidation runs cross-field validators for one row.
// It returns true if any cross-field validation error was found.
func (p *Processor) applyCrossFieldValidation(
	record []string,
	rowNum int,
	structInfo *structInfo,
	fieldNameToColIdx map[string]int,
	result *ProcessResult,
) bool {
	hasError := false

	for _, fieldInfo := range structInfo.Fields {
		if len(fieldInfo.CrossFieldValidators) == 0 {
			continue
		}

		colIdx := fieldInfo.ColumnIndex
		srcValue := ""
		if colIdx >= 0 && colIdx < len(record) {
			srcValue = record[colIdx]
		}
		colName := fieldInfo.ColumnName

		for _, crossValidator := range fieldInfo.CrossFieldValidators {
			targetFieldName := crossValidator.TargetField()
			targetColIdx, ok := fieldNameToColIdx[targetFieldName]
			if !ok || targetColIdx < 0 {
				result.Errors = append(result.Errors, newValidationError(
					rowNum, colName, fieldInfo.Name, srcValue,
					crossValidator.Name(),
					fmt.Sprintf("target field %s not found", targetFieldName),
				))
				hasError = true
				continue
			}

			if targetColIdx >= len(record) {
				result.Errors = append(result.Errors, newValidationError(
					rowNum, colName, fieldInfo.Name, srcValue,
					crossValidator.Name(),
					fmt.Sprintf("target field %s index out of range", targetFieldName),
				))
				hasError = true
				continue
			}

			targetValue := record[targetColIdx]
			if msg := crossValidator.Validate(srcValue, targetValue); msg != "" {
				result.Errors = append(result.Errors, newValidationError(
					rowNum, colName, fieldInfo.Name, srcValue,
					crossValidator.Name(), msg,
				))
				hasError = true
			}
		}
	}

	return hasError
}

// buildOutput generates the output io.Reader from processed records.
// When validRowsOnly is enabled, validRecords is used instead of all records.
func (p *Processor) buildOutput(headers []string, records [][]string, validRecords [][]string, isJSONFormat bool) (io.Reader, error) {
	// Select which records to include in output
	outputRecords := records
	if p.validRowsOnly {
		outputRecords = validRecords
	}

	// Pre-allocate buffer capacity based on estimated output size to reduce allocations
	var outputBuf bytes.Buffer
	estimatedSize := p.estimateOutputSize(headers, outputRecords)
	outputBuf.Grow(estimatedSize)
	if err := p.writeOutput(&outputBuf, headers, outputRecords); err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	// For JSON/JSONL, an empty output means all rows were empty after preprocessing.
	// This is a hard error because an empty JSONL stream is unparseable by downstream consumers.
	if isJSONFormat && outputBuf.Len() == 0 {
		return nil, ErrEmptyJSONOutput
	}

	return newStream(outputBuf.Bytes(), p.outputFormat(), p.fileType), nil
}

// outputFormat returns the actual output format for the stream.
// CSV, TSV, and LTSV preserve their format.
// JSON and JSONL are output as JSONL (one JSON value per line).
// XLSX and Parquet are converted to CSV.
func (p *Processor) outputFormat() fileparser.FileType {
	switch fileparser.BaseFileType(p.fileType) {
	case fileparser.CSV, fileparser.TSV, fileparser.LTSV:
		return fileparser.BaseFileType(p.fileType)
	case fileparser.JSON, fileparser.JSONL:
		return fileparser.JSONL
	default:
		// XLSX, Parquet output as CSV
		return fileparser.CSV
	}
}

// estimateOutputSize estimates the output buffer size based on headers and records.
// This helps reduce buffer reallocations during output generation.
func (p *Processor) estimateOutputSize(headers []string, records [][]string) int {
	// Estimate average field length (including delimiter and quotes)
	const avgFieldLen = 20
	const lineOverhead = 2 // newline characters

	headerSize := len(headers) * avgFieldLen
	recordSize := len(records) * (len(headers)*avgFieldLen + lineOverhead)

	return headerSize + recordSize
}

// writeOutput writes the processed data back in the original format.
//
// Output format by input type:
//   - CSV → CSV (comma-delimited)
//   - TSV → TSV (tab-delimited)
//   - LTSV → LTSV (label:value pairs, tab-separated)
//   - JSON → JSONL (one JSON value per line)
//   - JSONL → JSONL (one JSON value per line)
//   - XLSX → CSV (tabular data as comma-delimited)
//   - Parquet → CSV (tabular data as comma-delimited)
func (p *Processor) writeOutput(w io.Writer, headers []string, records [][]string) error {
	switch fileparser.BaseFileType(p.fileType) {
	case fileparser.TSV:
		return p.writeTSV(w, headers, records)
	case fileparser.LTSV:
		return p.writeLTSV(w, headers, records)
	case fileparser.JSON, fileparser.JSONL:
		return p.writeJSONL(w, records)
	default:
		// CSV, XLSX, Parquet all output as CSV (tabular format)
		return p.writeCSV(w, headers, records)
	}
}

// writeCSV writes data in CSV format
func (p *Processor) writeCSV(w io.Writer, headers []string, records [][]string) error {
	csvWriter := csv.NewWriter(w)

	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	for _, record := range records {
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

// writeTSV writes data in TSV format
func (p *Processor) writeTSV(w io.Writer, headers []string, records [][]string) error {
	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = '\t'

	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	for _, record := range records {
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

// writeLTSV writes data in LTSV format
func (p *Processor) writeLTSV(w io.Writer, headers []string, records [][]string) error {
	// Pre-allocate a reusable buffer for building each line
	var lineBuf strings.Builder
	// Estimate line size: header + ":" + avg_value_size + "\t" for each field
	estimatedLineSize := len(headers) * 20
	lineBuf.Grow(estimatedLineSize)

	for _, record := range records {
		lineBuf.Reset()
		for i, header := range headers {
			if i > 0 {
				lineBuf.WriteByte('\t')
			}
			lineBuf.WriteString(header)
			lineBuf.WriteByte(':')
			if i < len(record) {
				lineBuf.WriteString(record[i])
			}
		}
		lineBuf.WriteByte('\n')
		if _, err := io.WriteString(w, lineBuf.String()); err != nil {
			return err
		}
	}
	return nil
}

// writeJSONL writes data in JSONL format (one JSON value per line).
// For JSON/JSONL input, each record has a single "data" column containing
// a raw JSON string. The output writes each JSON value on its own line,
// producing valid JSONL that can be consumed by filesql.
// Empty strings are skipped to avoid writing blank lines.
//
// Each value is compacted via json.Compact to ensure it occupies exactly one line.
// Pretty-printed JSON from fileparser may contain newlines within a single element,
// which would break JSONL format without compaction.
func (p *Processor) writeJSONL(w io.Writer, records [][]string) error {
	var compactBuf bytes.Buffer
	for _, record := range records {
		// record[0] is the "data" column: fileparser stores each JSON element
		// as a single-column row for JSON/JSONL input.
		if len(record) == 0 || record[0] == "" {
			continue
		}
		compactBuf.Reset()
		if err := json.Compact(&compactBuf, []byte(record[0])); err != nil {
			// Should not happen: invalid JSON is caught by ErrInvalidJSONAfterPrep
			// before reaching writeJSONL. Return error rather than writing broken JSONL.
			return fmt.Errorf("failed to compact JSON at output: %w", err)
		}
		if _, err := compactBuf.WriteTo(w); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}
	return nil
}

// truncateForError truncates a string for inclusion in error messages.
// It truncates on rune boundaries to avoid splitting multi-byte characters.
func truncateForError(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// setFieldValue sets a struct field value from a string
func setFieldValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == "" {
			field.SetInt(0)
			return nil
		}
		intVal, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value == "" {
			field.SetUint(0)
			return nil
		}
		uintVal, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		if value == "" {
			field.SetFloat(0)
			return nil
		}
		floatVal, err := strconv.ParseFloat(value, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		if value == "" {
			field.SetBool(false)
			return nil
		}
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

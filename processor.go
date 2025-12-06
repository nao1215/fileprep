package fileprep

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Processor handles preprocessing and validation of file data
type Processor struct {
	fileType FileType
}

// NewProcessor creates a new Processor for the specified file type.
//
// Example:
//
//	processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
//	var records []MyRecord
//	reader, result, err := processor.Process(input, &records)
func NewProcessor(fileType FileType) *Processor {
	return &Processor{
		fileType: fileType,
	}
}

// Process reads from the input reader, applies preprocessing and validation,
// populates the struct slice, and returns an io.Reader with preprocessed data.
//
// The returned io.Reader preserves the original file format:
//   - CSV input → CSV output
//   - TSV input → TSV output (tab-delimited)
//   - LTSV input → LTSV output (label:value format)
//   - XLSX input → CSV output (tabular data)
//   - Parquet input → CSV output (tabular data)
//
// The returned io.Reader can be passed directly to filesql.AddReader:
//
//	reader, result, err := processor.Process(input, &records)
//	db.AddReader(reader, "table", filesql.FileTypeCSV)
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
//	processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
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

	structInfo, err := parseStructType(structType)
	if err != nil {
		return nil, nil, err
	}

	// Read and decompress if needed
	decompressedReader, cleanup, err := p.decompressIfNeeded(input)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if cleanup != nil {
			cleanup() //nolint:errcheck,gosec // cleanup errors are not critical
		}
	}()

	// Parse based on file type
	// For CSV/TSV/LTSV, pass the reader directly to avoid extra copy.
	// For XLSX/Parquet, we need to read all data first as these formats require random access.
	var parseRes *parseResult

	switch p.fileType.BaseType() {
	case FileTypeCSV:
		parseRes, err = parseCSV(decompressedReader, csvDelimiter)
	case FileTypeTSV:
		parseRes, err = parseCSV(decompressedReader, tsvDelimiter)
	case FileTypeLTSV:
		parseRes, err = parseLTSV(decompressedReader)
	case FileTypeXLSX, FileTypeParquet:
		// XLSX and Parquet require random access, so we need to read all data first
		var data []byte
		data, err = io.ReadAll(decompressedReader)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read input: %w", err)
		}
		if p.fileType.BaseType() == FileTypeXLSX {
			parseRes, err = parseXLSX(data)
		} else {
			parseRes, err = parseParquet(data)
		}
	default:
		return nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedFileType, p.fileType)
	}

	if err != nil {
		return nil, nil, err
	}

	headers := parseRes.headers
	records := parseRes.records

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
	result := &ProcessResult{
		Columns:        headers,
		OriginalFormat: p.fileType,
	}
	structSliceValue := reflect.ValueOf(structSlicePointer).Elem()

	// Build field name to column index map for cross-field validation
	fieldNameToColIdx := make(map[string]int)
	for _, fi := range structInfo.Fields {
		fieldNameToColIdx[fi.Name] = fi.ColumnIndex
	}

	headerLen := len(headers)

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
		rowHasError := false

		// First pass: apply preprocessing and single-field validation
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

		// Second pass: apply cross-field validation
		for _, fieldInfo := range structInfo.Fields {
			if len(fieldInfo.CrossFieldValidators) == 0 {
				continue
			}

			colIdx := fieldInfo.ColumnIndex
			if colIdx < 0 || colIdx >= len(record) {
				continue
			}

			srcValue := record[colIdx]
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
					rowHasError = true
					continue
				}

				if targetColIdx >= len(record) {
					result.Errors = append(result.Errors, newValidationError(
						rowNum, colName, fieldInfo.Name, srcValue,
						crossValidator.Name(),
						fmt.Sprintf("target field %s index out of range", targetFieldName),
					))
					rowHasError = true
					continue
				}

				targetValue := record[targetColIdx]
				if msg := crossValidator.Validate(srcValue, targetValue); msg != "" {
					result.Errors = append(result.Errors, newValidationError(
						rowNum, colName, fieldInfo.Name, srcValue,
						crossValidator.Name(), msg,
					))
					rowHasError = true
				}
			}
		}

		if !rowHasError {
			result.ValidRowCount++
		}

		structSliceValue.Set(reflect.Append(structSliceValue, structValue))
	}

	// Generate output for filesql using the modified records slice directly
	// Pre-allocate buffer capacity based on estimated output size to reduce allocations
	var outputBuf bytes.Buffer
	estimatedSize := p.estimateOutputSize(headers, records)
	outputBuf.Grow(estimatedSize)
	if err := p.writeOutput(&outputBuf, headers, records); err != nil {
		return nil, nil, fmt.Errorf("failed to write output: %w", err)
	}

	return newStream(outputBuf.Bytes(), p.outputFormat(), p.fileType), result, nil
}

// outputFormat returns the actual output format for the stream.
// CSV, TSV, and LTSV preserve their format.
// XLSX and Parquet are converted to CSV.
func (p *Processor) outputFormat() FileType {
	switch p.fileType.BaseType() {
	case FileTypeCSV, FileTypeTSV, FileTypeLTSV:
		return p.fileType.BaseType()
	default:
		// XLSX, Parquet output as CSV
		return FileTypeCSV
	}
}

// decompressIfNeeded wraps the reader with decompression if the file type is compressed
func (p *Processor) decompressIfNeeded(reader io.Reader) (io.Reader, func() error, error) {
	if !p.fileType.IsCompressed() {
		return reader, nil, nil
	}

	handler := newCompressionHandler(p.fileType.compressionTypeValue())
	return handler.CreateReader(reader)
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
//   - XLSX → CSV (tabular data as comma-delimited)
//   - Parquet → CSV (tabular data as comma-delimited)
func (p *Processor) writeOutput(w io.Writer, headers []string, records [][]string) error {
	switch p.fileType.BaseType() {
	case FileTypeTSV:
		return p.writeTSV(w, headers, records)
	case FileTypeLTSV:
		return p.writeLTSV(w, headers, records)
	default:
		// CSV, XLSX, Parquet all output as CSV (tabular format)
		return p.writeCSV(w, headers, records)
	}
}

// writeCSV writes data in CSV format
func (p *Processor) writeCSV(w io.Writer, headers []string, records [][]string) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	for _, record := range records {
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	return csvWriter.Error()
}

// writeTSV writes data in TSV format
func (p *Processor) writeTSV(w io.Writer, headers []string, records [][]string) error {
	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = '\t'
	defer csvWriter.Flush()

	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	for _, record := range records {
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	return csvWriter.Error()
}

// writeLTSV writes data in LTSV format
func (p *Processor) writeLTSV(w io.Writer, headers []string, records [][]string) error {
	for _, record := range records {
		var fields []string
		for i, header := range headers {
			value := ""
			if i < len(record) {
				value = record[i]
			}
			fields = append(fields, header+":"+value)
		}
		if _, err := fmt.Fprintln(w, strings.Join(fields, "\t")); err != nil {
			return err
		}
	}
	return nil
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

package fileprep

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strconv"
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

	// Read all data into memory for processing
	data, err := io.ReadAll(decompressedReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read input: %w", err)
	}

	// Parse based on file type
	var parseRes *parseResult

	switch p.fileType.BaseType() {
	case FileTypeCSV:
		parseRes, err = parseCSV(bytes.NewReader(data), csvDelimiter)
	case FileTypeTSV:
		parseRes, err = parseCSV(bytes.NewReader(data), tsvDelimiter)
	case FileTypeLTSV:
		parseRes, err = parseLTSV(bytes.NewReader(data))
	case FileTypeXLSX:
		parseRes, err = parseXLSX(data)
	case FileTypeParquet:
		parseRes, err = parseParquet(data)
	default:
		return nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedFileType, p.fileType)
	}

	if err != nil {
		return nil, nil, err
	}

	headers := parseRes.headers
	records := parseRes.records

	// Process records: apply preprocessing and validation
	result := &ProcessResult{
		Columns:        headers,
		OriginalFormat: p.fileType,
	}
	processedRecords := make([][]string, 0, len(records))
	structSliceValue := reflect.ValueOf(structSlicePointer).Elem()

	// Build field name to index map for cross-field validation
	fieldNameToIdx := make(map[string]int)
	for _, fi := range structInfo.Fields {
		fieldNameToIdx[fi.Name] = fi.Index
	}

	for rowIdx, record := range records {
		rowNum := rowIdx + 1 // 1-based row number (excluding header)
		result.RowCount++

		processedRecord := make([]string, len(record))
		copy(processedRecord, record)

		structValue := reflect.New(structType).Elem()
		rowHasError := false

		// First pass: apply preprocessing and single-field validation
		for _, fieldInfo := range structInfo.Fields {
			colIdx := fieldInfo.Index
			if colIdx >= len(record) {
				continue
			}

			value := record[colIdx]
			colName := ""
			if colIdx < len(headers) {
				colName = headers[colIdx]
			}

			// Apply preprocessing
			processedValue := fieldInfo.Preprocessors.Process(value)
			processedRecord[colIdx] = processedValue

			// Apply validation
			if tag, msg := fieldInfo.Validators.Validate(processedValue); msg != "" {
				result.Errors = append(result.Errors, newValidationError(
					rowNum, colName, fieldInfo.Name, processedValue, tag, msg,
				))
				rowHasError = true
			}

			// Set struct field value (non-fatal errors are ignored)
			setFieldValue(structValue.Field(colIdx), processedValue) //nolint:errcheck,gosec // type conversion errors are non-fatal
		}

		// Second pass: apply cross-field validation
		for _, fieldInfo := range structInfo.Fields {
			if len(fieldInfo.CrossFieldValidators) == 0 {
				continue
			}

			colIdx := fieldInfo.Index
			if colIdx >= len(processedRecord) {
				continue
			}

			srcValue := processedRecord[colIdx]
			colName := ""
			if colIdx < len(headers) {
				colName = headers[colIdx]
			}

			for _, crossValidator := range fieldInfo.CrossFieldValidators {
				targetFieldName := crossValidator.TargetField()
				targetIdx, ok := fieldNameToIdx[targetFieldName]
				if !ok {
					result.Errors = append(result.Errors, newValidationError(
						rowNum, colName, fieldInfo.Name, srcValue,
						crossValidator.Name(),
						fmt.Sprintf("target field %s not found", targetFieldName),
					))
					rowHasError = true
					continue
				}

				if targetIdx >= len(processedRecord) {
					result.Errors = append(result.Errors, newValidationError(
						rowNum, colName, fieldInfo.Name, srcValue,
						crossValidator.Name(),
						fmt.Sprintf("target field %s index out of range", targetFieldName),
					))
					rowHasError = true
					continue
				}

				targetValue := processedRecord[targetIdx]
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

		processedRecords = append(processedRecords, processedRecord)
		structSliceValue.Set(reflect.Append(structSliceValue, structValue))
	}

	// Generate output for filesql
	var outputBuf bytes.Buffer
	if err := p.writeOutput(&outputBuf, headers, processedRecords); err != nil {
		return nil, nil, fmt.Errorf("failed to write output: %w", err)
	}

	return newStream(outputBuf.Bytes(), p.fileType), result, nil
}

// decompressIfNeeded wraps the reader with decompression if the file type is compressed
func (p *Processor) decompressIfNeeded(reader io.Reader) (io.Reader, func() error, error) {
	if !p.fileType.IsCompressed() {
		return reader, nil, nil
	}

	handler := newCompressionHandler(p.fileType.compressionTypeValue())
	return handler.CreateReader(reader)
}

// writeOutput writes the processed data back to CSV format for filesql
func (p *Processor) writeOutput(w io.Writer, headers []string, records [][]string) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header
	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	// Write records
	for _, record := range records {
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	return csvWriter.Error()
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

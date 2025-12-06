package fileprep

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/parquet-go/parquet-go"
	"github.com/xuri/excelize/v2"
)

// parseResult holds the parsed data from any file format
type parseResult struct {
	headers []string
	records [][]string
}

// parseCSV parses CSV/TSV data with the given delimiter
func parseCSV(reader io.Reader, delimiter rune) (*parseResult, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = delimiter
	csvReader.FieldsPerRecord = -1 // Allow variable field counts

	allRecords, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(allRecords) == 0 {
		return nil, ErrEmptyFile
	}

	headers := allRecords[0]
	var records [][]string
	if len(allRecords) > 1 {
		records = allRecords[1:]
	}

	return &parseResult{headers: headers, records: records}, nil
}

// parseLTSV parses LTSV (Labeled Tab-Separated Values) data using bufio.Scanner.
// Values are preserved as-is without trimming whitespace; use prep:"trim" for whitespace removal.
func parseLTSV(reader io.Reader) (*parseResult, error) {
	scanner := bufio.NewScanner(reader)
	// Expand buffer to handle large lines (e.g., log-derived LTSV can exceed 64KB default)
	const maxLineSize = 10 * 1024 * 1024 // 10MB max line size
	scanner.Buffer(make([]byte, 64*1024), maxLineSize)

	// Collect all unique keys to form headers
	headerMap := make(map[string]int)
	var headerOrder []string
	var parsedRecords []map[string]string

	for scanner.Scan() {
		line := scanner.Text()
		// Skip empty lines (lines with only whitespace are not considered empty)
		if line == "" {
			continue
		}

		record := make(map[string]string)
		pairs := strings.Split(line, "\t")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				// Only trim key (label) as per LTSV spec; preserve value whitespace
				key := strings.TrimSpace(kv[0])
				value := kv[1] // Preserve whitespace in values
				record[key] = value
				if _, exists := headerMap[key]; !exists {
					headerMap[key] = len(headerOrder)
					headerOrder = append(headerOrder, key)
				}
			}
		}
		if len(record) > 0 {
			parsedRecords = append(parsedRecords, record)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read LTSV: %w", err)
	}

	if len(parsedRecords) == 0 {
		return nil, ErrEmptyFile
	}

	// Convert to standard record format
	records := make([][]string, len(parsedRecords))
	for i, recordMap := range parsedRecords {
		row := make([]string, len(headerOrder))
		for j, key := range headerOrder {
			if val, exists := recordMap[key]; exists {
				row[j] = val
			}
		}
		records[i] = row
	}

	return &parseResult{headers: headerOrder, records: records}, nil
}

// parseXLSX parses Excel XLSX data using streaming API to reduce memory usage.
// Note: Only the first sheet is processed.
func parseXLSX(data []byte) (*parseResult, error) {
	xlsxFile, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to open XLSX: %w", err)
	}
	defer func() {
		_ = xlsxFile.Close()
	}()

	sheetNames := xlsxFile.GetSheetList()
	if len(sheetNames) == 0 {
		return nil, ErrEmptyFile
	}

	// Use streaming Rows API to reduce memory usage for large files
	rowsIter, err := xlsxFile.Rows(sheetNames[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %w", err)
	}
	defer func() {
		_ = rowsIter.Close()
	}()

	var headers []string
	var records [][]string

	for rowsIter.Next() {
		row, err := rowsIter.Columns()
		if err != nil {
			return nil, fmt.Errorf("failed to read row: %w", err)
		}

		if headers == nil {
			// First row is headers
			headers = row
			continue
		}

		// Normalize row length to match header count
		if len(row) < len(headers) {
			padded := make([]string, len(headers))
			copy(padded, row)
			row = padded
		}
		records = append(records, row)
	}

	if err := rowsIter.Error(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}

	if len(headers) == 0 {
		return nil, ErrEmptyFile
	}

	return &parseResult{headers: headers, records: records}, nil
}

// parseParquet parses Parquet data using parquet-go library
func parseParquet(data []byte) (*parseResult, error) {
	if len(data) == 0 {
		return nil, ErrEmptyFile
	}

	// Open parquet file from bytes
	pqFile, err := parquet.OpenFile(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet file: %w", err)
	}
	defer func() {
		// Close the parquet file to release resources
		if closer, ok := any(pqFile).(io.Closer); ok {
			_ = closer.Close()
		}
	}()

	schema := pqFile.Schema()
	if schema == nil {
		return nil, errors.New("failed to get parquet schema")
	}

	// Extract headers from schema
	fields := schema.Fields()
	headers := make([]string, len(fields))
	for i, field := range fields {
		headers[i] = field.Name()
	}

	if len(headers) == 0 {
		return nil, ErrEmptyFile
	}

	// Read all row groups with reusable row buffer to reduce allocations
	var records [][]string
	rowGroups := pqFile.RowGroups()
	rowBuf := make([]parquet.Row, 100) // Reuse buffer across all row groups

	for _, rowGroup := range rowGroups {
		rows := rowGroup.Rows()

		var readErr error
		for {
			n, err := rows.ReadRows(rowBuf)
			if n == 0 {
				break
			}

			for j := range n {
				row := rowBuf[j]
				record := make([]string, len(headers))
				for k, val := range row {
					record[k] = formatParquetValue(val)
				}
				records = append(records, record)
			}

			if err != nil {
				if !errors.Is(err, io.EOF) {
					readErr = err
				}
				break
			}
		}
		_ = rows.Close()

		if readErr != nil {
			return nil, fmt.Errorf("failed to read parquet rows: %w", readErr)
		}
	}

	if len(records) == 0 {
		return nil, ErrEmptyFile
	}

	return &parseResult{headers: headers, records: records}, nil
}

// formatParquetValue converts a parquet.Value to string
func formatParquetValue(val parquet.Value) string {
	if val.IsNull() {
		return ""
	}

	switch val.Kind() {
	case parquet.Boolean:
		return strconv.FormatBool(val.Boolean())
	case parquet.Int32:
		return strconv.Itoa(int(val.Int32()))
	case parquet.Int64:
		return strconv.FormatInt(val.Int64(), 10)
	case parquet.Int96:
		// Int96 is typically used for timestamps in legacy Parquet
		return fmt.Sprintf("%v", val.Int96())
	case parquet.Float:
		return strconv.FormatFloat(float64(val.Float()), 'g', -1, 32)
	case parquet.Double:
		return strconv.FormatFloat(val.Double(), 'g', -1, 64)
	case parquet.ByteArray:
		return string(val.ByteArray())
	case parquet.FixedLenByteArray:
		return string(val.ByteArray())
	default:
		return fmt.Sprintf("%v", val)
	}
}

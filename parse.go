package fileprep

import (
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

// parseLTSV parses LTSV (Labeled Tab-Separated Values) data
func parseLTSV(reader io.Reader) (*parseResult, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read LTSV: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return nil, ErrEmptyFile
	}

	// Collect all unique keys to form headers
	headerMap := make(map[string]int)
	var headerOrder []string
	var parsedRecords []map[string]string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		record := make(map[string]string)
		pairs := strings.Split(line, "\t")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
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

// parseXLSX parses Excel XLSX data
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

	rows, err := xlsxFile.GetRows(sheetNames[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %w", err)
	}

	if len(rows) == 0 {
		return nil, ErrEmptyFile
	}

	headers := rows[0]
	var records [][]string
	if len(rows) > 1 {
		records = rows[1:]
	}

	// Normalize record lengths to match header count
	for i := range records {
		if len(records[i]) < len(headers) {
			padded := make([]string, len(headers))
			copy(padded, records[i])
			records[i] = padded
		}
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

	// Read all row groups
	var records [][]string
	rowGroups := pqFile.RowGroups()
	for _, rowGroup := range rowGroups {
		rows := rowGroup.Rows()

		for {
			rowBuf := make([]parquet.Row, 100)
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
				break
			}
		}
		_ = rows.Close()
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

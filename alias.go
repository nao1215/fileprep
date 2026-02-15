// Package fileprep re-exports fileparser types for backward compatibility.
package fileprep

import "github.com/nao1215/fileparser"

// FileType is an alias for fileparser.FileType for backward compatibility.
type FileType = fileparser.FileType

// File type constants re-exported from fileparser for backward compatibility.
const (
	FileTypeCSV         = fileparser.CSV
	FileTypeTSV         = fileparser.TSV
	FileTypeLTSV        = fileparser.LTSV
	FileTypeParquet     = fileparser.Parquet
	FileTypeXLSX        = fileparser.XLSX
	FileTypeCSVGZ       = fileparser.CSVGZ
	FileTypeCSVBZ2      = fileparser.CSVBZ2
	FileTypeCSVXZ       = fileparser.CSVXZ
	FileTypeCSVZSTD     = fileparser.CSVZSTD
	FileTypeTSVGZ       = fileparser.TSVGZ
	FileTypeTSVBZ2      = fileparser.TSVBZ2
	FileTypeTSVXZ       = fileparser.TSVXZ
	FileTypeTSVZSTD     = fileparser.TSVZSTD
	FileTypeLTSVGZ      = fileparser.LTSVGZ
	FileTypeLTSVBZ2     = fileparser.LTSVBZ2
	FileTypeLTSVXZ      = fileparser.LTSVXZ
	FileTypeLTSVZSTD    = fileparser.LTSVZSTD
	FileTypeParquetGZ   = fileparser.ParquetGZ
	FileTypeParquetBZ2  = fileparser.ParquetBZ2
	FileTypeParquetXZ   = fileparser.ParquetXZ
	FileTypeParquetZSTD = fileparser.ParquetZSTD
	FileTypeXLSXGZ      = fileparser.XLSXGZ
	FileTypeXLSXBZ2     = fileparser.XLSXBZ2
	FileTypeXLSXXZ      = fileparser.XLSXXZ
	FileTypeXLSXZSTD    = fileparser.XLSXZSTD

	// zlib compression formats (v0.2.0+)
	FileTypeCSVZLIB     = fileparser.CSVZLIB
	FileTypeTSVZLIB     = fileparser.TSVZLIB
	FileTypeLTSVZLIB    = fileparser.LTSVZLIB
	FileTypeParquetZLIB = fileparser.ParquetZLIB
	FileTypeXLSXZLIB    = fileparser.XLSXZLIB

	// snappy compression formats (v0.2.0+)
	FileTypeCSVSNAPPY     = fileparser.CSVSNAPPY
	FileTypeTSVSNAPPY     = fileparser.TSVSNAPPY
	FileTypeLTSVSNAPPY    = fileparser.LTSVSNAPPY
	FileTypeParquetSNAPPY = fileparser.ParquetSNAPPY
	FileTypeXLSXSNAPPY    = fileparser.XLSXSNAPPY

	// s2 compression formats (v0.2.0+)
	FileTypeCSVS2     = fileparser.CSVS2
	FileTypeTSVS2     = fileparser.TSVS2
	FileTypeLTSVS2    = fileparser.LTSVS2
	FileTypeParquetS2 = fileparser.ParquetS2
	FileTypeXLSXS2    = fileparser.XLSXS2

	// lz4 compression formats (v0.2.0+)
	FileTypeCSVLZ4     = fileparser.CSVLZ4
	FileTypeTSVLZ4     = fileparser.TSVLZ4
	FileTypeLTSVLZ4    = fileparser.LTSVLZ4
	FileTypeParquetLZ4 = fileparser.ParquetLZ4
	FileTypeXLSXLZ4    = fileparser.XLSXLZ4

	// JSON/JSONL file types (v0.5.0+)
	FileTypeJSON  = fileparser.JSON
	FileTypeJSONL = fileparser.JSONL

	// JSON compression formats (v0.5.0+)
	FileTypeJSONGZ     = fileparser.JSONGZ
	FileTypeJSONBZ2    = fileparser.JSONBZ2
	FileTypeJSONXZ     = fileparser.JSONXZ
	FileTypeJSONZSTD   = fileparser.JSONZSTD
	FileTypeJSONZLIB   = fileparser.JSONZLIB
	FileTypeJSONSNAPPY = fileparser.JSONSNAPPY
	FileTypeJSONS2     = fileparser.JSONS2
	FileTypeJSONLZ4    = fileparser.JSONLZ4

	// JSONL compression formats (v0.5.0+)
	FileTypeJSONLGZ     = fileparser.JSONLGZ
	FileTypeJSONLBZ2    = fileparser.JSONLBZ2
	FileTypeJSONLXZ     = fileparser.JSONLXZ
	FileTypeJSONLZSTD   = fileparser.JSONLZSTD
	FileTypeJSONLZLIB   = fileparser.JSONLZLIB
	FileTypeJSONLSNAPPY = fileparser.JSONLSNAPPY
	FileTypeJSONLS2     = fileparser.JSONLS2
	FileTypeJSONLLZ4    = fileparser.JSONLLZ4

	FileTypeUnsupported = fileparser.Unsupported
)

// DetectFileType detects file type from extension.
// This is a convenience wrapper around fileparser.DetectFileType.
func DetectFileType(path string) FileType {
	return fileparser.DetectFileType(path)
}

// IsCompressed returns true if the file type is compressed.
// This is a convenience wrapper around fileparser.IsCompressed.
func IsCompressed(ft FileType) bool {
	return fileparser.IsCompressed(ft)
}

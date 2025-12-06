package fileprep

import (
	"path/filepath"
	"strings"
)

// FileType represents supported file types including compression variants
type FileType int

const (
	// FileTypeCSV represents CSV file type
	FileTypeCSV FileType = iota
	// FileTypeTSV represents TSV file type
	FileTypeTSV
	// FileTypeLTSV represents LTSV file type
	FileTypeLTSV
	// FileTypeParquet represents Parquet file type
	FileTypeParquet
	// FileTypeXLSX represents Excel XLSX file type
	FileTypeXLSX
	// FileTypeCSVGZ represents gzip-compressed CSV file type
	FileTypeCSVGZ
	// FileTypeTSVGZ represents gzip-compressed TSV file type
	FileTypeTSVGZ
	// FileTypeLTSVGZ represents gzip-compressed LTSV file type
	FileTypeLTSVGZ
	// FileTypeParquetGZ represents gzip-compressed Parquet file type
	FileTypeParquetGZ
	// FileTypeCSVBZ2 represents bzip2-compressed CSV file type
	FileTypeCSVBZ2
	// FileTypeTSVBZ2 represents bzip2-compressed TSV file type
	FileTypeTSVBZ2
	// FileTypeLTSVBZ2 represents bzip2-compressed LTSV file type
	FileTypeLTSVBZ2
	// FileTypeParquetBZ2 represents bzip2-compressed Parquet file type
	FileTypeParquetBZ2
	// FileTypeCSVXZ represents xz-compressed CSV file type
	FileTypeCSVXZ
	// FileTypeTSVXZ represents xz-compressed TSV file type
	FileTypeTSVXZ
	// FileTypeLTSVXZ represents xz-compressed LTSV file type
	FileTypeLTSVXZ
	// FileTypeParquetXZ represents xz-compressed Parquet file type
	FileTypeParquetXZ
	// FileTypeCSVZSTD represents zstd-compressed CSV file type
	FileTypeCSVZSTD
	// FileTypeTSVZSTD represents zstd-compressed TSV file type
	FileTypeTSVZSTD
	// FileTypeLTSVZSTD represents zstd-compressed LTSV file type
	FileTypeLTSVZSTD
	// FileTypeParquetZSTD represents zstd-compressed Parquet file type
	FileTypeParquetZSTD
	// FileTypeXLSXGZ represents gzip-compressed Excel XLSX file type
	FileTypeXLSXGZ
	// FileTypeXLSXBZ2 represents bzip2-compressed Excel XLSX file type
	FileTypeXLSXBZ2
	// FileTypeXLSXXZ represents xz-compressed Excel XLSX file type
	FileTypeXLSXXZ
	// FileTypeXLSXZSTD represents zstd-compressed Excel XLSX file type
	FileTypeXLSXZSTD
	// FileTypeUnsupported represents unsupported file type
	FileTypeUnsupported
)

// File extensions
const (
	extCSV     = ".csv"
	extTSV     = ".tsv"
	extLTSV    = ".ltsv"
	extParquet = ".parquet"
	extXLSX    = ".xlsx"
	extGZ      = ".gz"
	extBZ2     = ".bz2"
	extXZ      = ".xz"
	extZSTD    = ".zst"
)

// File format delimiters
const (
	csvDelimiter = ','
	tsvDelimiter = '\t'
)

// Compression type strings
const (
	compTypeGZ   = "gz"
	compTypeBZ2  = "bz2"
	compTypeXZ   = "xz"
	compTypeZSTD = "zstd"
)

// String returns the string representation of the FileType
func (ft FileType) String() string {
	switch ft {
	case FileTypeCSV:
		return "CSV"
	case FileTypeTSV:
		return "TSV"
	case FileTypeLTSV:
		return "LTSV"
	case FileTypeParquet:
		return "Parquet"
	case FileTypeXLSX:
		return "XLSX"
	case FileTypeCSVGZ:
		return "CSV (gzip)"
	case FileTypeTSVGZ:
		return "TSV (gzip)"
	case FileTypeLTSVGZ:
		return "LTSV (gzip)"
	case FileTypeParquetGZ:
		return "Parquet (gzip)"
	case FileTypeXLSXGZ:
		return "XLSX (gzip)"
	case FileTypeCSVBZ2:
		return "CSV (bzip2)"
	case FileTypeTSVBZ2:
		return "TSV (bzip2)"
	case FileTypeLTSVBZ2:
		return "LTSV (bzip2)"
	case FileTypeParquetBZ2:
		return "Parquet (bzip2)"
	case FileTypeXLSXBZ2:
		return "XLSX (bzip2)"
	case FileTypeCSVXZ:
		return "CSV (xz)"
	case FileTypeTSVXZ:
		return "TSV (xz)"
	case FileTypeLTSVXZ:
		return "LTSV (xz)"
	case FileTypeParquetXZ:
		return "Parquet (xz)"
	case FileTypeXLSXXZ:
		return "XLSX (xz)"
	case FileTypeCSVZSTD:
		return "CSV (zstd)"
	case FileTypeTSVZSTD:
		return "TSV (zstd)"
	case FileTypeLTSVZSTD:
		return "LTSV (zstd)"
	case FileTypeParquetZSTD:
		return "Parquet (zstd)"
	case FileTypeXLSXZSTD:
		return "XLSX (zstd)"
	default:
		return "Unsupported"
	}
}

// Extension returns the file extension for the FileType
func (ft FileType) Extension() string {
	switch ft {
	case FileTypeCSV:
		return extCSV
	case FileTypeTSV:
		return extTSV
	case FileTypeLTSV:
		return extLTSV
	case FileTypeParquet:
		return extParquet
	case FileTypeXLSX:
		return extXLSX
	case FileTypeCSVGZ:
		return extCSV + extGZ
	case FileTypeTSVGZ:
		return extTSV + extGZ
	case FileTypeLTSVGZ:
		return extLTSV + extGZ
	case FileTypeParquetGZ:
		return extParquet + extGZ
	case FileTypeXLSXGZ:
		return extXLSX + extGZ
	case FileTypeCSVBZ2:
		return extCSV + extBZ2
	case FileTypeTSVBZ2:
		return extTSV + extBZ2
	case FileTypeLTSVBZ2:
		return extLTSV + extBZ2
	case FileTypeParquetBZ2:
		return extParquet + extBZ2
	case FileTypeXLSXBZ2:
		return extXLSX + extBZ2
	case FileTypeCSVXZ:
		return extCSV + extXZ
	case FileTypeTSVXZ:
		return extTSV + extXZ
	case FileTypeLTSVXZ:
		return extLTSV + extXZ
	case FileTypeParquetXZ:
		return extParquet + extXZ
	case FileTypeXLSXXZ:
		return extXLSX + extXZ
	case FileTypeCSVZSTD:
		return extCSV + extZSTD
	case FileTypeTSVZSTD:
		return extTSV + extZSTD
	case FileTypeLTSVZSTD:
		return extLTSV + extZSTD
	case FileTypeParquetZSTD:
		return extParquet + extZSTD
	case FileTypeXLSXZSTD:
		return extXLSX + extZSTD
	default:
		return ""
	}
}

// BaseType returns the base file type without compression
func (ft FileType) BaseType() FileType {
	switch ft {
	case FileTypeCSV, FileTypeCSVGZ, FileTypeCSVBZ2, FileTypeCSVXZ, FileTypeCSVZSTD:
		return FileTypeCSV
	case FileTypeTSV, FileTypeTSVGZ, FileTypeTSVBZ2, FileTypeTSVXZ, FileTypeTSVZSTD:
		return FileTypeTSV
	case FileTypeLTSV, FileTypeLTSVGZ, FileTypeLTSVBZ2, FileTypeLTSVXZ, FileTypeLTSVZSTD:
		return FileTypeLTSV
	case FileTypeParquet, FileTypeParquetGZ, FileTypeParquetBZ2, FileTypeParquetXZ, FileTypeParquetZSTD:
		return FileTypeParquet
	case FileTypeXLSX, FileTypeXLSXGZ, FileTypeXLSXBZ2, FileTypeXLSXXZ, FileTypeXLSXZSTD:
		return FileTypeXLSX
	default:
		return FileTypeUnsupported
	}
}

// IsCompressed returns true if the file type is compressed
func (ft FileType) IsCompressed() bool {
	switch ft {
	case FileTypeCSVGZ, FileTypeTSVGZ, FileTypeLTSVGZ, FileTypeParquetGZ, FileTypeXLSXGZ,
		FileTypeCSVBZ2, FileTypeTSVBZ2, FileTypeLTSVBZ2, FileTypeParquetBZ2, FileTypeXLSXBZ2,
		FileTypeCSVXZ, FileTypeTSVXZ, FileTypeLTSVXZ, FileTypeParquetXZ, FileTypeXLSXXZ,
		FileTypeCSVZSTD, FileTypeTSVZSTD, FileTypeLTSVZSTD, FileTypeParquetZSTD, FileTypeXLSXZSTD:
		return true
	default:
		return false
	}
}

// compressionTypeValue returns the compression type for the file type
func (ft FileType) compressionTypeValue() compressionType {
	switch ft {
	case FileTypeCSVGZ, FileTypeTSVGZ, FileTypeLTSVGZ, FileTypeParquetGZ, FileTypeXLSXGZ:
		return compressionGZ
	case FileTypeCSVBZ2, FileTypeTSVBZ2, FileTypeLTSVBZ2, FileTypeParquetBZ2, FileTypeXLSXBZ2:
		return compressionBZ2
	case FileTypeCSVXZ, FileTypeTSVXZ, FileTypeLTSVXZ, FileTypeParquetXZ, FileTypeXLSXXZ:
		return compressionXZ
	case FileTypeCSVZSTD, FileTypeTSVZSTD, FileTypeLTSVZSTD, FileTypeParquetZSTD, FileTypeXLSXZSTD:
		return compressionZSTD
	default:
		return compressionNone
	}
}

// DetectFileType detects file type from extension, considering compressed files.
//
// Example:
//
//	ft := fileprep.DetectFileType("data.csv")        // FileTypeCSV
//	ft := fileprep.DetectFileType("data.csv.gz")    // FileTypeCSVGZ
//	ft := fileprep.DetectFileType("report.xlsx")    // FileTypeXLSX
//	ft := fileprep.DetectFileType("data.parquet")   // FileTypeParquet
func DetectFileType(path string) FileType {
	basePath := path
	var compressionType string

	// Remove compression extensions
	switch {
	case strings.HasSuffix(strings.ToLower(path), extGZ):
		basePath = strings.TrimSuffix(path, extGZ)
		basePath = strings.TrimSuffix(basePath, strings.ToUpper(extGZ))
		compressionType = compTypeGZ
	case strings.HasSuffix(strings.ToLower(path), extBZ2):
		basePath = strings.TrimSuffix(path, extBZ2)
		basePath = strings.TrimSuffix(basePath, strings.ToUpper(extBZ2))
		compressionType = compTypeBZ2
	case strings.HasSuffix(strings.ToLower(path), extXZ):
		basePath = strings.TrimSuffix(path, extXZ)
		basePath = strings.TrimSuffix(basePath, strings.ToUpper(extXZ))
		compressionType = compTypeXZ
	case strings.HasSuffix(strings.ToLower(path), extZSTD):
		basePath = strings.TrimSuffix(path, extZSTD)
		basePath = strings.TrimSuffix(basePath, strings.ToUpper(extZSTD))
		compressionType = compTypeZSTD
	}

	ext := strings.ToLower(filepath.Ext(basePath))
	switch ext {
	case extCSV:
		switch compressionType {
		case compTypeGZ:
			return FileTypeCSVGZ
		case compTypeBZ2:
			return FileTypeCSVBZ2
		case compTypeXZ:
			return FileTypeCSVXZ
		case compTypeZSTD:
			return FileTypeCSVZSTD
		default:
			return FileTypeCSV
		}
	case extTSV:
		switch compressionType {
		case compTypeGZ:
			return FileTypeTSVGZ
		case compTypeBZ2:
			return FileTypeTSVBZ2
		case compTypeXZ:
			return FileTypeTSVXZ
		case compTypeZSTD:
			return FileTypeTSVZSTD
		default:
			return FileTypeTSV
		}
	case extLTSV:
		switch compressionType {
		case compTypeGZ:
			return FileTypeLTSVGZ
		case compTypeBZ2:
			return FileTypeLTSVBZ2
		case compTypeXZ:
			return FileTypeLTSVXZ
		case compTypeZSTD:
			return FileTypeLTSVZSTD
		default:
			return FileTypeLTSV
		}
	case extParquet:
		switch compressionType {
		case compTypeGZ:
			return FileTypeParquetGZ
		case compTypeBZ2:
			return FileTypeParquetBZ2
		case compTypeXZ:
			return FileTypeParquetXZ
		case compTypeZSTD:
			return FileTypeParquetZSTD
		default:
			return FileTypeParquet
		}
	case extXLSX:
		switch compressionType {
		case compTypeGZ:
			return FileTypeXLSXGZ
		case compTypeBZ2:
			return FileTypeXLSXBZ2
		case compTypeXZ:
			return FileTypeXLSXXZ
		case compTypeZSTD:
			return FileTypeXLSXZSTD
		default:
			return FileTypeXLSX
		}
	default:
		return FileTypeUnsupported
	}
}

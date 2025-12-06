package fileprep

import "testing"

func TestDetectFileType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want FileType
	}{
		// Basic formats
		{"CSV", "data.csv", FileTypeCSV},
		{"TSV", "data.tsv", FileTypeTSV},
		{"LTSV", "data.ltsv", FileTypeLTSV},
		{"Parquet", "data.parquet", FileTypeParquet},
		{"XLSX", "data.xlsx", FileTypeXLSX},

		// Compressed CSV
		{"CSV gzip", "data.csv.gz", FileTypeCSVGZ},
		{"CSV bzip2", "data.csv.bz2", FileTypeCSVBZ2},
		{"CSV xz", "data.csv.xz", FileTypeCSVXZ},
		{"CSV zstd", "data.csv.zst", FileTypeCSVZSTD},

		// Compressed TSV
		{"TSV gzip", "data.tsv.gz", FileTypeTSVGZ},
		{"TSV bzip2", "data.tsv.bz2", FileTypeTSVBZ2},
		{"TSV xz", "data.tsv.xz", FileTypeTSVXZ},
		{"TSV zstd", "data.tsv.zst", FileTypeTSVZSTD},

		// Compressed LTSV
		{"LTSV gzip", "data.ltsv.gz", FileTypeLTSVGZ},
		{"LTSV bzip2", "data.ltsv.bz2", FileTypeLTSVBZ2},
		{"LTSV xz", "data.ltsv.xz", FileTypeLTSVXZ},
		{"LTSV zstd", "data.ltsv.zst", FileTypeLTSVZSTD},

		// Compressed Parquet
		{"Parquet gzip", "data.parquet.gz", FileTypeParquetGZ},
		{"Parquet bzip2", "data.parquet.bz2", FileTypeParquetBZ2},
		{"Parquet xz", "data.parquet.xz", FileTypeParquetXZ},
		{"Parquet zstd", "data.parquet.zst", FileTypeParquetZSTD},

		// Compressed XLSX
		{"XLSX gzip", "data.xlsx.gz", FileTypeXLSXGZ},
		{"XLSX bzip2", "data.xlsx.bz2", FileTypeXLSXBZ2},
		{"XLSX xz", "data.xlsx.xz", FileTypeXLSXXZ},
		{"XLSX zstd", "data.xlsx.zst", FileTypeXLSXZSTD},

		// Unsupported
		{"Unknown", "data.json", FileTypeUnsupported},
		{"No extension", "data", FileTypeUnsupported},
		{"Only compression ext", "data.gz", FileTypeUnsupported},

		// Case insensitive
		{"CSV uppercase", "DATA.CSV", FileTypeCSV},
		{"CSV mixed case", "Data.Csv", FileTypeCSV},
		{"GZ uppercase", "data.csv.GZ", FileTypeCSVGZ},
		{"BZ2 uppercase", "data.csv.BZ2", FileTypeCSVBZ2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DetectFileType(tt.path); got != tt.want {
				t.Errorf("DetectFileType(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFileType_BaseType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileType FileType
		want     FileType
	}{
		{"CSV", FileTypeCSV, FileTypeCSV},
		{"CSV gzip", FileTypeCSVGZ, FileTypeCSV},
		{"CSV bzip2", FileTypeCSVBZ2, FileTypeCSV},
		{"TSV", FileTypeTSV, FileTypeTSV},
		{"TSV gzip", FileTypeTSVGZ, FileTypeTSV},
		{"XLSX", FileTypeXLSX, FileTypeXLSX},
		{"XLSX gzip", FileTypeXLSXGZ, FileTypeXLSX},
		{"Unsupported", FileTypeUnsupported, FileTypeUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.fileType.BaseType(); got != tt.want {
				t.Errorf("BaseType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileType_IsCompressed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileType FileType
		want     bool
	}{
		{"CSV not compressed", FileTypeCSV, false},
		{"CSV gzip compressed", FileTypeCSVGZ, true},
		{"CSV bzip2 compressed", FileTypeCSVBZ2, true},
		{"CSV xz compressed", FileTypeCSVXZ, true},
		{"CSV zstd compressed", FileTypeCSVZSTD, true},
		{"XLSX not compressed", FileTypeXLSX, false},
		{"XLSX gzip compressed", FileTypeXLSXGZ, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.fileType.IsCompressed(); got != tt.want {
				t.Errorf("IsCompressed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileType_Extension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileType FileType
		want     string
	}{
		{"CSV", FileTypeCSV, ".csv"},
		{"TSV", FileTypeTSV, ".tsv"},
		{"LTSV", FileTypeLTSV, ".ltsv"},
		{"Parquet", FileTypeParquet, ".parquet"},
		{"XLSX", FileTypeXLSX, ".xlsx"},
		{"CSV gzip", FileTypeCSVGZ, ".csv.gz"},
		{"CSV bzip2", FileTypeCSVBZ2, ".csv.bz2"},
		{"CSV xz", FileTypeCSVXZ, ".csv.xz"},
		{"CSV zstd", FileTypeCSVZSTD, ".csv.zst"},
		{"TSV gzip", FileTypeTSVGZ, ".tsv.gz"},
		{"TSV bzip2", FileTypeTSVBZ2, ".tsv.bz2"},
		{"TSV xz", FileTypeTSVXZ, ".tsv.xz"},
		{"TSV zstd", FileTypeTSVZSTD, ".tsv.zst"},
		{"LTSV gzip", FileTypeLTSVGZ, ".ltsv.gz"},
		{"LTSV bzip2", FileTypeLTSVBZ2, ".ltsv.bz2"},
		{"LTSV xz", FileTypeLTSVXZ, ".ltsv.xz"},
		{"LTSV zstd", FileTypeLTSVZSTD, ".ltsv.zst"},
		{"Parquet gzip", FileTypeParquetGZ, ".parquet.gz"},
		{"Parquet bzip2", FileTypeParquetBZ2, ".parquet.bz2"},
		{"Parquet xz", FileTypeParquetXZ, ".parquet.xz"},
		{"Parquet zstd", FileTypeParquetZSTD, ".parquet.zst"},
		{"XLSX gzip", FileTypeXLSXGZ, ".xlsx.gz"},
		{"XLSX bzip2", FileTypeXLSXBZ2, ".xlsx.bz2"},
		{"XLSX xz", FileTypeXLSXXZ, ".xlsx.xz"},
		{"XLSX zstd", FileTypeXLSXZSTD, ".xlsx.zst"},
		{"Unsupported", FileTypeUnsupported, ""},
		{"Unknown", FileType(99), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.fileType.Extension(); got != tt.want {
				t.Errorf("Extension() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileType FileType
		want     string
	}{
		{"CSV", FileTypeCSV, "CSV"},
		{"TSV", FileTypeTSV, "TSV"},
		{"LTSV", FileTypeLTSV, "LTSV"},
		{"Parquet", FileTypeParquet, "Parquet"},
		{"XLSX", FileTypeXLSX, "XLSX"},
		{"CSV gzip", FileTypeCSVGZ, "CSV (gzip)"},
		{"TSV gzip", FileTypeTSVGZ, "TSV (gzip)"},
		{"LTSV gzip", FileTypeLTSVGZ, "LTSV (gzip)"},
		{"Parquet gzip", FileTypeParquetGZ, "Parquet (gzip)"},
		{"XLSX gzip", FileTypeXLSXGZ, "XLSX (gzip)"},
		{"CSV bzip2", FileTypeCSVBZ2, "CSV (bzip2)"},
		{"TSV bzip2", FileTypeTSVBZ2, "TSV (bzip2)"},
		{"LTSV bzip2", FileTypeLTSVBZ2, "LTSV (bzip2)"},
		{"Parquet bzip2", FileTypeParquetBZ2, "Parquet (bzip2)"},
		{"XLSX bzip2", FileTypeXLSXBZ2, "XLSX (bzip2)"},
		{"CSV xz", FileTypeCSVXZ, "CSV (xz)"},
		{"TSV xz", FileTypeTSVXZ, "TSV (xz)"},
		{"LTSV xz", FileTypeLTSVXZ, "LTSV (xz)"},
		{"Parquet xz", FileTypeParquetXZ, "Parquet (xz)"},
		{"XLSX xz", FileTypeXLSXXZ, "XLSX (xz)"},
		{"CSV zstd", FileTypeCSVZSTD, "CSV (zstd)"},
		{"TSV zstd", FileTypeTSVZSTD, "TSV (zstd)"},
		{"LTSV zstd", FileTypeLTSVZSTD, "LTSV (zstd)"},
		{"Parquet zstd", FileTypeParquetZSTD, "Parquet (zstd)"},
		{"XLSX zstd", FileTypeXLSXZSTD, "XLSX (zstd)"},
		{"Unsupported", FileTypeUnsupported, "Unsupported"},
		{"Unknown", FileType(99), "Unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.fileType.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileType_compressionTypeValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileType FileType
		want     compressionType
	}{
		{"CSV no compression", FileTypeCSV, compressionNone},
		{"CSV gzip", FileTypeCSVGZ, compressionGZ},
		{"CSV bzip2", FileTypeCSVBZ2, compressionBZ2},
		{"CSV xz", FileTypeCSVXZ, compressionXZ},
		{"CSV zstd", FileTypeCSVZSTD, compressionZSTD},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.fileType.compressionTypeValue(); got != tt.want {
				t.Errorf("compressionTypeValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

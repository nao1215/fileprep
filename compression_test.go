package fileprep

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestCompressionType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ct   compressionType
		want string
	}{
		{"none", compressionNone, "none"},
		{"gzip", compressionGZ, "gzip"},
		{"bzip2", compressionBZ2, "bzip2"},
		{"xz", compressionXZ, "xz"},
		{"zstd", compressionZSTD, "zstd"},
		{"unknown", compressionType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.ct.String(); got != tt.want {
				t.Errorf("compressionType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompressionType_Extension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ct   compressionType
		want string
	}{
		{"none", compressionNone, ""},
		{"gzip", compressionGZ, ".gz"},
		{"bzip2", compressionBZ2, ".bz2"},
		{"xz", compressionXZ, ".xz"},
		{"zstd", compressionZSTD, ".zst"},
		{"unknown", compressionType(99), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.ct.Extension(); got != tt.want {
				t.Errorf("compressionType.Extension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewCompressionHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		ct            compressionType
		wantExtension string
	}{
		{"none", compressionNone, ""},
		{"gzip", compressionGZ, ".gz"},
		{"bzip2", compressionBZ2, ".bz2"},
		{"xz", compressionXZ, ".xz"},
		{"zstd", compressionZSTD, ".zst"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := newCompressionHandler(tt.ct)
			if handler.Extension() != tt.wantExtension {
				t.Errorf("Extension() = %v, want %v", handler.Extension(), tt.wantExtension)
			}
		})
	}
}

func TestCompressionFactory_DetectCompressionType(t *testing.T) {
	t.Parallel()

	factory := newCompressionFactory()

	tests := []struct {
		name string
		path string
		want compressionType
	}{
		{"gz", "file.csv.gz", compressionGZ},
		{"bz2", "file.csv.bz2", compressionBZ2},
		{"xz", "file.csv.xz", compressionXZ},
		{"zst", "file.csv.zst", compressionZSTD},
		{"none", "file.csv", compressionNone},
		{"uppercase gz", "FILE.CSV.GZ", compressionGZ},
		{"uppercase bz2", "FILE.CSV.BZ2", compressionBZ2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := factory.detectCompressionType(tt.path); got != tt.want {
				t.Errorf("detectCompressionType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompressionFactory_CreateHandlerForFile(t *testing.T) {
	t.Parallel()

	factory := newCompressionFactory()

	tests := []struct {
		name          string
		path          string
		wantExtension string
	}{
		{"gz file", "file.csv.gz", ".gz"},
		{"bz2 file", "file.csv.bz2", ".bz2"},
		{"xz file", "file.csv.xz", ".xz"},
		{"zst file", "file.csv.zst", ".zst"},
		{"uncompressed file", "file.csv", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := factory.createHandlerForFile(tt.path)
			if handler.Extension() != tt.wantExtension {
				t.Errorf("Extension() = %v, want %v", handler.Extension(), tt.wantExtension)
			}
		})
	}
}

func TestCompressionHandler_CreateReader_None(t *testing.T) {
	t.Parallel()

	handler := newCompressionHandler(compressionNone)
	input := strings.NewReader("test data")

	reader, cleanup, err := handler.CreateReader(input)
	if err != nil {
		t.Fatalf("CreateReader() error = %v", err)
	}
	defer func() { _ = cleanup() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != "test data" {
		t.Errorf("got %q, want %q", string(data), "test data")
	}
}

func TestCompressionHandler_CreateReader_GZ(t *testing.T) {
	t.Parallel()

	// Create gzip compressed data in memory
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	_, err := gzWriter.Write([]byte("test gzip data"))
	if err != nil {
		t.Fatalf("gzWriter.Write() error = %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("gzWriter.Close() error = %v", err)
	}

	handler := newCompressionHandler(compressionGZ)
	reader, cleanup, err := handler.CreateReader(&buf)
	if err != nil {
		t.Fatalf("CreateReader() error = %v", err)
	}
	defer func() { _ = cleanup() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != "test gzip data" {
		t.Errorf("got %q, want %q", string(data), "test gzip data")
	}
}

func TestCompressionHandler_CreateReader_GZ_Invalid(t *testing.T) {
	t.Parallel()

	handler := newCompressionHandler(compressionGZ)
	input := strings.NewReader("not gzip data")

	_, _, err := handler.CreateReader(input)
	if err == nil {
		t.Fatal("expected error for invalid gzip data")
	}
}

func TestCompressionHandler_CreateReader_ZSTD(t *testing.T) {
	t.Parallel()

	// Create zstd compressed data in memory
	var buf bytes.Buffer
	encoder, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("zstd.NewWriter() error = %v", err)
	}
	_, err = encoder.Write([]byte("test zstd data"))
	if err != nil {
		t.Fatalf("encoder.Write() error = %v", err)
	}
	if err := encoder.Close(); err != nil {
		t.Fatalf("encoder.Close() error = %v", err)
	}

	handler := newCompressionHandler(compressionZSTD)
	reader, cleanup, err := handler.CreateReader(&buf)
	if err != nil {
		t.Fatalf("CreateReader() error = %v", err)
	}
	defer func() { _ = cleanup() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != "test zstd data" {
		t.Errorf("got %q, want %q", string(data), "test zstd data")
	}
}

func TestCompressionHandler_CreateReader_Unknown(t *testing.T) {
	t.Parallel()

	handler := newCompressionHandler(compressionType(99))
	input := strings.NewReader("test data")

	_, _, err := handler.CreateReader(input)
	if err == nil {
		t.Fatal("expected error for unknown compression type")
	}
}

func TestCompressionFactory_CreateReaderForFile(t *testing.T) {
	t.Parallel()

	factory := newCompressionFactory()

	// Test with uncompressed CSV file
	t.Run("uncompressed csv", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join("testdata", "sample.csv")
		reader, cleanup, err := factory.createReaderForFile(path)
		if err != nil {
			t.Fatalf("createReaderForFile() error = %v", err)
		}
		defer func() { _ = cleanup() }()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if !strings.Contains(string(data), "name,email,age") {
			t.Error("expected CSV header in data")
		}
	})

	// Test with gzip compressed file
	t.Run("gzip csv", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join("testdata", "sample.csv.gz")
		reader, cleanup, err := factory.createReaderForFile(path)
		if err != nil {
			t.Fatalf("createReaderForFile() error = %v", err)
		}
		defer func() { _ = cleanup() }()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if !strings.Contains(string(data), "name,email,age") {
			t.Error("expected CSV header in data")
		}
	})

	// Test with bzip2 compressed file
	t.Run("bzip2 csv", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join("testdata", "sample.csv.bz2")
		reader, cleanup, err := factory.createReaderForFile(path)
		if err != nil {
			t.Fatalf("createReaderForFile() error = %v", err)
		}
		defer func() { _ = cleanup() }()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if !strings.Contains(string(data), "name,email,age") {
			t.Error("expected CSV header in data")
		}
	})

	// Test with xz compressed file
	t.Run("xz csv", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join("testdata", "sample.csv.xz")
		reader, cleanup, err := factory.createReaderForFile(path)
		if err != nil {
			t.Fatalf("createReaderForFile() error = %v", err)
		}
		defer func() { _ = cleanup() }()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if !strings.Contains(string(data), "name,email,age") {
			t.Error("expected CSV header in data")
		}
	})

	// Test with zstd compressed file
	t.Run("zstd csv", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join("testdata", "sample.csv.zst")
		reader, cleanup, err := factory.createReaderForFile(path)
		if err != nil {
			t.Fatalf("createReaderForFile() error = %v", err)
		}
		defer func() { _ = cleanup() }()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if !strings.Contains(string(data), "name,email,age") {
			t.Error("expected CSV header in data")
		}
	})

	// Test with non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join("testdata", "non_existent.csv")
		_, _, err := factory.createReaderForFile(path)
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}
	})
}

func TestCompressionFactory_CreateReaderFromReader(t *testing.T) {
	t.Parallel()

	factory := newCompressionFactory()

	t.Run("uncompressed", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader("test data")
		reader, cleanup, err := factory.createReaderFromReader(input, compressionNone)
		if err != nil {
			t.Fatalf("createReaderFromReader() error = %v", err)
		}
		defer func() { _ = cleanup() }()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if string(data) != "test data" {
			t.Errorf("got %q, want %q", string(data), "test data")
		}
	})

	t.Run("gzip", func(t *testing.T) {
		t.Parallel()
		// Create gzip compressed data
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		_, err := gzWriter.Write([]byte("compressed data"))
		if err != nil {
			t.Fatalf("gzWriter.Write() error = %v", err)
		}
		if err := gzWriter.Close(); err != nil {
			t.Fatalf("gzWriter.Close() error = %v", err)
		}

		reader, cleanup, err := factory.createReaderFromReader(&buf, compressionGZ)
		if err != nil {
			t.Fatalf("createReaderFromReader() error = %v", err)
		}
		defer func() { _ = cleanup() }()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if string(data) != "compressed data" {
			t.Errorf("got %q, want %q", string(data), "compressed data")
		}
	})
}

func TestCompressionFactory_CreateReaderForFile_InvalidGzip(t *testing.T) {
	t.Parallel()

	// Create a temporary file with invalid gzip content
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.csv.gz")
	if err := os.WriteFile(tmpFile, []byte("not gzip data"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	factory := newCompressionFactory()
	_, _, err := factory.createReaderForFile(tmpFile)
	if err == nil {
		t.Fatal("expected error for invalid gzip file")
	}
}

package fileprep

import (
	"io"
	"testing"
)

func TestStream_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		originalFormat FileType
		wantFormat     FileType
	}{
		{"CSV", FileTypeCSV, FileTypeCSV},
		{"CSV gzip returns CSV", FileTypeCSVGZ, FileTypeCSV},
		{"TSV bzip2 returns TSV", FileTypeTSVBZ2, FileTypeTSV},
		{"XLSX zstd returns XLSX", FileTypeXLSXZSTD, FileTypeXLSX},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newStream([]byte("test data"), tt.originalFormat)

			if got := s.Format(); got != tt.wantFormat {
				t.Errorf("Format() = %v, want %v", got, tt.wantFormat)
			}

			if got := s.OriginalFormat(); got != tt.originalFormat {
				t.Errorf("OriginalFormat() = %v, want %v", got, tt.originalFormat)
			}
		})
	}
}

func TestStream_Read(t *testing.T) {
	t.Parallel()

	data := []byte("hello, world")
	s := newStream(data, FileTypeCSV)

	// Read all data
	result, err := io.ReadAll(s)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if string(result) != string(data) {
		t.Errorf("Read() = %q, want %q", result, data)
	}
}

func TestStream_Seek(t *testing.T) {
	t.Parallel()

	data := []byte("hello, world")
	s := newStream(data, FileTypeCSV)

	// Read all
	if _, err := io.ReadAll(s); err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Seek to beginning
	pos, err := s.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek() error = %v", err)
	}
	if pos != 0 {
		t.Errorf("Seek() pos = %d, want 0", pos)
	}

	// Read again
	result, err := io.ReadAll(s)
	if err != nil {
		t.Fatalf("ReadAll() after Seek error = %v", err)
	}
	if string(result) != string(data) {
		t.Errorf("After Seek, Read() = %q, want %q", result, data)
	}
}

func TestStream_Len(t *testing.T) {
	t.Parallel()

	data := []byte("hello")
	s := newStream(data, FileTypeCSV)

	if got := s.Len(); got != len(data) {
		t.Errorf("Len() = %d, want %d", got, len(data))
	}

	// Read some bytes
	buf := make([]byte, 2)
	if _, err := s.Read(buf); err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if got := s.Len(); got != len(data)-2 {
		t.Errorf("After read, Len() = %d, want %d", got, len(data)-2)
	}
}

package fileprep

import (
	"bytes"
	"io"
)

// Stream represents a preprocessed data stream with format information.
// It implements io.Reader and provides metadata about the original file format.
type Stream interface {
	io.Reader
	// Format returns the base file type of the stream (without compression)
	Format() FileType
	// OriginalFormat returns the original file type including compression
	OriginalFormat() FileType
}

// stream implements the Stream interface
type stream struct {
	reader         *bytes.Reader
	format         FileType
	originalFormat FileType
}

// newStream creates a new Stream from data and format information
func newStream(data []byte, originalFormat FileType) *stream {
	return &stream{
		reader:         bytes.NewReader(data),
		format:         originalFormat.BaseType(),
		originalFormat: originalFormat,
	}
}

// Read implements io.Reader
func (s *stream) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
}

// Format returns the base file type (CSV, TSV, etc. without compression)
func (s *stream) Format() FileType {
	return s.format
}

// OriginalFormat returns the original file type including compression info
func (s *stream) OriginalFormat() FileType {
	return s.originalFormat
}

// Seek implements io.Seeker for rewinding the stream
func (s *stream) Seek(offset int64, whence int) (int64, error) {
	return s.reader.Seek(offset, whence)
}

// Len returns the number of bytes of the unread portion of the stream
func (s *stream) Len() int {
	return s.reader.Len()
}

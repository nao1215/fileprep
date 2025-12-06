package fileprep

import (
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

// compressionType represents the type of compression
type compressionType int

const (
	// compressionNone represents no compression
	compressionNone compressionType = iota
	// compressionGZ represents gzip compression
	compressionGZ
	// compressionBZ2 represents bzip2 compression
	compressionBZ2
	// compressionXZ represents xz compression
	compressionXZ
	// compressionZSTD represents zstd compression
	compressionZSTD
)

// String returns the string representation of the compressionType
func (ct compressionType) String() string {
	switch ct {
	case compressionNone:
		return "none"
	case compressionGZ:
		return "gzip"
	case compressionBZ2:
		return "bzip2"
	case compressionXZ:
		return "xz"
	case compressionZSTD:
		return "zstd"
	default:
		return "unknown"
	}
}

// Extension returns the file extension for the compressionType
func (ct compressionType) Extension() string {
	switch ct {
	case compressionGZ:
		return extGZ
	case compressionBZ2:
		return extBZ2
	case compressionXZ:
		return extXZ
	case compressionZSTD:
		return extZSTD
	default:
		return ""
	}
}

// compressionHandler defines the interface for handling file compression/decompression
type compressionHandler interface {
	// CreateReader wraps an io.Reader with a decompression reader if needed
	CreateReader(reader io.Reader) (io.Reader, func() error, error)
	// Extension returns the file extension for this compression type
	Extension() string
}

// compressionHandlerImpl implements the compressionHandler interface
type compressionHandlerImpl struct {
	ct compressionType
}

// CreateReader creates a decompression reader based on the compression type
func (h *compressionHandlerImpl) CreateReader(reader io.Reader) (io.Reader, func() error, error) {
	switch h.ct {
	case compressionNone:
		return reader, func() error { return nil }, nil

	case compressionGZ:
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return gzReader, gzReader.Close, nil

	case compressionBZ2:
		return bzip2.NewReader(reader), func() error { return nil }, nil

	case compressionXZ:
		xzReader, err := xz.NewReader(reader)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create xz reader: %w", err)
		}
		return xzReader, func() error { return nil }, nil

	case compressionZSTD:
		decoder, err := zstd.NewReader(reader)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create zstd reader: %w", err)
		}
		return decoder, func() error {
			decoder.Close()
			return nil
		}, nil

	default:
		return nil, nil, fmt.Errorf("unsupported compression type: %v", h.ct)
	}
}

// Extension returns the file extension for this compression type
func (h *compressionHandlerImpl) Extension() string {
	return h.ct.Extension()
}

// newCompressionHandler creates a new compression handler for the given compression type
func newCompressionHandler(ct compressionType) compressionHandler {
	return &compressionHandlerImpl{
		ct: ct,
	}
}

// compressionFactory provides factory methods for compression handling
type compressionFactory struct{}

// newCompressionFactory creates a new compression factory
func newCompressionFactory() *compressionFactory {
	return &compressionFactory{}
}

// detectCompressionType detects the compression type from a file path
func (f *compressionFactory) detectCompressionType(path string) compressionType {
	path = strings.ToLower(path)

	switch {
	case strings.HasSuffix(path, extGZ):
		return compressionGZ
	case strings.HasSuffix(path, extBZ2):
		return compressionBZ2
	case strings.HasSuffix(path, extXZ):
		return compressionXZ
	case strings.HasSuffix(path, extZSTD):
		return compressionZSTD
	default:
		return compressionNone
	}
}

// createHandlerForFile creates an appropriate compression handler for a given file path
func (f *compressionFactory) createHandlerForFile(path string) compressionHandler {
	ct := f.detectCompressionType(path)
	return newCompressionHandler(ct)
}

// createReaderForFile opens a file and returns a reader that handles decompression
func (f *compressionFactory) createReaderForFile(path string) (io.Reader, func() error, error) {
	file, err := os.Open(path) //nolint:gosec // User-provided path is necessary for file operations
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	handler := f.createHandlerForFile(path)
	reader, cleanup, err := handler.CreateReader(file)
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}

	compositeCleanup := func() error {
		var cleanupErr error
		if cleanup != nil {
			cleanupErr = cleanup()
		}
		if closeErr := file.Close(); closeErr != nil && cleanupErr == nil {
			cleanupErr = closeErr
		}
		return cleanupErr
	}

	return reader, compositeCleanup, nil
}

// createReaderFromReader wraps an io.Reader with decompression based on compression type
func (f *compressionFactory) createReaderFromReader(reader io.Reader, ct compressionType) (io.Reader, func() error, error) {
	handler := newCompressionHandler(ct)
	return handler.CreateReader(reader)
}

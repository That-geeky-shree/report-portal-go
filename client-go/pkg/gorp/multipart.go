package gorp

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
)

var (
	errMultipartFilename = errors.New("multipart filename must not be empty")
	errNoMultipartFile   = errors.New("multipart file must not be nil")
)

// Multipart is implemented by types that can supply a file part for a multipart log upload.
type Multipart interface {
	Load() (fileName, contentType string, reader io.Reader, err error)
}

// FileMultipart wraps an *os.File as a Multipart source.
type FileMultipart struct {
	*os.File
}

func (fm *FileMultipart) Load() (fileName, contentType string, reader io.Reader, err error) {
	if fm.File == nil {
		return "", "", nil, errNoMultipartFile
	}
	fName := fm.File.Name()
	if _, sErr := os.Stat(fName); os.IsNotExist(sErr) {
		return "", "", nil, fmt.Errorf("file %s does not exist: %w", fName, errNoMultipartFile)
	}
	// Rewind so that callers (including retry loops) always read from the start.
	if _, sErr := fm.File.Seek(0, io.SeekStart); sErr != nil {
		return "", "", nil, fmt.Errorf("file %s: seek failed: %w", fName, sErr)
	}
	contentType = mime.TypeByExtension(filepath.Ext(fName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return filepath.Base(fName), contentType, fm.File, nil
}

// ReaderMultipart wraps an io.Reader as a Multipart source.
// FileName must match the Attachment.Name field in the corresponding SaveLogRQ.
type ReaderMultipart struct {
	FileName    string
	ContentType string
	io.Reader
}

func (fm *ReaderMultipart) Load() (fileName, contentType string, reader io.Reader, err error) {
	if fm.FileName == "" {
		return "", "", nil, errMultipartFilename
	}
	if fm.Reader == nil {
		return "", "", nil, fmt.Errorf("ReaderMultipart %q: reader must not be nil", fm.FileName)
	}
	return fm.FileName, fm.ContentType, fm, nil
}

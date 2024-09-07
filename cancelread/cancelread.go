package cancelread

import (
	"context"
	"io"
	"io/fs"
	"log"
	"os"
)

type Reader struct {
	Path   string
	file   *os.File
	Info   fs.FileInfo
	Ctx    context.Context
	reader io.Reader
	cancel context.CancelFunc
}

func New(path string) *Reader {
	// open file
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return nil
	}

	// get file info
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		return nil
	}

	// create context
	ctx, cancel := context.WithCancel(context.Background())

	return &Reader{
		Path:   path,
		file:   file,
		Info:   info,
		Ctx:    ctx,
		reader: file,
		cancel: cancel,
	}
}

func (cr *Reader) Read(p []byte) (int, error) {
	select {
	case <-cr.Ctx.Done():
		return 0, io.ErrClosedPipe
	default:
		n, err := cr.reader.Read(p)
		if err != nil && err != io.EOF {
			log.Printf("Error reading: %v", err)
			return n, err
		}
		return n, io.EOF
	}
}

func (cr *Reader) Seek(offset int64, whence int) (int64, error) {
	select {
	case <-cr.Ctx.Done():
		return 0, io.ErrClosedPipe
	default:
		if _, err := cr.reader.(io.Seeker).Seek(offset, whence); err != nil {
			log.Printf("Error seeking: %v", err)
			return 0, err
		}
		return cr.reader.(io.Seeker).Seek(0, 1) // Reset position to end
	}
}

func (cr *Reader) Cancel() error {
	cr.cancel()
	err := cr.file.Close()
	if err != nil {
		log.Printf("Error closing file: %v", err)
	}
	cr.reader = nil
	cr.file = nil
	return err
}

// verify that cancelableReader implements io.ReadSeeker's interface
var _ io.ReadSeeker = (*Reader)(nil)

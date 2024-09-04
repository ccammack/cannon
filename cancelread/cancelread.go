package cancelread

import (
	"context"
	"fmt"
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
	// active bool
}

func New(path string) *Reader {
	log.Println("reader.New()")

	// open file
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return nil
	}

	// get file info
	info, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
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
		// active: true,
	}
}

func (cr *Reader) Read(p []byte) (int, error) {
	log.Println("reader.Read()")
	// if !cr.active {
	// 	return 0, io.ErrClosedPipe
	// }

	select {
	case <-cr.Ctx.Done():
		fmt.Println("read cancelled")
		return 0, io.ErrClosedPipe
	default:
		log.Println("read...")
		n, err := cr.reader.Read(p)
		if err != nil && err != io.EOF {
			log.Printf("error reading: %v", err)
			return n, err
		}
		return n, io.EOF
	}
}

func (cr *Reader) Seek(offset int64, whence int) (int64, error) {
	log.Println("reader.Seek()")
	// if !cr.active {
	// 	return 0, io.ErrClosedPipe
	// }

	select {
	case <-cr.Ctx.Done():
		log.Println("seek cancelled")
		return 0, io.ErrClosedPipe
	default:
		log.Println("seek...")
		if _, err := cr.reader.(io.Seeker).Seek(offset, whence); err != nil {
			log.Printf("error seeking: %v", err)
			return 0, err
		}
		return cr.reader.(io.Seeker).Seek(0, 1) // Reset position to end
	}
}

func (cr *Reader) Cancel() error {
	log.Println("reader.Cancel()")
	// cr.active = false
	cr.cancel()
	err := cr.file.Close()
	if err != nil {
		log.Printf("error closing file: %v", err)
	}
	return err
}

// verify that cancelableReader implements io.ReadSeeker's interface
var _ io.ReadSeeker = (*Reader)(nil)

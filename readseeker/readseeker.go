package readseeker

// mmap-backed io.readSeeker with a Cancel() function

import (
	"context"
	"io"
	"io/fs"
	"log"
	"os"

	"github.com/edsrzf/mmap-go"
)

type ReadSeeker struct {
	Info   fs.FileInfo
	mmap   mmap.MMap
	pos    int64
	ctx    context.Context
	cancel context.CancelFunc
}

func New(path string) *ReadSeeker {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("error opening file: %v", err)
		return nil
	}
	defer file.Close()

	info, err := os.Stat(path)
	if err != nil {
		log.Printf("error getting file info: %v", err)
		return nil
	}

	m, err := mmap.Map(file, mmap.RDONLY, 0)
	if err != nil {
		// log.Printf("error creating memory map: %v", err)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ReadSeeker{
		Info:   info,
		mmap:   m,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (rs *ReadSeeker) Read(p []byte) (int, error) {
	// log.Println("read", len(p))

	if rs.ctx.Err() != nil {
		// log.Println("read canceled")
		return 0, io.ErrClosedPipe
	}

	select {
	case <-rs.ctx.Done():
		// log.Println("read canceled")
		return 0, io.ErrClosedPipe
	default:
		n := copy(p, rs.mmap[rs.pos:rs.pos+int64(len(p))])
		rs.pos += int64(n)
		return n, nil
	}
}

func (rs *ReadSeeker) Seek(offset int64, whence int) (int64, error) {
	// log.Println("seek")

	if rs.ctx.Err() != nil {
		// log.Println("seek canceled")
		return 0, io.ErrClosedPipe
	}

	select {
	case <-rs.ctx.Done():
		// log.Println("seek canceled")
		return 0, io.ErrClosedPipe
	default:
		switch whence {
		case io.SeekStart:
			rs.pos = offset
		case io.SeekCurrent:
			rs.pos += offset
		case io.SeekEnd:
			rs.pos = rs.Info.Size() - offset
		}
		if rs.pos < 0 || rs.pos > rs.Info.Size() {
			return 0, io.EOF
		}
		return rs.pos, nil
	}
}

func (rs *ReadSeeker) Cancel() {
	rs.cancel()
	if err := rs.mmap.Unmap(); err != nil {
		log.Printf("error closing mmap: %v", err)
	}
}

// func (rs *ReadSeeker) Timeout(t int) *ReadSeeker {
// 	go func() {
// 		time.Sleep(time.Duration(t) * time.Second)
// 		log.Println("timeout")
// 		rs.Cancel()
// 	}()
// 	return rs
// }

// verify that this implements io.ReadSeeker's interface
var _ io.ReadSeeker = (*ReadSeeker)(nil)

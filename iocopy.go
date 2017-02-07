package rsync

import (
	"io"
	"strings"
)

// copyBuffer is the actual implementation of Copy and CopyBuffer.
// if buf is nil, one is allocated.
// from go stdlib with added support for EAGAIN
func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}
	if buf == nil {
		buf = make([]byte, 32*1024)
	}
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			var nw int
			var ew error
			for {
				nw, ew = dst.Write(buf[0:nr])
				if nw > 0 {
					written += int64(nw)
				}
				if ew != nil && strings.Contains(ew.Error(), "resource temporarily unavailable") {
					continue
				}
				break
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	return written, err
}

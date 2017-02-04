package rsync

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	RegisterPrimaryInit()
}

func testTransfer(dir bool, mode bool, t *testing.T) {
	tmpSender, err := ioutil.TempDir("", "gorsync-sender")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpSender)

	tmpReceiver, err := ioutil.TempDir("", "gorsync-receiver")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpReceiver)

	srcFile := filepath.Join(tmpSender, "testfile")
	destFile := filepath.Join(tmpReceiver, "testfile")
	realDestFile := destFile

	err = ioutil.WriteFile(srcFile, []byte("foobar"), 0600)
	assert.NoError(t, err)

	if dir {
		srcFile = tmpSender + "/"
		destFile = tmpReceiver
	}

	c1, c2 := sockPair()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		err := Push(Secondary, c1, srcFile, TransferOpt{})
		assert.NoError(t, err)
		wg.Done()
	}()

	go func() {
		err := Pull(Primary, c2, destFile, TransferOpt{})
		assert.NoError(t, err)
		wg.Done()
	}()

	wg.Wait()

	dt, err := ioutil.ReadFile(realDestFile)
	assert.NoError(t, err)
	assert.Equal(t, string(dt), "foobar")

}

func TestTransferDirectoryPriSec(t *testing.T) {
	testTransfer(true, true, t)
}

func TestTransferDirectorySecPri(t *testing.T) {
	testTransfer(true, false, t)
}

func TestTransferFilePriSec(t *testing.T) {
	testTransfer(true, true, t)
}
func TestTransferFileSecPri(t *testing.T) {
	testTransfer(true, false, t)
}

func sockPair() (io.ReadWriteCloser, io.ReadWriteCloser) {
	type fakeConn struct {
		io.Reader
		io.Writer
		io.Closer
	}

	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()
	c1 := &fakeConn{Reader: pr1, Writer: pw2, Closer: pw2}
	c2 := &fakeConn{Reader: pr2, Writer: pw1, Closer: pw1}
	return c1, c2
}

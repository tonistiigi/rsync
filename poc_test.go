package rsync

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	if os.Getenv("REEXEC_TestSendReceive") == "1" && len(os.Args) > 2 && os.Args[2] == "rsync" {
		args := os.Args[3:]
		recv := exec.Command("rsync", args...)
		ioutil.WriteFile("/tmp/cmdline", []byte(fmt.Sprintf("%+v %+v", recv, os.Environ())), 0600)
		recv.Stdin = os.Stdin
		recv.Stdout = os.Stdout
		err := recv.Run()
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}
}

func TestSendReceivePOC(t *testing.T) {
	tmpSender, err := ioutil.TempDir("", "gorsync-sender")
	assert.NoError(t, err)

	tmpReceiver, err := ioutil.TempDir("", "gorsync-receiver")
	assert.NoError(t, err)

	testFile := filepath.Join(tmpSender, "testfile")
	destFile := filepath.Join(tmpReceiver, "testfile")
	err = ioutil.WriteFile(testFile, []byte("foobar"), 0600)
	assert.NoError(t, err)

	fmt.Println(destFile)

	sender := exec.Command("rsync", "-ar", "-e", os.Args[0], tmpSender+"/", "localhost:"+tmpReceiver+"/")

	sender.Env = append(os.Environ(), "REEXEC_TestSendReceive=1")
	sender.Stderr = os.Stdout
	err = sender.Run()
	assert.NoError(t, err)
}

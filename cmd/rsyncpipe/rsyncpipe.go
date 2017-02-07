package main

import (
	"flag"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/tonistiigi/rsync"
)

func init() {
	rsync.RegisterPrimaryInit()
}

func main() {
	sender := flag.Bool("sender", false, "")
	primary := flag.Bool("primary", false, "")
	flag.Parse()

	if len(flag.Args()) != 1 {
		panic(errors.Errorf("invalid arguments %v", flag.Args()))
	}

	p := flag.Args()[0]

	c := &fakeConn{os.Stdin, os.Stdout, os.Stdout}

	mode := rsync.Primary
	if !*primary {
		mode = rsync.Secondary
	}

	var err error
	switch *sender {
	case true:
		err = rsync.Push(mode, c, p, rsync.TransferOpt{})
	default:
		err = rsync.Pull(mode, c, p, rsync.TransferOpt{})
	}
	if err != nil {
		panic(err)
	}
}

type fakeConn struct {
	io.Reader
	io.Writer
	io.Closer
}

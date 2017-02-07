package rsync

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/tonistiigi/fifo"
)

var primaryInited bool

type TransferOpt struct {
	Delete bool
}

type Mode int

const (
	Primary   Mode = 1
	Secondary Mode = 2
)

func Push(mode Mode, conn io.ReadWriteCloser, source string, opt TransferOpt) error {
	switch mode {
	case Primary:
		return primary(conn, source, opt, true)
	case Secondary:
		return secondary(conn, source, opt, true)
	}
	return errors.New("invalid mode")
}

func Pull(mode Mode, conn io.ReadWriteCloser, destination string, opt TransferOpt) error {
	switch mode {
	case Primary:
		return primary(conn, destination, opt, false)
	case Secondary:
		return secondary(conn, destination, opt, false)
	}
	return errors.New("invalid mode")
}

func primary(conn io.ReadWriteCloser, localFile string, opt TransferOpt, sender bool) error {
	if !primaryInited {
		return errors.New("primary not initialized")
	}

	tmpDir, err := ioutil.TempDir("", "rsyncfifo")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary directory")
	}
	defer os.RemoveAll(tmpDir)
	finPath := filepath.Join(tmpDir, "stdin")
	foutPath := filepath.Join(tmpDir, "stdout")
	fin, err := fifo.OpenFifo(context.Background(), finPath, syscall.O_CREAT|syscall.O_NONBLOCK|syscall.O_RDONLY, 0600)
	if err != nil {
		return errors.Wrapf(err, "failed to create %s", finPath)
	}
	defer fin.Close()
	fout, err := fifo.OpenFifo(context.Background(), foutPath, syscall.O_CREAT|syscall.O_NONBLOCK|syscall.O_WRONLY, 0600)
	if err != nil {
		return errors.Wrapf(err, "failed to create %s", foutPath)
	}
	defer fout.Close()

	args := []string{"rsync", "-ar", "-e", naiveSelf()}
	remote := "localhost:"
	if sender {
		args = append(args, localFile, remote)
	} else {
		args = append(args, remote, localFile)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(os.Environ(),
		"GORSYNC_FIFO_STDIN="+finPath,
		"GORSYNC_FIFO_STDOUT="+foutPath,
		"GORSYNC_INIT_PRIMARY=1",
	)

	go func() {
		io.Copy(fout, conn)
		fout.Close()
	}()
	go io.Copy(conn, fin)
	return cmd.Run()
}

func secondary(conn io.ReadWriteCloser, localFile string, opt TransferOpt, sender bool) error {
	// args := []string{"rsync", "--server", "-logDtpre.iLsfx"}
	args := []string{"rsync", "--server", "-logDtpre.iLsfxC"}

	if sender {
		args = append(args, "--sender", ".", localFile)
	} else {
		args = append(args, ".", localFile)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = conn
	cmd.Stdout = conn
	err := cmd.Start()
	if err != nil {
		return err
	}
	_, err = cmd.Process.Wait()
	conn.Close()
	return err
}

func RegisterPrimaryInit() {
	primaryInited = true
	if os.Getenv("GORSYNC_INIT_PRIMARY") == "1" {
		initPrimary()
	}
}

func initPrimary() {
	if err := runPrimaryInit(); err != nil {
		panic(err)
	}
	os.Exit(0)
}

func runPrimaryInit() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fin, err := fifo.OpenFifo(ctx, os.Getenv("GORSYNC_FIFO_STDIN"), syscall.O_WRONLY, 0600)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", os.Getenv("GORSYNC_FIFO_STDIN"))
	}
	fout, err := fifo.OpenFifo(ctx, os.Getenv("GORSYNC_FIFO_STDOUT"), syscall.O_RDONLY, 0600)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", os.Getenv("GORSYNC_FIFO_STDOUT"))
	}

	errs := make(chan error)
	go func() {
		_, err = io.Copy(fin, os.Stdin)
		if err != nil {
			errs <- err
		}
	}()
	go func() {
		_, err = io.Copy(os.Stdout, fout)
		errs <- err
	}()

	if err := <-errs; err != nil {
		return err
	}
	return nil
}

func naiveSelf() string {
	name := os.Args[0]
	if filepath.Base(name) == name {
		if lp, err := exec.LookPath(name); err == nil {
			return lp
		}
	}
	// handle conversion of relative paths to absolute
	if absName, err := filepath.Abs(name); err == nil {
		return absName
	}
	// if we couldn't get absolute name, return original
	// (NOTE: Go only errors on Abs() if os.Getwd fails)
	return name
}

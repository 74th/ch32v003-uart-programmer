package ch32v003uartprogrammer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/goburrow/serial"
)

const (
	syncHead1  byte = 0x57
	syncHead2  byte = 0xab
	CmdProgram byte = 0x80
	CmdErase   byte = 0x81
	CmdVerify  byte = 0x82
	CmdEnd     byte = 0x83

	resSuccess byte = 0x00
	resFail    byte = 0x00
	resEnd     byte = 0x00
)

type flasher struct {
	uart      serial.Port
	firmware  *bytes.Buffer
	buf       *bytes.Buffer
	checkByte byte
}

func (f *flasher) writeCmd(cmd byte, length byte, arg1 byte, arg2 byte) {
	f.buf.WriteByte(syncHead1)
	f.buf.WriteByte(syncHead2)
	f.appendByteWithCheckByte(cmd)
	f.appendByteWithCheckByte(length)
	f.appendByteWithCheckByte(arg1)
	f.appendByteWithCheckByte(arg2)
}

func (f *flasher) appendByteWithCheckByte(b byte) {
	f.buf.WriteByte(b)
	f.checkByte += b
}

func (f *flasher) flash(ctx context.Context) error {
	var (
		err       error
		writeSize int64
	)

	f.buf.WriteByte(f.checkByte)
	bufLength := f.buf.Len()
	f.checkByte = 0x00

	done := make(chan struct{})

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	go func() {
		writeSize, err = io.Copy(f.uart, f.buf)
		done <- struct{}{}
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return fmt.Errorf("UART write timeout")
	}
	if err != nil && err != io.EOF {
		return fmt.Errorf("cannot write UART: %w", err)
	}
	if int64(bufLength) != writeSize {
		return fmt.Errorf("cannot write all buffer")
	}

	return nil
}

func (f *flasher) receiveMessage(ctx context.Context) error {
	var (
		buf  [2]byte
		err  error
		size int
	)

	done := make(chan struct{})

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	go func() {
		size, err = f.uart.Read(buf[:])
		done <- struct{}{}
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return fmt.Errorf("UART response timeout")
	}

	if err == io.EOF {
		return fmt.Errorf("UART response closed: %w", err)
	}

	if size != 2 {
		return fmt.Errorf("cannot get response")
	}

	if buf[0] != resSuccess {
		return fmt.Errorf("response status is not success: %d", buf[0])
	}

	return nil
}

func (f *flasher) erase(ctx context.Context) error {
	fmt.Print("Erase...")

	f.writeCmd(CmdErase, 2, 0, 0)
	err := f.flash(ctx)
	if err != nil {
		return fmt.Errorf("erase error: %w", err)
	}

	err = f.receiveMessage(ctx)
	if err != nil {
		return fmt.Errorf("erase error: %w", err)
	}

	fmt.Println("done")

	return nil
}

func (f *flasher) program(ctx context.Context, isVerify bool) error {
	cmd := CmdProgram
	cmdString := "write"
	if isVerify {
		cmd = CmdVerify
		cmdString = "verify"
	}

	fmt.Printf("%s...", cmdString)

	pos := 0

	for f.firmware.Len() > 0 {

		size := 0x3c

		if f.firmware.Len() < size {
			size = f.firmware.Len()
		}

		f.writeCmd(cmd, byte(size), byte(pos%256), byte(pos/256))

		for i := 0; i < size; i++ {
			b, err := f.firmware.ReadByte()
			if err == io.EOF {
				break
			}
			f.appendByteWithCheckByte(b)
		}

		err := f.flash(ctx)
		if err != nil {
			return fmt.Errorf("%s error: %w", cmdString, err)
		}

		err = f.receiveMessage(ctx)
		if err != nil {
			return fmt.Errorf("%s error: %w", cmdString, err)
		}

		pos += size
	}

	fmt.Println("done")

	return nil
}

func (f *flasher) sendEnd(ctx context.Context) error {
	f.writeCmd(CmdEnd, 2, 0, 0)
	return f.flash(ctx)
}

func OpenUART(ctx context.Context, device string, baud int) (serial.Port, error) {
	if runtime.GOOS == "windows" && strings.HasPrefix(device, "COM") {
		device = "\\\\.\\" + device
	}

	config := serial.Config{
		Address:  device,
		BaudRate: baud,
	}

	uart, err := serial.Open(&config)
	if err != nil {
		return nil, fmt.Errorf("cannot open port: %w", err)
	}

	return uart, nil
}

func (f *flasher) loadFirmware(firmPath string) error {
	fi, err := os.Open(firmPath)
	if err != nil {
		return fmt.Errorf("cannot open firmware: %w", err)
	}
	_, err = io.Copy(f.firmware, fi)
	if err != nil && err != io.EOF {
		return fmt.Errorf("cannot open firmware: %w", err)
	}
	return nil
}

func Program(ctx context.Context, uart serial.Port, firmPath string) error {
	f := flasher{
		firmware: bytes.NewBuffer(nil),
		buf:      bytes.NewBuffer(nil),
		uart:     uart,
	}

	if err := f.loadFirmware(firmPath); err != nil {
		return err
	}

	if f.buf.Len() > 16<<10 {
		return fmt.Errorf("over firmware size: %s", humanize.Comma(int64(f.firmware.Len())))
	}

	if err := f.erase(ctx); err != nil {
		return err
	}

	if err := f.program(ctx, false); err != nil {
		return err
	}

	if err := f.loadFirmware(firmPath); err != nil {
		return err
	}

	if err := f.program(ctx, true); err != nil {
		return err
	}

	if err := f.sendEnd(ctx); err != nil {
		return err
	}

	return nil
}

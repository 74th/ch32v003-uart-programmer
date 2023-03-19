package ch32v003uartprogrammer

import (
	"context"
	"log"
	"testing"

	"github.com/goburrow/serial"
)

type dummySerial struct {
}

func (d *dummySerial) Read(b []byte) (int, error) {
	b[0] = 0x00
	b[1] = 0x00
	log.Printf("read[%d]", len(b))
	return 2, nil
}

func (d *dummySerial) Write(b []byte) (int, error) {
	log.Printf("write[%d]:%x", len(b), b)
	return len(b), nil
}

func (d *dummySerial) Close() error {
	return nil
}

func (d *dummySerial) Open(*serial.Config) error {
	return nil
}

func TestProgram(t *testing.T) {
	uart := new(dummySerial)
	ctx := context.Background()

	err := Program(ctx, uart, "testdata/blink.bin")
	if err != nil {
		t.Errorf("error: %s", err)
	}
}

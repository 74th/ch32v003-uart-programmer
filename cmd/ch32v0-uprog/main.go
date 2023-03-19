package main

import (
	"context"
	"fmt"
	"os"

	prog "github.com/74th/ch32v003-uart-programmer"
	"github.com/jessevdk/go-flags"
)

func main() {

	var opts struct {
		Version bool   `short:"v" long:"version" description:"show version"`
		Device  string `short:"d" long:"device" required:"yes" description:"device such as /dev/ttyUSB0, COM10"`
		Baud    int    `short:"b" long:"baud" default:"115200" description:"baud rate such as 115200, 460800"`
		Args    struct {
			Path string `description:"firmware path firmware.bin"`
		} `positional-args:"yes" required:"yes"`
	}

	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(0)
	}

	if opts.Version {
		fmt.Println("0.1.0")
		os.Exit(0)
	}

	ctx := context.Background()

	uart, err := prog.OpenUART(ctx, opts.Device, opts.Baud)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer uart.Close()

	err = prog.Program(ctx, uart, opts.Args.Path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

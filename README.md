# CH32V003 UART Programmer

This tool need USART IAP Boot-loader.

https://github.com/openwch/ch32v003/tree/main/EVT/EXAM/USART_IAP

## How to use

Follow this document to flash the USART IAP boot-loader using WCH-LinkE hardware and WCH-LinkUtility software.

https://github.com/openwch/ch32v003/blob/main/CH32V003_IAP_Use_Introduction.pdf

When this bootloader is powered up with PC0 high (it may not function satisfactorily with NRST), it is in a state waiting to be written on the UART.

```
ch32v0-uprog --baud 460800 --device /dev/ttyUSB0 firmware.bin
```

## install

```
go install github.com/74th/ch32v003-uart-programmer/cmd/ch32v0-uprog@latest
```

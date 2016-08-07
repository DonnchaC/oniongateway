package main

/*  Entry Proxy for Onion Gateway

See also:

  * https://github.com/DonnchaC/oniongateway/blob/master/docs/design.rst#32-entry-proxy
  * https://gist.github.com/Yawning/bac58e08a05fc378a8cc (SOCKS5 client, Tor)
  * https://habrahabr.ru/post/142527/ (TCP proxy)
*/

import (
	"flag"
	"os"
	"syscall"
	"unsafe"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("roflcoptor")

var logFormat = logging.MustStringFormatter(
	"%{level:.4s} %{id:03x} %{message}",
)
var ttyFormat = logging.MustStringFormatter(
	"%{color}%{time:15:04:05} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}",
)

const ioctlReadTermios = 0x5401

func isTerminal(fd int) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlReadTermios, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

func setupLoggerBackend() logging.LeveledBackend {
	format := logFormat
	if isTerminal(int(os.Stderr.Fd())) {
		format = ttyFormat
	}
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	formatter := logging.NewBackendFormatter(backend, format)
	leveler := logging.AddModuleLevel(formatter)
	leveler.SetLevel(logging.INFO, "roflcoptor")
	return leveler
}

func main() {
	var (
		proxyNet  = flag.String("proxyNet", "tcp", "Proxy network type")
		proxyAddr = flag.String("proxyAddr", "127.0.0.1:9050", "Proxy address")
		listenOn  = flag.String("listen-on", ":443", "Where to listen")
		onionPort = flag.Int("onion-port", 443, "Port on onion site to use")
	)

	flag.Parse()

	logBackend := setupLoggerBackend()
	log.SetBackend(logBackend)

	proxy := NewTLSProxy(*onionPort, *proxyNet, *proxyAddr)
	proxy.Start("tcp", *listenOn)
}

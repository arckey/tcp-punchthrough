package helpers

import "syscall"

var sockOpts = [...]int{
	syscall.SO_REUSEADDR,
	syscall.SO_REUSEPORT,
	syscall.SO_KEEPALIVE,
}

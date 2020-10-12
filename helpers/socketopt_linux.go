package helpers

import "syscall"

var sockOpts = [...]int{
	syscall.SO_REUSEADDR,
	syscall.IP_TTL,
	syscall.TCP_CONNECTIONTIMEOUT,
	syscall.TCP_NODELAY,
}

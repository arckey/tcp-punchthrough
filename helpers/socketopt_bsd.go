package helpers

import "syscall"

var sockOpts = [...]int{
	syscall.SO_REUSEADDR,
	syscall.SO_REUSEPORT,
	syscall.IP_TTL,
	syscall.TCP_CONNECTIONTIMEOUT,
	syscall.TCP_NODELAY,
}

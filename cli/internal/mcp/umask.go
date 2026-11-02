package mcp

import "syscall"

// The admin token file must be private (the script: umask 077 around init-server).
func setUmask077() int     { return syscall.Umask(0o077) }
func restoreUmask(old int) { syscall.Umask(old) }

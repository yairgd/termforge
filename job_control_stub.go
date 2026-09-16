//go:build !linux && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd

package termforge

func stopForShellJobControl() error { return nil }

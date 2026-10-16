package main

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/egoist/mygo"
	"golang.org/x/sys/unix"
)

// detachedEnv marks the Mayoi detach started, which the programs it runs,
// such as its terminals' shells, do not inherit.
const detachedEnv = "MAYOI_DETACHED"

// detach starts Mayoi again in a session of its own, away from the
// terminal it was started from, and reports whether it did: the shell has
// its prompt back at once, and closing the terminal leaves the windows, as
// the mayoi command does with open on macOS. Development builds, and
// MAYOI_FOREGROUND=1, stay in the terminal, which shows what they log.
func detach() bool {
	if os.Getenv(detachedEnv) != "" {
		os.Unsetenv(detachedEnv)
		return false
	}
	if os.Getenv("MAYOI_FOREGROUND") != "" || !mygo.App.IsPackaged() || !onTerminal() {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return false
	}
	defer null.Close()
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), detachedEnv+"=1")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = null, null, null
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return false
	}
	cmd.Process.Release()
	return true
}

// onTerminal reports whether Mayoi was started with a terminal for its
// input or output.
func onTerminal() bool {
	for _, f := range []*os.File{os.Stdin, os.Stdout, os.Stderr} {
		if _, err := unix.IoctlGetTermios(int(f.Fd()), unix.TCGETS); err == nil {
			return true
		}
	}
	return false
}

//go:build !linux

package main

// detach does nothing: on macOS the mayoi command opens the app with open,
// and Windows starts a program with a window of its own away from the
// console.
func detach() bool { return false }

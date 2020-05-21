package cli

import "os"

// Run gojq.
func Run() int {
	return (&Cli{
		InStream:  os.Stdin,
		OutStream: os.Stdout,
		ErrStream: os.Stderr,
	}).run(os.Args[1:])
}

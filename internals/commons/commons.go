package commons

import (
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

var (
	// Private variable to keep a track if output stream has been set once or not.
	oStreamSet = false

	// The main output stream - can be set only once during the lifetime of the
	// application, any output to be sent to the user will be written to this stream.
	outStream io.Writer = nil
)

/*
SetOutput is a simple setter that is designed to be called exactly once during the
lifetime of the application. This method will simply use the parameter as the stream
to which all output messages sent by the application are written.

Note: Any attempts to call this function more than once will result in a crash
*/
func SetOutput(stream io.Writer) {
	if !oStreamSet {
		oStreamSet = true
		outStream = stream
	} else if oStreamSet {
		// Force-stop
		log.Warnf(
			"(commons/SetOutput) attempt to set the value of output stream " +
				"when it has a value already",
		)

		Printf(
			"Error: This error is should not occur. \n\nIf you're seeing this " +
				"message, someone isn't doing their job properly\n\n\t\t(0_0/)\n\n",
		)

		os.Exit(YouAreStupid)
	}
}

/*
GetOutput is a simple getter that returns the private output stream
*/
func GetOutput() io.Writer {
	return outStream
}

/*
Printf is a simple method that acts as a bridge between the application and the user

It is designed to print messages to the console, and provides the same interface as
`fmt.Printf` - providing a layer of abstraction along ease of modification.
*/
func Printf(format string, printable ...interface{}) {
	_, _ = fmt.Fprintf(
		outStream,
		format,
		printable...,
	)
}

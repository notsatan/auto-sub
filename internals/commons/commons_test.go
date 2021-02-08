package commons

import (
	"bytes"
	"os"
	"testing"

	"bou.ke/monkey"
)

func TestGetOutput(t *testing.T) {
	if GetOutput() != outStream {
		t.Errorf(
			"(commons/GetOutput) outStream returned is not the one being used!",
		)
	}
}

func TestSetOutput(t *testing.T) {
	// Fixing the internal variables to isolate tests
	outStream = nil
	oStreamSet = false

	// Set stdout as out-stream
	SetOutput(os.Stdout) // will force-stop the application if this fails

	// If `SetOutput()` is called again, it should attempt to force-stop the application
	detect := false

	defer monkey.Unpatch(os.Exit)
	monkey.Patch(os.Exit, func(code int) {
		if code != YouAreStupid {
			t.Errorf(
				"(commons/SetOutput) unknown exit code received \ncode "+
					"expected: %d \ncode received: %d",
				YouAreStupid,
				code,
			)
		}

		detect = true
	})

	// Attempt to modify the output stream again - should cause a failure
	SetOutput(os.Stderr)
	if !detect {
		t.Errorf(
			"(commons/SetOutput) failed to prevent out-stream from modification"+
				"\noutput stream set: %v",
			oStreamSet,
		)
	}
}

func TestPrintf(t *testing.T) {
	// Create a buffer stream and set is as the output stream
	stream := bytes.NewBufferString("")
	outStream = stream

	// The test message to be used
	msg := "hello, this is a test message"
	Printf(msg)

	if stream.String() != msg {
		t.Errorf(
			"(commons/Printf) message being printed does not match the original"+
				"\noriginal message: \"%s\" \nmessage received: \"%s\"",
			msg,
			stream.String(),
		)
	}
}

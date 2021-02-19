package commons

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
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

	// Test to ensure a call to `Printf` is ignored in case out-stream is null
	outStream = nil
	defer monkey.Unpatch(fmt.Fprintf)
	monkey.Patch(fmt.Fprintf, func(io.Writer, string, ...interface{}) (int, error) {
		t.Errorf(
			"(commons/Printf) running `Printf()` when `outStream` is null!",
		)

		return 0, nil
	})

	Printf("this won't be printed!")
}

func TestStringify(t *testing.T) {
	testdata := ""
	if root, err := os.Getwd(); err != nil {
		t.Errorf(
			"(commons/Stringify) unable to fetch working directory \nerror: %v",
			err,
		)
	} else {
		testdata = filepath.Join(filepath.Dir(filepath.Dir(root)), "testdata")
	}

	// Get a list of files present in testdata
	files, err := ioutil.ReadDir(testdata)
	if err != nil {
		t.Errorf(
			"(commons/Stringify) failed to read contents of testdata \nerror: %v",
			err,
		)
	} else if len(files) < 2 {
		t.Errorf(
			"(commons/Stringify) testdata dir does not contain enough files!"+
				"files found: %+v",
			files,
		)
	}

	// Match the output returned by the function
	if inp := []os.FileInfo{files[0]}; fmt.Sprintf(
		`["%s"]`,
		files[0].Name(),
	) != Stringify(&inp) {
		t.Errorf(
			"(commons/Stringify) unexpected output for single file input! "+
				"\noutput received: `%s`\noutput expected: `%s`",
			Stringify(&inp),
			fmt.Sprintf(`["%s"]`, files[0].Name()),
		)
	}

	if inp := []os.FileInfo{}; Stringify(&inp) != "[]" {
		t.Errorf(
			"(commons/Stringify) unexpected result for empty input! "+
				"\nreceived: %s",
			Stringify(&inp),
		)
	}

	expOutput := fmt.Sprintf(`["%s", "%s"]`, files[0].Name(), files[1].Name())
	if inp := []os.FileInfo{files[0], files[1]}; expOutput != Stringify(&inp) {
		t.Errorf(
			"(commons/Stringify) unexpected output for multi-file input "+
				"\nexpected output: `%s` \nresult: `%s",
			expOutput,
			Stringify(&inp),
		)
	}
}

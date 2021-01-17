package internals

import (
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"bou.ke/monkey"
)

/*
Test the method that attempts to fetch the default locations for ffmpeg and ffprobe.

Testing involves ensuring that the value of both strings remains null in case the method
fails to fetch path to the executables, and also test the scenario when the method is
able to successfully fetch paths to the executables to check the updated value in the
strings
*/
func TestFetchLocation(t *testing.T) {
	// Important
	defer monkey.UnpatchAll()

	// Testing the situation when `ffmpeg` and `ffprobe` both cannot be found - will
	// have `exec.LookPath` return an error regardless of the input.
	monkey.Patch(exec.LookPath, func(string) (string, error) {
		return "", errors.New("")
	})

	// Patch `os.Exit`, when an incorrect path is used as a test case, `userInput` will
	// detect the same the and force-stop the app - this gets in the way of being able
	// to isolate this function for test. Simply overriding the functionality of
	// `os.Exit` to prevent this.
	monkey.Patch(os.Exit, func(int) {})

	// Running the method - both the variables should contain an error, and the global
	// strings should be empty
	ffmpegPath, ffprobePath := fetchLocation()

	if ffmpegPath != "" || ffprobePath != "" {
		t.Errorf(
			"path to executable is not empty even when not found \n"+
				"ffmpeg: %v \nffprobe: %v",
			ffmpegPath,
			ffprobePath,
		)
	}

	// If the executables are found using `exec.LookPath`, testing to ensure that the
	// value of global variables is also updated.
	const testReturn = "test path"
	monkey.Patch(exec.LookPath, func(input string) (string, error) {
		return testReturn, nil
	})

	monkey.Patch(os.Exit, func(int) {})
	ffmpegPath, ffprobePath = fetchLocation()

	if ffmpegPath != testReturn || ffprobePath != testReturn {
		t.Errorf(
			"function `fetchLocation` fails to update global variable "+
				"\nffprobe: %v \nffmpeg: %v",
			ffmpegPath,
			ffprobePath,
		)
	}
}

func TestExecute(t *testing.T) {
	// Important
	defer monkey.UnpatchAll()

	// Patch the `Execute()` method to throw error - will check if the application force
	// stops with the correct error code or not.
	monkey.PatchInstanceMethod(
		reflect.TypeOf(rootCommand),
		"Execute",
		func(command *cobra.Command) error {
			return errors.New("temporary error")
		},
	)

	monkey.Patch(os.Exit, func(code int) {
		if code != UnexpectedError {
			t.Errorf(
				"unexpected exit code, expected %v found %v",
				UnexpectedError,
				code,
			)
		}
	})

	// Running the method.
	Execute()
}

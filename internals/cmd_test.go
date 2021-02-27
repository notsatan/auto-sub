package internals

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/demon-rem/auto-sub/internals/commons"

	"github.com/spf13/cobra"

	"bou.ke/monkey"
)

/*
TestFetchLocation runs tests on edge-cases for the `FetchLocation` method - throwing an
error if the output deviates from the expected output.

Testing involves ensuring that the value of both strings remains null in case the method
fails to fetch path to the executables, and a test to ensure that the value returned
by the method is correct, i.e. actual path to the executables.
*/
func TestFetchLocation(t *testing.T) {
	/*
		First part of the test - ensure that the function returns empty strings in case
		the executables can't be located. Patch `exec.LookPath` to always throw an error
		to ensure this.
	*/

	defer monkey.Unpatch(exec.LookPath)
	monkey.Patch(exec.LookPath, func(string) (string, error) {
		return "", errors.New("")
	})

	// Prevent direct quits
	defer monkey.Unpatch(os.Exit)
	monkey.Patch(os.Exit, func(int) {})

	// Running the method - because of patch(es), both strings should be empty
	ffmpegPath, ffprobePath := findBinaries()

	// handlerTest fails if either one of the returned values are not empty.
	if ffmpegPath != "" || ffprobePath != "" {
		t.Errorf(
			"(entryPoint/FetchLocation) path to executable not empty "+
				"\nffmpeg: %v \nffprobe: %v",
			ffmpegPath,
			ffprobePath,
		)
	}

	/*
		Second part of the test - check if the function is returning correct values of
		or not.

		Patch `os.LookPath` method to return a fixed value and check the value returned
		by the method against this fixed value.
	*/

	const testReturn = "test path"
	defer monkey.Unpatch(exec.LookPath)
	monkey.Patch(exec.LookPath, func(input string) (string, error) {
		// Return the fixed value regardless of the expected input.
		return testReturn, nil
	})

	// Run the method - both the variables should contain the fixed value
	ffmpegPath, ffprobePath = findBinaries()

	// Fail test if either one of them does not match the fixed value
	if ffmpegPath != testReturn || ffprobePath != testReturn {
		t.Errorf(
			"(entryPoint/FetchLocation) returned value does not match expected "+
				"value. \nexpected: `%v` \nffprobe: `%v` \nffmpeg: `%v`",
			testReturn,
			ffmpegPath,
			ffprobePath,
		)
	}
}

/*
TestExecute runs tests on the Execute method.

Testing involves checking if the `Execute()` method fails, or runs into an error, the
application will be force-stopped with the correct exit code.
*/
func TestExecute(t *testing.T) {
	/*
		First part of the test - check to ensure that the application force-stops in
		case the root command returns an error while running - also check the error
		code being returned.

		Patch the `Execute()` method of the root command to always throw an error.
	*/

	// Generate a root command
	cmd := *cmd // use a copy

	monkey.PatchInstanceMethod(
		reflect.TypeOf(&cmd),
		"Execute",
		func(command *cobra.Command) error {
			return errors.New("temporary error")
		},
	)

	defer monkey.UnpatchInstanceMethod(
		reflect.TypeOf(&cmd),
		"Execute",
	)

	// Patch the exit method to fail in case of an unexpected error code
	defer monkey.Unpatch(os.Exit)
	monkey.Patch(os.Exit, func(code int) {
		if code != commons.UnexpectedError {
			t.Errorf(
				"(entryPoint/Execute) unexpected exit code, expected %v found %v",
				commons.UnexpectedError,
				code,
			)
		}
	})

	// Running the method.
	Execute()
}

func TestStringFlags(t *testing.T) {
	// The functioning of `stringFlags()` involves adding flags and marking them as
	// required if needed; the former doesn't need to be tested (no chance of failure)
	// and the latter can't be tested (API limitations)
	//
	// This test function will simply use patches to imitate failure where needed to
	// improve coverage score - failure can't be tested either since failure handling
	// just involves logging the failure.

	// Template command
	rootCmd := &cobra.Command{}
	input := commons.UserInput{}

	val := "template path"

	for _, in := range []struct {
		ffmpegPath, ffprobePath string
	}{
		{val, ""},
		{"", ""},
		{"", val},
		{val, val},
	} {
		// Reset all flags
		rootCmd.ResetFlags()

		// Run the function
		stringFlags(
			rootCmd,
			&input,
			&in.ffmpegPath,
			&in.ffprobePath,
		)
	}

	defer monkey.UnpatchInstanceMethod(
		reflect.TypeOf(rootCmd),
		"MarkFlagDirname",
	)

	monkey.PatchInstanceMethod(
		reflect.TypeOf(rootCmd),
		"MarkFlagDirname",
		func(*cobra.Command, string) error { return errors.New("test error") },
	)

	defer monkey.UnpatchInstanceMethod(
		reflect.TypeOf(rootCmd),
		"MarkFlagRequired",
	)

	monkey.PatchInstanceMethod(
		reflect.TypeOf(rootCmd),
		"MarkFlagRequired",
		func(*cobra.Command, string) error { return errors.New("testo error") },
	)

	blank := ""

	rootCmd.ResetFlags()
	stringFlags(rootCmd, &input, &blank, &blank)
}

/*
TestHandlerTest checks the handler function that will be run in case the test flag is
used

Testing involves three cases, when either `ffmpeg` or `ffprobe` commands can't be run,
or when both of them can't be run. Checking the output in each of these cases to ensure
that the test handler function runs as expected.

It is expected that the test handler function will return a blank string instead of the
version if fails to fetch the version for any case.
*/
func TestHandlerTest(t *testing.T) {
	/*
		Testing the scenario when attempting to run the commands to fetch versions
		results in a failure - expect to get a blank corresponding string as a result
		for the particular entry.
	*/

	// Temporary command - used to monkey patch instance methods.
	tempCmd := &exec.Cmd{}

	// String containing the version being used for testing - will be used to apply
	// patches and then verify if the method can correctly find the version
	version := "4.31.12"

	// Patch applied in the loop
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(tempCmd), "Output")

	// Iterating through the possibility. Failure to run the command for `ffmpeg`, or
	// `ffprobe` or for both (blank string)
	for _, seq := range []string{
		userInput.FFmpegPath,
		userInput.FFprobePath,
		version, // Value returned only in this case.
		"",
	} {
		// Pin - take a look at https://github.com/kyoh86/scopelint/ for this.
		// Will probably remove this in future. Using this just to pass tests for now.
		seq := seq

		// Applying instance patch such that if `seq` contains an empty string, the
		// method will directly throw an error. Apart from this, if `seq` matches the
		// command path, the method will throw an error.
		//
		// This ensures testing each scenario separately - if either one of the two
		// commands can't be run, or if both fail.
		monkey.PatchInstanceMethod(
			reflect.TypeOf(tempCmd),
			"Output", // Patching the `Output` method to return error.
			func(cmd *exec.Cmd) ([]byte, error) {
				if seq == "" {
					return nil, errors.New("test error")
				} else if cmd.Path == seq {
					return nil, errors.New("test error")
				}

				// Note: The string being returned as result should be such that
				// it matches the regex being used by the function.
				return []byte("test here version " + version + " extra text"), nil
			},
		)

		// Once the patch is applied, running the method and checking the result
		ffmpegVersion, ffprobeVersion := handlerTest()

		msg := ""

		if (seq == "" || seq == userInput.FFmpegPath) && ffmpegVersion != "" {
			// FFmpeg version should be blank.
			msg += fmt.Sprintf(
				"\nmanaged to fetch ffmpeg version instead of error"+
					"\nffmpeg version: %v",
				ffmpegVersion,
			)
		} else if ffmpegVersion != version {
			// Incorrect version detected - possibly due to incorrect regex
			msg += fmt.Sprintf(
				"incorrect ffmpeg version detected! \nexpected version: %v "+
					"\nversion fetched: %v",
				version,
				ffmpegVersion,
			)
		}

		if (seq == "" || seq == userInput.FFprobePath) && ffprobeVersion != "" {
			// FFprobe version should be blank
			msg += fmt.Sprintf(
				"managed to fetch ffprobe version instead of error "+
					"\nffprobe version: %v",
				ffmpegVersion,
			)
		} else if ffprobeVersion != version {
			// Incorrect version detected - possibly due to incorrect regex.
			msg += fmt.Sprintf(
				"incorrect ffprobe version detected \nexpected version: %v "+
					"\ndetected version: %v",
				version,
				ffprobeVersion,
			)
		}
	}
}

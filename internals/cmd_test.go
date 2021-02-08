package internals

import (
	"errors"
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
	cmd := getRootCommand()

	monkey.PatchInstanceMethod(
		reflect.TypeOf(cmd),
		"Execute",
		func(command *cobra.Command) error {
			return errors.New("temporary error")
		},
	)

	defer monkey.UnpatchInstanceMethod(
		reflect.TypeOf(cmd),
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
	cmd := &cobra.Command{}
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
		cmd.ResetFlags()

		// Run the function
		stringFlags(
			cmd,
			&input,
			&in.ffmpegPath,
			&in.ffprobePath,
		)
	}

	defer monkey.UnpatchInstanceMethod(
		reflect.TypeOf(cmd),
		"MarkFlagDirname",
	)

	monkey.PatchInstanceMethod(
		reflect.TypeOf(cmd),
		"MarkFlagDirname",
		func(*cobra.Command, string) error { return errors.New("test error") },
	)

	defer monkey.UnpatchInstanceMethod(
		reflect.TypeOf(cmd),
		"MarkFlagRequired",
	)

	monkey.PatchInstanceMethod(
		reflect.TypeOf(cmd),
		"MarkFlagRequired",
		func(*cobra.Command, string) error { return errors.New("testo error") },
	)

	blank := ""

	cmd.ResetFlags()
	stringFlags(cmd, &input, &blank, &blank)
}

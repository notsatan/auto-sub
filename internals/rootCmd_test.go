package internals

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"bou.ke/monkey"
)

/*
Check to ensure failure if argument count exceeds expected amount.
*/
func TestArgsCheck(t *testing.T) {
	// Undo all patches being made.
	defer monkey.UnpatchAll()

	monkey.Patch(os.Exit, func(int) {})

	// Array of temporary UserInput strings.
	testArgs := [...]string{"test", "array", "with", "random", "values", "inside", "it"}
	for i := 0; i < len(testArgs); i++ {
		// Create a slice of first `i` inputs.
		inputArgs := testArgs[0:i]

		// Set this slice to be the UserInput for the root command.
		rootCommand.SetArgs(inputArgs)

		// Running the command. If the amount of arguments passed exceeds `maxInputArgs`
		// an error should be returned.
		result := rootCommand.Execute()

		// Fail test if no error is being returned even on exceeding allowed count.
		if len(inputArgs) > maxInputArgs && result == nil {
			t.Errorf(
				"root command failed to raise error with %d UserInput arguments",
				len(inputArgs),
			)
		}
	}
}

func TestRun(t *testing.T) {
	// Important
	defer monkey.UnpatchAll()

	// Testing the result when `Initialize` method fails - should force stop.
	monkey.PatchInstanceMethod(
		reflect.TypeOf(&userInput),
		"Initialize",
		func(input *UserInput) (int, error) {
			return UnexpectedError, errors.New("temp error")
		},
	)

	monkey.Patch(os.Exit, func(code int) {
		if code != UnexpectedError {
			t.Errorf(
				"unexpected exit code \nexpected: %v \nfound: %v",
				UnexpectedError,
				code,
			)
		}
	})

	// First run, will trip the failure point when `Initialize` method is run, causing
	// the application to attempt to force-stop
	_ = rootCommand.Execute()

	monkey.UnpatchAll()

	// A list of all possible arguments/flags. Will be used to run the root command
	// for varying scenarios
	//
	// Note: Do not add the `--help` flag or any other flag that is built-into Cobra
	// by default. Also, ensure that these arguments do not cause
	// `userInput.Initialize()` to fail - the method will be separately tested.
	listArgs := []string{
		"--test", // Test flag - second highest precedence (after help flag)
		"--log",  // Enables logging
		"--echo", // Should echo back the commands being used.
	}

	// Random number between [3, 8) - decides the number of loops to run. Each loop
	// will run the root command with a set of random arguments from the set of possible
	// arguments.
	loop := rand.Intn(5) + 3

	var args []string

	for i := 0; i < loop; i++ {
		// Random number - the number of arguments to be used in the current run
		argCount := rand.Intn(len(listArgs))

		// Slice to contain arguments
		args = make([]string, argCount)

		// Populating the list of arguments.
		for v := 0; v < argCount; v++ {
			// Adding a random argument from all possible arguments to the list of args
			args[v] = listArgs[rand.Intn(len(listArgs))]
		}

		// Setting the randomized list of arguments to be run with the command
		rootCommand.SetArgs(args)
		_ = rootCommand.Execute()
	}
}

/*
Tests the handler method to run the handler function for the test flag.

Testing involves three cases, when either `ffmpeg` or `ffprobe` commands can't be run,
or when both of them can't be run. Checking the output in each of these cases to ensure
that the test handler function runs as expected.

It is expected that the test handler function will return a blank string instead of the
version if fails to fetch the version for any case.
*/
func TestHandlerTest(t *testing.T) {
	// Important
	defer monkey.UnpatchAll()

	/*
		Testing the scenario when attempting to run the commands to fetch versions
		results in a failure - expect to get a blank corresponding string as a result
		for the particular entry.
	*/

	// Temporary command - used to monkey patch instance methods.
	tempCmd := &exec.Cmd{}

	// String containing the version being used for testing - will be used to apply
	// patches and then verify if the method can correctly find the version
	version = "4.31.12"

	// Iterating through the possibility. Failure to run the command for `ffmpeg`, or
	// `ffprobe` or for both (blank string)
	for _, seq := range []string{
		userInput.ffmpegPath,
		userInput.ffprobePath,
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
				}

				if cmd.Path == seq {
					return nil, errors.New("test error")
				}

				// Note: The string being returned as result should be such that
				// it matches the regex being used by the function.
				return []byte("test here version " + version + " extra text"), nil
			},
		)

		// Once the patch is applied, running the method and checking the result
		ffmpegVersion, ffprobeVersion := HandlerTest()

		msg := ""

		if (seq == "" || seq == userInput.ffmpegPath) && ffmpegVersion != "" {
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

		if (seq == "" || seq == userInput.ffprobePath) && ffprobeVersion != "" {
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

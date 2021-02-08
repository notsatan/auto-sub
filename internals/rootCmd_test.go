package internals

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/demon-rem/auto-sub/internals/commons"
	"github.com/demon-rem/auto-sub/internals/ffmpeg"

	"bou.ke/monkey"
)

/*
Helper method designed to create a test config for user input that points the root
directory to `testdata` directory - ideal to run tests.
*/
func testConfig(t *testing.T) commons.UserInput {
	// Fetch the path to the test data directory - will be set as the root directory
	// for tests.
	path, err := os.Getwd()
	if err != nil {
		t.Errorf(
			"failed to fetch current working directory \n(traceback): \n%v",
			err,
		)
	} else {
		// If path is fetched successfully, modify it to point to the test directory.
		// This part might need to be modified if this part of the code is moved around
		path = filepath.Join(filepath.Dir(path), "testdata")
	}

	input := commons.UserInput{
		RootPath: path,
	}

	return input
}

/*
Helper method to reset the value of `userInput` global variable - ideally, a deferred
call should be made to this function whenever the value of `userInput` is modified;
would prevent contamination of data across tests.
*/
func resetConfig() {
	userInput = commons.UserInput{}
}

/*
TestArgs runs tests on the root command with argument(s) being passed in.

Testing involves supplying the command with more arguments than required to ensure that
the application force-stops with the correct exit code, ensuring failure in case
incorrect path is being used as the root path and more.
*/
func TestArgsCheck(t *testing.T) {
	/*
		Perform check by using one more argument than needed, i.e. `maxInputArgs+1` or
		more arguments being passed to the command.
	*/

	cmd := getRootCommand()

	defer monkey.Unpatch(os.Exit)
	monkey.Patch(os.Exit, func(int) {})

	// Array of temporary UserInput strings.
	testArgs := [...]string{"test", "array", "with", "random", "values", "inside", "it"}
	for i := maxInputArgs + 1; i < len(testArgs); i++ {
		// Create a slice of first `i` inputs.
		inputArgs := testArgs[0:i]

		// Set this slice to be the UserInput for the root command.
		cmd.SetArgs(inputArgs)

		// Running the command. If the amount of arguments passed exceeds `maxInputArgs`
		// an error should be returned.
		result := cmd.Execute()

		// Fail test if no error is returned
		if result == nil {
			t.Errorf(
				"(rootCmd/Args) root command failed to raise error with %d args",
				len(inputArgs),
			)
		}
	}

	// Using test data directory as the root
	root, err := os.Getwd()
	if err != nil {
		t.Errorf(
			"(rootCmd/Args) failed to fetch current working dir "+
				"\n(traceback): %v\n",
			err,
		)
	} else {
		root = filepath.Join(filepath.Dir(root), "testdata")
	}

	/*
		Ensure failure if the arguments accepted count is simply increased - without
		handling the new argument in the switch block
	*/

	// Increase number of arguments allowed
	const inc = 3
	maxInputArgs += inc

	// Array of (valid) arguments - ensures that the first `maxInputArgs` arguments
	// pass checks, without this, error will be thrown because arguments are invalid
	args := []string{root}

	// Array of additional arguments, any random gibberish goes
	addArgs := [inc]string{"test", "args", "here"}

	// Note: Start the loop with `1`, at `i = 0` new argument(s) will not be added,
	// causing no failure
	for i := 1; i <= inc; i++ {
		// Input for each case - append incorrect arguments after the correct ones
		in := append(
			args,
			addArgs[0:i]...,
		)

		// Running the function to be test - should fail every time
		res := cmd.Args(
			cmd,
			in,
		)

		// Test fails if no error occurs
		if res == nil {
			t.Errorf(
				"(rootCmd/Args) function accepts more arguments than required\n"+
					`args passed: ["%v"]`,
				strings.Join(in, `", "`),
			)
		}
	}

	// Reset the value of the variable once done - will corrupt other tests without this
	maxInputArgs -= inc
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
	version = "4.31.12"

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

/*
TestInitializeFailure runs a test against a singular point of failure - ensuring that
the application quits with the correct exit code in case `userInput.Initialize()` fails,
this will happen in case of incorrect input data.
*/
func TestInitializeFailure(t *testing.T) {
	/*
		Ensure the application is force-stopped if `userInput.Initialize()` fails.
		Mimic this with a patch.
	*/

	userInput = testConfig(t)
	defer resetConfig()

	cmd := getRootCommand()

	defer monkey.Unpatch(ffmpeg.TraverseRoot)
	monkey.Patch(
		ffmpeg.TraverseRoot,
		func(*commons.UserInput, string) {},
	)

	monkey.PatchInstanceMethod(
		reflect.TypeOf(&userInput),
		"Initialize",
		func(input *commons.UserInput) (int, error) {
			return commons.UnexpectedError, errors.New("temp error")
		},
	)

	defer monkey.UnpatchInstanceMethod(
		reflect.TypeOf(&userInput),
		"Initialize",
	)

	defer monkey.Unpatch(os.Exit)
	monkey.Patch(os.Exit, func(code int) {
		if code != commons.UnexpectedError {
			t.Errorf(
				"(rootCmd/RootCommand) unexpected exit code \nexpected: %v "+
					"\nfound: %v",
				commons.UnexpectedError,
				code,
			)
		}
	})

	// Will trip the failure point when `Initialize` method is run
	_ = cmd.PreRunE(cmd, []string{})
}

func TestRun(t *testing.T) {
	// Generate test config
	userInput = testConfig(t)
	defer resetConfig()

	cmd := getRootCommand()

	defer monkey.Unpatch(ffmpeg.TraverseRoot)
	monkey.Patch(
		ffmpeg.TraverseRoot,
		func(*commons.UserInput, string) {},
	)

	/*
		Verify the test flag - patch `handlerTest()` function to ensure isolation.

		Check the exit code used in case the function can return a ver, or if it
		fails to.
	*/

	ver := "v3.2.1" // Version code being returned (if at all)

	// Enable the test flag
	userInput.IsTest = true

	// Create temporary structure to contain two strings, an array of such structures
	// will be used as the values returned by `handlerTest()`, with a new patch being
	// applied with every iteration of the loop.
	for i, res := range []struct{ key, value string }{
		{"", ""},   // Complete failure
		{ver, ""},  // Partial failure
		{"", ver},  // Partial failure
		{ver, ver}, // Success
	} {
		// Applying the patch
		monkey.Patch(handlerTest, func() (string, string) {
			return res.key, res.value
		})

		// Patch `os.Exit()` to check the exit code being used - fail if incorrect.
		monkey.Patch(os.Exit, func(code int) {
			if res.key == "" || res.value == "" {
				if code != commons.ExecNotFound {
					t.Errorf(
						"(rootCmd/RunE) exit code incorrect when executables "+
							"cannot be found.\nexpected code: %v \nfound: %v"+
							"\ninput set: %d",
						commons.ExecNotFound,
						code,
						i,
					)
				}
			} else if code != commons.StatusOK {
				t.Errorf(
					"(rootCmd/RunE) incorrect exit code returned, expected a "+
						"clean exit. \nexit code found: %v\ninput set: %d",
					code,
					i,
				)
			}
		})

		// Finally, run the main method
		if err := cmd.RunE(cmd, []string{}); err != nil {
			t.Errorf(
				"(rootCmd/RunE) fail to run the main method! \nerror: %v",
				err,
			)
		}
	}

	// Undo the patches applied, and disable the test flag
	monkey.Unpatch(handlerTest)
	monkey.Unpatch(os.Exit)
	userInput.IsTest = false

	// nolint
	// Temporary patch - ensure application does not force-stop due to failure in
	// `userInput.Initialize()`
	monkey.PatchInstanceMethod(
		reflect.TypeOf(&userInput),
		"Initialize",
		func(input *commons.UserInput) (int, error) {
			return commons.StatusOK, nil
		},
	)

	// Reset user input
	userInput = testConfig(t)
	defer resetConfig()

	for _, path := range []string{
		userInput.RootPath, // correct path
		"",                 // Blank
		filepath.Join(userInput.RootPath, ".gitkeep"),       // file
		filepath.Join(userInput.RootPath, "incorrect_path"), // incorrect path
	} {
		// Using `path` as root path
		userInput.RootPath = path

		monkey.Patch(os.Exit, func(code int) {
			if (path == "" && code != commons.RootDirectoryIncorrect) ||
				(path != "" && code != commons.UnexpectedError) {
				t.Errorf(
					"(rootCmd/RunE) unexpected exit code found! \nroot path: "+
						"`%v` \nexpected exit code: %v \nexit code found: %v",
					path,
					commons.UnexpectedError,
					code,
				)
			}
		})

		_ = cmd.PreRunE(cmd, []string{})
	}

	monkey.UnpatchInstanceMethod(reflect.TypeOf(&userInput), "Initialize")
	monkey.Unpatch(os.Exit)
}

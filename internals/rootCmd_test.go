package internals

import (
	"testing"

	"os"

	"bou.ke/monkey"
	log "github.com/sirupsen/logrus"
)

func TestExecute(t *testing.T) {
	// Ensures all patches are undone once this function is executed.
	defer monkey.UnpatchAll()

	/*
		Check to ensure the program exits if `performCheck` returns false.
	*/

	// Ensuring failure.
	monkey.Patch(performCheck, func() (bool, string) { return false, "error" })

	// Catch calls being made to `os.Exit` and check the exit code.
	monkey.Patch(os.Exit, func(code int) {
		if code != codeInitialCheckFail {
			t.Errorf("expected code `%d`, got `%d`", codeInitialCheckFail, code)
		}
	})

	Execute()
}

/*
Check to ensure failure if argument count exceeds expected amount.
*/
func TestArgsCheck(t *testing.T) {
	// Undo all patches being made.
	defer monkey.UnpatchAll()

	// Patch the global check to always return true.
	monkey.Patch(performCheck, func() (bool, string) { return true, "" })

	// Array of temporary input strings.
	testArgs := [...]string{"test", "array", "with", "random", "values", "inside", "it"}
	for i := 0; i < len(testArgs); i++ {
		// Create a slice of first `i` inputs.
		inputArgs := testArgs[0:i]

		// Set this slice to be the input for the root command.
		rootCommand.SetArgs(inputArgs)

		// Running the command. If the amount of arguments passed exceeds `maxInputArgs`
		// an error should be returned.
		result := rootCommand.Execute()

		// Fail test if no error is being returned even on exceeding allowed count.
		if len(inputArgs) > maxInputArgs && result == nil {
			t.Errorf(
				"root command failed to raise error with %d input arguments",
				len(inputArgs),
			)
		}
	}
}

func TestFlagLog(t *testing.T) {
	// When test starts, log level will be at `info` (default). Depending upon the input
	// passed to the `flagLog` function, the log level should change. Running a loop
	// over the possible values {`true`, `false`} to check both scenarios.
	for _, value := range []bool{true, false} {
		flagLog(value)

		// If `value` is true, logging is to be enabled, and log level should be at
		// `trace`, conversely, if `value` is false, log level should be at `warn`.
		if logLevel := log.GetLevel(); (value && logLevel != log.TraceLevel) ||
			(!value && logLevel != log.WarnLevel) {
			t.Errorf(
				"failed to modify log level \nInput: %v \nLogLevel: %v",
				value,
				logLevel,
			)
		}
	}
}

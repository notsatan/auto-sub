package internals

import (
	"testing"

	"bou.ke/monkey"
)

/*
Check to ensure failure if argument count exceeds expected amount.
*/
func TestArgsCheck(t *testing.T) {
	// Undo all patches being made.
	defer monkey.UnpatchAll()

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

func TestExecute(t *testing.T) {
	// Ensures all patches are undone once this function is executed.
	defer monkey.UnpatchAll()
}

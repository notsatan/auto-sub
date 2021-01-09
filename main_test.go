package main

import (
	"errors"
	"os"
	"testing"

	"bou.ke/monkey"
	"github.com/demon-rem/auto-sub/internals"
)

/*
TestMainFunction will run tests on the main function - involves moderate usage of
monkey patching to replace actual function with dummy functions.

Choosing this weird name because `TestMain` has a designated purpose, which this
function won't be doing - stupid Go stuff :(
*/
func TestMainFunction(t *testing.T) {
	// Boolean flag to detect if a method is executed.
	flag := false

	// Note:
	// This test function will internally use MonkeyPatches to isolate and run certain
	// tests - take special take to ensure that the functions are un-patched when they
	// are no longer needed.
	defer monkey.UnpatchAll() // This should handle all patches made in this method

	// Replace function call to the execute command with a dummy function.
	monkey.Patch(internals.Execute, func() { flag = true })

	// A blank call to the main method should run successfully - ensuring that the
	// execute command function is run at the end of the test case.
	main()
	if !flag {
		t.Errorf("failed initial run - main")
	}

	// Replacing call to `os.OpenFile` with a template function throwing an error
	monkey.Patch(os.OpenFile, func(_ string, _ int, _ os.FileMode) (*os.File, error) {
		return nil, errors.New("")
	})

	// Initiating call to main
	main() // Error should be handled internally
}

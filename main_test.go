package main

import (
	"errors"
	"os"
	"reflect"
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

	defer monkey.UnpatchAll() // Removes all patches made in this method

	// Replace function call to the execute command with a dummy function.
	monkey.Patch(internals.Execute, func() { flag = true })

	// Blank call to the main method should run successfully - ensuring that the
	// execute command function is run at the end of the test case.
	main()
	if !flag {
		t.Errorf("failed initial run - main")
	}

	// Replacing call to `os.OpenFile` with a template function throwing an error
	monkey.Patch(os.OpenFile, func(string, int, os.FileMode) (*os.File, error) {
		return nil, errors.New("(main/main) test to emulate failure in opening file")
	})

	// Initiating call to main
	main() // Error should be handled internally

	// Unpatch the previous patch
	monkey.Unpatch(os.OpenFile)

	/*
		Emulate scenario if the main method fails to close connection to the log file,
	*/
	var file os.File
	monkey.PatchInstanceMethod(
		reflect.TypeOf(&file),
		"Close",
		func(*os.File) error {
			return errors.New("(main/main) test failure if a file fails to close")
		},
	)

	main()
}

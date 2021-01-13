package internals

import (
	"errors"
	"os/exec"
	"testing"

	"bou.ke/monkey"
)

/*
Tests the abstraction method that adds boolean flags to the root command.

Testing involves ensuring that each key-value pair present in the map is a boolean flag
present in the root command, the root command does not have any flags by default.
*/
func TestSetBoolFlags(t *testing.T) {
	// Emptying root command - ensures previous tests won't cause changes.
	rootCommand = getRootCommand()

	// The root command should not have any flags by default.
	if rootCommand.HasFlags() {
		t.Errorf("default flags found in root command")
	}

	// Running the method directly
	setBoolFlags()

	if !rootCommand.HasFlags() {
		t.Errorf("root command has no flags set")
	}

	// Checking the contents of the map, and the flags present in the root command
	for key := range boolFlags {
		_, err := rootCommand.Flags().GetBool(key)
		if err != nil {
			t.Errorf(
				"could not extract map value from flag \ntraceback: \n%v",
				err,
			)
		}
	}
}

func TestFetchLocation(t *testing.T) {
	// Important
	defer monkey.UnpatchAll()

	// Testing the situation when `ffmpeg` and `ffprobe` both cannot be found - will
	// have `exec.LookPath` return an error regardless of the input.
	monkey.Patch(exec.LookPath, func(string) (string, error) {
		return "", errors.New("")
	})

	// Running the method - both the variables should contain an error, and the global
	// strings should be empty
	err01, err02 := fetchLocation()

	if err01 == nil || err02 == nil {
		t.Errorf("`fetchLocation` failed to return an error")
	}

	if ffmpegLocation != "" || ffprobeLocation != "" {
		t.Errorf(
			"path to executable is not empty even when not found \n"+
				"ffmpeg: %v \nffprobe: %v",
			ffmpegLocation,
			ffprobeLocation,
		)
	}

	// If the executables are found using `exec.LookPath`, testing to ensure that the
	// value of global variables is also updated.
	const testReturn = "test path"
	monkey.Patch(exec.LookPath, func(input string) (string, error) {
		return testReturn, nil
	})

	err01, err02 = fetchLocation()
	if err01 != nil || err02 != nil {
		t.Errorf(
			"ran into an error while attempting to locate executables "+
				"\ntraceback: \n%v\n%v",
			err01,
			err02,
		)
	}

	if ffmpegLocation != testReturn || ffprobeLocation != testReturn {
		t.Errorf(
			"function `fetchLocation` fails to update global variable "+
				"\nffprobe: %v \nffmpeg: %v",
			ffprobeLocation,
			ffmpegLocation,
		)
	}
}

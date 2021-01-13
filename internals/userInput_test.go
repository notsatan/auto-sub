package internals

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"bou.ke/monkey"
	log "github.com/sirupsen/logrus"
)

/*
Global variables, act as static variables. These variables should not be accessed
directly.
*/
var (
	dummyFileInfo os.FileInfo = nil
	dummyDirInfo  os.FileInfo = nil
)

/*
Helper method to get dummy file and directory info.

Designed to fetch info only once. Subsequent runs will directly return the previously
fetched info.

The directory for which info is fetched will be the current working directory, and the
file will be the first file present in the testdata directory.
*/
func getDummies(t *testing.T) (directoryInfo, fileInfo os.FileInfo) {
	if dummyDirInfo != nil && dummyFileInfo != nil {
		return dummyDirInfo, dummyFileInfo
	}

	var cwd string
	var err error

	// Fetching path to current directory and the test directory.
	if cwd, err = os.Getwd(); err != nil {
		t.Errorf("unable to fetch path to current directory! \nError: %v", err)
	}

	testDir := path.Join(path.Dir(cwd), "testdata")

	dummyDirInfo, err = os.Stat(cwd)
	if err != nil {
		t.Errorf("unable to get stats for the current directory \n%v", err)
	}

	// Fetching the first file from test data directory as the temporary file info.
	err = filepath.Walk(
		testDir,
		func(path string, info os.FileInfo, e error) error {
			if e == nil && !info.IsDir() && dummyFileInfo == nil {
				dummyFileInfo = info
			}

			return nil
		},
	)

	// Fail the test if no file is present among test data.
	if err != nil || dummyFileInfo == nil {
		t.Errorf("failed to fetch a file to generate test info \nerror: %v", err)
	}

	return dummyDirInfo, dummyFileInfo
}

/*
Check to ensure that the internal method actually changes the log level.

Testing involves attempting to turn the logger on and off and checking if the log
level is actually being modified when the method is called or not.

Assumes that logging being turned off sets the log level at `Warn` and logging being
enabled sets the log level at `Trace`.
*/
func TestSetLogging(t *testing.T) {
	// Log level will be at `info` (default). Running a loop to detect changes in both
	// scenario's
	for _, value := range []bool{true, false} {
		userInput.SetLogging(value)

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

/*
Test the setter method to set a directory as the root directory.

Testing involves checking that the application is being terminated if incorrect path
is used as the root directory, and if the path supplied points to a file instead of a
directory.
*/
func TestSetRoot(t *testing.T) {
	// Important
	defer monkey.UnpatchAll()

	/*
		This test involves patching the `os.Stat` method, before patching the method,
		fetching info for a directory, and a file - this data will be used as dummy
		values when needed
	*/

	// Fetching dummy values for directory info and file info.
	var dirInfo, fileInfo os.FileInfo = getDummies(t)

	/*
		The first part of test involves causing call to `os.Stat` fail. This should
		result in the application being force-stopped with an error code of
		`RootDirectoryIncorrect` - this will likely occur when a non-existent path is
		passed in as a value.

		Patch call to `os.Stat` to ensure an error is raised, and call to `os.Exit` to
		check the error code being used.
	*/

	// Cause failure regardless of path
	monkey.Patch(os.Stat, func(path string) (os.FileInfo, error) {
		return dirInfo, errors.New("random error")
	})

	// CFail if exit code is unexpected.
	monkey.Patch(os.Exit, func(code int) {
		if code != RootDirectoryIncorrect {
			t.Errorf(
				"expected to quit with exit code %d received %d",
				RootDirectoryIncorrect,
				code,
			)
		}
	})

	// Running the method once the patches are in place.
	userInput.SetRoot("incorrect path")

	/*
		The second part of test involves failing if the path passed for the root
		directory points to a file instead - something likely to occur in real-world.
	*/

	// Making any call to `os.Stat` return info for the dummy file
	monkey.Patch(os.Stat, func(path string) (os.FileInfo, error) {
		return fileInfo, nil
	})

	// Calling the method for a test run.
	userInput.SetRoot("path in here")
}

/*
Tests setter method to set the path to ffmpeg and ffprobe executable(s).

Testing involves ensuring that the application terminates if a non-existent path is
supplied (`os.Stat` throws an error), or if the path points to a directory.
*/
func TestDependencies(t *testing.T) {
	// Important
	defer monkey.UnpatchAll()

	tempDir, tempFile := getDummies(t)

	// List of functions that are to be tested.
	functions := []func(string){
		userInput.SetFFmpegPath,
		userInput.SetFFprobePath,
	}

	// Iterating once through the tests for every function
	for _, function := range functions {
		/*
			Checking if application force-stops in case `os.Stat` throws error - will
			occur if the path is incorrect - also checks the exit code.
		*/

		monkey.Patch(os.Stat, func(string) (os.FileInfo, error) {
			return tempFile, errors.New("temporary error")
		})

		monkey.Patch(os.Exit, func(exitCode int) {
			if exitCode != ExecutableNotFound {
				t.Errorf(
					"unexpected exit code. \nexpected: %d\nfound: %d",
					ExecutableNotFound,
					exitCode,
				)
			}
		})

		// Executing the method
		function("temporary path")

		/*
			Test to ensure that the application is force-stopped if path used belongs
			to a directory instead of file.
		*/
		monkey.Patch(os.Stat, func(string) (os.FileInfo, error) {
			return tempDir, nil
		})

		// Executing the method again
		function("temporary path")
	}
}

/*
Tests the setter method to accept a list of files that are to be ignored.

Testing includes ensuring that excess space is stripped off, trailing back/forward
slashes are also removed.
*/
func TestExcludeFiles(t *testing.T) {
	input := make(map[string][]string, 10)

	// Normal values, with extra spacing - ensures spaces are trimmed.
	input[" tests/result , tests   ,  test,"] = []string{
		"tests/result",
		"tests",
		"test",
	}

	// Values with trailing spaces, and dangling forward/backward slashes
	input[" result/, /result/new/, /home/new dir\\"] = []string{
		"result",
		"/result/new", // Slashes in the left not to be disturbed - full paths
		"/home/new dir",
	}

	for key, value := range input {
		// Executing the method with the key.
		userInput.SetExcludeFiles(key)

		// Quick check for potential failure.
		if len(value) != len(userInput.excludeFiles) {
			t.Errorf(
				"error expected result to contain `%v` values, found `%v`",
				len(value),
				len(userInput.excludeFiles),
			)
		}

		// Iterating over the array of known values to this key. Ensuring that each
		// element in this array is present in the internal array instantiated by
		// `SetExcludeFiles` method.
		for _, val01 := range value {
			found := false
			for _, val02 := range userInput.excludeFiles {
				if val01 == val02 {
					found = true
				}
			}

			if !found {
				t.Errorf("unable to find value `%v` in result. \nInput: `%v` "+
					"\nOutput: %v",
					val01,
					key,
					userInput.excludeFiles,
				)
			}
		}
	}
}

/*
Test setter method to use a regex pattern to exclude files/folders

Testing performed includes ensuring that the application force-quits if the regex string
being used is incorrect, and checks the regex value being compiled and stored
*/
func TestExcludeRegex(t *testing.T) {
	// Important
	defer monkey.UnpatchAll()

	monkey.Patch(os.Exit, func(exitCode int) {
		if exitCode != RegexError {
			t.Errorf(
				"unexpected exit code. \nexpected: %d \nfound: %d",
				RegexError,
				exitCode,
			)
		}
	})

	// Initial test with faulty pattern sequence - ensuring failure.
	for _, regex := range []string{
		"g([az]+ng",
		"go(lang",
	} {
		userInput.SetExcludeRegex(regex)
	}

	// Remove patch to ensure that the setter method can run with valid pattern
	monkey.Unpatch(os.Exit)

	for _, pattern := range []string{
		"[test]",
		"golang.*",
		".*test\\-pattern.*",
	} {
		userInput.SetExcludeRegex(pattern)

		// Check the stored pattern sequence - note the usage of `MustCompile`. Ensure
		// `pattern` is a valid pattern pattern
		result := regexp.MustCompile(pattern)
		if result.String() != userInput.excludeRegex.String() {
			t.Errorf(
				"failed to match pattern sequence \nTest Input: %v "+
					"\nStored Sequence: %v",
				pattern,
				userInput.excludeRegex,
			)
		}
	}
}

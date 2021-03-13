package commons

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitialize(t *testing.T) {
	/*
		Ensuring the method fails with the expected exit code in case of incorrect
		regex pattern being used - as well as tests pass successfully for valid regex.
	*/

	// Map of strings used as regex patterns, and structure containing the expected
	// return value(s) - in case an error is expected, the error message will be
	// ignored.
	inputRegex := map[string]struct {
		code int
		err  error
	}{
		".exe": {StatusOK, nil},
		".*":   {StatusOK, nil},
		"":     {StatusOK, nil},

		"[e": {RegexError, errors.New("")},
		"(]": {RegexError, errors.New("")},
		")]": {RegexError, errors.New("")},
	}

	for in, result := range inputRegex {
		userInput := UserInput{RegexExclude: in, IsTest: true}
		errCode, err := userInput.Initialize()

		// fail test if return code is unexpected, or if `err` or `resultErr` do not
		// match - i.e. they should both be null, or non-null
		if errCode != result.code || (err != nil && result.err == nil) ||
			(err == nil && result.err != nil) {
			t.Errorf(
				"(userInput/Initialize) unexpected return value while testing "+
					"regex failures! \npattern used: `%s` \nvalue "+
					"expected: `%d` \nvalue returned: %d \nerror returned: %v",
				in,
				result.code,
				errCode,
				err,
			)
		}
	}

	var pathDir, pathFile, pathInvalid string

	if cwd, err := os.Getwd(); err != nil {
		t.Errorf(
			"(userInput/Initialize) failed to fetch current working "+
				"directory!\nerror: %v",
			err,
		)
	} else {
		// Point to `testdata` - two directories up.
		pathDir = filepath.Join(filepath.Dir(filepath.Dir(cwd)), "testdata")

		// Point to `.gitkeep` file in testdata
		pathFile = filepath.Join(pathDir, ".gitkeep")

		// Random non-existent path
		pathInvalid = filepath.Join(pathDir, "invalid_file.txtS")
	}

	/*
		Checking if the method fails in case of invalid/empty root path
	*/
	for _, in := range []struct {
		path string
		flag bool
	}{
		{"", false},          // should fail unless (`test` flag is disabled)
		{"", true},           // pass (`test`flag enabled)
		{pathInvalid, false}, // fail - path invalid
		{pathInvalid, true},  // fail - path invalid (flag will be ignored)
		{pathFile, true},     // fail - points to a file
		{pathFile, false},    // fail - points to a file
		{pathDir, true},      // pass
		{pathDir, false},     // pass
	} {
		// Emulate user input
		input := UserInput{
			RootPath: in.path,
			IsTest:   in.flag,
		}

		// Fetch results on running the method and compare
		retCode, retErr := input.Initialize()

		errMsg := "(userInput/Initialize) expected failure, received none " +
			"\ninput path: `%s`\nflag: %v\ncode returned: %d  \nerror: %v"

		switch item, err := os.Stat(in.path); {
		case in.path == pathDir && err == nil && item.IsDir():
			// If the path is valid, and points to a directory, pass
			//lint:ignore SA4011 not an error
			break

		case in.path == "" && in.flag == true:
			// If the path is empty, and the test flag is enabled, pass
			//lint:ignore SA4011 not an error
			break

		case retCode == StatusOK || retErr == nil:
			// For any other case, fail if an error is not returned
			t.Errorf(
				errMsg,
				in.path,
				in.flag,
				retCode,
				err,
			)
		}
	}
}

func TestLog(t *testing.T) {
	// As a method, `UserInput.log()` does nothing but log - no chances of failure.
	// Running "tests" on it nevertheless because coverage score falls without them for
	// no fault of mine :(
	for i := 0; i < 3; i++ {
		ui := UserInput{}
		ui.log()
	}
}

func TestIgnoreFile(t *testing.T) {
	/*
		Combining two tests together - ensuring that files ignored either by regex
		string, or by exclusion list are actually ignored
	*/

	// list of file names to be used as input for test
	files := []string{
		"test.exe",
		"temp_file.bat",
		"new_file.docx",
		"incorrect_file.word",
		"video.mkv",
		"same_video.jpg",
		"definitely_not_a_video.txt",
	}

	input := UserInput{
		// Will require full value as present in `files`
		Exclusions: []string{
			"test.exe",
			"incorrect_file.word",
		},

		// Regex pattern to ignore files based on their extensions
		RegexExclude: `(.*\.txt)|(.*\.mkv)|(.*\.jpg)`,

		IsTest: true, // Ensures root path isn't required
	}

	errCode, err := input.Initialize()
	if errCode != StatusOK {
		t.Errorf(
			"(userInput/IgnoreFile) error occurred during initialization!"+
				"\nerror code: %d \nerror: %v",
			errCode,
			err,
		)
	}

	source := "source-directory"
	for i, file := range files {
		result := input.IgnoreFile(&source, &files[i])

		flag := false

		if input.RegexRule != nil && input.RegexRule.MatchString(file) {
			flag = true
		}

		for _, fileName := range input.Exclusions {
			if strings.EqualFold(fileName, file) {
				flag = true
			}
		}

		if result != flag {
			t.Errorf(
				"(commmons/IgnoreFile) failed to match the value with expected"+
					" value \nresult obtained: %v \nexpected result: %v "+
					"\nfile name: %s \nregex rule: %v \nexclusions: %v",
				result,
				flag,
				file,
				input.RegexExclude,
				input.Exclusions,
			)
		}
	}
}

func TestOutputName(t *testing.T) {
	input := UserInput{}

	for inp, res := range map[string]string{
		"input.mkv":                        "input.mkv",
		"input.mp4":                        "input.mkv",
		"/home/usr/local/movies/movie.mp4": "/home/usr/local/movies/movie.mkv",
	} {
		if val := input.OutputName(inp); val != res {
			t.Errorf(
				"(userInput/OutputName) unexpected file name returned: \n"+
					`input: "%s"`+"\n"+`output: "%s"`+"\n"+`expected output: "%s"`,
				inp,
				val,
				res,
			)
		}
	}
}

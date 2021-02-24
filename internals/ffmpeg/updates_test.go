package ffmpeg

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"bou.ke/monkey"
	"github.com/demon-rem/auto-sub/internals/commons"
	log "github.com/sirupsen/logrus"
)

var update = Updates{
	userInput: &commons.UserInput{
		RootPath:    "",
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
	},
}

/*
A simple structure to contain data regarding media files present in the testdata
directory. Used to store/contain data that will be used in tests
*/
type tFiles struct {
	// Path to media file, *inside* the `testdata` directory
	filePath string

	frames      int64  // frames present in the file
	fileSize    int64  // file size in bytes
	strFileSize string // file size as expected in response from the method
}

func generateFileData(path string, frames, fileSize int64, readableSize string) tFiles {
	testdata := ""
	if cwd, err := os.Getwd(); err != nil {
		log.Debugf("unable to fetch current working directory! \nerror: %v", err)
		os.Exit(-20)
	} else {
		testdata = filepath.Join(filepath.Dir(filepath.Dir(cwd)), "testdata")
	}

	return tFiles{
		filePath:    filepath.Join(testdata, path),
		frames:      frames,
		fileSize:    fileSize,
		strFileSize: readableSize,
	}
}

// A set of media-files present in the tests
var testFiles = []tFiles{
	generateFileData(
		"test 01/sample_640x360.mkv",
		400,
		573_066,
		"559.63 KiB",
	),

	generateFileData(
		"test 02/sample_960x540.mkv",
		400,
		1_319_066,
		"1.26 MiB",
	),
}

func TestConvertor(t *testing.T) {
	set := map[string]int64{
		"100":     100,
		"0":       0,
		"-1000":   -1000,
		"500":     500,
		"failure": convFail,
	}

	for input, result := range set {
		if val := update.convertor(input); val != result {
			t.Errorf(
				"(Updates/convertor) result does not match expected output "+
					"\ninput: %s \nresult: %d \nexpected: %d",
				input,
				val,
				result,
			)
		}
	}
}

func TestTrimString(t *testing.T) {
	// Change the test set if trim length changes
	set := map[string]string{
		"this string remains unmodified": "this string remains unmodified",
		"too short":                      "too short",
		"pass":                           "pass",

		"this string exceeds limit and will be truncated to fit in length": "this " +
			"string exceeds li....cated to fit in length",
		"this string shall also be truncated for exceeding max length :(": "this " +
			"string shall also....xceeding max length :(",
	}

	for input, result := range set {
		if val := update.trimString(&input); val != result {
			t.Errorf(
				"(Updates/trimString) result does not match expected value\n"+
					"input: \"%s\" \nresult: \"%s\" \nexpected: \"%s\"",
				input,
				val,
				result,
			)
		}
	}
}

func TestGetTotalFrames(t *testing.T) {
	defer monkey.UnpatchAll()

	monkey.UnpatchAll()

	// Helper function to run test on all test files. Depending on whether failure
	// or success is expected, cause the test to pass or fail.
	test := func(fail bool) {
		for _, testFile := range testFiles {
			frames, err := update.getTotalFrames(testFile.filePath)

			errMsg := ""
			if fail {
				// If failure is to be expected, ensure that it does
				if frames != 0 || err == nil {
					errMsg = "(Updates/getTotalFrames) test passing when failure was" +
						"expected \nframes found: %d \nerror: %v"
				}
			} else {
				// If failure isn't expected, ensure there is no failure
				if frames != testFile.frames || err != nil {
					errMsg = "(Updates/getTotalFrames) test passing when failure was " +
						"NOT expected \nframes found: %d \nerror: %v"
				}
			}

			if errMsg != "" {
				t.Errorf(errMsg, frames, err)
			}
		}
	}

	// Emulate failure to execute the command internally
	cmd := exec.Cmd{}
	monkey.PatchInstanceMethod(
		reflect.TypeOf(&cmd),
		"Run",
		func(*exec.Cmd) error {
			return errors.New("(Updates/getTotalFrames) error thrown as a test")
		},
	)

	// Run tests; expecting failure.
	test(true)

	// First set of tests end; unpatch
	monkey.UnpatchInstanceMethod(reflect.TypeOf(&cmd), "Run")

	originalRegex := regexFrames // save a copy

	// Second part of the tests, modify regex, emulating the scenario where regex isn't
	// matched.
	regexFrames = regexp.MustCompile(``) // empty regex, no match possible!

	// Run tests expecting failure
	test(true)

	// Third part of the tests, modify the regex pattern to select incorrect values,
	// emulates the scenario where regex fetches wrong text (or non-numeric text)
	regexFrames = regexp.MustCompile(`(^|\s+)fram(e=.)`)

	test(true) // run tests again, expecting failure.

	// Reset the regex pattern back
	regexFrames = originalRegex

	// Once everything is in order, run the test again, this time, it should pass
	test(false)
}

func TestFileSize(t *testing.T) {
	// Ensuring failure in case path to a directory is being used.
	if val := update.getFileSize(filepath.Dir(testFiles[0].filePath)); val > 0 {
		// If the function manages to return a valid size for a directory, fail the test
		t.Errorf(
			"(Updates/getFileSize) method returning positive value for file "+
				"size of a directory \n"+`input path: "%s"`+"\nsize calculated: %d",
			filepath.Dir(testFiles[0].filePath),
			val,
		)
	}

	// Ensure failure in case of invalid path
	if path := filepath.Join(testFiles[0].filePath, "invalid_path"); update.
		getFileSize(path) > 0 {
		// If function can calculate and return size for an invalid path, fail
		t.Errorf(
			"(Updates/getFileSize) method can get file size for invalid path \n"+
				`path: %s`+"\nsize returned: %d",
			path,
			update.getFileSize(path),
		)
	}

	// Run normal tests on all files, expect success
	for _, test := range testFiles {
		if val := update.getFileSize(test.filePath); val != test.fileSize {
			t.Errorf(
				"(Updates/getFileSize) file size fetched by method does not"+
					"match expected file size!\n"+`file path: "%s"`+
					"\nexpected size: %d \nsize returned: %d",
				test.filePath,
				test.fileSize,
				val,
			)
		}
	}
}

func TestReadableSize(t *testing.T) {
	for _, test := range testFiles {
		if v := update.readableFileSize(float64(test.fileSize)); v != test.strFileSize {
			t.Errorf(
				"(Updates/readableFileSize) value returned by method does not "+
					"match expected response \ninput: %d \nvalue returned: %s "+
					"\nexpected: %s",
				test.fileSize,
				v,
				test.strFileSize,
			)
		}
	}

	// Catch a panic statement when the test function ends
	defer func() {
		if err := recover(); err == nil {
			t.Errorf(
				"(Updates/readableFileSize) method did not throw an error with " +
					"negative file size",
			)
		}
	}()

	// Ensure failure in case of negative file size - will cause a panic, which will
	// then be caught by the deferred function call
	update.readableFileSize(-10)
}

func TestProgress(t *testing.T) {
	// This function doesn't actually perform tests, just firing the progress bar method
	// to increase test coverage - progress bar and other stuff related to updates
	// on the screen will have to be verified manually (visually :p)
	update.progressBar(-10)
	update.progressBar(200)
	update.progressBar(50)

	tempAnimationProgress = 50
	update.progressBar(200)
	tempAnimationProgress = 0
}

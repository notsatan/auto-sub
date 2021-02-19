package ffmpeg

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bou.ke/monkey"

	"github.com/demon-rem/auto-sub/internals/commons"
)

/*
TestCheckExt runs a test on the `checkExt` method.

Tests being run include a using a list of file names, and combining them with the values
*/
func TestCheckExt(t *testing.T) {
	/*
		List of file names, one of these will be picked at random for each iteration
		in the test.

		Note: Extensions (if present) in these values will be ignored - new extensions
		will be added during run-time to dynamically generate file names
	*/
	fileNames := []string{
		"test-file",
		"movie",
		"subtitle-file.mp3",
		"tv-series",
		"random-file.mkv",
	}

	// Picking up a slice randomly from the slices of extensions
	for _, extensions := range [][]string{
		subsExt,
		videoExt,
		chaptersExt,
		attachmentExt,
	} {
		// Iterating through the list of file names - each file name will then be
		// combined with every extension present in `extensions` and tested
		for _, fName := range fileNames {
			// Iterate through a list of all extensions in the slice - in each test
			// a randomly probability will be used to decide the value to be returned
			// by the method before-hand. If `false` is to be returned, modify the
			// the file name by adding another (unrecognized) extension to ensure
			// that the latter is detected.
			for _, ext := range extensions {
				// Randomly decide if the result is supposed to be correct or not
				result := rand.Intn(2)%2 == 0 // random number is even or not

				// Generate file name
				fileName := fmt.Sprintf("%s.%s", fName, ext)

				// Modify file name to make extension unrecognizable depending on the
				// value of `result`
				if !result {
					// Add a new extension that isn't recognized
					fileName += ".mp3"
				}

				// Run the function with the current slice, and check the result
				if res := checkExt(fileName, extensions); res != result {
					// Failure
					t.Errorf(
						"(handler/checkExt) failed to match extension with "+
							"list of recognized extensions \n file name: `%s` "+
							"\nextensions: [`%v`]\nexpected result: %v "+
							"\nresult found: %v",
						fileName,
						strings.Join(extensions, "`, "),
						result,
						res,
					)
				}
			}
		}
	}
}

//nolint:gocyclo // will edit this test to reduce its complexity later
func TestGroupFiles(t *testing.T) {
	// Form path to `testdata` directory
	testdata := ""
	if cwd, err := os.Getwd(); err != nil {
		t.Errorf(
			"(handler/groupFiles) unable to fetch current path! \nError: \n%v",
			err,
		)
	} else {
		// Move up the file tree (twice)
		testdata = filepath.Join(filepath.Dir(filepath.Dir(cwd)), "testdata")
	}

	// Ensure the function fails with incorrect path
	o01, o02, o03, o04 := groupFiles("invalid/path", &commons.UserInput{})
	if o01 != nil || o02 != nil || o03 != nil || o04 != nil {
		t.Errorf(
			"(handler/groupFiles) expected the function to fail with invalid "+
				"path! \nvalues: %s \n%s \n%s \n%s",
			commons.Stringify(&o01),
			commons.Stringify(&o02),
			commons.Stringify(&o03),
			commons.Stringify(&o04),
		)
	}

	// Fetch a list of all items present in the directory
	var dirs []os.FileInfo
	if dir, err := ioutil.ReadDir(testdata); err != nil {
		t.Errorf(
			"(handler/groupFiles) unable to fetch a list of directories in the"+
				"path: `%v` \nerror: %v",
			testdata,
			err,
		)
	} else {
		dirs = dir
	}

	// Iterate through every item in `testdata` directory - ignore non-directory items
	for _, dir := range dirs {
		if !dir.IsDir() {
			// Skip if `item` is not a directory
			continue
		}

		sourceDir := filepath.Join(testdata, dir.Name())

		// Forcing to ignore `exe` files to achieve complete coverage for the function
		input := &commons.UserInput{RegexExclude: `.*\.exe`}
		_, _ = input.Initialize()

		// Run the function to sort the files present in the directory
		retMedia, retSubs, retAttachments, retChapters := groupFiles(
			sourceDir,
			input,
		)

		var mediaFiles, subs, attachments, chapters []os.FileInfo

		// Fetch list of items in the source directory, sort them using `checkExt`
		items, _ := ioutil.ReadDir(sourceDir)
		for _, file := range items {
			switch fName := file.Name(); {
			case input.IgnoreFile(&sourceDir, &fName):
				// Ignore the file directly if required
				continue
			case checkExt(fName, videoExt):
				mediaFiles = append(mediaFiles, file)
			case checkExt(fName, subsExt):
				subs = append(subs, file)
			case checkExt(fName, attachmentExt):
				attachments = append(attachments, file)
			case checkExt(fName, chaptersExt):
				chapters = append(chapters, file)
			}
		}

		/*
			Creating 2d slices, one to contain the values returned while testing the
			function, the other to contain values obtained by iterating the source
			directories manually.

			Note: Ensure that the order of the same value remains the same in both
			slices, i.e. media files returned and calculated for example should be at
			the same index in both slices (zero for now)
		*/

		returned := [][]os.FileInfo{
			retMedia,
			retSubs,
			retAttachments,
			retChapters,
		}

		determined := [][]os.FileInfo{
			mediaFiles,
			subs,
			attachments,
			chapters,
		}

		// Quick comparison - match lengths
		for i := 0; i < len(determined); i++ {
			if len(returned[i]) != len(determined[i]) {
				t.Errorf(
					"(handler/groupFiles) array lengths fail to match "+
						"[iteration %d]\nvalues returned: %s\nvalues calculated: %s",
					i,
					commons.Stringify(&returned[i]),
					commons.Stringify(&determined[i]),
				)
			}
		}

		// Check values in both 2d slices
		for index := range determined {
			// Pick up a file in from `returned[index]` and look for the same file
			// in `determined[index]`
			for _, searchFile := range returned[index] {
				flag := false

				for _, curFile := range determined[index] {
					if curFile.Name() == searchFile.Name() {
						flag = true
					}
				}

				// If the flag is still incorrect, fail
				if !flag {
					t.Errorf(
						"(handler/groupFiles) unable to locate file: `%s`"+
							"\nsource directory: `%s` \nreturned values: %s"+
							"\ndetermined values: %s",
						searchFile.Name(),
						sourceDir,
						commons.Stringify(&returned[index]),
						commons.Stringify(&determined[index]),
					)
				}
			}
		}
	}
}

func TestTraverseRoot(t *testing.T) {
	// Fetch path to test data
	root := ""
	if path, err := os.Getwd(); err != nil {
		t.Errorf(
			"(handler/TraverseRoot) failed to fetch path to current working "+
				"directory \nerror: %v",
			err,
		)
	} else {
		root = filepath.Join(filepath.Dir(filepath.Dir(path)), "testdata")
	}

	// Template user input
	in := commons.UserInput{RootPath: root}
	if errCode, err := in.Initialize(); errCode != commons.StatusOK || err != nil {
		t.Errorf(
			"(handler/TraverseRoot) failed to initialize template user input"+
				"\nerror: %v \nexit code: %d",
			err,
			errCode,
		)
	}

	// Test to ensure the function fails if result directory points to an existing
	// non-directory item
	//nolint
	if errCode, err := TraverseRoot(&in, filepath.Join(root, ".gitkeep"));
		errCode != commons.UnexpectedError || err == nil {
		t.Errorf(
			"(handler/TraverseRoot) function does not fail even if path to "+
				"result directory points to an existing file \nerror: %v \nstatus: %d",
			err,
			errCode,
		)
	}

	/*
		Test to ensure failure occurs if unable to perform a check for existence of
		result directory
	*/
	defer monkey.Unpatch(os.Stat)
	monkey.Patch(os.Stat, func(string) (os.FileInfo, error) {
		return nil, errors.New("fail `os.Stat()` through a patch for tests")
	})

	// Patch function call to `sourceDir` to isolate the function being tested
	defer monkey.Unpatch(sourceDir)
	monkey.Patch(sourceDir, func(string, string, *commons.UserInput) int {
		return commons.StatusOK
	})

	//nolint
	if errCode, err := TraverseRoot(&in, root);
		err == nil || errCode != commons.UnexpectedError {
		t.Errorf(
			"(handler/TraverseRoot) function does not force stop even when " +
				"`os.Stat()` check fails!",
		)
	}

	// Test to ensure result directory is being created if it does not already exist
	defer monkey.Unpatch(os.Stat)
	monkey.Patch(os.Stat, func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	})

	// Result directory fails to be created
	defer monkey.Unpatch(os.Mkdir)
	monkey.Patch(os.Mkdir, func(string, os.FileMode) error {
		return errors.New("failing `os.Mkdir()` through a patch for tests")
	})

	//nolint
	if errCode, err := TraverseRoot(&in, root);
		err == nil || errCode != commons.UnexpectedError {
		t.Errorf(
			"(handler/TraverseRoot) function does not fail even when result " +
				"directory cannot be created",
		)
	}

	flag := false
	createPath := filepath.Join(root, "create dir")

	// Patch `os.Mkdir` to succeed (without actually creating a directory)
	defer monkey.Unpatch(os.Mkdir)
	monkey.Patch(os.Mkdir, func(path string, mode os.FileMode) error {
		if path != createPath {
			t.Errorf(
				"(hander/TraverseRoot) function attempting to create a "+
					"directory that is not the result directory "+
					"\nexpected dir: \"%s\" \ncreating: \"%s\"",
				createPath,
				path,
			)
		}

		flag = true
		monkey.Unpatch(os.Mkdir) // Removes the patch, the patch works once
		return nil
	})

	if _, _ = TraverseRoot(&in, createPath); !flag {
		t.Errorf(
			"(handler/TraverseRoot) function did not attempt to create result " +
				"directory if it does not exist",
		)
	}
}

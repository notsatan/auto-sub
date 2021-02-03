package ffmpeg

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestGroupFiles(t *testing.T) {
	// Form path to `testdata` directory
	testdata := ""

	if cwd, err := os.Getwd(); err != nil {
		t.Errorf(
			"(handler/groupFiles) unable to fetch current path! \nError: \n%v",
			err,
		)
	} else {
		// Move to up the file tree (twice)
		testdata = filepath.Join(filepath.Dir(filepath.Dir(cwd)), "testdata")
	}

	// Run the method for all directories present in test data
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

		// Run the function to sort the files present in the directory
		retMedia, retSubs, retAttachments, retChapters := groupFiles(
			sourceDir,
			&commons.UserInput{}, // keeping the test simple and isolated
		)

		var mediaFiles, subs, attachments, chapters []os.FileInfo

		// Fetch a list of items in the source directory, and sort them using `checkExt`
		items, _ := ioutil.ReadDir(sourceDir)
		for _, file := range items {
			switch {
			case checkExt(file.Name(), videoExt):
				mediaFiles = append(mediaFiles, file)
			case checkExt(file.Name(), subsExt):
				subs = append(subs, file)
			case checkExt(file.Name(), attachmentExt):
				attachments = append(attachments, file)
			case checkExt(file.Name(), chaptersExt):
				chapters = append(chapters, file)
			}
		}

		stringify := func(files []os.FileInfo) string {
			result := "["

			for _, file := range files {
				result += fmt.Sprintf("`%v` ", file.Name())
			}

			result = strings.TrimSpace(result) + "]"

			return result
		}

		/*
			Create 2d slices, one to contain the values returned while testing the
			function, the other to contain the values obtained by iterating the source
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
					"(groupFiles) array lengths fail to match [iteration %d] "+
						"\nvalues returned: %s\nvalues calculated: %s",
					i,
					stringify(returned[i]),
					stringify(determined[i]),
				)
			}
		}

		// Check for values in both 2d slices
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
						stringify(returned[index]),
						stringify(determined[index]),
					)
				}
			}
		}
	}
}

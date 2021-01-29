/*
Package FFmpeg contains the internal implementation of the backend for the application.

This package contains the code that will actually traverse the root directory, find the
files present in the source directories and recognize them based on their extensions.
And then, finally, fire a ffmpeg command to combine the subtitles/chapters/attachments
with the media file.

This package will abstract the irrelevant but important details from the rest of the
code providing just a single function call.
*/
package ffmpeg

import (
	"fmt"
	"github.com/demon-rem/auto-sub/internals/commons"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

/*
Set of arrays containing extensions for various recognized file-types.

Files ending with one of these known extensions will be treated as such (for example,
a file have an extension from `videoExt` will be considered as the main video file).

Note: While the elements in these array(s) are file extensions, they do not contain
a period - this will be added during runtime comparison - just for simplicity.
*/
var (
	videoExt = []string{
		"mkv",
		"mp4",
		"webm",
		"m2ts",
	}

	subsExt = []string{
		"srt",
		"ass",
		"sup",
		"pgs",
		"vtt",
	}

	attachmentExt = []string{
		"ttf",
		"otf",
	}

	// Ideally should be present with `attachments` - creating a separate array since
	// the mime type for chapters/tags (or any XML file) will be different.
	chaptersExt = []string{
		"xml",
	}
)

/*
TraverseRoot is the central function to which the flow-of-control will be passed. Once
the step of input validation and pre-emptive processing is completed, this function
will the traverse the root directory and use ffmpeg in the backend to be soft-sub
the video files as required.
*/
func TraverseRoot(
	userInput *commons.UserInput,
	files *[]os.FileInfo,
	stderr func(string, ...interface{}),
) {

	// Path to the output directory
	outputDir := filepath.Join(userInput.RootPath, "result")

	// Check if result directory exists in the root directory, if not, attempt to
	// create one - force stop if the latter fails.
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		log.Debugf(
			"(rootCmd/TraverseRoot) creating result dir in: `%v`",
			userInput.RootPath,
		)

		if err = os.Mkdir(outputDir, os.ModeDir); err != nil {
			log.Warnf(
				"(rootCmd/TraverseRoot) failed to create result directory "+
					"in: `%v`\n Error Traceback: %v\n",
				userInput.RootPath,
				err,
			)

			stderr("Error: Failed to create destination directory: `%v`\n\n", outputDir)
			os.Exit(commons.UnexpectedError)
		}
	}

	// Iterate through the root directory.
	for _, f := range *files {
		if f.IsDir() && filepath.Join(userInput.RootPath, f.Name()) != outputDir {
			// If the item is a directory - treat it as the source directory.
			output, status, cmd := generateCmd(
				filepath.Join(userInput.RootPath, f.Name()),
				userInput,
				outputDir,
			)

			// Output will contain message to be printed to the user
			if output != "" {
				// If the output message is not empty, printing it.
				stderr(output)
			}

			// If the status code returned is not ok, force-stop the application.
			if status != commons.StatusOK {
				os.Exit(status)
			}

			// The flow-of-control reaches this point only if the status returned by
			// is okay - finally running the ffmpeg command
			_, _ = cmd.Output() // Will process output of the command in future
		}
	}
}

/*
generateCmd is the central function which will generate the ffmpeg command to be used to
soft-sub the media file along with additional chapters/attachments.

This function will internally traverse a source directory, and identify the files
present inside it as media file, subtitle, attachments, or chapters based on their
extensions. Using this, the function will then generate a FFmpeg command that will
combine all these files into a matroska container in the output directory.

Once formed, this command will then be returned to the calling function.
*/
func generateCmd(
	root string,
	userInput *commons.UserInput,
	outDir string,
) (output string, status int, cmd *exec.Cmd) {
	// Variables to contain files as they are detected in the directory. Each string
	// will be the complete path to a file
	//
	// While all the lists are optional - at least one of them needs to have one or more
	// members, if not an error will be thrown
	var mFileFound os.FileInfo        // Media file - required.
	var attachmentFound []os.FileInfo // Attachments - optional.
	var subsFound []os.FileInfo       // Subtitles - optional.
	var chaptersFound []os.FileInfo   // Chapters - optional.

	// Fetching a list of all files present in this directory - `ioutil.ReadDir`
	// returns list of files sorted by filename - sorting not required
	files, err := ioutil.ReadDir(root)
	if err != nil {
		log.Warnf(
			"(rootCmd/generateCmd) ran into an error traversing source "+
				"directory \ndirectory: `%v` \n(traceback): %v",
			root,
			err,
		)

		// Force-stop since the error cannot be recovered from.
		return "Error: Ran into an unexpected error",
			commons.UnexpectedError,
			cmd
	}

	/*
		Simple helper function designed specifically to compare if a file contains a
		recognized extension from a list of extensions or not.

		Note: The elements of the array may (or may not) contain a period as a prefix,
		regardless, they will be treated as file extensions internally.
	*/
	checkExt := func(file string, array []string) bool {
		for _, ext := range array {
			// Trim any number of period(s) - if present - from the left, and add one.
			// Converts `mp4` -> `.mp4`; and ensures that `.mp4` stays the same.
			ext = "." + strings.TrimLeft(ext, ".")

			// Check if the file name ends with this extension
			if strings.HasSuffix(file, ext) {
				return true
			}
		}

		// Indicate that the file extension is not present in this array.
		return false
	}

	// Iterate through the list of files in the source directory
	for _, file := range files {

		/*
			Check for ignore rules - if the current file name matches an ignore rule,
			jump to the next iteration.
		*/

		// Check to ensure regex rule is not null before using it
		if userInput.RegexRule != nil && userInput.RegexRule.MatchString(file.Name()) {
			log.Debugf(
				"(rootCmd/generateCmd) skip file `%v` - regex exclusion rule",
				filepath.Join(root, file.Name()),
			)

			// Jump to the next file
			continue
		}

		// If regex does not match, check if file name is present in the list of files
		// to be ignored - case insensitive.
		for _, exclusion := range userInput.Exclusions {
			// Value of `exclusion` will always be lowercase, no need to convert it
			if strings.ToLower(file.Name()) == exclusion {
				log.Debugf(
					"(rootCmd/generateCmd) skip file `%v` - ignore rule `%v`",
					filepath.Join(root, file.Name()),
					exclusion,
				)

				// Jump to next file
				continue
			}
		}

		/*
			Since the file is not to be ignored, attempt to recognize the file as a
			media file, subtitle, attachment or chapter(s) - skip if none matches
		*/

		if checkExt(file.Name(), videoExt) {
			// If the file is recognized as a media file, check if the variable is
			// empty or not - if the variable has a value, it means that the source
			// directory has one or more media file(s) - throw an error.
			if mFileFound != nil {
				log.Warnf(
					"(rootCmd/generateCmd) multiple media files found in a "+
						"source directory \ndirectory: %v \nfiles: (`%v`, `%v`)",
					root,
					mFileFound.Name(),
					file.Name(),
				)

				return fmt.Sprintf(
					"Error: Multiple media files found in a source directory\n"+
						"Directory: %v\nFiles: [`%v`, `%v`]\n\n",
					root,
					mFileFound.Name(),
					file.Name(),
				), commons.UnexpectedError, cmd
			}

			mFileFound = file
		} else if checkExt(file.Name(), subsExt) {
			// Add this file to the list of subtitle files
			subsFound = append(subsFound, file)
		} else if checkExt(file.Name(), attachmentExt) {
			// Add this file to the list of attachments
			attachmentFound = append(attachmentFound, file)
		} else if checkExt(file.Name(), chaptersExt) {
			// Add this file to the list of chapters.
			chaptersFound = append(chaptersFound, file)
		}
	}

	/*
		Files present in the current source directory have been segregated into a
		type based on their extensions - perform a basic check.

		For one, there should be exactly one media file present in the source directory,
		if no file is found till now, throw an error regarding the same. Also, there
		should be at least one additional file that can be recognized (i.e. subtitle,
		chapter, or attachment) - if either of these two check fails, throw an error.
	*/

	if mFileFound == nil {
		log.Debugf("(rootCmd/generateCmd) no media file found in path: `%v`", root)

		return fmt.Sprintf(
			"Error: Failed to locate media file in the source directory: `%v`",
			root,
		), commons.UnexpectedError, cmd
	} else if len(subsFound) == 0 && len(attachmentFound) == 0 &&
		len(chaptersFound) == 0 {
		// There should be at least one subtitle/chapter/attachment file that is to be
		// attached to the media file
		log.Debugf(
			"(rootCmd/generateCmd) failed to locate any additional file in `%v`",
			root,
		)

		return fmt.Sprintf(
			"Error: Could not detect any additional file in the path: `%v`",
			root,
		), commons.UnexpectedError, cmd
	}

	/*
		If the checks are successful, form the ffmpeg command to attach these files to
		the media file.
	*/

	// The internal command that will be fired - each string should be an individual
	// argument to be used. Wrapping in arguments in double-quotes is not required.
	//
	// Code beyond this point simply pertains to forming ffmpeg command that is to be
	// used. If unsure, check FFmpeg documentation at: https://ffmpeg.org/ffmpeg.html
	//
	// Note: Be sure to use full-path for any input/source files being used in the
	// command internally.
	cmdRaw := []string{
		"-i",
		filepath.Join(root, mFileFound.Name()),
	}

	/*
		Adding subtitle streams to the source file - since subtitle streams are to be
		added as an input source, this process will be carried out in two separate steps
		the first step involves adding the subtitle streams as source to the command

		The next step involves marking the metadata related to the subtitle streams
		being used - this will be done after copy markers are added to the command.
	*/
	for _, sub := range subsFound {

		cmdRaw = append(
			cmdRaw,
			"-i",

			// full path to the subtitle file
			filepath.Join(root, sub.Name()),
		)
	}

	/*
		Adding copy markers to the command - these markers ensure that the original
		stream(s) are copied as original - without any stream selection or processing
		by ffmpeg. This step ensures that video with multiple audio/subtitle streams
		copied over; the default ffmpeg behaviour is to select one stream of each
		type - i.e. a single audio stream, a single video stream and a single subtitle
		stream
	*/

	cmdRaw = append(
		cmdRaw,

		// Ensure streams from the original file are being copied directly
		"-c",
		"copy",
	)

	/*
		Mapping the input streams - extension of the above `-c copy` flag; this ensures
		that in case of multiple subtitles being soft-subbed, all of them are mapped
		correctly.
	*/
	for i := 0; i < len(subsFound)+1; i++ {
		cmdRaw = append(
			cmdRaw,
			"-map",
			strconv.Itoa(i),
		)
	}

	for i, sub := range subsFound {
		var title string

		if userInput.SubTitleString == "" {
			title = strings.TrimSuffix(sub.Name(), filepath.Ext(sub.Name()))
		} else {
			title = userInput.SubTitleString
		}

		cmdRaw = append(
			cmdRaw,

			// Metadata for the subtitle stream - using the name of the file as the
			// title inside the container.
			fmt.Sprintf("-metadata:s:s:%v", i),
			fmt.Sprintf("title=%s", title),
		)

		// Setting language only if present - if not `language` will be a blank string
		if userInput.SubLang != "" {
			cmdRaw = append(
				cmdRaw,
				fmt.Sprintf("-metadata:s:s:%v", i),
				fmt.Sprintf("language=%s", userInput.SubLang),
			)
		}
	}

	/*
		Adding chapters found to the source file
	*/
	streams := 0
	for _, chapter := range chaptersFound {
		cmdRaw = append(
			cmdRaw,
			"-attach",
			filepath.Join(root, chapter.Name()),

			// Metadata for a chapter file
			fmt.Sprintf("-metadata:s:t:%v", streams),
			"mimetype=text/xml",
		)

		streams++
	}

	/*
		Adding attachments found to the source file
	*/
	for _, attachment := range attachmentFound {
		cmdRaw = append(
			cmdRaw,
			"-attach",
			filepath.Join(root, attachment.Name()),

			// Metadata for an attachment file
			fmt.Sprintf("-metadata:s:t:%v", streams),
			"mimetype=application/x-truetype-font",
		)

		streams++
	}

	// At the end, naming the output file - using the same name as the original file,
	// while changing the extension to be `.mkv` - ensures that the resultant container
	// is matroska; allowing multiple subtitles and attachments as required.
	cmdRaw = append(
		cmdRaw,
		filepath.Join(
			outDir,
			fmt.Sprintf(
				"%s.mkv",

				// Trim extension from original file name
				strings.TrimSuffix(
					mFileFound.Name(),
					filepath.Ext(mFileFound.Name()),
				),
			),
		),
	)

	// Simple function to convert a list of files into a string - used to log the files
	// present found in the source directory and the category they are recognized as
	stringify := func(files *[]os.FileInfo) string {
		count := len(*files)
		res := ""

		for i := 0; i < count; i++ {
			res += fmt.Sprintf("`%s`", (*files)[i].Name())
			if i < count-1 {
				res += ", "
			}
		}

		return res
	}

	log.Debugf(
		"Source Directory: %s \nMediaFile: %s \nSubtitles: %s \nChapters: %s"+
			"\nAttachments: %s\n\n",
		root,
		mFileFound.Name(),
		stringify(&subsFound),
		stringify(&chaptersFound),
		stringify(&attachmentFound),
	)

	cmd = exec.Command(
		userInput.FFmpegPath,
		cmdRaw...,
	)

	// Form the final command to be fired at ffmpeg.
	return "", commons.StatusOK, cmd
}

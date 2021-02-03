/*
Package ffmpeg contains the internal implementation of the backend for the application.

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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/demon-rem/auto-sub/internals/commons"
	log "github.com/sirupsen/logrus"
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
TraverseRoot is the main function to which the flow-of-control will be passed.
Once input validation and pre-emptive processing are completed, this function
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
CheckExt is a helper function designed to check if a file contains a extension from a
list of extensions.

Strings to be treated as extensions may (or may not) contain period as a prefix;
regardless, they will be treated as file extension(s) internally.
*/
func checkExt(fileName string, extensions []string) bool {
	for _, ext := range extensions {
		// Trim any number of period(s) - if present - from the left, and add one.
		// Converts `mp4` -> `.mp4`; and ensures that `.mp4` stays the same.
		ext = "." + strings.TrimLeft(ext, ".")

		// Check if the file name ends with this extension
		if strings.HasSuffix(fileName, ext) {
			return true
		}
	}

	// Indicate that the file extension is not present in this array.
	return false
}

/*
GroupFiles acts as a helper function that traverses through a source directory, lists
out all the file present in the source directory, and groups them into slices based on
file extensions.

Note: The function will internally assume `sourceDir` is a valid directory, the calling
function should perform a check before using the value
*/
func groupFiles(sourceDir string, userInput *commons.UserInput) (
	mediaFiles,
	subtitles,
	attachments,
	chapters []os.FileInfo,
) {
	// Fetch list of files present in this directory - `ioutil.ReadDir` sorts using
	// filename by default. Source path has been verified - skip checking again
	files, _ := ioutil.ReadDir(sourceDir)

	// Iterate through the list of files in the source directory
	for _, file := range files {
		if fName := file.Name(); userInput.IgnoreFile(&sourceDir, &fName) {
			// Check if file name is to be skipped - jump to next iteration if current
			// file is to be skipped
			continue
		}

		/*
			If the file is not to be ignored, attempt to recognize the file as a
			media file, subtitle, attachment or chapter(s) - skip if none matches
		*/

		switch {
		case checkExt(file.Name(), videoExt):
			mediaFiles = append(mediaFiles, file)

		case checkExt(file.Name(), subsExt):
			subtitles = append(subtitles, file)

		case checkExt(file.Name(), attachmentExt):
			attachments = append(attachments, file)

		case checkExt(file.Name(), chaptersExt):
			chapters = append(chapters, file)
		}
	}

	return mediaFiles, subtitles, attachments, chapters
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

	mediaFile, subsFound, attachmentFound, chaptersFound := groupFiles(
		root,
		userInput,
	)

	switch {
	case len(mediaFile) == 0:
		log.Debugf("(rootCmd/generateCmd) no media file in path: `%v`", root)

		return fmt.Sprintf(
			"Error: Failed to locate media file in source directory: `%v`",
			root,
		), commons.UnexpectedError, cmd

	case len(mediaFile) > 1:
		str := stringify(&mediaFile)

		log.Debugf(
			"(rootCmd/generateCmd) mutiple media files found in directory "+
				"\ndirectory: `%v` \nfiles: %v",
			root,
			str,
		)

		return fmt.Sprintf(
			"Error: Multiple media files found in source directory! "+
				"\nSource Directory: `%v` \nFiles Found: [`%v`]",
			root,
			str,
		), commons.UnexpectedError, cmd

	case len(subsFound) == 0 && len(attachmentFound) == 0 &&
		len(chaptersFound) == 0:
		// There should be at least one subtitle/chapter/attachment file that is to be
		// attached to the media file
		log.Debugf(
			"(rootCmd/generateCmd) failed to locate additional files `%v`",
			root,
		)

		return fmt.Sprintf(
			"Error: Could not detect any additional file in the path: `%v`",
			root,
		), commons.UnexpectedError, cmd
	}

	/*
		Forming the ffmpeg command to attach these files to the media file.
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
		filepath.Join(root, mediaFile[0].Name()),
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
		copied over; the default ffmpeg behavior is to select one stream of each
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
					mediaFile[0].Name(),
					filepath.Ext(mediaFile[0].Name()),
				),
			),
		),
	)

	log.Debugf(
		"Source Directory: %s \nMediaFile: %s \nSubtitles: %s \nChapters: %s"+
			"\nAttachments: %s\n\n",
		root,
		mediaFile[0].Name(),
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

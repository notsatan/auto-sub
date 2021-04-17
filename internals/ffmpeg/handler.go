/*
Package ffmpeg implements the central backend for the application.

This package will internally traverse the root directory, select the files that are to
be modified, excluded based on user input and then fire the calls to FFmpeg to merge
these files together as required.
*/
package ffmpeg

import (
	"errors"
	"fmt"
	"io"
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
period - will be added during runtime while comparing. Also, ensure all the extensions
are in LOWER case; ensures comparisons be case insensitive.
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
TraverseRoot is the public function that traverses the root directory working on
sub-directories present in it.

This function acts as a public interface connecting the internal workings of the
package to the rest of the application.

In case of failure, the function will internally print error message to the screen,
returning an error code as the result.
*/
func TraverseRoot(
	input *commons.UserInput, // user input
	resDir string,            // full path to output directory
) (exitCode int, err error) {
	log.Debugf(
		`(ffmpeg/TraverseRoot) traversing root directory: "%s"`+"\n"+
			`storing result in: "%s"`,
		input.RootPath,
		resDir,
	)

	// Check if result directory exists in the root directory, if not, attempt to
	// create one - return error if the latter fails
	item, err := os.Stat(resDir)
	if os.IsNotExist(err) {
		log.Debugf(
			"(ffmpeg/TraverseRoot) creating result dir in: `%v`",
			input.RootPath,
		)

		if err = os.Mkdir(resDir, os.ModeDir); err != nil {
			log.Warnf(
				`(ffmpeg/TraverseRoot) failed to create directory: "%s"`+
					"\nerror traceback: `%v`\n",
				resDir,
				err,
			)

			return commons.UnexpectedError,
				errors.New("unable to create destination directory")
		}
	} else if err != nil || !item.IsDir() {
		// Error if the check failed, or root path points to non-directory item
		log.Debugf(
			"(ffmpeg/TraverseRoot) failed to check for result directory "+
				"\nerror: %v \nitem type: %+v",
			err,
			item,
		)

		return commons.UnexpectedError,
			errors.New("an unexpected internal error occurred")
	}

	// Iterate through the root directory, fetching a list of all items present in it
	files, err := ioutil.ReadDir(input.RootPath)
	if err != nil {
		log.Debugf(
			"(ffmpeg/TraverseRoot) failed to fetch items present in root "+
				"directory! \nerror: `%v`",
			err,
		)

		return commons.UnexpectedError, errors.New("unable to read root directory")
	}

	if input.IsDirect {
		// The root directory is to be used as the source directory
		sourceDir(
			input.RootPath,
			resDir,
			input,
		)

		return commons.StatusOK, nil
	}

	// Variable to keep a track of source directories preset in the root directory;
	// used to throw an error in case root directory is empty
	dirsFound := 0

	// Iterate through the items present in root directory, treating each directory
	// as a source directory!
	for _, f := range files {
		if !f.IsDir() {
			continue
		}

		dirsFound++ // increment for each directory found
		sourcePath := filepath.Join(input.RootPath, f.Name())

		if sourcePath == resDir {
			// Don't use the directory containing results as a source directory
			continue
		}

		// The method call will handle the rest of the part for the source directory
		sourceDir(sourcePath, resDir, input)
	}

	if dirsFound == 0 {
		// Fail if the root directory does not contain any source directories
		return commons.RootDirectoryIncorrect,
			errors.New("root directory does not contain any source directories")
	}

	return commons.StatusOK, nil
}

/*
SourceDir is the central function that makes calls to FFmpeg to soft-sub media file(s)
with extras found in the source directory.

Once the command is fired, the function will then internally monitor the encoding
progress via a goroutine.
*/
func sourceDir(sourceDir, resDir string, input *commons.UserInput) (exitCode int) {
	log.Debugf(`(ffmpeg/sourceDir) processing source directory: "%s"`, sourceDir)

	// Fetch grouped list of files present in the source directory
	mediaFiles, subtitles, attachments, chapters := groupFiles(
		sourceDir,
		input,
	)

	log.Debugf(
		`(ffmpeg/sourceDir) grouped files for source directory "%s"`+
			"\nMediafile: %s \nChapters: %s \nSubtitles: %s \nAttachments: %s",
		sourceDir,
		commons.Stringify(&mediaFiles),
		commons.Stringify(&chapters),
		commons.Stringify(&subtitles),
		commons.Stringify(&attachments),
	)

	/*
		Performing basic checks on list of file(s) found, ensuring the directory
		contains exactly one media file, and at least one attachment/subtitle/chapter
		file, etc.

		Force-stop in case any thing is missing
	*/
	switch {
	case len(mediaFiles) == 0:
		log.Debugf(`(ffmpeg/sourceDir) no media file in path: "%s"`, sourceDir)
		commons.Printf(
			`Error: failed to locate any media file \n\tPath: "%s"`,
			sourceDir,
		)

		return commons.SourceDirectoryError
	case len(mediaFiles) > 1:
		log.Debugf(
			"(ffmpeg/sourceDir) mutiple media files found in source directory"+
				"\ndirectory: `%v` \nfiles: %v",
			sourceDir,
			commons.Stringify(&mediaFiles),
		)

		commons.Printf(
			"Error: multiple media files in source directory\n\t"+`Path: "%s"`+
				"\n\nFiles found: \n%s",
			sourceDir,
			commons.Stringify(&mediaFiles),
		)

		return commons.SourceDirectoryError
	case len(subtitles) == 0 && len(attachments) == 0 && len(chapters) == 0:
		// There should be at least one subtitle/chapter/attachment file
		log.Debugf(
			`"(ffmpeg/sourceDir) failed to locate additional files.\npath: "%v"`,
			sourceDir,
		)

		commons.Printf(
			"Error: failed to find any additional files in source directory\n"+
				`Path: "%s"`,
			sourceDir,
		)

		return commons.SourceDirectoryError
	}

	// Generate the FFmpeg command to run for the source directory
	cmd := generateCmd(
		sourceDir,
		input,
		resDir,

		// grouped list of files present inside the source directory
		mediaFiles[0], // flow-of-control ensures the array has exactly one item
		subtitles,
		attachments,
		chapters,
	)

	/*
		Two buffers; will be used to read command output as the command runs

		One of buffer will be used to actively track (and update) the progress using a
		goroutine in the background - this buffer will be cleared by the background
		thread when required.

		Second buffer will be used as a log dump, i.e. to log the output if needed in
		case of a crash.
	*/
	var progBuf strings.Builder
	var logBuf strings.Builder

	// Redirecting output from `stderr` to both buffers at once.
	cmd.Stderr = io.MultiWriter(&progBuf, &logBuf)

	// Channel to send signal to the background thread performing updates. The channel
	// ensures that flow-of-control is retained by this function as long as updates
	// are being performed in the background.
	signal := make(chan bool)

	// Deferred function call to ensure the goroutine stops before this function ends
	defer func(sig *chan bool) {
		log.Debugf(
			"(ffmpeg/sourceDir) wrapping up progress thread for source "+
				`directory: "%s"`,
			sourceDir,
		)

		// Emitting a signal; informs the goroutine that that the ffmpeg command has
		// completed its execution.
		*sig <- true

		// Receive a value from the signal - acts as an indicator from the goroutine
		// that it has performed final updates and closed.
		<-*sig

		// Finally, close the channel as well.
		close(*sig)

		log.Debugf(
			`(ffmpeg/sourceDir) completed processing source directory: "%s"`,
			sourceDir,
		)
	}(&signal)

	// An instance of the updates structure; will perform updates in the background
	updateThread := Updates{
		userInput:   input,
		filePath:    filepath.Join(sourceDir, mediaFiles[0].Name()),
		fileName:    mediaFiles[0].Name(),
		sourceDir:   sourceDir,
		resDir:      resDir,
		totalFrames: 0,
	}

	// Initializing the updates variable; performs internal household chores
	updateThread.Initialize()

	// Firing a goroutine; this function will track (and update) progress of the running
	// command
	go updateThread.DisplayUpdates(&progBuf, signal)

	// Running the command. This statement will block the main thread until the
	// ffmpeg process completes in the background. Will be the slowest step in the
	// function
	if err := cmd.Run(); err != nil {
		log.Debugf(
			"(ffmpeg/sourceDir) ffmpeg command failed while running in "+
				"background \nerror: %v \n\nlog buffer: %s",
			err,
			logBuf.String(),
		)
	}

	return commons.StatusOK
}

/*
CheckExt is a helper function designed to check if a file contains a extension from a
list of extensions.

Strings to be treated as extensions may (or may not) contain period as a prefix;
regardless, they will be treated as file extension(s).
*/
func checkExt(fileName string, extensions []string) bool {
	for _, ext := range extensions {
		// Trim any number of period(s) - (if present) from the left of the extension,
		// and add one. Converts `mp4` -> `.mp4`; while `.mp4` remains unaffected
		ext = "." + strings.TrimLeft(ext, ".")

		// Check if the file name ends with this extension - extensions are lowercase,
		// convert the file name to lower case to ensure case insensitive comparisons.
		if strings.HasSuffix(strings.ToLower(fileName), ext) {
			return true
		}
	}

	// The file extension is not present in this array.
	return false
}

/*
GroupFiles is a helper function designed to traverse a source directory, grouping all
the file(s) present in the directory based on their extensions.
*/
func groupFiles(sourceDir string, userInput *commons.UserInput) (
	mediaFiles,
	subtitles,
	attachments,
	chapters []os.FileInfo,
) {
	// Fetch list of files present in this directory - `ioutil.ReadDir` sorts using
	// filename by default. Source path has been verified - skip checking again
	files, err := ioutil.ReadDir(sourceDir)
	if err != nil {
		log.Debugf(
			"(ffmpeg/groupFiles) unable to read source directory: \"%s\""+
				"\nerror: %v",
			sourceDir,
			err,
		)

		// Empty return
		return nil, nil, nil, nil
	}

	// Iterate through files present in the source directory - check if a file is to be
	// ignored using the ignore rules, if not, group the file if its extension matches
	// a recognized extension
	for _, file := range files {
		if file.IsDir() {
			// Ignore directories - jump to the next item.
			continue
		}

		if fName := file.Name(); userInput.IgnoreFile(&sourceDir, &fName) {
			// Check if file name is to be skipped - jump to next iteration if current
			// file is to be skipped; the function call will log internally if a file
			// is to be skipped!
			continue
		}

		/*
			If the file is not to be ignored, attempt to group the file as a
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

		default:
			log.Debugf(
				"(ffmpeg/groupFiles) failed to group file: \"%s\"",
				filepath.Join(sourceDir, file.Name()),
			)
		}
	}

	return mediaFiles, subtitles, attachments, chapters
}

/*
GenerateCmd is the central function which will generate the ffmpeg command to soft-sub
the media file along with additional chapters/attachments, this function will form and
return the command, the calling-method will be responsible for running the command
*/
func generateCmd(
	sourceDir string,
	userInput *commons.UserInput,
	outDir string,

	mediaFile os.FileInfo,
	subsFound,
	attachmentFound,
	chaptersFound []os.FileInfo,
) (cmd *exec.Cmd) {
	// String array containing the command, each argument must be an individual element
	// in the array.
	//
	// Code beyond this point simply pertains to forming FFmpeg command. If unsure,
	// check FFmpeg documentation at: https://ffmpeg.org/ffmpeg.html
	//
	// Note: Use full-path for any input/source files used in the command, arguments
	// passed are NOT to be wrapped in double-quotes.
	cmdRaw := []string{
		"-i",
		filepath.Join(sourceDir, mediaFile.Name()),
	}

	/*
		Adding subtitle streams to the source file - since subtitle streams are to be
		added as an input source, this process will be carried out in two separate steps
		the first step involves adding subtitle streams as source to the command

		The second step will be marking metadata (language/title) for the subtitle
		streams being used - this will be done after copy markers are added to
		the command.
	*/
	for _, sub := range subsFound {
		cmdRaw = append(
			cmdRaw,
			"-i",

			// full path to the subtitle file
			filepath.Join(sourceDir, sub.Name()),
		)
	}

	/*
		Adding copy markers to the command - these ensure the input	stream(s) are
		copied as original (no implicit stream selection or processing) done by FFmpeg.
		This step ensures that video(s) with multiple audio/subtitle streams are exactly
		copied over;

		The default ffmpeg behavior is to select one stream of each type from every
		input file - i.e. a single audio stream, a single video stream and a single
		subtitle stream, etc
	*/

	cmdRaw = append(
		cmdRaw,

		// Ensure streams from the original file are being copied directly
		// Selectively mapping just the audio and video streams
		"-c",
		"copy",
	)

	/*
		Mapping the input streams - extension of the above `-c copy` flag; ensures in
		case of multiple subtitles being soft-subbed, all of them are mapped correctly.

		Starting from index 1 since index zero will be the media file.
	*/
	for i := 1; i < len(subsFound)+1; i++ {
		cmdRaw = append(
			cmdRaw,
			"-map",
			strconv.Itoa(i),
		)
	}

	/*
		Finally, the second (and last) step for attaching subtitle files - adding
		metadata to them, this step involves setting titles for the subtitle files,
		and language.
	*/
	for i, sub := range subsFound {
		var title string

		if userInput.SubTitleString == "" {
			// If a custom title is not to be used, use the name of the subtitle
			// file minus its extension.
			title = strings.TrimSuffix(sub.Name(), filepath.Ext(sub.Name()))
		} else {
			title = userInput.SubTitleString
		}

		cmdRaw = append(
			cmdRaw,

			// The first argument decides the (subtitle) stream for which metadata is
			// being added, the defines the metadata to be added (and its value)
			fmt.Sprintf("-metadata:s:s:%d", i),
			fmt.Sprintf("title=%s", title),
		)

		// Setting language only if present - if not `language` will be a blank string
		if userInput.SubLang != "" {
			cmdRaw = append(
				cmdRaw,

				// Same step as above, first argument selects the stream, the second
				// argument defines the metadata to be added and its value
				fmt.Sprintf("-metadata:s:s:%d", i),
				fmt.Sprintf("language=%s", userInput.SubLang),
			)
		}
	}

	/*
		Adding chapters found.
	*/
	streams := 0
	for _, chapter := range chaptersFound {
		cmdRaw = append(
			cmdRaw,
			"-attach",
			filepath.Join(sourceDir, chapter.Name()),

			// Metadata for a chapter file
			fmt.Sprintf("-metadata:s:t:%d", streams),
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
			filepath.Join(sourceDir, attachment.Name()),

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

			// Fetch final name for the output file
			userInput.OutputName(mediaFile.Name()),
		),
	)

	if userInput.Force {
		// Force flag is enabled
		cmdRaw = append(cmdRaw, "-f")
	}

	cmd = exec.Command(
		userInput.FFmpegPath, // path to the FFmpeg executable
		cmdRaw...,
	)

	// Return the final command formed
	return cmd
}

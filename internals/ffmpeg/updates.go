package ffmpeg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/demon-rem/auto-sub/internals/commons"
	log "github.com/sirupsen/logrus"
	escapes "github.com/snugfox/ansi-escapes"
)

const (
	// Constant defining the regex group from which values are to be extracted
	regexPos int = 2

	// Error code to indicate failure, used specifically by `convertor()`
	convFail = -1

	// Maximum number of characters in a string returned by `trimString`
	strTrimLen = 48
)

// Counter to keep a track of template animation progress across method calls.
var tempAnimationProgress = 0

/*
Compiled regex patterns to extract progress from FFmpeg command

Important: Ensure that the group at `regexPos` is the one containing the value to be
extracted, and the group should contain ONLY digits (not even floats)
*/
//nolint:gocritic
var (
	regexFrames = regexp.MustCompile(`.*(\s+|^)frame=\s*(\d+)`)
	regexFps    = regexp.MustCompile(`.*(\s+|^)fps=\s*(\d+)`)
	regexSize   = regexp.MustCompile(`.*(\s+|^)L?size=\s*(\d+)`)
)

/*
Updates is a simple structure that acts as an easy-abstraction for the main thread to
spawn a background thread that will display updates from an ongoing command to the
screen.

Designed to be run as a goroutine, the methods present in this structure will track the
progress of an ongoing command, providing updates to the screen for the same.

Use the method Updates.Initialize() to have the structure fetch the number of frames
present in the media file.

Use the method Updates.DisplayUpdates() as a goroutine to use the structure display
the progress of an ongoing encode on the screen.
*/
type Updates struct {
	// Input passed by the user
	userInput *commons.UserInput

	filePath  string // Full path to the media file
	fileName  string // Name of the media file
	sourceDir string // Path to the source directory
	resDir    string // Path to the result directory

	// Total frames present in media file; use `Initialize()` method to set its value
	totalFrames int64
}

/*
Initialize is a simple helper function designed to fetch the total number of frames
present in the destination media file implicitly.
*/
func (update *Updates) Initialize() {
	if frames, err := update.getTotalFrames(update.filePath); err != nil {
		log.Debugf(
			`(updates/Initialize) unable to fetch frame count for file "%s"`+
				"\nerror: %v",
			update.filePath,
			err,
		)

		update.totalFrames = 0 // default value in case of failure
	} else {
		update.totalFrames = frames
	}

	// Reset this counter, ensures the template animation also starts from scratch.
	tempAnimationProgress = 0
}

/*
DisplayUpdates is the main interface for the structure to the rest of the application,
this method should be fired as a goroutine to independently track the progress of
a command running in the background.

The contents from `stderr` of the running command should be redirected to the `buffer`
object supplied as a parameter to this function.

The interrupt channel is used as a two-way stream between the main thread and this
method.

The main thread should fire a signal on the channel when the command completes
its execution. Once the signal is received, this method will then internally
complete its own operation(s) and fire the signal (again) to indicate that the main
thread can move on.
*/
func (update *Updates) DisplayUpdates(buffer *strings.Builder, interrupt chan bool) {
	// Integer to keep a track of the number of lines to move up. The value will be
	// updated every time an update is made on the screen.
	lineCount := 0

	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		// Extract frames processed, FPS and current output size from the buffer.
		frames, fps, size := update.extractData(buffer)

		// Depending on the values fetched, set the contents of the progress message
		var progress string
		if frames == convFail && fps == convFail && size == convFail {
			// Let the user know something went wrong; unlikely scenario.
			progress = "If you're seeing this message, something broke :(" +
				"\nPlease file a bug report!"
		} else {
			progress = update.getProgress(
				frames,
				fps,
				size,
			)
		}

		// Make the cursor jump `lineCount` lines up - if any error were to occur,
		// the flow-of-control will not reach here.
		jumpCursor(lineCount)

		// Print progress dialog
		commons.Printf(progress)

		// Count the number of newline(s) present in the string - will be used in the
		// next iteration of the loop
		lineCount = strings.Count(progress, "\n")

		// Clear the buffer - ensures only the latest updates are present in the buffer
		buffer.Reset()

		select {
		case <-interrupt:
			// Interrupt received, time to kill the goroutine!
			log.Debugf(
				`(Updates/DisplayUpdates) received signal to kill background thread`,
			)

			/*
				Update value of the progress bar to display 100% completion (since the
				goroutine runs at one-second interval, the last update could be from
				one-second ago), pushing a final update to cover the edge-case
			*/

			// Use the total frame count, and fetch the final file size.
			frames = update.totalFrames
			size = update.getFileSize(filepath.Join(update.resDir, update.fileName))

			// Have the cursor jump upwards (again).
			jumpCursor(lineCount)

			// Printing the latest values, FPS counter can remain unchanged
			commons.Printf(update.getProgress(frames, fps, size) + "\n\n\n")

			log.Debugf(`(Updates/DisplayUpdates) killing the background thread`)
			interrupt <- true // indicates the goroutine is done
			return

		default:
			// ignore
		}
	}
}

/*
ExtractData is a helper function to extract values from the buffer input, and return
the same to the calling method.
*/
//nolint:interfacer // stupid suggestion
func (update *Updates) extractData(buffer *strings.Builder) (
	curFrames,
	curFps,
	curSize int64,
) {
	// Fetch updates from the buffer
	bufString := buffer.String()
	if res := regexFrames.FindSubmatch([]byte(bufString)); len(res) >= regexPos {
		curFrames = update.convertor(string(res[regexPos]))
	}

	if res := regexFps.FindSubmatch([]byte(bufString)); len(res) >= regexPos {
		curFps = update.convertor(string(res[regexPos]))
	}

	if res := regexSize.FindSubmatch([]byte(bufString)); len(res) >= regexPos {
		curSize = update.convertor(string(res[regexPos])) * 1000 // convert kB to bytes
	}

	return curFrames, curFps, curSize
}

/*
GetProgress is a simple method to generate the text to be printed on the screen, this
includes the name of the file being processed, progress bar depicting the current
progress and other statistics required.
*/
func (update *Updates) getProgress(
	curFrames,
	fps,
	size int64,
) (progress string) {
	// Calculating current progress percentage
	//nolint // again, stupid
	curProgress := float32(curFrames*100) / float32(update.totalFrames)

	// String to pad the left of each line, increase/decrease number of spaces on left
	padLeft := "  "

	// Excess spaces padding the right of each line, ensures existing text (if any) will
	// be overwritten by these spaces
	padRight := "\t\t"

	// String slice, each element being a line of the final progress dialog.
	contents := []string{
		fmt.Sprintf(`File: "%s"`, update.trimString(&update.fileName)),

		// The progress bar
		fmt.Sprintf(
			"\n\t%s\t%.2f", // rounding off the progress to two decimals
			update.progressBar(int(curProgress)),
			curProgress,
		) + "%%",

		"", // blank line
		fmt.Sprintf("Frames Processed: %d", curFrames),
		fmt.Sprintf("Average FPS: %d", fps),
		fmt.Sprintf("Output Size: %s", update.readableFileSize(float64(size))),
	}

	// Join string slice with a newline character, and return the same
	return padLeft + // left padding before the first element
		strings.Join(contents, padRight+"\n"+padLeft)
}

/*
ProgressBar generates a progress bar using the total frame count and the frames
processed currently and returns the same to the calling function
*/
func (update *Updates) progressBar(progress int) (progressBar string) {
	// The length of the progress bar. Will be padded by a space and opening/closing
	// character on both sides (i.e. four extra characters)
	const pbLen = 40

	/*
		Constant values used to draw the progress bar, should contain exactly one
		character. Can be modified as required.
	*/
	const (
		// Character present at the start of progress bar
		pbStart = "["

		// Character present at the end of progress bar
		pbEnd = "]"

		// Character to mark completed progress
		pbComplete = "="

		// Placed at the end of the completed progress.
		pbHead = ">"

		// Character to mark incomplete progress
		pbIncomplete = " "
	)

	fills := (pbLen * progress) / 100
	if fills == 0 {
		// In case the number of progress bars to be filled turns out to be zero,
		// increment the count by one.
		fills++ // number of progress bars drawn is `fills - 1`
	}

	if fills < 0 || pbLen-fills < 0 {
		// This occurs when the progress exceeds 100% or is negative (i.e. total frame
		// count fetched is incorrect). Replacing normal progress bar with question-mark
		// symbols - indicates something went wrong, and prevents a crash

		tempAnimationProgress += 5
		if pbLen-tempAnimationProgress <= 0 {
			// Reset the variable if it exceeds the length of progress bar
			tempAnimationProgress = 0
		}

		return fmt.Sprintf(
			"%s %s%s %s",
			pbStart,
			strings.Repeat("?", tempAnimationProgress),
			strings.Repeat(pbIncomplete, pbLen-tempAnimationProgress),
			pbEnd,
		)
	}

	return fmt.Sprintf(
		"%s %s%s%s %s",
		pbStart,
		strings.Repeat(pbComplete, fills-1),
		pbHead,
		strings.Repeat(pbIncomplete, pbLen-fills),
		pbEnd,
	)
}

/*
ReadableFileSize is a helper method to convert bytes into human-readable format. Should
not be used with negative values.

Example:
	fmt.Println(Updates.readableFileSize(1855425871872))

Will print 1.69 TiB as the result
*/
func (*Updates) readableFileSize(fileSize float64) string {
	// File size units, more units can be added if required.
	units := []string{
		"B",
		"KiB",
		"MiB",
		"GiB",
		"TiB",
		"PiB",
	}

	if fileSize < 0 {
		// Throw an error for a zero/negative file size
		log.Debugf("(updates/readableFileSize) file size: %f", fileSize)
		panic("(updates/readableFileSize) found negative file size as input")
	}

	counter := 0
	for fileSize >= 1024 {
		counter++        // increment the counter
		fileSize /= 1024 // divide file size by 1024
	}

	// Return the result, rounding it up to two decimals
	return fmt.Sprintf("%.2f %s", fileSize, units[counter])
}

/*
GetFileSize is a wrapper function to directly get the size of a file.

Note: Will return negative result in case the function fails to fetch the actual file
size, for example, in case of a directory.
*/
func (*Updates) getFileSize(path string) int64 {
	file, err := os.Stat(path)
	switch {
	case err != nil:
		log.Debugf("(updates/getFileSize) failed to get file size \nerror: %v", err)
		return -10

	case file.IsDir():
		log.Debugf("(updates/getFileSize) path belongs to directory \npath: %s", path)
		return -10
	}

	return file.Size()
}

/*
JumpCursor makes the cursor jump `count` lines vertically upwards.

Note: The number of lines (`count`) should NOT be negative
*/
func jumpCursor(count int) {
	commons.Printf(
		"%s%s",
		escapes.CursorPosX(0),         // resets x-coordinate of cursor
		escapes.CursorMove(0, -count), // moves `lineCount` lines up
	)
}

/*
GetTotalFrames will internally fire an FFmpeg command to attempt to fetch the total
number of frames present in the media file.

The frame count returned will be for the first video stream present in the input file.
No checks for validating the location of the media file are performed - should be
managed by the calling function.
*/
func (update *Updates) getTotalFrames(mediaFile string) (frames int64, err error) {
	// Command being fired: `ffmpeg -i <input.mkv> -map 0:v:0 -c copy -f null -`
	// Basically, will use FFmpeg to copy the first video stream from input to `null`;
	// ensuring that no copy actually takes place. The output produced by this command
	// will be
	cmd := exec.Command(
		update.userInput.FFmpegPath, // path to FFmpeg executable

		// arguments for the command being fired
		"-i", mediaFile, "-map", "0:v:0", "-c", "copy", "-f", "null", "-",
	)

	// Redirect stderr to string builder. Output of the command is dumped at `stderr`,
	// it can't be fetched through `cmd.Output()`
	var output strings.Builder
	cmd.Stderr = &output

	// Executing the command, output will be redirected to the string builder implicitly
	if err = cmd.Run(); err != nil {
		log.Debugf(
			`(updates/getTotalFrames) failed to fetch output for file: "%s"`+
				"\nerror: %v",
			mediaFile,
			err,
		)

		return 0, err
	}

	if res := regexFrames.FindSubmatch([]byte(output.String())); len(res) >= regexPos {
		if val := update.convertor(string(res[regexPos])); val != convFail {
			log.Debugf(
				`(updates/getTotalFrames) found %d frames in file "%s"`,
				val,
				mediaFile,
			)

			return val, nil
		}

		log.Debugf(
			`(updates/getTotalFrames) convertor failed for file: "%s"`+"\n"+
				`extracted value: "%s"`,
			mediaFile,
			res[regexPos],
		)
	} else {
		// Flow-of-control reaches here only if the regex pattern match fails
		log.Debugf(
			`(updates/getTotalFrames) regex pattern match failed for file: "%s"`+
				"\nlength(res): %d \n\noutput: %v",
			mediaFile,
			len(res),
			output.String(),
		)
	}

	return 0, errors.New("regex pattern match failed")
}

/*
TrimString trims the input string to fit a pre-determined length

Example:
	fmt.Println("this string exceeds the max character limit :/")

Will print "this string exceeds ....x character limit :/"

The maximum number of characters allowed is a constant value.
*/
func (*Updates) trimString(in *string) string {
	if len(*in) <= strTrimLen {
		// String is too small to be trimmed
		return *in
	}

	separator := "...."
	half := (strTrimLen - len(separator)) / 2
	res := fmt.Sprintf(
		"%s%s%s",
		(*in)[:half],
		separator,
		(*in)[len(*in)-half:],
	)

	return res
}

/*
Convertor is a utility function to convert byte slices into integers

The function simplifies the process for conversion through abstraction. If the byte
slice can be converted into an integer, the result will be returned, in case the
conversion fails midway, the default value will be returned.
*/
func (update *Updates) convertor(in string) int64 {
	if res, err := strconv.ParseInt(
		in,
		10,
		64,
	); err == nil {
		return res
	}

	return convFail
}

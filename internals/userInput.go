package internals

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

/*
UserInput is a basic structure to store and operate upon data passed by the user using
CLI.

Helper methods can be used to modify the values of these variables using arguments
passed by the user from command-line.
*/
type UserInput struct {
	// Path to the root directory containing the files
	rootPath string

	// Path to ffmpeg executable
	ffmpegPath string

	// Path to ffprobe executable
	ffprobePath string

	// Indicates if logging is required or not. True indicates logging is required.
	logging bool

	// Boolean containing value of the direct flag
	isDirect bool

	// Boolean containing value of test flag
	isTest bool

	// Boolean containing value of echo flag.
	echo bool

	// File names that are to be ignored.
	excludeFiles []string

	// Regex-friendly file names that are to be ignored.
	excludeRegex *regexp.Regexp
}

/***********************************************************************************
 *																				   *
 *					Methods to interact with boolean flags						   *
 *																				   *
 ***********************************************************************************/

/*
SetLogging is a handler method to handle the event when logging flag is used.

Depending on whether logging is required or not, will lower or increasing the value
of log messages that are to be logged.
*/
func (userInput *UserInput) SetLogging(shouldLog bool) {
	if shouldLog {
		log.SetLevel(log.TraceLevel)
		_, _ = fmt.Fprintf(
			rootCommand.OutOrStderr(),
			"\nLogging enabled \nLog level set to `Trace`\n",
		)
	} else {
		log.SetLevel(log.WarnLevel)
	}

	log.Debugf("(flag) Log level set to %v", log.GetLevel())
	userInput.logging = shouldLog
}

/*
SetDirect is a handler method to handle the event when direct flag is used.
*/
func (userInput *UserInput) SetDirect(direct bool) {
	userInput.isDirect = direct
}

/*
SetTest acts as a setter function to modify the value of the test flag.
*/
func (userInput *UserInput) SetTest(test bool) {
	userInput.isTest = test
}

/*
SetEcho acts as a setter function to modify the value of the echo flag.
*/
func (userInput *UserInput) SetEcho(echo bool) {
	userInput.echo = echo
}

/***********************************************************************************
 *																				   *
 *					Methods to interact with string flags						   *
 *																				   *
 ***********************************************************************************/

/*
SetRoot sets the path to the directory which is to be used as root directory.

Internally, this method will verify to ensure root directory exists. In case of failure,
the application will be force-stopped.
*/
func (userInput *UserInput) SetRoot(root string) {
	log.Debugf("(flag)(arg) Input for root directory: %v", root)

	item, err := os.Stat(root)
	if err != nil || !item.IsDir() {
		if err != nil {
			log.Warnf("ran into error with root directory")
			log.Warnf("(traceback): %v", err)

			_, _ = fmt.Fprintf(
				rootCommand.ErrOrStderr(),
				"Error: Ran into error while attempting to locate root "+
					"directory. \n\nError Traceback: \n%v\n",
				err,
			)
		} else {
			log.Warnf("root path does not point to a directory")
			log.Debugf("(traceback): %v", err)

			_, _ = fmt.Fprintf(
				rootCommand.ErrOrStderr(),
				"Error: Path to root directory is not a directory. \n\n"+
					"Error Traceback: \n%v\n",
				err,
			)
		}

		// Finally, force-stop the application as the value of root directory is
		// incorrect.
		os.Exit(RootDirectoryIncorrect)
	}

	// Setting this directory as the root directory if the flow-of-control reaches here.
	userInput.rootPath = root
}

/*
SetFFmpegPath is a setter to set a custom location for FFmpeg executable.

Returns false if the path is invalid
*/
func (userInput *UserInput) SetFFmpegPath(path string) {
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		log.Errorf("ffmpeg path invalid: `%v`", path)
		log.Debugf("(traceback) %v", err)
		_, _ = fmt.Fprintf(
			rootCommand.ErrOrStderr(),
			"Error: ffmpeg path invalid: `%v`\n",
			path,
		)

		os.Exit(ExecutableNotFound)
	}

	userInput.ffmpegPath = path
	log.Debugf("(flag) Set ffmpeg path at: %v", path)
}

/*
SetFFprobePath is a setter to set-up a custom location FFprobe executable.

Returns false if the location is invalid.
*/
func (userInput *UserInput) SetFFprobePath(path string) {
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		log.Errorf("(flag) ffprobe path invalid: `%v`", path)
		log.Debugf("(traceback) %v", err)
		_, _ = fmt.Fprintf(
			rootCommand.ErrOrStderr(),
			"Error: ffprobe path invalid: `%v`\n",
			path,
		)

		os.Exit(ExecutableNotFound)
	}

	userInput.ffprobePath = path
	log.Debugf("(flag) Set ffprobe path at: %v", path)
}

/*
SetExcludeFiles parses a string and forms a list of the files that are to be excluded.

This method will internally parse the string, breaking the string into multiple values
based on commas as required.
*/
func (userInput *UserInput) SetExcludeFiles(exclude string) {
	// This string can consist of one or more files/paths separated by a comma.
	// Splitting the string based on the value of comma.
	values := strings.Split(exclude, ",")

	for i := range values {
		// Trimming excess spaces, and forward/backward trailing slashes from the
		// string. The latter does not include slashes at the beginning since they
		// could be valid in case the complete path is being used as the input.
		values[i] = strings.TrimRight(strings.TrimSpace(values[i]), "/\\")

		if values[i] == "" {
			// If string is empty, removing it - replacing it with the last element
			// of the slice, and reducing the size of slice by 1.
			values[len(values)-1], values[i] = values[i], values[len(values)-1]
			values = values[:len(values)-1]
		}
	}

	userInput.excludeFiles = values
}

/*
SetExcludeRegex is a setter method to compile a string into a regex pattern and set
the pattern internally.

In case of failure, the application will be force-stopped.
*/
func (userInput *UserInput) SetExcludeRegex(regex string) {
	result, err := regexp.Compile(regex)
	if err != nil {
		log.Warnf("ran into error while compiling regex pattern: %v", regex)
		log.Debugf("(traceback): %v", err)

		_, _ = fmt.Fprintf(
			rootCommand.ErrOrStderr(),
			"Error: Ran into error while compiling regex pattern. "+
				"\n\nTraceback: %v\n",
			err,
		)

		os.Exit(RegexError)
	}

	userInput.excludeRegex = result
}

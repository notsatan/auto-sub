/*
Package internals contains the central command that is invoked when the script is run.
This includes the various flags that are available for use with the command, helper
functions/methods/structures and any other internal components that are required.
*/
package internals

import (
	"fmt"
	"github.com/demon-rem/auto-sub/internals/commons"
	"github.com/demon-rem/auto-sub/internals/ffmpeg"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
)

/*
GetRootCommand is a convenience function to create and return a pointer to root command.

The main usage of a generator function is to be able to generate a fresh copy of the
root command when required in tests.
*/
func getRootCommand() *cobra.Command {
	return &cobra.Command{
		// Shortened usage sample
		Use: fmt.Sprintf("%s [\"/path/to/root\"] [flags]", title),

		// Short description - starting line in help.
		Short: fmt.Sprintf(
			"%s \n  Command line utility to simplify soft-subbing videos",
			title,
		),

		Long:    "",
		Example: "",
		Version: version,

		// Ensure number of arguments does not exceed expectations.
		Args: func(cmd *cobra.Command, args []string) error {
			// This function is run before Logging can be modified - as such, stuff
			// inside this function should be logged at `Warn` or higher.

			if len(args) > maxInputArgs {
				// Directly return an error if arguments passed exceeds allowed count.
				log.Warnf(
					"(rootCmd/Args) found %v args instead of %v! \n\nArgs: `%v`",
					len(args),
					maxInputArgs,
					args,
				)

				return fmt.Errorf(
					"At most %d argument(s) required. Found %v.\n",
					maxInputArgs,
					len(args),
				)
			}

			// If the command has `maxInputArgs` or less, making sense of each argument
			// and assigning values to them.

			// Iterate through each argument and use the value in it as required - using
			// a for-loop with a switch statement; ensures that changes in the future
			// can be easily implemented. Throwing an error default case - ensuring that
			// unhandled cases will result in a crash instead of simply ignoring the
			// input - easier to detect and fix.
			for i := 0; i < len(args); i++ {
				switch i {
				case 0:
					// The first argument will be full path to the directory which is
					// to be used as root
					if dir, err := os.Stat(args[0]); err != nil || !dir.IsDir() {
						log.Warnf("(rootCmd/Args) invalid root: `%v`", args[i])

						return fmt.Errorf(
							"Invalid value for root directory\n",
						)
					} else {
						if userInput.RootPath == "" {
							// Use this value only if the root path is not set already,
							// ensures flag has higher priority
							userInput.RootPath = args[i]
						}
					}

				default:
					// This default case is set to throw an error - if more arguments
					// are added in the future, each case will need to be individually
					// handled. If the count for allowed argument(s) is increased
					// without handling the argument in this switch statement, a crash
					// will occur, converting a logic bug into a runtime error - easier
					// to detect and fix.
					log.Warnf("(rootCmd/Args) unexpected case on argument %v", i)

					return fmt.Errorf("Unexpected internal error\n")
				}
			}

			return nil
		},

		// The main method that will handle the back-end when the command is executed.
		Run: func(cmd *cobra.Command, args []string) {
			/*
				Simple internal method to be able to directly send stuff to `stderr`
				acts as a layer of abstraction - and for ease of use.

				Note: Output that is meant to be read by the user should be sent to
				`stderr` instead of `stdout`.
			*/
			stderr := func(format string, printable ...interface{}) {
				_, _ = fmt.Fprintf(
					cmd.OutOrStderr(),
					format,
					printable...,
				)
			}

			// When the flow-of-control reaches this part, values have been inferred
			// from the flags being used and have been assigned to the variables
			// as required.
			//
			// Running initialization method to allow the structure to internally
			// process user input as the first step.
			errCode, err := userInput.Initialize()

			// If user input was unexpected/incorrect, the method call above will
			// return an error for the same. Using that error to force-stop the app
			if err != nil || errCode != commons.StatusOK {
				// Force-stop if an error occurred.
				log.Warnf("(rootCmd/Run) unexpected user input: %v", err)
				stderr(
					"Error: Ran into an error while processing input. " +
						"Check logs for details!\n\n",
				)

				os.Exit(errCode)
			}

			// Modify the log level if needed.
			if userInput.Logging {
				log.SetLevel(log.TraceLevel)
				stderr("\nLogging enabled \nLog level set to `Trace`\n")
			}

			// Log user input - logs user input if Logging is allowed.
			userInput.Log()

			// Print the versions found, and quit - ensuring that the test flag can't
			// be combined with any other flag.
			if userInput.IsTest {
				ffmpegVersion, ffprobeVersion := handlerTest()
				if ffmpegVersion == "" || ffprobeVersion == "" {
					stderr(
						"Ran into an unexpected error! Attempting fallback \n\t"+
							"FFmpeg Version: `%v` \n\tFFprobe Version: `%v`",
						ffmpegVersion,
						ffprobeVersion,
					)

					os.Exit(commons.ExecNotFound)
				} else {
					stderr(
						"FFmpeg version found: `%v`"+
							"\nFFprobe version found: `%v`",
						ffmpegVersion,
						ffprobeVersion,
					)

					os.Exit(commons.StatusOK)
				}
			}

			// Walk through root directory - will result in error if path is incorrect.
			if files, err := ioutil.ReadDir(userInput.RootPath);
				userInput.RootPath == "" {
				log.Debugf("(rootCmd/Run) root path empty!")
				stderr(
					"Error: Empty root path detected",
				)

				os.Exit(commons.RootDirectoryIncorrect)
			} else if err != nil {
				// Force-stop the application if it runs into an unexpected error.
				log.Warnf(
					"(rootCmd/Run) ran into error traversing root directory: %v"+
						"\n(traceback): %v",
					userInput.RootPath,
					err,
				)

				stderr("Error: Ran into an error with the root directory\n\n")
				os.Exit(commons.UnexpectedError)
			} else {
				ffmpeg.TraverseRoot(
					&userInput,
					&files,
					stderr,
				)
			}
		},
	}
}

/*
handlerTest is a function designed to consume the `test` flag of the root command. This
function will attempt to test the entire setup - to be used by users after to check if
all dependencies are present as required.

Will attempt to fetch the versions for `ffmpeg` and `ffprobe` in the back-end and return
the same to calling method.

Return value of empty string(s) signifies an error occurred while attempting to call
the executable(s) - in case of an error, the traceback will be logged.
*/
func handlerTest() (ffmpegVersion, ffprobeVersion string) {
	// Compiling a regex pattern to fetch the next word after the word `version`.'
	// Specifically designed to be able to fetch the version tag from the output of
	// `-version` flag. Might need to change it if the output of ffmpeg is modified in
	// the future.
	//
	// Note: Using `MustCompile` - will fail if regex is incorrect.
	regex := regexp.MustCompile(`version (\S*)`)

	// Running ffmpeg executable with a `-version` flag.
	output, err := exec.Command(userInput.FFmpegPath, "-version").Output()
	if err != nil {
		// If error occurs, log and proceed normally - `ffmpegVersion` will remain blank
		log.Warnf("(rootCmd/handlerTest) failed to fetch ffmpeg version: \n%v", err)
	} else {
		// Extracting version from the output of the command.
		//
		// Note: Using `regex.FindSubmatch` to be able to extract data from the capture
		// group present in regex. The first index in the result will be the entire
		// string that matches the regex patter, following this, (index 1 and on) will
		// be the contents matching the capture group(s) sequentially.
		//
		// Extracting info from the first capture group (at index 1) directly. If the
		// output of `ffmpeg -version` command changes in the future, this may need
		// to be modified.
		ffmpegVersion = string(regex.FindSubmatch(output)[1])
	}

	// Running the same command for ffprobe
	output, err = exec.Command(userInput.FFprobePath, "-version").Output()
	if err != nil {
		// If error occurs, log and proceed - `ffprobeVersion` will be a blank string.
		log.Warnf("(rootCmd/handlerTest) failed to fetch ffprobe version: \n%v", err)
	} else {
		// Note: Using `regex.FindSubmatch` - same as above. Might need to modify this
		// if the output of version command changes.
		ffprobeVersion = string(regex.FindSubmatch(output)[1])
	}

	// If `err` was not null in any scenario, the string will be empty.
	return ffmpegVersion, ffprobeVersion
}

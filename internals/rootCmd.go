/*
Package internals contains the central command that is invoked when the script is run.
This includes the various flags that are available for use with the command, helper
functions/methods/structures and any other internal components that are required.
*/
package internals

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
			if len(args) > maxInputArgs {
				// Directly return an error if arguments passed exceeds allowed count.
				log.Warnf("running the script with %v arguments", len(args))
				log.Debugf("Arguments detected: %v", args)
				return fmt.Errorf(
					"accepts at most %d arguments. Found %v",
					maxInputArgs,
					len(args),
				)
			}

			return nil
		},

		// The main method that will handle the back-end when the command is executed.
		RunE: func(cmd *cobra.Command, args []string) error {
			// When the flow-of-control reaches this part, values have been inferred
			// from the flags being used and have been assigned to the variables
			// as required.
			//
			// Running initialization method to allow the structure to internally
			// process user input as the first step.
			errCode, err := userInput.Initialize()

			// A simple internal method to be able to directly send stuff to `stderr`
			// acts as a layer of abstraction - and for ease of use.
			stderr := func(format string, printable ...interface{}) {
				_, _ = fmt.Fprintf(
					cmd.OutOrStderr(),
					format,
					printable...,
				)
			}

			// If user input was unexpected/incorrect, the method call above will
			// return an error for the same. Using that error to force-stop the app
			if err != nil || errCode != StatusOK {
				// Force-stop if an error occurred.
				log.Warnf("ran into an error processing user input: %v", err)
				stderr(
					"Error: Ran into an error while processing input. " +
						"Check logs for details!\n",
				)

				os.Exit(errCode)
			}

			// Setting up the log level as required.
			if userInput.logging {
				log.SetLevel(log.TraceLevel)
				stderr("\nLogging enabled \nLog level set to `Trace`\n")
			} else {
				log.SetLevel(log.WarnLevel)
			}

			// Logging user input - this logs sorted version of the user input.
			userInput.Log()

			if userInput.isTest {
				output := ""

				ffmpegVersion, ffprobeVersion := HandlerTest()
				if ffmpegVersion == "" && ffprobeVersion == "" {
					output = "Ran into an unexpected error! Attempting fallback \n" +
						"Detected FFmpeg Version: %v \nDetected FFprobe Version: %v"
				} else {
					output = "Detected FFmpeg Version: %v" +
						"\nDetected FFprobe Version: %v\n"
				}

				stderr(
					output,
					ffmpegVersion,
					ffprobeVersion,
				)

				// Direct exit. Test flag can't be combined with normal flags.
				os.Exit(StatusOK)
			}

			return err // Will be null
		},
	}
}

/*
HandlerTest will attempt to test the entire setup - designed to be used by users after
installation to ensure all dependencies are present as required.

Will simply fetch the versions of `ffmpeg` and `ffprobe` in the back-end and return them
to the calling method.

Return value of empty string(s) signifies an error occurred while attempting to call
the executable(s) - in case of an error, the traceback will be logged.
*/
func HandlerTest() (ffmpegVersion, ffprobeVersion string) {
	// Compiling a regex pattern to fetch the next word after the word `version`.'
	// Specifically designed to be able to fetch the version tag from the output of
	// `-version` flag. Might need to change it if the output of ffmpeg is modified in
	// the future.
	//
	// Note: Using `MustCompile` - will fail if regex is incorrect.
	regex := regexp.MustCompile(`version (\S*)`)

	// Running ffmpeg executable with a `-version` flag.
	output, err := exec.Command(userInput.ffmpegPath, "-version").Output()
	if err != nil {
		log.Warnf(
			"failed to fetch ffmpeg version: \n%v",
			err,
		)
	} else {
		// Extracting version from the output of the command.
		//
		// Note: Using `regex.FindSubmatch` to be able to extract data from the capture
		// group present in regex. The first index in the result will be the entire
		// string that matched, following this, from second position (index 1) will
		// be the contents matching the capture group(s) sequentially.
		//
		// Extracting info from the first capture group (at index 1) directly. If the
		// output of `ffmpeg -version` command changes in the future, this may need
		// to be modified.
		ffmpegVersion = string(regex.FindSubmatch(output)[1])
	}

	// Running the same command for ffprobe
	output, err = exec.Command(userInput.ffprobePath, "-version").Output()
	if err != nil {
		log.Warnf(
			"failed to fetch ffprobe version: \n%v",
			err,
		)
	} else {
		// Note: Using `regex.FindSubmatch` - same as above. Might need to modify this
		// if the output of version command changes.
		ffprobeVersion = string(regex.FindSubmatch(output)[1])
	}

	// If `err` was null in any scenario, the string will be empty.
	return ffmpegVersion, ffprobeVersion
}

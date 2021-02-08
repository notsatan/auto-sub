/*
Package internals contains the central command that is invoked when the script is run.
This includes the various flags that are available for use with the command, helper
functions/methods/structures and any other internal components that are required.
*/
package internals

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/demon-rem/auto-sub/internals/commons"
	"github.com/demon-rem/auto-sub/internals/ffmpeg"
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

		Args: func(cmd *cobra.Command, args []string) error {
			// Flags will already be set, changing the value of the logger if required
			if userInput.Logging {
				log.SetLevel(log.TraceLevel)
			}

			// Iterate through each arg. Using a switch block to improve readability.
			// Default case should throw an error - ensures each argument is handled,
			// and if one isn't, the application crashes
			for i := 0; i < len(args); i++ {
				switch i {
				case 0:
					// Path to root directory - skip checking validity. Will check
					// root path obtained through the flag or as an argument at once
					//
					// Skip this value if the variable is set already - ensures the
					// flag has a higher priority
					if userInput.RootPath == "" {
						userInput.RootPath = args[i]
					} else {
						log.Debugf(
							"(rootCmd/Args) ignoring argument #%d, root path "+
								"already present \nignoring value: \"%s\"",
							i,
							args[i],
						)
					}

				default:
					// Throw an error if any valid argument is being ignored - converts
					// logical error into runtime error; at least easier to detect :p
					log.Warnf(
						"(rootCmd/Args) unexpected argument %d\nvalue: `%s`",
						i,
						args[i],
					)

					return fmt.Errorf("unexpected internal error")
				}
			}

			return nil
		},

		/*
			Runs after `command.Args()`, by the time this function runs, only the flags
			and args have been parsed.

			This method will simply validate user input - failing if the input is
			invalid.
		*/
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Setting up the output stream if not set already - this check will be
			// useful when multiple copies of the root command are being created,
			// especially during tests.
			if commons.GetOutput() == nil {
				// Deciding the stream to which output is to be written.
				oStream := cmd.OutOrStderr()
				commons.SetOutput(oStream)
			}

			// Validate user input. Force-stop if this step fails. The method call will
			// internally validate the root path, and log user input.
			//
			// Note: The function will allow flow-of-control to pass even if the root
			// path is empty as long as the test flag is present
			if errCode, err := userInput.Initialize(); err != nil ||
				errCode != commons.StatusOK {
				log.Warnf(
					"(rootCmd/RunE) unexpected input\nerror: `%v`\nexit code: %d",
					err,
					errCode,
				)

				// Form output message depending on exit code
				outMsg := ""
				switch errCode {
				case commons.RegexError:
					outMsg = "Error: Failed to compile the regex pattern"

				case commons.UnexpectedError:
					outMsg = "Error: Path to root directory is incorrect"

				case commons.RootDirectoryIncorrect:
					// Will be the case if path to root directory is not present and
					// `test` flag is not used.
					//
					// Returning error to have `RunE` function display command help
					return errors.New("path to root directory not found")

				default:
					// Will end up here in case of `UnexpectedError` or any other
					// error code introduced in the future.

					outMsg = "Error: Ran into an unexpected error. Check the logs for" +
						" details"
				}

				commons.Printf(outMsg + "\n\n")
				os.Exit(errCode)
			}

			return nil
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			if userInput.Logging {
				// Log level will be internally modified while validating user input,
				// printing confirmation to the screen in here
				commons.Printf("\nLogging enabled \nLog level set to `Trace`\n")
			}

			// Handle the test flag - once done, direct exit, ensuring that the test
			// flag can't be combined with any other flag
			if userInput.IsTest {
				ffmpegVersion, ffprobeVersion := handlerTest()
				if ffmpegVersion == "" || ffprobeVersion == "" {
					commons.Printf(
						"Ran into an unexpected error! Attempting fallback \n\t"+
							"FFmpeg Version: %v\n\tFFprobe Version: %v\n\n",
						ffmpegVersion,
						ffprobeVersion,
					)

					os.Exit(commons.ExecNotFound)
				} else {
					commons.Printf(
						"FFmpeg version found: %v\n"+
							"FFprobe version found: %v\n\n",
						ffmpegVersion,
						ffprobeVersion,
					)

					// Exit using this exit code - this is being tested against
					os.Exit(commons.StatusOK)
				}
			}

			// TODO: The root directory is to be treated as the source directory

			// Root path has already been validated, simply pass the flow of control
			// to the next section
			ffmpeg.TraverseRoot(
				&userInput,

				// Defaulting output directory to `<root-dir>/auto-sub [output]`
				filepath.Join(
					userInput.RootPath,
					fmt.Sprintf("%s [output]", title),
				),
			)

			return nil
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

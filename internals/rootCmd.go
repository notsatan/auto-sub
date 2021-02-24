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
	"path/filepath"

	"github.com/demon-rem/auto-sub/internals/commons"
	"github.com/demon-rem/auto-sub/internals/ffmpeg"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cmd = &cobra.Command{
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

	/*
		Runs after `command.Args()`, by the time this function runs, only the flags
		and args have been parsed.

		This method will simply validate user input - failing if the input is
		invalid.
	*/
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Setting up the output stream if not set already - this check will be
		// useful when multiple copies of the root command are being created,
		// especially for tests.
		if commons.GetOutput() == nil {
			commons.SetOutput(cmd.OutOrStderr())
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

		log.Debugf("(rootCmd/PreRunE) user input initialized")
		return nil
	},

	Args: func(cmd *cobra.Command, args []string) error {
		// Changing the value of the logger if required; making this change in here
		// as this method is run before the other methods (even before `PreRunE`) :/
		if userInput.Logging {
			log.SetLevel(log.TraceLevel)
			log.Debugf("(rootCmd/Args) modify logger level to `trace`")
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
					log.Debugf(`(rootCmd/Args) root path: "%s"`, args[i])
					userInput.RootPath = args[i]
				} else {
					log.Debugf(
						"(rootCmd/Args) ignore argument #%d \nroot path "+
							"already present\n ignored input: \"%s\"",
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

		log.Debugf("(rootCmd/Args) user input received through arguments")
		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		if userInput.Logging {
			// Log level will be internally modified while validating user input,
			// printing confirmation to the screen in here
			commons.Printf("\nLogging enabled \nLog level set to `Trace`\n\n")
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

				// Direct exit - test flag can't be combined with any other flag
				log.Debugf("(rootCmd/RunE) test flag found, direct exit")
				os.Exit(commons.StatusOK)
			}
		}

		// Root path has been validated already
		_, _ = ffmpeg.TraverseRoot( // TODO: Don't ignore these values
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

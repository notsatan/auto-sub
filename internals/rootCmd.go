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

const (
	// String containing current version - should be updated with new(er) releases. Do
	// not add `v` or `Version` or any other prefixes to this.
	version = "0.0.1"

	// Project title - used in sample commands and stuff
	title = "auto-sub"
)

var cmd = &cobra.Command{
	// Shortened usage sample
	Use: fmt.Sprintf("%s [\"/path/to/root\"] [flags]", title),

	// Short description - starting line in help.
	Short: fmt.Sprintf(
		"%s \n  Command line utility to simplify soft-subbing videos",
		title,
	),

	Long: `
A command-line utility tool to batch add subtitles, attachments
and/or chapters to multiple media files using FFmpeg.

**Important**: Requires FFmpeg in the backend. Make sure to have
FFmpeg installed, test your setup with the ` + "`--test`" + ` flag to verify.

File types are recognized through their extensions, the resultant
file will always be in a matroska (mkv) container.

The subtitle stream language/title can be modified using flags
`,

	Version: version,

	/*
		Runs after `command.Args()`, by the time this function runs, only the flags
		and args have been parsed.

		This method will simply validate user input - failing if the input is
		invalid.
	*/
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Setting up the output stream, the check will be useful when the main method
		// is being called multiple times (during tests)
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
		// Changing the value of the logger if required; making this change here
		// since this method is run before the other methods (even before `PreRunE`) :/
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

				return fmt.Errorf(`unexpected argument "%s"`, args[i])
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

		if userInput.IsTest {
			// Handle the test flag - once done, direct exit, ensuring that the test
			// flag can't be combined with any other flag
			exitCode := handleTestFlag()

			// Direct exit
			log.Debugf("(rootCmd/RunE) test flag found, direct exit")
			os.Exit(exitCode)
		}

		// Root path has been validated already
		exitCode, err := ffmpeg.TraverseRoot(
			&userInput,

			// Defaulting output directory to `<root-dir>/auto-sub [output]`
			filepath.Join(
				userInput.RootPath,
				fmt.Sprintf("%s [output]", title),
			),
		)

		if exitCode != commons.StatusOK || err != nil {
			if exitCode == commons.StatusOK {
				// If `err` is not null, the application will be force-stopped. In the
				// unlikely scenario when `err` is not null, but the exit code is
				// normal, this block of code will change the exit code to signify a
				// crash.
				log.Debugf(
					"(rootCmd/RunE) modify value of exit code received."+
						"\noriginal value: %d \nupdated value: %d",
					commons.StatusOK,
					commons.UnexpectedError,
				)

				exitCode = commons.UnexpectedError
			}

			log.Debugf(
				"(rootCmd/RunE) force-kill due to failure in `ffmpeg.Traverse()`"+
					"\nexit code: %d \nerror: %v",
				exitCode,
				err,
			)

			commons.Printf("Error: %v", err)
			if err := cmd.Help(); err != nil {
				log.Debugf(
					"(rootCmd/RunE) an error occurred while printing the help "+
						"message \ntraceback: %v",
					err,
				)
			}

			os.Exit(exitCode)
		}

		return nil
	},
}

func handleTestFlag() (exitCode int) {
	ffmpegVersion, ffprobeVersion := handlerTest()
	if ffmpegVersion == "" || ffprobeVersion == "" {
		commons.Printf(
			"Ran into an unexpected error! Attempting fallback\n\t"+
				"FFmpeg Version: %v\n\tFFprobe Version: %v\n\n",
			ffmpegVersion,
			ffprobeVersion,
		)

		return commons.ExecNotFound
	}

	commons.Printf(
		"FFmpeg version found: %v\n"+
			"FFprobe version found: %v\n\n",
		ffmpegVersion,
		ffprobeVersion,
	)

	return commons.StatusOK
}

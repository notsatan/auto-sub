/*
Package internals contains the central command that is invoked when the script is run.
This includes the various flags that are available for use with the command, and helper
functions as well.
*/
package internals

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Declaring maps containing strings as the key and functions as the result.
// This project will consist of a single root command with multiple flags.
//
// This would result in a single handler having a large if-else ladder to handle each
// flag. Instead, avoiding this using two maps.
//
// One map will contain a list of functions accepting strings as key-value pairs, and
// the other map containing a list of functions using bool key-value pairs.
var boolFlags = make(map[string]func(bool))
var stringFlags = make(map[string]func(string))

var (
	// String containing current version - should be updated with new(er) releases. Do
	// not add `v` or `version` or any other prefixes to this.
	version = "0.0.1"

	// Executable that is to be run - will be used in the command when fired.
	executable = "ffmpeg"

	// Project title - used in sample commands, and as the project repo url.
	title = "auto-sub"
)

// Exit code when initial check fails.
var codeInitialCheckFail = 10

// Maximum number of input arguments allowed - acts as layer of abstraction; ensuring
// changes to this value do not break tests. All arguments are to be optional.
var maxInputArgs = 1

// The root command. Defines the behavior of the base command when fired. The
// flow-of-control will be decided depending on values of arguments and flags.
var rootCommand = &cobra.Command{
	// Shortened usage sample
	Use: fmt.Sprintf("%s [\"/path/to/root\"] [flags]", title),

	// Short description - starting line in help.
	Short: fmt.Sprintf(
		"%s \n  Command line utility to simplify soft-subbing videos",
		title,
	),

	Long:    "",
	Example: "",

	// Supplying this value here as a backup - the version flag will be overridden with
	// more stylized output.
	Version: version,

	// Validate argument count; ensure number of arguments does not exceed expectations.
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > maxInputArgs {
			// Directly return an error if arguments passed exceeds allowed count.
			log.Debugf("Attempt to run the script with %v arguments\n", len(args))
			log.Debugf("Arguments detected: %v\n", args)
			return fmt.Errorf(
				"accepts at most %d arguments. Found %v",
				maxInputArgs,
				len(args),
			)
		}

		return nil
	},

	// The main method that will handle the back-end when the command is executed.
	Run: func(cmd *cobra.Command, args []string) {
		// Iterating over all keys present in the map, and firing the corresponding
		// functions if the flag is detected.
		for key, value := range boolFlags {
			if result, err := cmd.Flags().GetBool(key); err == nil {
				// Invoking the function with its value.
				value(result)
			}
		}

		// Repeating the same for the map of string key-value pairs.
		for key, value := range stringFlags {
			if result, err := cmd.Flags().GetString(key); err == nil {
				// Invoking the function with string value.
				value(result)
			}
		}
	},
}

/*
Execute is the point of contact between the main function and the internal workings of
the script.

This method will directly run the root command - from where the flow of control will be
modified depending the flag used by the user and more.
*/
func Execute() {
	// Performing initial check - ensures that everything required is in order. Failure
	// in this step indicates missing requirements.
	if result, errorMessage := performCheck(); !result {
		log.Errorf(errorMessage)
		log.Errorf("force stop execution")
		os.Exit(codeInitialCheckFail)
	}

	// Template version string - overrides the output displayed when `--version` flag
	// is run - the default output is ugly.
	rootCommand.SetVersionTemplate(
		fmt.Sprintf(
			"\n%s v%s\n"+
				"Licensed under MIT"+
				"\n",
			title,
			version,
		),
	)

	// Flag to enable/disable logging - using this will internally reduce the level
	// at which log messages are being recorded.
	boolFlags["log"] = flagLog
	rootCommand.Flags().Bool(
		"log",
		false,
		"Generate logs for the current run",
	)

	if rootErr := rootCommand.Execute(); rootErr != nil {
		log.Warn(fmt.Printf("\nEncountered error while running: %v\n", rootErr))
	}
}

/*
Handler method to handle the event when the logging flag is used.

Will lower the logging level to be at trace - ensuring log messages are now captured.
*/
func flagLog(shouldLog bool) {
	if shouldLog {
		log.SetLevel(log.TraceLevel)
		log.Warn("Logging enabled.")
		log.Debug("Log level set to trace.")
	} else {
		log.SetLevel(log.WarnLevel)
		log.Debug("Logging disabled")
		log.Debug("Log level set to warn.")
	}
}

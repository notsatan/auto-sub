/*
Package internals contains the central command that is invoked when the script is run.
This includes the various flags that are available for use with the command, and helper
functions as well.
*/
package internals

import (
	"fmt"

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
}

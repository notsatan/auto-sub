package internals

import (
	"fmt"
	"os/exec"

	log "github.com/sirupsen/logrus"
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

	// Project title - used in sample commands, and as the project repo url.
	title = "auto-sub"
)

/*
Strings containing the locations for FFMPEG and FFProbe executables.

Should default to empty strings.
*/
var (
	ffmpegLocation  = ""
	ffprobeLocation = ""
)

// Maximum number of UserInput arguments allowed - acts as layer of abstraction;
// ensuring changes to this value do not break tests. All arguments are to be optional.
var maxInputArgs = 1

var userInput UserInput

/*
Possible exit codes
*/
var (
	// One or more of required executables couldn't be found.
	ExecutableNotFound = 10

	// Path to root directory is invalid / item is not a directory.
	RootDirectoryIncorrect = 11

	// Ran into an error while attempting to compile regex pattern
	RegexError = 12
)

// Central copy of the root command.
var rootCommand = getRootCommand()

/*
Execute is the point of contact between the main function and the internal workings of
the script.

This method will directly run the root command - from where the flow of control will be
modified depending the flag used by the user and more.
*/
func Execute() {
	// Fetch current location for `ffmpeg` and `ffprobe` executables. The function call
	// will implicitly populate the global variables if an executable is found.
	_, _ = fetchLocation()

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

	setBoolFlags()

	stringFlags["ffmpeg"] = userInput.SetFFmpegPath
	rootCommand.Flags().String(
		"ffmpeg",
		ffmpegLocation, // Empty string if not found
		"Path to ffmpeg executable",
	)

	stringFlags["ffprobe"] = userInput.SetFFprobePath
	rootCommand.Flags().String(
		"ffprobe",
		ffprobeLocation, // Empty string if not found
		"Path to ffprobe executable",
	)

	if rootErr := rootCommand.Execute(); rootErr != nil {
		log.Warn(fmt.Printf("\nEncountered error while running: %v", rootErr))
	}
}

/*
SetBoolFlags is a simple function that acts as a layer of abstraction and adds the
required boolean flags to the root command. At the same time, each of these flags is
also added to the `boolFlags` map that keeps a track of the boolean flags and the
functions that will handle them as needed.
*/
func setBoolFlags() {
	// Flag to enable/disable logging - using this will internally reduce the level
	// at which log messages are being recorded.
	boolFlags["log"] = userInput.SetLogging
	rootCommand.Flags().Bool(
		"log",
		false,
		"Generate logs for the current run",
	)

	boolFlags["direct"] = userInput.SetDirect
	rootCommand.Flags().Bool(
		"direct",
		false,
		"Use the root directory as the main directory",
	)

	boolFlags["test"] = userInput.SetTest
	rootCommand.Flags().Bool(
		"test",
		false,
		"Run test to verify dependencies",
	)

	boolFlags["echo"] = userInput.SetEcho
	rootCommand.Flags().Bool(
		"echo",
		false,
		"Echo the commands being fired instead of executing them",
	)
}

/*
FetchLocation fetches the location for ffmpeg and ffprobe executables if found.

If either value is not found, the function will return an error in the corresponding
result, and leave global string empty.
*/
func fetchLocation() (error, error) {
	var err01 error
	var err02 error

	ffmpegLocation, err01 = exec.LookPath("ffmpeg")
	if err01 != nil {
		ffmpegLocation = ""
		log.Warnf("Unable to locate ffmpeg executable: %v", err01)
	} else {
		log.Debugf("(default) Found ffmpeg at: %v", ffmpegLocation)
	}

	ffprobeLocation, err02 = exec.LookPath("ffprobe")
	if err02 != nil {
		ffprobeLocation = ""
		log.Warnf("Unable to locate ffprobe executable: %v", err02)
	} else {
		log.Debugf("(default) Found ffprobe at: %v", ffprobeLocation)
	}

	return err01, err02
}

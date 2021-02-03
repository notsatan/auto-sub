package internals

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/demon-rem/auto-sub/internals/commons"

	log "github.com/sirupsen/logrus"
)

var (
	// String containing current Version - should be updated with new(er) releases. Do
	// not add `v` or `Version` or any other prefixes to this.
	version = "0.0.1"

	// Project Title - used in sample commands, and as the project repo url.
	title = "auto-sub"
)

// Maximum number of UserInput arguments allowed - acts as layer of abstraction;
// ensuring changes to this value do not break tests. All arguments are to be optional.
var maxInputArgs = 1

var userInput commons.UserInput

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
	ffmpegPath, ffprobePath := fetchLocation()

	// Template Version string - overrides the output displayed when `--Version` flag
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

	// Adding flags to root command.
	addBoolFlags()
	addStringFlags(
		ffmpegPath,
		ffprobePath,
	)

	if rootErr := rootCommand.Execute(); rootErr != nil {
		log.Errorf("\nEncountered error while running: %v", rootErr)
		_, _ = fmt.Fprintf(
			rootCommand.OutOrStderr(),
			"\nEncountered an unexpected error! Check logs for details\n",
		)

		os.Exit(commons.UnexpectedError)
	}
}

/*
SetBoolFlags is a simple function that acts as a layer of abstraction and adds the
required boolean flags to the root command. At the same time, each of these flags is
also added to the `boolFlags` map that keeps a track of the boolean flags and the
functions that will handle them as needed.
*/
func addBoolFlags() {
	// Flag to enable/disable Logging - using this will internally reduce the level
	// at which log messages are being recorded.
	rootCommand.Flags().BoolVar(
		&userInput.Logging,
		"log",
		false,
		"Enable Logging for the current run",
	)

	rootCommand.Flags().BoolVar(
		&userInput.IsTest,
		"test",
		false,
		"Run test to verify dependencies",
	)

	rootCommand.Flags().BoolVar(
		&userInput.IsDirect,
		"direct",
		false,
		"Use the root direct as the main directory",
	)
}

/*
Simple function that will attach all string flags to the root command.

Enables easy testing, and segregation of the codebase based on similarities.
*/
func addStringFlags(ffmpegPath, ffprobePath string) {
	rootCommand.Flags().StringVar(
		&userInput.RootPath,
		"root",
		"",
		"Path to the root directory",
	)

	rootCommand.Flags().StringVar(
		&userInput.FFmpegPath,
		"ffmpeg",
		ffmpegPath, // Empty string if not found
		"Path to ffmpeg executable",
	)

	rootCommand.Flags().StringVar(
		&userInput.FFprobePath,
		"ffprobe",
		ffprobePath, // Empty string if not found
		"Path to ffprobe executable",
	)

	rootCommand.Flags().StringVar(
		&userInput.ExcludeFiles,
		"exclude",
		"",
		"List of files to be ignored",
	)

	rootCommand.Flags().StringVar(
		&userInput.RegexExclude,
		"rexclude",
		"",
		"Regex pattern to dictate files to be ignored",
	)

	rootCommand.Flags().StringVarP(
		&userInput.SubTitleString,
		"subtitle",
		"T",
		"",
		"Custom title for subtitles files",
	)

	rootCommand.Flags().StringVarP(
		&userInput.SubLang,
		"language",
		"L",
		"",
		"Subtitle language",
	)
}

/*
FetchLocation fetches the location for ffmpeg and ffprobe executables if found.

If either value is not found, the function will return an error in the corresponding
result, and leave global string empty.
*/
func fetchLocation() (ffmpegPath, ffprobePath string) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		ffmpegPath = ""
		log.Warnf("Unable to locate ffmpeg executable: %v", err)
	} else {
		log.Debugf("(default) Found ffmpeg at: %v", ffmpegPath)
	}

	ffprobePath, err = exec.LookPath("ffprobe")
	if err != nil {
		ffprobePath = ""
		log.Warnf("Unable to locate ffprobe executable: %v", err)
	} else {
		log.Debugf("(default) Found ffprobe at: %v", ffprobePath)
	}

	return ffmpegPath, ffprobePath
}

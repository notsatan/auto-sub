package internals

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/demon-rem/auto-sub/internals/commons"

	log "github.com/sirupsen/logrus"
)

var (
	// String containing current version - should be updated with new(er) releases. Do
	// not add `v` or `Version` or any other prefixes to this.
	version = "0.0.1"

	// Project title - used in sample commands and stuff
	title = "auto-sub"
)

// Maximum input arguments allowed - acts as layer of abstraction; ensuring changes to
// this value do not break tests. All arguments to be optional.
var maxInputArgs = 1

// Central copy of user input variable - used to keep a track of user input. Using a
// global variable is needed since the same variable will be read by the central
// root command (can't pass in custom parameters :/)
var userInput commons.UserInput

/*
Execute acts as the point of contact between the main function and the internal workings
of script.

This method will directly run the root command - from where flow of control is branched
to various methods/functions depending on user input
*/
func Execute() {
	// Fetch current location for `ffmpeg` and `ffprobe` executables - used by default
	// unless custom path is supplied by the user, or the executables can't be found
	ffmpegPath, ffprobePath := findBinaries()

	// Override the output for `--version` flag - default output is (relatively) ugly
	cmd.SetVersionTemplate(
		fmt.Sprintf(
			`
%s v%s
 - os/arch: %s/%s
 - go version: %s

Licensed under MIT
`,
			title,
			version,
			runtime.GOOS,
			runtime.GOARCH,
			runtime.Version(),
		),
	)

	// Add flags to root command
	boolFlags(cmd, &userInput)
	stringFlags(
		cmd,
		&userInput,
		&ffmpegPath,
		&ffprobePath,
	)

	if rootErr := cmd.Execute(); rootErr != nil {
		// Force-quit in case an error is encountered.
		log.Errorf("(cmd/Execute) encountered an error: \n%v", rootErr)
		commons.Printf(
			"\nEncountered an unexpected error! Check logs for details\n",
		)

		// Non-zero exit code
		os.Exit(commons.UnexpectedError)
	}
}

/*
BoolFlags is a simple helper function to attach boolean flags to the command
*/
func boolFlags(command *cobra.Command, input *commons.UserInput) {
	command.Flags().BoolVar(
		&input.Logging,
		"log",
		false,
		"Generate logs for the current run",
	)

	command.Flags().BoolVar(
		&input.IsTest,
		"test",
		false,
		"Run test(s) to verify your setup",
	)

	command.Flags().BoolVar(
		&input.IsDirect,
		"direct",
		false,
		"Use root directory as source directory",
	)

	// Override `help` and `version` flags - for a better output
	command.Flags().BoolP(
		"help",
		"h",
		false,
		"Show this help for auto-sub command and flags",
	)

	command.Flags().BoolP(
		"version",
		"v",
		false,
		"Display the current version number for "+title,
	)
}

/*
StringFlags is a simple helper function to add all string flags to the command.
*/
func stringFlags(command *cobra.Command, input *commons.UserInput, ffmpegPath,
	ffprobePath *string) {
	// Message to log if a flag can't be marked as required
	failMsg := "(cmd/stringFlags) failed to mark `%s` flag as required\nerror; %v"

	// Do not mark the root flag as required - it can be passed in as an argument too!
	rootFlag := "root" // easy access/modification
	command.Flags().StringVar(
		&input.RootPath,
		rootFlag,
		"",
		"Full path to root directory",
	)

	// Mark root flag as directory name (limits auto-completion) for better results
	if err := command.MarkFlagDirname(rootFlag); err != nil {
		log.Debugf(
			"(cmd/stringFlags) failed to restrict `%s` flag!\nerror; %v",
			rootFlag,
			err,
		)
	}

	ffmpegFlag := "ffmpeg" // easy modification
	command.Flags().StringVar(
		&input.FFmpegPath,
		ffmpegFlag,
		*ffmpegPath, // empty string if not found
		"Path to ffmpeg executable",
	)

	// Mark ffmpeg flag as required if the executable could not be located implicitly
	if *ffmpegPath == "" {
		if err := command.MarkFlagRequired(ffmpegFlag); err != nil {
			log.Debugf(
				failMsg,
				ffmpegFlag,
				err,
			)
		}
	}

	ffprobeFlag := "ffprobe" // easy modification
	command.Flags().StringVar(
		&input.FFprobePath,
		ffprobeFlag,
		*ffprobePath, // empty string if not found
		"Path to ffprobe executable",
	)

	// Mark ffprobe flag as required if the executable could not be located
	if *ffprobePath == "" {
		if err := command.MarkFlagRequired(ffprobeFlag); err != nil {
			log.Debugf(
				failMsg,
				ffprobeFlag,
				err,
			)
		}
	}

	command.Flags().StringSliceVarP(
		&input.Exclusions,
		"exclude",
		"E",
		[]string{},
		"List of files to be ignored",
	)

	command.Flags().StringVar(
		&input.RegexExclude,
		"rexclude",
		"",
		"Regex pattern to dictate files to be ignored",
	)

	command.Flags().StringVar(
		&input.SubTitleString,
		"subtitle",
		"",
		"Custom title for subtitles files",
	)

	command.Flags().StringVarP(
		&input.SubLang,
		"language",
		"l",
		"eng", // set default subtitle language to english
		"Subtitle language",
	)
}

/*
FindBinaries attempts to fetch location(s) for ffmpeg and ffprobe executables.

If either value is not found, the corresponding string in the result will be left
empty and the error will be internally logged (if logging is enabled)

P.S. Better name for the function would have been `fetchExecutables` - but was too long
for a function that will be used just once, and `fetchExecs` looked weird :(
*/
func findBinaries() (ffmpegPath, ffprobePath string) {
	if path, err := exec.LookPath("ffmpeg"); err != nil {
		ffmpegPath = "" // empty any existing value
		log.Debugf("(cmd/findBinaries) unable to locate ffmpeg! \n`%v`", err)
	} else {
		ffmpegPath = path
		log.Debugf("(cmd/findBinaries) ffmpeg binary found at: `%s`", ffmpegPath)
	}

	if path, err := exec.LookPath("ffprobe"); err != nil {
		ffprobePath = "" // empty any existing value
		log.Debugf("(cmd/findBinaries) unable to locate ffprobe! \n`%v`", err)
	} else {
		ffprobePath = path
		log.Debugf("(cmd/findBinaries) ffprobe found at: `%s`", ffprobePath)
	}

	return ffmpegPath, ffprobePath
}

/*
HandlerTest is a function designed to consume `test` flag. This function will attempt to
test the entire setup - to be used by users after to check if dependencies are present
as required.

Will attempt to fetch the versions for `ffmpeg` and `ffprobe` in the back-end and return
the same to calling method.

Return value of empty string(s) signifies an error occurred while attempting to call
the executable(s) - in case of an error, the traceback will be logged implicitly
*/
func handlerTest() (ffmpegVersion, ffprobeVersion string) {
	// Regex pattern to fetch the next word after the word `version` to fetch the
	// version tag from the output of the command. Might need to change it if the
	// output of ffmpeg is modified.
	regex := regexp.MustCompile(`version (\S*)`)

	// Running ffmpeg executable with a `-version` flag.
	output, err := exec.Command(userInput.FFmpegPath, "-version").Output()
	if err != nil {
		// If error occurs, log and proceed normally - `ffmpegVersion` will remain blank
		log.Warnf("(rootCmd/handlerTest) failed to fetch ffmpeg version: \n%v", err)
	} else {
		// Extracting version from the output of the command.
		//
		// Note: The first index in the result will be the entire string that matches
		// the regex pattern, following this, (index 1 and on) will be contents from the
		// capture group(s) sequentially.
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

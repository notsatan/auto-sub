package internals

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

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

	// Exit code for an unexpected internal error.
	UnexpectedError = 13

	// Exit code for a successful termination.
	StatusOK = 0
)

/*
UserInput is a basic structure to store and operate upon data passed by the user using
CLI.

Any object of this structure should make a call to `Init()` method that will internally
handle the chores as required by the method.
*/
type UserInput struct {
	// Path to the root directory containing the files
	rootPath string

	// Path to ffmpeg executable
	ffmpegPath string

	// Path to ffprobe executable
	ffprobePath string

	// Indicates if logging is required or not. True indicates logging is required.
	logging bool

	// Boolean containing value of the direct flag
	isDirect bool

	// Boolean containing value of test flag
	isTest bool

	// Boolean containing value of echo flag.
	echo bool

	// String with file names that are to be ignored.
	excludeFiles string

	// Array of strings with each string being a name of the file that is to be ignored.
	exclusions []string

	// Regex-friendly file names that are to be ignored.
	regexExclude string

	// Compiled regex expression - will be slightly faster than the normal Version.
	regexRule *regexp.Regexp
}

/*
Initialize method will initialize the values present in the structure.

This includes any sort of background house-keeping required as well. For example, one of
the main tasks performed by this method is to split the string acquired as input in
`excludeFiles` variable into an array of strings.
*/
func (userInput *UserInput) Initialize() (int, error) {
	// Splitting `excludeFiles` string into an array and processing it before using this
	// as the value for `exclusions` variable.
	result := strings.Split(userInput.excludeFiles, ",")

	// Trimming spaces from each value in the array, and removing trailing slashes.
	for i := range result {
		result[i] = strings.TrimRight(strings.TrimSpace(result[i]), "\\/")
	}

	// Setting this as the value for `exclusions`
	userInput.exclusions = result

	// Compiling the regex string into a compiled regex expression - compiled regex
	// expressions are easy to compare against.
	regex, err := regexp.Compile(userInput.regexExclude)
	if err != nil {
		return RegexError, err
	}

	userInput.regexRule = regex
	return StatusOK, err // will be nil
}

/*
Log simply logs the values values present in the structure. Acts as a convenience
method, a simple call to this method ensures that all values in the structure will be
logged as required.
*/
func (userInput *UserInput) Log() {
	log.Debugf("Logging user input: \n\nRoot path: `%v`\n"+
		"FFmpeg Executable: `%v`\n"+
		"FFprobe Executable: %v\n"+
		"Logging Enabled: %v\n"+
		"Test Mode: %v\n"+
		"Echo Commands: %v\n"+
		"Exclusions: `%v`\n"+
		"Inferred Exclusions: `%v`\n"+
		"Regex Exclusions: `%v`\n",
		userInput.rootPath,
		userInput.ffmpegPath,
		userInput.ffprobePath,
		userInput.logging,
		userInput.isTest,
		userInput.echo,
		userInput.excludeFiles,
		userInput.exclusions,
		userInput.regexExclude,
	)
}

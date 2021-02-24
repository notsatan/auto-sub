/*
Package commons contains the common resources that are to be shared by other packages
in the application - in order to avoid a cyclic dependencies, these resources are moved
under this package.
*/
package commons

import (
	"errors"
	"os"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

/*
UserInput is a simple structure to store and operate upon data passed by the user using
CLI.

Any object of this structure should make a call to `Init()` method that will internally
handle the chores as required by the method.
*/
type UserInput struct {
	// Path to the root directory containing the files
	RootPath string

	// Path to ffmpeg executable
	FFmpegPath string

	// Path to ffprobe executable
	FFprobePath string

	// Indicates if logging is required or not. True indicates Logging is required.
	Logging bool

	// Boolean containing value of the direct flag
	IsDirect bool

	// Boolean containing value of test flag
	IsTest bool

	// Array of strings with each string being a name of the file that is to be ignored.
	Exclusions []string

	// Regex-friendly file names that are to be ignored.
	RegexExclude string

	// Compiled regex expression - will be slightly faster than the normal Version.
	RegexRule *regexp.Regexp

	// Custom title for the subs file being attached
	SubTitleString string

	// Subtitle language
	SubLang string
}

/*
Initialize method will initialize the values present in the structure.

This involves validating user input, compiling the regex pattern (if any) to ensure
that the pattern is valid, validating the root path, trimming spaces/trailing slashes
from the list of exclusions, etc.

Note: This function will safely exit in case root path is empty - this check is
supposed to be made by the calling method
*/
func (userInput *UserInput) Initialize() (int, error) {
	// Trimming spaces from each value in the array, removing trailing slashes - do not
	// convert cases, messes up if a value is a full path
	for i := range userInput.Exclusions {
		userInput.Exclusions[i] = strings.TrimRight(
			strings.TrimSpace(userInput.Exclusions[i]),
			"\\/",
		)
	}

	// Compiling the regex string into a compiled regex expression - compiled regex
	// expressions are easy to compare against.
	var regex *regexp.Regexp
	if exp, err := regexp.Compile(userInput.RegexExclude); err == nil {
		// Doing it this way to make GoLint stop throwing errors :(
		regex = exp
	} else {
		// fail if regex pattern can't be compiled
		return RegexError, err
	}

	if userInput.RegexExclude != "" {
		userInput.RegexRule = regex
	} else {
		userInput.RegexRule = nil
	}

	// log user input
	userInput.log()

	switch item, err := os.Stat(userInput.RootPath); {
	case userInput.RootPath == "" && userInput.IsTest:
		// Allow an empty root path only if the test flag is present. If path to root
		// directory is preset, it will be validated (even if `test` flag is used)
		return StatusOK, nil

	case userInput.RootPath == "":
		// Explicitly handling this case for more specific exit code
		log.Debugf("(userInput/Initilaize) path to root directory is empty!")
		return RootDirectoryIncorrect,
			errors.New("path to root directory not specified")

	case err != nil:
		// Fail if root path is invalid
		log.Debugf(
			"(userInput/Initialize) non-existent path to root directory! \npath "+
				"used: \"%s\" \nerror: `%v`",
			userInput.RootPath,
			err,
		)

		return UnexpectedError, err

	case !item.IsDir():
		// Fail if path to root directory points to a file instead
		log.Debugf(
			`(userInput/Initialize) invalid path to root directory: "%s"`,
			userInput.RootPath,
		)

		return RootDirectoryIncorrect, errors.New("path to root directory invalid")

	default:
		// Pass if the root path is correct
		return StatusOK, err
	}
}

/*
IgnoreFile acts as a wrapper method that internally decides if a file is supposed to
be ignored or not based on the name of the file.

This function will internally use the value of `userInput.Exclusions` and
`userInput.RegexRule` to match against the name of the file. A response of true
indicates that the file is to be skipped
*/
func (userInput *UserInput) IgnoreFile(sourceDir, fileName *string) bool {
	// Match file name against regex pattern
	if userInput.RegexRule != nil && userInput.RegexRule.MatchString(*fileName) {
		log.Debugf(
			"(userInput/IgnoreFile) skip file; match against regex exclusion! "+
				"\nsource dir: `%v` \nfile name: `%v`",
			*sourceDir,
			*fileName,
		)

		return true
	}

	// Compare file name against all the list of file names to be excluded
	for _, exclude := range userInput.Exclusions {
		if strings.EqualFold(*fileName, exclude) {
			log.Debugf(
				"(userInput/IgnoreFile) skip file; match with exclusion rule!"+
					"\nexclusion rule: `%v` \nsource dir: `%v` \nfile name: `%v`",
				exclude,
				*sourceDir,
				*fileName,
			)

			return true
		}
	}

	// No match occurred.
	return false
}

/*
Log simply logs the values values present in the structure. Acts as a convenience
method, a simple call to this method ensures that all values in the structure will be
logged as required.
*/
func (userInput *UserInput) log() {
	log.Debugf(
		"Logging user input: \n\nRoot path: `%s`\n"+
			"FFmpeg Executable: `%s`\n"+
			"FFprobe Executable: `%s`\n"+
			"Logging Enabled: %v\n"+
			"Test Mode: %v\n"+
			`Exclusions: ["%v"]`+
			"\nRegex Exclusions: `%v`\n",
		userInput.RootPath,
		userInput.FFmpegPath,
		userInput.FFprobePath,
		userInput.Logging,
		userInput.IsTest,
		strings.Join(userInput.Exclusions, `", "`),
		userInput.RegexExclude,
	)
}

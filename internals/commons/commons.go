/*
Package commons contains the common resources that are to be shared by other packages
in the application - in order to avoid a cyclic dependencies, these resources are moved
under this package.
*/
package commons

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

/*
Possible exit codes
*/
const (
	// Path supplied for root directory is incorrect - points to a non-existent location
	// or to a file.
	RootDirectoryIncorrect = 11

	// Ran into an error while attempting to compile regex pattern
	RegexError = 12

	// Exit code for an unexpected internal error.
	UnexpectedError = 13

	// Executables not found - typically used in case FFmpeg or FFprobe can't be located
	ExecNotFound = 14

	// Exit code for a successful termination.
	StatusOK = 0
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

	// Indicates if Logging is required or not. True indicates Logging is required.
	Logging bool

	// Boolean containing value of the direct flag
	IsDirect bool

	// Boolean containing value of test flag
	IsTest bool

	// String with file names that are to be ignored.
	ExcludeFiles string

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

This includes any sort of background house-keeping required as well. For example, one of
the main tasks performed by this method is to split the string acquired as input in
`ExcludeFiles` variable into an array of strings.
*/
func (userInput *UserInput) Initialize() (int, error) {
	// Splitting `ExcludeFiles` string into an array and processing it before using this
	// as the value for `Exclusions` variable.
	result := strings.Split(userInput.ExcludeFiles, ",")

	// Trimming spaces from each value in the array, removing trailing slashes, and
	// converting it all to lower case.
	for i := range result {
		result[i] = strings.ToLower(strings.TrimRight(
			strings.TrimSpace(result[i]),
			"\\/"),
		)
	}

	// Setting this as the value for `Exclusions`
	userInput.Exclusions = result

	// Compiling the regex string into a compiled regex expression - compiled regex
	// expressions are easy to compare against.
	regex, err := regexp.Compile(userInput.RegexExclude)
	if err != nil {
		return RegexError, err
	}

	if userInput.RegexExclude != "" {
		userInput.RegexRule = regex
	} else {
		userInput.RegexRule = nil
	}

	return StatusOK, err // will be nil
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
func (userInput *UserInput) Log() {
	log.Debugf("Logging user input: \n\nRoot path: `%v`\n"+
		"FFmpeg Executable: `%v`\n"+
		"FFprobe Executable: %v\n"+
		"Logging Enabled: %v\n"+
		"Test Mode: %v\n"+
		"Exclusions: `%v`\n"+
		"Inferred Exclusions: `%v`\n"+
		"Regex Exclusions: `%v`\n",
		userInput.RootPath,
		userInput.FFmpegPath,
		userInput.FFprobePath,
		userInput.Logging,
		userInput.IsTest,
		userInput.ExcludeFiles,
		userInput.Exclusions,
		userInput.RegexExclude,
	)
}

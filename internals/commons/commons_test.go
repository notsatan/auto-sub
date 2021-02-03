package commons

import (
	"strings"
	"testing"
)

func TestIgnoreFile(t *testing.T) {
	files := []string{
		"test.exe",
		"temp_file.bat",
		"new_file.docx",
		"incorrect_file.word",
		"video.mkv",
		"same_video.jpg",
		"definitely_not_a_video.txt",
	}

	input := UserInput{
		Exclusions: []string{
			"video.mkv",
			"same_video.jpg",
		},

		RegexExclude: `\.exe`,
	}

	errCode, err := input.Initialize()
	if errCode != StatusOK {
		t.Errorf(
			"(commons/IgnoreFile) error occurred during initialization! "+
				"\nerror code: %d \nerror: %v",
			errCode,
			err,
		)
	}

	source := "source-directory"
	for i, file := range files {
		result := input.IgnoreFile(&source, &files[i])

		flag := false

		if input.RegexRule != nil && input.RegexRule.MatchString(file) {
			flag = true
		}

		for _, fileName := range input.Exclusions {
			if strings.EqualFold(fileName, file) {
				flag = true
			}
		}

		if result != flag {
			t.Errorf(
				"(commmons/IgnoreFile) failed to match the value with expected"+
					" value \nresult obtained: %v \nexpected result: %v "+
					"\nfile name: %s \nregex rule: %v \nexclusions: %v",
				result,
				flag,
				file,
				input.RegexExclude,
				input.Exclusions,
			)
		}
	}
}

package internals

import (
	"fmt"
	"os/exec"
)

/*
Helper function to perform a check to ensure everything is in order.
*/
func performCheck() (bool, string) {
	_, err := exec.LookPath(executable)
	if err != nil {
		return false, fmt.Sprintf("unable to locate package `%v`", executable)
	}

	return true, ""
}

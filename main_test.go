package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	rescueStdout := os.Stdout
	r, w, e := os.Pipe()
	if e != nil {
		t.Errorf("Ran into an error while opening a connection to pipe: %v\n", e)
	}

	os.Stdout = w

	// Running the main method to generate output - which will then be fetched
	// and tested against.
	main()

	if res := w.Close(); res != nil {
		t.Errorf("Ran into an error while closing stream: %v\n", w)
	}

	out, e := ioutil.ReadAll(r)
	if e != nil {
		t.Errorf("Ran into an error while reading input: %v\n", e)
	}

	os.Stdout = rescueStdout

	if strings.Trim(string(out), "\n") != "Hello World" {
		t.Errorf("Expected %s, got %s", "Hello World", string(out))
	}
}

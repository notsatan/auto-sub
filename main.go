package main

import (
	"io"
	"os"

	"github.com/demon-rem/auto-sub/internals"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

// Name of the text file containing logs
var logFile = "logs.txt"

// Entry point when the script is run - sets up a logger, and hands over the flow
// of control to the central command.
func main() {
	// Logging will be enabled - by default with the log level at warn. If the
	// logging is explicitly enabled (using the flag) the level will be reduced.
	log.SetLevel(log.WarnLevel)

	// Modify the formatter, prettifies log output
	log.SetFormatter(&easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "[%lvl%]: %time% - %msg%\n",
	})

	// Logging will always be enabled, if the user does not enable logging, messages
	// above `Warn` level will be logged.

	// #nosec G302 - GoSec not working with the `-exclude` tag for some reason.
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		// If the file can't be opened, setting the output channel to be stderr
		log.SetOutput(os.Stderr)

		log.Warn("Error; failed to open a connection to the log file")
	} else {
		// Using MultiWriter to write to stderr and log file at the same time.
		log.SetOutput(io.MultiWriter(os.Stderr, file))

		// Closing the log file when the main function ends.
		defer func() {
			if err := file.Close(); err != nil {
				log.Warn("Failed to close connection to the log file")
			}
		}()
	}

	internals.Execute()
}

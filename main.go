package main

import (
	"os"

	"github.com/demon-rem/auto-sub/internals"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

// Name of the text file containing logs
const logFile = "logs.txt"

// Entry point when the script is run - sets up a logger, and hands over the flow
// of control to the central command.
func main() {
	// Logging will be enabled - by default with the log level at warn. If logging is
	// explicitly enabled (using a flag) log level will be reduced.
	log.SetLevel(log.WarnLevel)

	// Modify the formatter, prettifies log output
	log.SetFormatter(&easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "[%lvl%]: %time% - %msg%\n",
	})

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		// If the file can't be opened, set the output channel to be stderr
		log.SetOutput(os.Stderr)
		log.Warn("(main/main) failed to open a connection to the log file")
	} else {
		// Writing logs to the log file.
		log.SetOutput(file)

		// Close the log file when the function ends.
		defer func() {
			if err := file.Close(); err != nil {
				log.Warn("(main/main) failed to close connection to the log file")
			}
		}()
	}

	// Call the main internal method
	internals.Execute()
}

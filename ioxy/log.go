package main

import (
	"fmt"
	"io"
	"os"
	. "os"

	"github.com/serverhorror/rog-go/reverse"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func initializeLogging(logFile string) {
	formatter := new(prefixed.TextFormatter)
	formatter.DisableColors = true
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "2006-01-02 15:04:05.00"
	formatter.ForceFormatting = true
	log.SetFormatter(formatter)
	var file, err = OpenFile(logFile, O_RDWR|O_CREATE|O_APPEND, 0666)
	if err != nil {
		fmt.Println("Could Not Open Log File : " + err.Error())
	}
	file.WriteString("\n\n")
	multi := io.MultiWriter(file, oldStd, os.Stdout)
	log.SetOutput(multi)
}

func sessionLogs(logFile string) []string {
	var file, err = OpenFile(logFile, O_RDWR|O_CREATE|O_APPEND, 0666)
	var logs []string
	if err != nil {
		fmt.Println("Could Not Open Log File : " + err.Error())
	}
	f := reverse.NewScanner(file)
	var lastBackslash bool
	for f.Scan() {
		if f.Text() == "" && lastBackslash {
			break
		}
		if f.Text() == "" {
			lastBackslash = true
		} else {
			lastBackslash = false
			logs = append([]string{f.Text()}, logs...)
		}
	}
	return logs
}

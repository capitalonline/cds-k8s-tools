package utils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"strings"
	"time"
)

const (
	LogType       = "LOG_TYPE"
	LogfilePrefix = "/var/log/cds/"
	MBSize        = 1024 * 1024
)

// SetLogAttribute
// rotate log file by 2M bytes
// default print log to stdout and file both.
func SetLogAttribute(logName string) {
	logType := os.Getenv(LogType)
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		//PrettyPrint:     true,
	})
	outputMulti := false
	logType = strings.ToLower(logType)
	if logType == "stdout" {
		return
	} else if logType != "host" {
		outputMulti = true
	}

	// make file stream
	if err := os.MkdirAll(LogfilePrefix, os.FileMode(0755)); err != nil {
		log.Errorf("failed to create the log directory %s: %s", LogfilePrefix, err.Error())
	}
	logFile := fmt.Sprintf("%s%s.log", LogfilePrefix, logName)
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	// rotate the log file if too large
	if fi, err := f.Stat(); err == nil && fi.Size() > 2*MBSize {
		if err := f.Close(); err != nil {
			log.Errorf("failed to close the log file %s: %s", f.Name(), err.Error())
		}
		timeStr := time.Now().Format("-2006-01-02-15:04:05")
		timedLogfile := fmt.Sprintf("%s%s%s.log", LogfilePrefix, logName, timeStr)
		if err := os.Rename(logFile, timedLogfile); err != nil {
			log.Errorf("failed to rename file from %s to %s: %s", logFile, timedLogfile, err.Error())
		}
		f, err = os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
	}
	if outputMulti {
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
	} else {
		log.SetOutput(f)
	}
}

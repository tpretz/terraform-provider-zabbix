package provider

import (
	logger "log"
	"os"
)

// this changes and no longer works if accessed later
var stderr = os.Stderr

type Log struct{}

func (Log) Trace(msg string, args ...interface{}) {
	logger.Printf("[TRACE] "+msg, args...)
}
func (Log) Debug(msg string, args ...interface{}) {
	logger.Printf("[DEBUG] "+msg, args...)
}
func (Log) Info(msg string, args ...interface{}) {
	logger.Printf("[INFO] "+msg, args...)
}
func (Log) Warn(msg string, args ...interface{}) {
	logger.Printf("[WARN] "+msg, args...)
}
func (Log) Error(msg string, args ...interface{}) {
	logger.Printf("[ERROR] "+msg, args...)
}

var log = &Log{}

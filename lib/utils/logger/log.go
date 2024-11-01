package logger

import (
	"io"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	log  *logrus.Logger
	once sync.Once
)

func InitializeTestnetLogger() {
	once.Do(func() {
		log = logrus.New()
		// We do not want to log by default
		log.SetOutput(io.Discard)
		log.SetLevel(logrus.PanicLevel)
		// Check if DEBUG_I2P is set
		if logLevel := os.Getenv("DEBUG_TESTNET"); logLevel != "" {
			log.SetOutput(os.Stdout)
			switch strings.ToLower(logLevel) {
			case "debug":
				log.SetLevel(logrus.DebugLevel)
			case "warn":
				log.SetLevel(logrus.WarnLevel)
			case "error":
				log.SetLevel(logrus.ErrorLevel)
			default:
				log.SetLevel(logrus.DebugLevel)
			}
			log.WithField("level", log.GetLevel()).Debug("Logging enabled.")
		}
	})
}

// GetTestnetLogger returns the initialized logger
func GetTestnetLogger() *logrus.Logger {
	if log == nil {
		InitializeTestnetLogger()
	}
	return log
}

func init() {
	InitializeTestnetLogger()
}

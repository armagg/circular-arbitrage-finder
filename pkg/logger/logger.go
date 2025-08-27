package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

func Init(level string) error {
	Log.SetOutput(os.Stdout)
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if level == "" {
		level = "info"
	}

	logLevel, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		return err
	}
	Log.SetLevel(logLevel)
	return nil
}

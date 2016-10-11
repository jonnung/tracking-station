package tracking_station

import (
	"os"

	"github.com/Sirupsen/logrus"
)

var LogAccess *logrus.Logger
var LogError *logrus.Logger

func SetupLogger() {
	LogAccess = logrus.New()
	LogError = logrus.New()

	LogAccess.Formatter = &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     false,
		FullTimestamp:   true,
	}
	LogError.Formatter = &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     false,
		FullTimestamp:   true,
	}

	LogAccess.Level = logrus.InfoLevel
	LogError.Level = logrus.ErrorLevel

	LogAccess.Out, _ = os.OpenFile("log/tracking-station.access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
	LogError.Out, _ = os.OpenFile("log/tracking-station.error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
}

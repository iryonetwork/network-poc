package logger

import (
	"log"

	"github.com/iryonetwork/network-poc/config"
)

type Log struct {
	doDebug bool
}

func New(cfg *config.Config) *Log {
	return &Log{doDebug: cfg.Debug}
}

func (l *Log) Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

func (l *Log) Println(v ...interface{}) {
	log.Println(v...)
	log.Println(v...)
}

func (l *Log) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *Log) Debugln(v ...interface{}) {
	if l.doDebug {
		log.Println(v...)
	}
}

func (l *Log) Debugf(format string, v ...interface{}) {
	if l.doDebug {
		log.Printf(format, v...)
	}
}

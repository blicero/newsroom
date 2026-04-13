// /home/krylon/go/src/github.com/blicero/newsroom/common/common.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-13 12:38:58 krylon>

package common

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blicero/krylib"
	"github.com/blicero/newsroom/logdomain"
	"github.com/google/uuid"
	"github.com/hashicorp/logutils"
)

//go:generate ./build_time_stamp.pl

// AppName is the name under which the application identifies itself.
// Version is the version number.
// Debug, if true, causes the application to log additional messages and perform
// additional sanity checks.
// TimestampFormat is the default format for timestamp used throughout the
// application.
const (
	AppName                  = "Newsroom"
	Version                  = "0.1.0"
	Debug                    = true
	TimestampFormatMinute    = "2006-01-02 15:04"
	TimestampFormat          = "2006-01-02 15:04:05"
	TimestampFormatSubSecond = "2006-01-02 15:04:05.0000 MST"
	TimestampFormatDate      = "2006-01-02"
	TimestampFormatTime      = "15:04:05"
	NetName                  = "udp4"
	BufSize                  = 65536
	LiveTimeout              = time.Minute * 5
	ActiveTimeout            = time.Second * 5
	WebPort                  = 4200
)

// LogLevels are the names of the log levels supported by the logger.
var LogLevels = []logutils.LogLevel{
	"TRACE",
	"DEBUG",
	"INFO",
	"WARN",
	"ERROR",
	"CRITICAL",
	"CANTHAPPEN",
	"SILENT",
}

// SuffixPattern is a regular expression that matches the suffix of a file name.
// For "text.txt", it should match ".txt" and capture "txt".
var SuffixPattern = regexp.MustCompile("([.][^.]+)$")

// PackageLevels defines minimum log levels per package.
var PackageLevels = make(map[logdomain.ID]logutils.LogLevel, len(LogLevels))

func init() {
	for _, id := range logdomain.All() {
		PackageLevels[id] = MinLogLevel
	}
} // func init()

var tildeRe = regexp.MustCompile(`^~`)

// MinLogLevel is the minimum level a log message must
// have to be written out to the log.
// This value is configurable to reduce log verbosity
// in regular use.
var MinLogLevel logutils.LogLevel = "TRACE"

// DoTrace causes the log level to be lowered to TRACE when set.
var DoTrace = true

// BaseDir is the folder where all application-specific files are stored.
// It defaults to $HOME/.Kuang2.d
var BaseDir = filepath.Join(
	krylib.GetHomeDirectory(),
	fmt.Sprintf(".%s.d", strings.ToLower(AppName)))

// LogPath is the filename of the log file.
var LogPath = filepath.Join(BaseDir, fmt.Sprintf("%s.log", strings.ToLower(AppName)))

// DbPath is the filename of the database.
var DbPath = filepath.Join(BaseDir, fmt.Sprintf("%s.db", strings.ToLower(AppName)))

// CachePath is the directory where the various cache stores live.
var CachePath = filepath.Join(BaseDir, "cache.d")

// CfgPath is the path to the config file (should we ever adopt using one).
var CfgPath = filepath.Join(BaseDir, fmt.Sprintf("%s.toml", strings.ToLower(AppName)))

// BlacklistPath is the path to the Blacklist.
var BlacklistPath = filepath.Join(BaseDir, "blacklist.db")

// This needs a little refinement, but should clear up the race condition.
var (
	lock   sync.RWMutex
	isInit atomic.Bool
)

// InitApp performs some basic preparations for the application to run.
// Currently, this means creating the BaseDir folder.
func InitApp() (e error) {
	var err error

	if isInit.Load() {
		return nil
	}

	lock.Lock()
	defer func() {
		if e == nil {
			isInit.Store(true)
		}
		lock.Unlock()
	}()

	if err = os.Mkdir(BaseDir, 0700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("error creating BaseDir %s: %w", BaseDir, err)
	}

	LogPath = filepath.Join(BaseDir, fmt.Sprintf("%s.log", strings.ToLower(AppName)))
	DbPath = filepath.Join(BaseDir, fmt.Sprintf("%s.db", strings.ToLower(AppName)))
	CachePath = filepath.Join(BaseDir, "cache.d")
	CfgPath = filepath.Join(BaseDir, fmt.Sprintf("%s.toml", strings.ToLower(AppName)))
	BlacklistPath = filepath.Join(BaseDir, "blacklist.db")

	if err = os.Mkdir(CachePath, 0700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("error creating cache directory %s: %s",
			CachePath,
			err.Error())
	}

	return nil
} // func InitApp() error

// SetBaseDir sets the application's base directory. This should only be
// done during initialization.
// Once the log file and the database are opened, this
// is useless at best and opens a world of confusion at worst, so this function
// should only be called at the very beginning of the program.
func SetBaseDir(path string) error {
	if tildeRe.MatchString(path) {
		path = tildeRe.ReplaceAllString(path, krylib.GetHomeDirectory())
	}

	fmt.Printf("Setting BASE_DIR to %s\n", path)

	BaseDir = path
	LogPath = filepath.Join(BaseDir, fmt.Sprintf("%s.log", strings.ToLower(AppName)))
	DbPath = filepath.Join(BaseDir, fmt.Sprintf("%s.db", strings.ToLower(AppName)))
	CachePath = filepath.Join(BaseDir, "cache.d")
	CfgPath = filepath.Join(BaseDir, fmt.Sprintf("%s.toml", strings.ToLower(AppName)))
	BlacklistPath = filepath.Join(BaseDir, "blacklist.db")

	var (
		err error
		msg string
	)

	if err = InitApp(); err != nil {
		msg = fmt.Sprintf("Error initializing application environment: %s\n",
			err.Error())
		fmt.Println(msg)
		return errors.New(msg)
	}

	return nil
} // func SetBaseDir(path string)

// GetLogger tries to create a named logger instance and return it.
// If the directory to hold the log file does not exist, try to create it.
func GetLogger(domain logdomain.ID) (*log.Logger, error) { // nolint: interfacer
	var (
		err     error
		logfile *os.File
		logName = fmt.Sprintf("%s ",
			strings.ToLower(domain.String()))
	)

	if err = InitApp(); err != nil {
		return nil, fmt.Errorf("error initializing application environment: %s", err.Error())
	}

	if logfile, err = os.OpenFile(LogPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600); err != nil {
		msg := fmt.Sprintf("Error opening log file: %s\n", err.Error())
		fmt.Println(msg)
		return nil, errors.New(msg)
	}

	var (
		writer io.Writer
	)

	if Debug {
		writer = io.MultiWriter(os.Stdout, logfile)
	} else {
		writer = io.MultiWriter(logfile)
	}

	var lvl = PackageLevels[domain]

	filter := &logutils.LevelFilter{
		Levels:   LogLevels,
		MinLevel: lvl,
		Writer:   writer,
	}

	logger := log.New(filter, logName, log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
	return logger, nil
} // func GetLogger(name string) (*log.Logger, error)

// GetLoggerStdout returns a Logger that will log to stdout AND the log file.
func GetLoggerStdout(domain logdomain.ID) (*log.Logger, error) { // nolint: interfacer
	var err error

	if err = InitApp(); err != nil {
		return nil, fmt.Errorf("error initializing application environment: %s", err.Error())
	}

	var (
		logfile *os.File
		writer  io.Writer
		lvl     logutils.LogLevel
		logName = fmt.Sprintf("%s ",
			strings.ToLower(domain.String()))
	)

	if logfile, err = os.OpenFile(LogPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600); err != nil {
		msg := fmt.Sprintf("Error opening log file: %s\n", err.Error())
		fmt.Println(msg)
		return nil, errors.New(msg)
	}

	writer = io.MultiWriter(os.Stdout, logfile)

	lvl = PackageLevels[domain]

	filter := &logutils.LevelFilter{
		Levels:   LogLevels,
		MinLevel: lvl,
		Writer:   writer,
	}

	logger := log.New(filter, logName, log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
	return logger, nil
} // func GetLoggerStdout(name string) (*log.Logger, error)

// GetUUID returns a randomized UUID
func GetUUID() string {
	var (
		id  uuid.UUID
		err error
	)

	if id, err = uuid.NewRandom(); err != nil {
		panic(err)
	}

	return id.String()
} // func GetUUID() string

// TimeEqual returns true if the two timestamps are less than one second apart.
//
// I suppose the name is bad. Should be "TimeEqualish" or something.
func TimeEqual(t1, t2 time.Time) bool {
	var delta = t1.Sub(t2)

	if delta < 0 {
		delta = -delta
	}

	return delta < time.Second
} // func TimeEqual(t1, t2 time.Time) bool

// GetChecksum computes the SHA512 checksum of the given data.
func GetChecksum(data []byte) (string, error) {
	var err error
	var hash = sha512.New()

	if _, err = hash.Write(data); err != nil {
		fmt.Fprintf( // nolint: errcheck
			os.Stderr,
			"Error computing checksum: %s\n",
			err.Error(),
		)
		return "", err
	}

	var checkSumBinary = hash.Sum(nil)
	var checkSumText = fmt.Sprintf("%x", checkSumBinary)

	return checkSumText, nil
} // func getChecksum(data []byte) (string, error)

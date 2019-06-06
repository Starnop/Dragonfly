package app

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/dragonflyoss/Dragonfly/common/dflog"
	"github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/constant"
	"github.com/sirupsen/logrus"
)

func initDfdaemon(cfg config.Properties) {
	if cfg.Verbose {
		logrus.Infoln("use verbose logging")
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	// if Options.MaxProcs <= 0, programs run with GOMAXPROCS set to the number of cores available.
	if cfg.MaxProcs > 0 {
		runtime.GOMAXPROCS(cfg.MaxProcs)
	}

	if err := cfg.Validate(); err != nil {
		logrus.Error(err)
		os.Exit(err.Code())
	}

	initLogger()

	if err := os.MkdirAll(cfg.DFRepo, 0755); err != nil {
		logrus.Errorf("create local repo:%s err:%v", cfg.DFRepo, err)
		os.Exit(constant.CodeExitRepoCreateFail)
	}

	go cleanLocalRepo(cfg.DFRepo)

	dfgetVersion, err := exec.Command(cfg.DFPath, "version").CombinedOutput()
	if err != nil {
		logrus.Errorf("failed to get dfget version: %v", err)
	}
	logrus.Infof("use dfget %s from %s", bytes.TrimSpace(dfgetVersion), cfg.DFPath)
}

func initLogger() {
	current, err := user.Current()
	exitOnError(err, "get current user")
	logFilePath := filepath.Join(current.HomeDir, ".small-dragonfly/logs/dfdaemon.log")
	logrus.Infof("use log file %s", logFilePath)

	exitOnError(
		dflog.InitLog(false, logFilePath, fmt.Sprintf("%d", os.Getpid())),
		"init log file",
	)

	if logFile, ok := (logrus.StandardLogger().Out).(*os.File); ok {
		go func(logFile *os.File) {
			logrus.Infof("rotate %s every 60 seconds", logFilePath)
			ticker := time.NewTicker(60 * time.Second)
			for range ticker.C {
				if err := rotateLog(logFile); err != nil {
					logrus.Errorf("failed to rotate log %s: %v", logFile.Name(), err)
				}
			}
		}(logFile)
	}
}

// rotateLog truncates the logs file by a certain amount bytes.
func rotateLog(logFile *os.File) error {
	stat, err := logFile.Stat()
	if err != nil {
		return err
	}
	logSizeLimit := int64(20 * 1024 * 1024)
	// if it exceeds the 20MB limitation
	if stat.Size() > logSizeLimit {
		log.SetOutput(ioutil.Discard)
		// make sure set the output of log back to logFile when error be raised.
		defer log.SetOutput(logFile)
		logFile.Sync()
		truncateSize := logSizeLimit/2 - 1
		mem, err := syscall.Mmap(int(logFile.Fd()), 0, int(stat.Size()),
			syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		if err != nil {
			return err
		}
		copy(mem[0:], mem[truncateSize:])
		if err := syscall.Munmap(mem); err != nil {
			return err
		}
		if err := logFile.Truncate(stat.Size() - truncateSize); err != nil {
			return err
		}
		if _, err := logFile.Seek(truncateSize, 0); err != nil {
			return err
		}
	}
	return nil
}

// cleanLocalRepo checks the files at local periodically, and delete the file when
// it comes to a certain age(counted by the last access time).
// TODO: what happens if the disk usage comes to high level?
func cleanLocalRepo(dfpath string) {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("recover cleanLocalRepo from err:%v", err)
			go cleanLocalRepo(dfpath)
		}
	}()
	for {
		time.Sleep(time.Minute * 2)
		logrus.Info("scan repo and clean expired files")
		filepath.Walk(dfpath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logrus.Warnf("walk file:%s error:%v", path, err)
				return nil
			}
			if !info.Mode().IsRegular() {
				logrus.Infof("ignore %s: not a regular file", path)
				return nil
			}
			// get the last access time
			statT, ok := util.GetSys(info)
			if !ok {
				logrus.Warnf("ignore %s: failed to get last access time", path)
				return nil
			}
			// if the last access time is 1 hour ago
			if time.Since(util.Atime(statT)) > time.Hour {
				if err := os.Remove(path); err == nil {
					logrus.Infof("remove file:%s success", path)
				} else {
					logrus.Warnf("remove file:%s error:%v", path, err)
				}
			}
			return nil
		})
	}
}

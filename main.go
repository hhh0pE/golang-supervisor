package golang_supervisor

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var RunningProcess *os.Process

func OriginalExecutablePath() string {
	exeName, _ := os.Executable()
	if strings.HasSuffix(exeName, ".exe") {
		exeName = strings.Replace(exeName, ".supervisor.exe", ".exe", 1)
		exeName = strings.Replace(exeName, ".running.exe", ".exe", 1)
	} else {
		exeName = strings.TrimSuffix(exeName, ".supervisor")
		exeName = strings.TrimSuffix(exeName, ".running")
	}
	return exeName
}

var logFile *os.File

func writeToLog(args ...interface{}) {
	var params []interface{}
	params = append(params, time.Now().Format(time.Stamp))
	for _, a := range args {
		params = append(params, a)
	}
	fmt.Fprintln(logFile, params...)
}
func init() {
	if flag.Lookup("test.v") != nil { // if there is gotest run - skip supervisor
		return
	}
	var log_file_err error
	logFile, log_file_err = os.OpenFile("log_supervisor.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
	if log_file_err != nil {
		panic(log_file_err)
	}
	//log.SetOutput(logFile)

	var isSupervisor, isSupervised, withoutSupervisor bool
	for _, arg := range os.Args[1:] {
		if arg == "-without-supervisor" {
			withoutSupervisor = true
		}
		if arg == "-supervisor" {
			isSupervisor = true
			break
		}
		if arg == "-supervised" {
			isSupervised = true
			break
		}
	}

	if (withoutSupervisor) {
		return
	}
	writeToLog("golang-supervisor init", os.Args, os.Getpid())
	wd, _ := os.Getwd()

	if !isSupervisor && !isSupervised {
		newExePath := duplicateExecutable("supervisor")
		args := os.Args[1:]
		args = append(args, "-supervisor")

		os.Chdir(filepath.Dir(newExePath))
		exeName := getExecutableName(newExePath)
		if runtime.GOOS == "linux" {
			exeName = "./"+exeName
		}
		cmd := exec.Command(exeName, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if starting_err := cmd.Start(); starting_err != nil {
			panic(starting_err)
		}
		os.Exit(0)
	}

	if isSupervisor {
		var killSignalReceived = make(chan os.Signal)
		signal.Notify(killSignalReceived, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)
		go func() {
				<-killSignalReceived
				if RunningProcess != nil {
					RunningProcess.Kill()
					RunningProcess.Signal(syscall.SIGTERM)
				}
				os.Exit(0)
		}()
		args := os.Args[1:]
		for i, arg := range args {
			if arg == "-supervisor" {
				args = append(args[:i], args[i+1:]...)
				i--
			}
		}
		args = append(args, "-supervised")
		for {
			exePath := duplicateExecutable("running")
			os.Chdir(filepath.Dir(exePath))

			exeName := getExecutableName(exePath)
			if runtime.GOOS == "linux" {
				exeName = "./"+exeName
			}
			cmd := exec.Command(exeName, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err != nil {
				writeToLog("error when starting new process", err.Error())
				time.Sleep(time.Second)
				continue
			}

			os.Chdir(wd)

			RunningProcess = cmd.Process

			if cmd_err := cmd.Wait(); cmd_err != nil {
				writeToLog("process finished with different from zero code, restarting..")
				writeToLog("Finished with response: ", cmd_err.Error())
			} else {
				writeToLog("process finished with code 0, supervisor shutting down..")
				os.Exit(0)
			}
			RunningProcess = nil
		}
	}

}

func duplicateExecutable(suffix string) string {
	selfFile, opening_err := os.Open(OriginalExecutablePath())
	if opening_err != nil {
		panic(opening_err)
	}

	newLocation, creating_err := os.Create(addSuffix(selfFile.Name(), suffix))
	if creating_err != nil {
		panic(creating_err)
	}

	if runtime.GOOS != "windows" {
		if chmod_err := newLocation.Chmod(0754); chmod_err != nil {
			panic("chmod err: " + chmod_err.Error())
		}
	}

	_, copy_err := io.Copy(newLocation, selfFile)
	if copy_err != nil {
		panic(copy_err)
	}

	selfFile.Close()
	newLocation.Close()

	return newLocation.Name()
}

func addSuffix(name, suffix string) string {
	if strings.HasSuffix(name, ".exe") {
		return strings.TrimSuffix(name, ".exe") + "." + suffix + ".exe"
	}
	name = strings.TrimSuffix(name, ".")
	return name + "." + suffix
}

func getExecutableName(path string) string {
	parts := strings.Split(path, string(os.PathSeparator))
	var name = parts[len(parts)-1]
	return name
}
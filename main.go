package golang_supervisor

import (
	"os"
	"os/exec"
	"log"
	"io"
	"strings"
	"runtime"
	"time"
	"os/signal"
	"syscall"
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

func init() {
	logFile, log_file_err := os.OpenFile("log_supervisor.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
	if log_file_err != nil {
		panic(log_file_err)
	}
	log.SetOutput(logFile)

	var isSupervisor, isSupervised bool
	for _, arg := range os.Args[1:] {
		if arg == "-supervisor" {
			isSupervisor = true
			break
		}
		if arg == "-supervised" {
			isSupervised = true
			break
		}
	}

	log.Println("golang-supervisor init", os.Args, os.Getpid())

	if !isSupervisor && !isSupervised {
		newExeName := duplicateExecutable("supervisor")
		args := os.Args[1:]
		args = append(args, "-supervisor")

		cmd := exec.Command(newExeName, args...)
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
				RunningProcess.Kill()
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
			exeName := duplicateExecutable("running")
			cmd := exec.Command(exeName, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err != nil {
				log.Println("error when starting new process", err.Error())
				time.Sleep(time.Second)
				continue
			}

			RunningProcess = cmd.Process

			if cmd_err := cmd.Wait(); cmd_err != nil {
				log.Println("process finished with different from zero code, restarting..")
				log.Println("Finished with response: ", cmd_err.Error())
			} else {
				log.Println("process finished with code 0, supervisor shutting down..")
				os.Exit(0)
			}
			RunningProcess = nil
		}
	}

}

func duplicateExecutable(suffix string) string {
	selfFile, opening_err  := os.Open(OriginalExecutablePath())
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
		return strings.TrimSuffix(name, ".exe")+"."+suffix+".exe"
	}
	name = strings.TrimSuffix(name, ".")
	return name + "."+suffix
}
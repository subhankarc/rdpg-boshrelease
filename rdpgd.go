package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/starkandwayne/rdpgd/admin"
	"github.com/starkandwayne/rdpgd/cfsb"
	"github.com/starkandwayne/rdpgd/log"
	"github.com/starkandwayne/rdpgd/rdpg"
	"github.com/starkandwayne/rdpgd/tasks"
)

var (
	VERSION string
	pidFile string
	Role    string
)

func init() {
	pidFile = os.Getenv("RDPGD_PIDFILE")
}

func main() {
	go writePidFile()

	parseArgs()

	switch Role {
	case "manager":
		manager()
	case "service":
		service()
	default:
		if len(Role) > 0 {
			fmt.Fprintf(os.Stderr, `ERROR: Unknown Role: %s, valid Roles: manager / service`, Role)
			usage()
			os.Exit(1)
		} else {
			fmt.Fprintf(os.Stderr, `ERROR: Role must be specified on the command line.`)
			usage()
			os.Exit(1)
		}
	}
}

func parseArgs() {
	for index, arg := range os.Args {
		if index == 0 {
			continue
		}

		switch arg {
		case "manager":
			Role = "manager"
		case "service":
			Role = "service"
		case "version", "--version", "-version":
			fmt.Fprintf(os.Stdout, "%s\n", VERSION)
			os.Exit(0)
		case "help", "-h", "?", "--help":
			usage()
			os.Exit(0)
		default:
			usage()
			os.Exit(1)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stdout, `
rdpgd - Reliable Distributed PostgreSQL Daemon

Usage:

	rdpgd [Flag(s)] <Action>

Actions:

	manager   Run in Management Cluster mode.
	service   Run in Service Cluster mode.
	bootstrap Bootstrap RDPG schemas, filesystem etc...
	version   print rdpg version
	help      print this message

Flags:

	--version  print rdpgd version and exit
	--help     print this message and exit

	`)
}

func manager() (err error) {
	log.Info(`Starting with 'manager' role...`)
	go admin.API()
	err = bootstrap()
	if err != nil {
		log.Error(fmt.Sprintf(`main.manager() bootstrap() ! %s`, err))
		os.Exit(1)
	}
	go cfsb.API()
	go tasks.Scheduler(Role)
	go tasks.Work(Role)
	err = signalHandler()
	return
}

func service() (err error) {
	log.Info(`Starting with 'service' role...`)
	go admin.API()
	err = bootstrap()
	if err != nil {
		log.Error(fmt.Sprintf(`main.service() bootstrap() ! %s`, err))
		os.Exit(1)
	}
	go tasks.Scheduler(Role)
	go tasks.Work(Role)
	err = signalHandler()
	return
}

func bootstrap() (err error) {
	err = rdpg.Bootstrap(Role)
	if err != nil {
		log.Error(fmt.Sprintf(`Bootstrap(%s) failed`, Role))
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(syscall.SIGTERM)
	}
	return
}

func writePidFile() {
	if pidFile != "" {
		pid := os.Getpid()
		log.Trace(fmt.Sprintf(`main.writePidFile() Writing pid %d to %s`, pid, pidFile))
		err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
		if err != nil {
			log.Error(fmt.Sprintf(`main.writePidFile() Error while writing pid '%d' to '%s' :: %s`, pid, pidFile, err))
			os.Exit(1)
		}
	}
	return
}

func signalHandler() (err error) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	for sig := range ch {
		log.Info(fmt.Sprintf("main.signalHandler() Received signal %v, shutting down gracefully...", sig))
		if _, err := os.Stat(pidFile); err == nil {
			if err := os.Remove(pidFile); err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
		}
		os.Exit(0)
	}
	return
}

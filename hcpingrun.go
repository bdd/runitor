package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/bdd/hcpingrun/api"
)

var checkUUID string
var runEvery time.Duration
var tee, noStart, noOutput bool

func initFlags() {
	flag.StringVar(&checkUUID, "check", "", "UUID of check")
	flag.DurationVar(&runEvery, "every", 0, "Run the command periodically every N time unit")
	flag.BoolVar(&tee, "tee", false, "Passthrough stdout and stderr")
	flag.BoolVar(&noStart, "nostart", false, "Do not send start ping")
	flag.BoolVar(&noOutput, "nooutput", false, "Do not attach stdout and stderr to pings")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [flags] [-check UUID] [-every duration] -- command [arg]...\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
}

func main() {
	initFlags()

	if len(checkUUID) == 0 {
		v, ok := os.LookupEnv("HCIO_CHECK")
		if !ok {
			log.Fatal("Must pass check UUID either with '-check UUID' param or HCIO_CHECK environment variable")
		}
		checkUUID = v
	}

	if flag.NArg() < 1 {
		log.Fatal("missing command")
	}

	chk := api.NewCheck(checkUUID)
	task := func() (int, error) { return runOnce(chk, flag.Args()) }

	if runEvery > 0 {
		sigC := make(chan os.Signal, 1)
		signal.Notify(sigC, syscall.SIGALRM)
		ticker := time.NewTicker(runEvery)
		for {
			select {
			case <-ticker.C:
				task()
			case <-sigC:
				task()
			}
		}
	} else {
		exitCode, err := task()
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(exitCode)
	}
}

func runOnce(p api.Pinger, cmd []string) (exitCode int, err error) {
	body := io.ReadWriter(new(bytes.Buffer))

	if !noStart {
		p.PingStart(body)
	}

	c := exec.Command(cmd[0], cmd[1:]...)
	if noOutput {
		goto Run
	}

	if tee {
		c.Stdout, c.Stderr = io.MultiWriter(body, os.Stdout), io.MultiWriter(body, os.Stderr)
	} else {
		c.Stdout, c.Stderr = body, body
	}

Run:
	err = c.Run()
	if err != nil {
		v, ok := err.(*exec.ExitError)
		if !ok {
			log.Fatal(err)
		}
		exitCode = v.ProcessState.ExitCode()
		fmt.Fprintf(body, "Command exited with code %d\n", exitCode)
	}

	if exitCode == 0 {
		p.PingSuccess(body)
	} else {
		p.PingFailure(body)
	}

	return
}

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/bdd/hcpingrun/pkg/api"
	"github.com/bdd/hcpingrun/pkg/api/healthchecks"
)

type RunConfig struct {
	Quiet          bool // Don't tee command stdout to stdout
	QuietErrors    bool // Don't tee command stderr to stderr
	NoStartPing    bool // Don't send Start ping.
	NoOutputInPing bool // Don't send command std{out, err} with Success and Failure pings.
}

func main() {
	var (
		check          = flag.String("check", "", "UUID of check")
		every          = flag.Duration("every", 0, "Run the command periodically at specified interval")
		quiet          = flag.Bool("quiet", false, "Don't relay stdout of command to terminal")
		silent         = flag.Bool("silent", false, "Don't relay  stdout and stderr of command to terminal")
		noStartPing    = flag.Bool("no-start-ping", false, "Don't send start ping")
		noOutputInPing = flag.Bool("no-output-in-ping", false, "Don't send stdout and stderr with pings")
	)
	flag.Parse()

	if len(*check) == 0 {
		v, ok := os.LookupEnv("HCIO_CHECK")
		if !ok {
			log.Fatal("Must pass check UUID either with '-check UUID' param or HCIO_CHECK environment variable")
		}
		check = &v
	}

	if flag.NArg() < 1 {
		log.Fatal("missing command")
	}

	command := flag.Args()
	pinger := healthchecks.NewPinger(*check)

	task := func() (int, error) {
		return Run(command, pinger, RunConfig{
			Quiet:          *quiet || *silent,
			QuietErrors:    *silent,
			NoStartPing:    *noStartPing,
			NoOutputInPing: *noOutputInPing,
		})
	}

	if *every == 0 {
		exitCode, _ := task()
		os.Exit(exitCode)
	}

	task()

	runNow := make(chan os.Signal, 1)
	signal.Notify(runNow, syscall.SIGALRM)
	ticker := time.NewTicker(*every)
	for {
		select {
		case <-ticker.C:
			task()
		case <-runNow:
			task()
		}
	}
}

func Run(cmd []string, p api.Pinger, cfg RunConfig) (exitCode int, err error) {
	body := io.ReadWriter(new(bytes.Buffer))

	var stdoutWriter, stderrWriter, bodyWriter io.Writer = os.Stdout, os.Stderr, body
	if cfg.Quiet {
		stdoutWriter = ioutil.Discard
	}
	if cfg.QuietErrors {
		stderrWriter = ioutil.Discard
	}
	if cfg.NoOutputInPing {
		bodyWriter = ioutil.Discard
	}

	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout = io.MultiWriter(stdoutWriter, bodyWriter)
	c.Stderr = io.MultiWriter(stderrWriter, bodyWriter)

	if !cfg.NoStartPing {
		p.PingStart(body)
	}
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

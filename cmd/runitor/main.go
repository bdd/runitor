package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"bdd.fi/x/runitor/pkg/api"
	"bdd.fi/x/runitor/pkg/api/healthchecks"
)

// RunConfig sets the behavior of a Run
type RunConfig struct {
	Quiet          bool // Don't tee command stdout to stdout
	QuietErrors    bool // Don't tee command stderr to stderr
	NoStartPing    bool // Don't send Start ping.
	NoOutputInPing bool // Don't send command std{out, err} with Success and Failure pings.
}

func main() {
	var (
		apiURL         = flag.String("api-url", healthchecks.DefaultBaseURL, "API URL Base of Healthchecks instance")
		apiTries       = flag.Int("api-tries", healthchecks.DefaultMaxTries, "Number of times an API request will be tried for transient errors.")
		apiTimeout     = flag.Duration("api-timeout", healthchecks.DefaultTimeout, "Client timeout for a single API request")
		uuid           = flag.String("uuid", "", "UUID of check")
		every          = flag.Duration("every", 0, "Run the command periodically at specified interval")
		quiet          = flag.Bool("quiet", false, "Don't relay stdout of command to terminal")
		silent         = flag.Bool("silent", false, "Don't relay  stdout and stderr of command to terminal")
		noStartPing    = flag.Bool("no-start-ping", false, "Don't send start ping")
		noOutputInPing = flag.Bool("no-output-in-ping", false, "Don't send stdout and stderr with pings")
	)
	flag.Parse()

	if len(*uuid) == 0 {
		v, ok := os.LookupEnv("CHECK_UUID")
		if !ok || len(v) == 0 {
			log.Fatal("Must pass check UUID either with '-uuid UUID' param or CHECK_UUID environment variable")
		}
		uuid = &v
	}

	if flag.NArg() < 1 {
		log.Fatal("missing command")
	}

	command := flag.Args()
	pinger := &healthchecks.APIClient{
		BaseURL:  *apiURL,
		MaxTries: int(math.Max(1, float64(*apiTries))), // has to be >=1
		Client:   &http.Client{Timeout: *apiTimeout},
	}

	runConfig := RunConfig{
		Quiet:          *quiet || *silent,
		QuietErrors:    *silent,
		NoStartPing:    *noStartPing,
		NoOutputInPing: *noOutputInPing,
	}

	task := func() (int, error) {
		return Run(command, *uuid, pinger, runConfig)
	}

	if *every == 0 {
		exitCode, err := task()
		if err == nil {
			os.Exit(exitCode)
		}

		v, ok := err.(*api.PingError)
		if ok {
			log.Fatal("Ping Error: ", v)
		}
		log.Fatal("Command execution error: ", err)
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
			ticker.Stop()
			ticker = time.NewTicker(*every)
			task()
		}
	}
}

// Run function executes cmd[0] with parameters cmd[1:].
// Pinger is used to signal start, success, or failure of execution.
func Run(cmd []string, uuid string, p api.Pinger, cfg RunConfig) (exitCode int, err error) {
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
		if err := p.PingStart(uuid, body); err != nil {
			log.Print("Error trying to ping (start): ", err)
		}
	}
	err = c.Run()
	if err != nil {
		v, ok := err.(*exec.ExitError)
		if !ok {
			return
		}
		exitCode = v.ProcessState.ExitCode()
		fmt.Fprintf(body, "Command exited with code %d\n", exitCode)
	}

	if exitCode == 0 {
		err = p.PingSuccess(uuid, body)
	} else {
		err = p.PingFailure(uuid, body)
	}
	if err != nil {
		log.Print("Error trying to ping (success/failure): ", err)
	}

	return
}

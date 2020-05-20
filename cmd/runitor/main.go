package main // import "bdd.fi/x/runitor/cmd/runitor"

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"bdd.fi/x/runitor/internal"
)

// RunConfig sets the behavior of a run.
type RunConfig struct {
	Quiet          bool // Don't tee command stdout to stdout
	QuietErrors    bool // Don't tee command stderr to stderr
	NoStartPing    bool // Don't send Start ping.
	NoOutputInPing bool // Don't send command std{out, err} with Success and Failure pings.
}

// Globals used for building help and identification strings.

// Name is the name of this command.
const Name string = "runitor"

// Homepage is the URL to the canonical website describing this command.
const Homepage string = "https://bdd.fi/x/runitor"

// Version is the version string that gets overridden at link time for releases.
var Version string = "HEAD"

func main() {
	var (
		apiURL         = flag.String("api-url", internal.DefaultBaseURL, "API base URL. Defaults to healthchecks.io hosted service one.")
		apiTries       = flag.Int("api-tries", internal.DefaultMaxTries, "Number of times an API request will be attempted")
		apiTimeout     = flag.Duration("api-timeout", internal.DefaultTimeout, "Client timeout per request")
		uuid           = flag.String("uuid", "", "UUID of check. Takes precedence over CHECK_UUID env var")
		every          = flag.Duration("every", 0, "When non-zero periodically run command at specified interval")
		quiet          = flag.Bool("quiet", false, "Don't tee stdout of the command to terminal")
		silent         = flag.Bool("silent", false, "Don't tee stout and stderr of the command to terminal")
		noStartPing    = flag.Bool("no-start-ping", false, "Don't send start ping")
		noOutputInPing = flag.Bool("no-output-in-ping", false, "Don't send stdout and stderr with pings")
		version        = flag.Bool("version", false, "Show version")
	)

	flag.Parse()

	if *version {
		fmt.Println(Name, Version)
		os.Exit(0)
	}

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

	cmd := flag.Args()
	client := &internal.APIClient{
		BaseURL:   *apiURL,
		MaxTries:  int(math.Max(1, float64(*apiTries))), // has to be >=1
		Client:    &http.Client{Timeout: *apiTimeout},
		UserAgent: fmt.Sprintf("%s/%s (+%s)", Name, Version, Homepage),
	}

	cfg := RunConfig{
		Quiet:          *quiet || *silent,
		QuietErrors:    *silent,
		NoStartPing:    *noStartPing,
		NoOutputInPing: *noOutputInPing,
	}

	// Save this invocation so we don't repeat ourselves.
	task := func() int {
		return Do(cmd, cfg, *uuid, client)
	}

	exitCode := task()

	// One-shot mode. Exit with command's exit code.
	if *every == 0 {
		os.Exit(exitCode)
	}

	// Task scheduler mode. Run the command periodically at specified interval.
	ticker := time.NewTicker(*every)
	runNow := make(chan os.Signal, 1)
	signal.Notify(runNow, syscall.SIGALRM)

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

// Do function runs the cmd line, tees its output to terminal & ping body as configured in cfg
// and pings the monitoring API to signal start, and then success or failure of execution.
func Do(cmd []string, cfg RunConfig, uuid string, p internal.Pinger) (exitCode int) {
	var (
		stdoutReceivers, stderrReceivers []io.Writer
		pingBody                         io.ReadWriter = new(bytes.Buffer)
	)

	if !cfg.NoStartPing {
		if err := p.PingStart(uuid, pingBody); err != nil {
			log.Print("PingStart: ", err)
		}
	}

	if !cfg.NoOutputInPing {
		stdoutReceivers = append(stdoutReceivers, pingBody)
		stderrReceivers = append(stderrReceivers, pingBody)
	}

	if !cfg.Quiet {
		stdoutReceivers = append(stdoutReceivers, os.Stdout)
	}

	if !cfg.QuietErrors {
		stderrReceivers = append(stderrReceivers, os.Stderr)
	}

	stdout := io.MultiWriter(stdoutReceivers...)
	stderr := io.MultiWriter(stderrReceivers...)

	exitCode, err := Run(cmd, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stdout, "Command execution failed: %v", err)
		// Use POSIX EXIT_FAILURE (1) for cases where the specified
		// command fails to execute.  Execution will continue and a
		// failure ping will be sent due to non-zero exit code.
		exitCode = 1
	}

	if exitCode != 0 {
		fmt.Fprintf(pingBody, "\nCommand exited with code %d\n", exitCode)

		if err := p.PingFailure(uuid, pingBody); err != nil {
			log.Print("PingFailure: ", err)
		}

		return exitCode
	}

	if err := p.PingSuccess(uuid, pingBody); err != nil {
		log.Print("PingSuccess: ", err)
	}

	return exitCode
}

// Run function executes cmd[0] with parameters cmd[1:] and redirects its stdout & stderr to passed
// writers of corresponding parameter names.
func Run(cmd []string, stdout, stderr io.Writer) (exitCode int, err error) {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout, c.Stderr = stdout, stderr

	err = c.Run()
	if err != nil {
		// Convert *exec.ExitError to just exit code and no error.
		// From our point of view, it's not really an error but a value.
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ProcessState.ExitCode(), nil
		}
	}

	// Here we either have:
	// a) we couldn't execute the command and we have a real error in our hands.
	//    exitCode's zero value is '0' but it doesn't matter as we'll return non-nil err.
	// b) the command ran successfully and exit with code 0.
	//    exitCode hasn't been mutated, so its zero value of '0' is what we would like to return
	//    anyway.
	return
}

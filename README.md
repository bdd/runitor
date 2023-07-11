# runitor

Runitor runs the supplied command, captures its output, and based on its exit
code reports successful or failed execution to https://healthchecks.io or your
private instance.

Healthchecks.io is a web service for monitoring periodic tasks. It's like a
dead man's switch for your cron jobs. You get alerted if they don't run on time
or terminate with a failure.

## Installation

### Download Signed Release Binaries

Binaries of the latest release for popular platforms are at
https://github.com/bdd/runitor/releases/latest

#### Verify Signatures

SHA256 checksum manifest of the releases are signed with one of the SSH
keys published at https://bdd.fi/x/runitor.pub.

An [example verification script](scripts/verify) shows how to use `ssh-keygen`
and `sha256sum` to verify downloads.

### Build Locally

If you have Go 1.18 or newer installed, you can use the command:

	go install bdd.fi/x/runitor/cmd/runitor@latest

...and the binary will be at `$GOPATH/bin/runitor` or if `GOPATH` isn't set,
under `$HOME/go/bin/runitor`.

#### Cross Compilation

If you need to build the binary on a platform different than the target, you
can pass the target operating system and architecture with GOOS and GOARCH
environment variables to the `go install` command.

	GOOS=plan9 GOARCH=arm go install bdd.fi/x/runitor/cmd/runitor@latest

...and the binary will be under `$GOPATH/bin/plan9_arm/runitor`.


### Container Image

If you prefer to run your workloads in containers,
[runitor/runitor](https://hub.docker.com/r/runitor/runitor) on Docker Hub
provides images based on Alpine, Debian, and Ubuntu for x86_64, arm64v8, and
armv7 architectures.


## Why Do I Need This Instead of Calling curl from a Shell Script?

In addition to clean separation of concerns from the thing that needs to run and
the act of calling an external monitor, runitor packs a few neat extra features
that are bit more involved than single line additions to a script.

It can capture the stdout and stderr of the command to send it along with
execution reports, a.k.a. "pings". When you respond to an alert you can quickly
start investigating the issue with the relevant context already available.

It can be used as a long running process acting as a task scheduler, executing
the command at specified intervals. This feature comes in handy when you don't
readily have access to a job scheduler like crond or systemd.timer. Works well
in one process per container environments.


## Example Uses

### Certificate Renewal

	# (Using per check UUIDs.)
	export CHECK_UUID=8116e449-d71c-4112-8f5d-a66f60902091
	runitor -- dehydrated --cron --config example.com.conf

### Repository Maintenance

	# Run the maintenance script 10 times a day (24h/10 = 2h 24m)
	# (Using per-project ping key, and check slugs.)
	export HC_PING_KEY=edf72661-62ff-49e7-a921-33b41502d7e7
	runitor -slug git-repo-maintenance \
		-every 2h24m -- \
		/script/git-maint

### Backup

	# Do not attach output to ping.
	# Backup software may leak filenames and paths.

	runitor -no-output-in-ping -- restic backup /home /etc

### Triggering an Immediate Run in Periodic Mode

When invoked with `-every <duration>` flag, runitor will also act as a basic
task scheduler.

Sometimes you may not want to restart the process or the container just to force
an immediate run. Instead, you can send SIGALRM to runitor to get it run the
command right away and reset the interval.

	pkill -ALRM runitor


## Usage

	runitor [-uuid uuid] -- command

### Flags

	-api-retries uint
	  Number of times an API request will be retried if it fails with a transient error (default 2)
	-api-timeout duration
	  Client timeout per request (default 5s)
  	 -api-url string
	  API URL. Takes precedence over HC_API_URL environment variable (default "https://hc-ping.com"). Must not contain a trailing slash. 
   	  When using with a self-hosted Healthchecks instance, where ping endpoints are hosted under the "/ping" path (this will show in the ping URLs in the web UI), the API URL must include this path. 
          Example for a Healthchecks instance running on https://example.org: -api-url=https://example.org/ping
	-every duration
	  If non-zero, periodically run command at specified interval
	-no-output-in-ping
	  Don't send command's output in pings
	-no-run-id
	  Don't generate and send a run id per run in pings
	-no-start-ping
	  Don't send start ping
	-on-exec-fail value
	  Ping type to send when runitor cannot execute the command (exit-code|success|fail|log (default fail))
	-on-nonzero-exit value
	  Ping type to send when command exits with a nonzero code (exit-code|success|fail|log (default exit-code))
	-on-success value
	  Ping type to send when command exits successfully (exit-code|success|fail|log (default success))
	-ping-body-limit uint
	  If non-zero, truncate the ping body to its last N bytes, including a truncation notice. (default 10000)
	-ping-key string
	  Ping Key. Takes precedence over HC_PING_KEY environment variable
	-quiet
	  Don't capture command's stdout
	-req-header value
	  Additional request header as "key: value" string
	-silent
	  Don't capture command's stdout or stderr
	-slug string
	  Slug of check. Requires a ping key. Takes precedence over CHECK_SLUG environment variable
	-uuid string
	  UUID of check. Takes precedence over CHECK_UUID environment variable
	-version
	  Show version


## More on What healthchecks.io Provides

* It listens for HTTP requests (pings) from services being monitored.

* It keeps silent as long as pings arrive on time. It raises an alert as soon
  as a ping does not arrive on time.

* Pings can signal start of execution so run durations can be tracked.

* Pings can explicitly signal failure of execution so an alert can be send.

* Pings can attach up to 100KB of logs. Runitor automatically handles truncation if needed.

* It can alert you via email, SMS, WhatsApp, Slack, and many more services.

* It has a free tier with up to 20 checks.

* Software behind the service is open source. You can run your own instance if
  you'd like to.

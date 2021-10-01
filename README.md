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

### Clone and Build Locally

If you need to cross compile for a certain operating system and architecture
pair, you can clone the repository and use the build script.

	git clone https://github.com/bdd/runitor
	GOOS=plan9 GOARCH=arm runitor/scripts/build dist


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

	export CHECK_UUID=8116e449-d71c-4112-8f5d-a66f60902091
	runitor -- dehydrated --cron --config example.com.conf

### Repository Maintenance

	# Run the maintenance script 10 times a day (24h/10 = 2h 24m)

	runitor -uuid edf72661-62ff-49e7-a921-33b41502d7e7 \
		-every 2h24m -- /script/git-maint

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

	-uuid=""
		UUID of check. Takes precedence over CHECK_UUID environment variable.
	-every="0s"
		If non-zero, periodically run command at specified interval.
	-quiet=false
		Don't capture command's stdout.
	-silent=false
		Don't capture command's stdout or stderr.
	-no-start-ping=false
		Don't send start ping.
	-no-output-in-ping=false
		Don't send command's output in pings.
	-ping-body-limit=10000
		If non-zero, truncate the ping body to its last N bytes,
		including a truncation notice.
		Default value of 10KB is equal to healthchecks.io instance's
		ping body limit.
	-api-url="https://hc-ping.com"
		API URL. Takes precedence over HC_API_URL environment variable. Defaults to healthchecks.io hosted service.
	-api-retries=2
		Number of times an API request will be retried if it fails with
		a transient error.
	-api-timeout="5s"
		Client timeout per request.


## More on What healthchecks.io Provides

* It listens for HTTP requests (pings) from services being monitored.

* It keeps silent as long as pings arrive on time. It raises an alert as soon
  as a ping does not arrive on time.

* Pings can signal start of execution so run durations can be tracked.

* Pings can explicitly signal failure of execution so an alert can be send.

* Pings can attach up to 10KB of logs. Runitor automatically handles truncation if needed.

* It can alert you via email, SMS, WhatsApp, Slack, and many more services.

* It has a free tier with up to 20 checks.

* Software behind the service is open source. You can run your own instance if
  you'd like to.

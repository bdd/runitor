# runitor

`runitor` runs the supplied command, captures its output, and based on its exit
code reports successful or failed execution to https://healthchecks.io.

Healthchecks.io is a web service for monitoring periodic tasks. It's like a
dead man's switch for your cron jobs. You get alerted if they don't run on time
or terminate with a failure.

## Install:

	go get bdd.fi/x/runitor/cmd/runitor

## Usage:

	runitor -uuid uuid -- command

### The flags are:

	-api-url="https://hc-ping.com"
		API base URL. Defaults to healthchecks.io hosted service one
	-api-tries=3
		Number of times an API request will be attempted
	-api-timeout="5s"
		Client timeout per request
	-uuid=""
		UUID of check. Takes precedence over CHECK_UUID env var
	-every="0s"
		When non-zero periodically run command at specified interval
	-quiet=false
		Don't tee stdout of the command to terminal
	-silent=false
		Don't tee stout and stderr of the command to terminal
	-no-start-ping=false
		Don't send start ping
	-no-output-in-ping=false
		Don't send stdout and stderr with pings


## "Why do I need this? I can just run curl from my shell script."

You sure can but `runitor` offers a clean separation of concerns from the thing
that needs to run and signalling its execution to an external monitoring
service. You would want to get alerted if your script abrubtly exits without a
chance to signal failure.

It also offers a few neat extra features.

* It can capture the stdout and stderr of the command to send it along with
  execution reports, a.k.a. "pings". When you respond to an alert
  you can quickly start investigating the issue with the relevant context
  already available.

* It can be used as a long running process executing the command at specified
  intervals. This feature comes in handy when you don't readily have access
  to a job scheduler like crond or systemd.timer. Works well in one process
  per container environments.


## Example Use:

### Certificate Renewal

	runitor -uuid <check-uuid> -- dehydrated --cron --config example.com.conf


### Mirror Git Repositories

	# Run the mirror script every 30 minutes 5 seconds.
	runitor -uuid <check-uuid> -every 30m5s -- /script/git-mirror


### Periodic Backup

	# Do not attach output to ping.
	# Backup software may leak filenames and paths.

	runitor -uuid <check-uuid> -no-output-in-ping -- restic backup /home /etc


### Triggering an immediate run in periodic mode

When invoked with `-every <duration>` argument, `runitor` will act as a
a lo-fi process manager and scheduler. Sometimes you may not want to restart the
pprocess or the container to force an immediate run. Instead you can send
SIGALRM to `runitor`. It will run the command right away and reset the periodic
timer.

	pkill -ALRM runitor


## "Remind me again, what is this healthchecks.io service?"

* healthchecks.io listens for HTTP requests--"pings" from services being
  monitored.
* It keeps silent as long as pings arrive on time. It raises an alert as soon
  as a ping does not arrive on time.
* Pings can signal start of execution so run durations can be tracked.
* Pings can explicitly signal failure of execution so an alert can be send.
* Pings can attach up to 10KB of logs to
* It can alert you via email, SMS, WhatsApp, Slack, and many more services.
* It has a free tier with up to 20 checks (last checked April 2020)
* Software behind the service is open source. You can run your own instance if
  you'd like to.



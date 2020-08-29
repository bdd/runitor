/*

Runitor runs the supplied command, captures its output, and based on its exit
code reports successful or failed execution to https://healthchecks.io.

Healthchecks.io is a web service for monitoring periodic tasks. It's like a
dead man's switch for your cron jobs. You get alerted if they don't run on time
or terminate with a failure.

Install:

	go get bdd.fi/x/runitor/cmd/runitor

Usage:

	runitor -uuid uuid -- command

Flags:

	-api-url="https://hc-ping.com"
		API base URL. Defaults to healthchecks.io hosted service one
	-api-retries=2
		Number of times an API request will be retried if it fails with a transient error
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


Why Do I Need This Instead of Calling curl from a Shell Script

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


Example Use

Certificate Renewal:

	runitor -uuid 2f9-a5c-0123 -- dehydrated --cron --config example.com.conf

Repository Maintenance:

	# Run the maintenance script every 100 minutes (1 hour 40 minutes)

	runitor -uuid 2f9-a5c-0123 -every 1h40m -- /script/git-maint

Backup:

	# Do not attach output to ping.
	# Backup software may leak filenames and paths.

	runitor -uuid 2f9-a5c-0123 -no-output-in-ping -- restic backup /home /etc


Triggering an Immediate Run in Periodic Mode

When invoked with `-every <duration>` flag, runitor will also act as a basic
task scheduler.

Sometimes you may not want to restart the process or the container just to force
an immediate run. Instead, you can send SIGALRM to runitor to get it run the
command right away and reset the interval.

	pkill -ALRM runitor


More on What healthchecks.io Provides

It listens for HTTP requests (pings) from services being monitored.

It keeps silent as long as pings arrive on time. It raises an alert as soon
as a ping does not arrive on time.

Pings can signal start of execution so run durations can be tracked.

Pings can explicitly signal failure of execution so an alert can be send.

Pings can attach up to 10KB of logs.

It can alert you via email, SMS, WhatsApp, Slack, and many more services.

It has a free tier with up to 20 checks (last checked April 2020)

Software behind the service is open source. You can run your own instance if
you'd like to.

*/
package main // import "bdd.fi/x/runitor/cmd/runitor"

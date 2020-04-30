# runitor

`runitor` runs the supplied command, captures its output, and based on its exit
code reports successful of failed execution to
[healthchecks.io](https://healthchecks.io).

Healthchecks.io is a web service for monitoring periodic tasks. It's like a
[dead man's switch](https://en.wikipedia.org/wiki/Dead_man%27s_switch) for your
cron jobs. You get alerted if they don't run on time or terminate with a
failure.

## Why do I need this? I can just run curl from my shell script.

You sure can but `runitor` offers a clean separation of concerns from the thing
that needs to run and signalling its execution to an external monitoring
service. You would want to get alerted if your script abrubtly exits without a
chance to signal failure.

This little utility also offers a few extra neat features.

  * It can capture the stdout and stderr of the command to send it along with
    execution reports, _a.k.a._ "pings". When you respond to an alert
    you can quickly start investigating the issue with the relevant context
    already available.

  * It can be used as a long running process executing the command at specified
    intervals. This feature comes in handy when you don't readily have access
    to a job scheduler like crond or systemd.timer. Works well in one process
    per container environments.

## Install
```
% go get bdd.fi/x/runitor/cmd/runitor
```

## Example Use

### Certificate Renewal

```
% runitor -uuid <check-uuid> -- dehydrated --cron --config example.com.conf
```

### Mirror Git Repositories

```
# Run the mirror script every 30 minutes 5 seconds.

% runitor -uuid <check-uuid> -every 30m5s -- /script/git-mirror
```

### Periodic Backup

```
# Do not attach output to ping. Backup software may leak filenames and paths.

% runitor -uuid <check-uuid> -no-output-in-ping -- restic backup /home /etc
```

### Triggering an immediate run in periodic mode

When invoked with `-every <duration>` argument, `runitor` will act as a
a lo-fi process manager and scheduler. Sometimes you may not want to restart 
the pprocess or the container to force an immediate run. Instead you can send
SIGALRM to `runitor`. It will run the command right away and reset the periodic
timer.

```
% pkill -ALRM runitor
```


## Remind me again, what is this healthchecks.io service?

  * healthchecks.io listens for HTTP requests ("pings") from services being
    monitored.
  * It keeps silent as long as pings arrive on time.
  * It raises an alert as soon as a ping does not arrive on time.
  * Checks can signal start of execution so run durations can be tracked.
  * Checks can explicitly signal failure of execution so an alert can be send.
  * Checks can attach up to 10KB of logs to pings.
  * It can alert you via email, SMS, WhatsApp, Slack, and many more services.
  * It has a free tier with up to 20 checks (last checked April 2020)
  * Software behind the service is open source. You can run your own instance if
    you'd like to.

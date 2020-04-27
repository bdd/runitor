# hcpingrun

`hcpingrun` runs your commands and notifies
[healthchecks.io](https://healthchecks.io) if they successfully executed or not.

Healthchecks.io is a service for monitoring similar periodic tasks. It's like a
dead man's switch for your cron jobs so you can get alerted if they don't run
when they were expected to or fail.

## Example usage:

### Certificate renewal

```
% hcpingrun -uuid <check-uuid> -- dehydrated --cron --config example.com.conf
```

### Mirror Git Repositories

```
# Run the mirror script every 30 minutes 5 seconds.

% hcpingrun -uuid <check-uuid> -every 30m5s -- /script/git-mirror
```

### Periodic Backup

```
# Do not attach output to ping. Backup software may leak filenames and paths.

% hcpingrun -uuid <check-uuid> -no-output-in-ping -- restic backup /home /etc
```

### Don't wait until the end of period, run now
When invoked with `-every <duration>` argument, `hcpingrun` will act as a
process manager and lo-fi scheduler. Sometimes you don't want to kill the
process or container to force an immediate run. For thatm you can send SIGALRM
to `hcpingrun` to trigger an immediate run of the command.

```
% pkill -ALRM hcpingrun
```

## Why do I need this? I can just run curl from my shell script.

You sure can but hcpingrun offers a clean separation of concerns from the thing
that needs to be executed and signalling its execution to an external monitoring
service. You probably want to get alerted if your script abrubtly exits without
a chance to signal failure.

This little utility also offers a few other neat features like:

  * Ability to capture the stdout and stderr of the command and send it
    along with the ping. When you respond to an alert, you can start
    invetigating the issue with relevant context right in front of you.

  * Act as a long running process, executing the  command at specified intervals.
    This feature comes in handy when you don't readily have access to a job
    scheduler like cron or systemd timers.  Works well in one process per
    container environments.


## Remind me again, what is this healthchecks.io service?

  * healthchecks.io listens for HTTP requests
    ("pings") from services being monitored.
  * It keeps silent as long as pings arrive on time.
  * It raises an alert as soon as a ping does not arrive on time.
  * Checks can signal start of execution so run durations can be tracked.
  * Checks can explicitly signal failure of execution so an alert can be raised.
  * Checks can attach up to 10KB of logs to success/failure pings.
  * It can alert you via email, SMS, WhatsApp, Slack, and many more services.
  * It has a free tier with up to 20 checks (as of Apr 2020)
  * Software behind the service is open source. You can run your own instance if you'd like to.

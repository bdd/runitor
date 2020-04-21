# hcpingrun

`hcpingrun` runs your commands and notifies healthchecks.io if they successfully
successfuly exited or not.

[Healthchecks.io](https://healthchecks.io) is a service for monitoring cron
jobs and similar periodic processes. It's like a dead man's switch for your
cron jobs so you can get alerted if they fail to run by the expected period.

## Example usage:

### Certificate renewal

```
% hcpingrun --check <check-uuid> -- dehydrated --cron --config example.com.conf
```

### Mirror Git Repositories

```
# Run the mirror script every 30 minutes 5 seconds.

% hcpingrun --check <check-uuid> -every 30m5s -- /script/git-mirror
```

### Periodic Backup

```
# Do not attach output to ping. Backup software may leak filenames and paths.

% hcpingrun --check <check-uuid> -nooutput -- restic backup /home /etc
```

## Why do I need this? I can just run curl from my shell script.

You sure can but hcpingrun offers a clean separation of concerns from the thing
that needs to be executed and signalling its execution to an external monitoring
service. You probably want to get alerted if your script abrubtly exits without
a chance signal failure.  This little utility also offer a few other neat
features like:

  * Ability to capture the stdout and stderr output of the command and send it
    along with the ping so when you respond to an alert, you can start
    invetigating the issue with some relevant context.

  * Act as a long running process, executing the command periodically if you
    don't readily have access to a job scheduler like crond or systemd timers.
    Works well in single process containers. It's small and simple.


## Remind me again, what is this healthchecks.io service?

  * [healthchecks.io](https://healthchecks.io) listens for HTTP requests
    ("pings") from services being monitored.
  * It keeps silent as long as pings arrive on time.
  * It raises an alert as soon as a ping does not arrive on time.
  * Checks can signal start of execution so run durations can be tracked.
  * Checks can explicitly signal failure of execution so an alert can be raised.
  * Checks can attach up to 10KB of logs to success/failure pings.
  * It has a free tier option with up to 20 checks (as of Apr 2020)
  * It can alert you via email, sms, whatsapp, slack, and many more services.

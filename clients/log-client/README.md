# Log Client
Log client collects AccessLogs and Metrics from SentryFlow and prints them to the terminal or saves them to a log file.

## Log Client Deployment
Log client can be deployed using kubectl command. The deployment can be accomplished with the following
commands:
```bash
$ cd SentryFlow/deployments
$ kubectl apply -f log-client.yaml
```

## Log client options
These are the default env value in the log-client.yaml file.
```bash
env:
- name: LOG_CFG
  value: "stdout"
- name: METRIC_CFG
  value: "stdout"
- name: METRIC_FILTER
  value: "api"
```

If you want to change the default env value, you can refer to the following options.
```bash
env:
- name: LOG_CFG
  value: {"stdout"|"file"|"none"}
- name: METRIC_CFG
  value: {"stdout"|"file"|"none"}
- name: METRIC_FILTER
  value: {"all"|"api"|"envoy"}
```

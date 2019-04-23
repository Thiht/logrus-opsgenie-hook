# logrus-opsgenie-hook

logrus-opsgenie-hook is a [Logrus](https://github.com/sirupsen/logrus) hook used to push alerts on OpsGenie.

The goal is to be more flexible than [JackFazackerley/logrus-opsgenie-hook](https://github.com/JackFazackerley/logrus-opsgenie-hook). The usage should also be simpler since it doesn't force to use the [alertsv2.CreateAlertRequest](https://godoc.org/github.com/opsgenie/opsgenie-go-sdk/alertsv2#CreateAlertRequest) structure in your code.

## Quick start

```go
import (
  "github.com/opsgenie/opsgenie-go-sdk/alertsv2"
  "github.com/sirupsen/logrus"
  "github.com/Thiht/logrus-opsgenie-hook"
)

func main() {
  opsgenieHook, _ := opsgenie.NewHook("my-api-token", opsgenie.EndpointEU, opsgenie.HookConfig{
    DefaultTeams: []alertsv2.Team{
      { Name: "my-team-name" },
    },
    DefaultSource: "my-app",
    DefaultPriority: alertsv2.P1,
  })

  log.AddHook(opsgenieHook)
}
```

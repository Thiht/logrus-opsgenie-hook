package opsgenie

import (
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"

	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/sirupsen/logrus"
)

const (
	// EndpointEU is the OpsGenie API URL located in Europe
	EndpointEU = "https://api.eu.opsgenie.com"
	// EndpointUS is the OpsGenie API URL located in the USA
	EndpointUS = "https://api.opsgenie.com"
)

const (
	// OverridePrefix defines a prefix used to override some configuration on runtime using `WithField`/`WithFields`
	// ogh stands for "OpsGenie Hook"
	OverridePrefix = "ogh:"
	OverrideAlias  = OverridePrefix + "alias"
	OverrideSource = OverridePrefix + "source"
	// OverrideTags *appends* tags to the default tags, it does not replace them
	OverrideTags     = OverridePrefix + "tags"
	OverrideEntity   = OverridePrefix + "entity"
	OverridePriority = OverridePrefix + "priority"
)

// HookConfig allows to declare a default configuration for the OpsGenie alerts
type HookConfig struct {
	DefaultTeams  []alertsv2.Team
	DefaultTags   []string
	DefaultEntity string
	DefaultSource string
	// DefaultPriority will fallback to P3 if it's not set
	// It can be overridden on runtime with the Logrus field `ogh:priority`
	DefaultPriority alertsv2.Priority
}

// Validate checks the content of the hook configuration and sanitizes it
func (c *HookConfig) Validate() error {
	if c.DefaultTeams == nil {
		c.DefaultTeams = []alertsv2.Team{}
	}

	if c.DefaultTags == nil {
		c.DefaultTags = []string{}
	}

	if c.DefaultPriority == "" {
		c.DefaultPriority = alertsv2.P3
	}
	if !isValidPriority(c.DefaultPriority) {
		return fmt.Errorf("invalid priority")
	}

	return nil
}

type hook struct {
	client *ogcli.OpsGenieAlertV2Client
	config HookConfig
}

func NewHook(apiKey, endpoint string, config HookConfig) (logrus.Hook, error) {
	// Sanity checks
	if apiKey == "" {
		return nil, fmt.Errorf("api key must be specified")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint must be specified")
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(apiKey)
	cli.SetOpsGenieAPIUrl(endpoint)

	client, err := cli.AlertV2()
	if err != nil {
		return nil, err
	}

	return &hook{
		client: client,
		config: config,
	}, nil
}

func (h *hook) Fire(entry *logrus.Entry) error {
	alert := alertsv2.CreateAlertRequest{
		Message:     entry.Message,
		Alias:       h.alias(entry),
		Description: h.description(entry),
		Teams:       h.teams(entry),
		Tags:        h.tags(entry),
		Details:     h.details(entry),
		Entity:      h.entity(entry),
		Source:      h.source(entry),
		Priority:    h.priority(entry),
	}

	_, err := h.client.Create(alert)
	return err
}

// Levels indicates that the hook will be triggered on the levels Error, Fatal, and Panic
func (*hook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}

// alias returns:
// - the content of the `ogh:alias` field if it's present
// - or the CRC32 checksum of the entry message
func (*hook) alias(entry *logrus.Entry) string {
	if aliasOverride, ok := entry.Data[OverrideAlias].(string); ok {
		return aliasOverride
	}

	// we don't need to be cryptographically secure
	h := crc32.ChecksumIEEE([]byte(entry.Message))
	return strconv.FormatUint(uint64(h), 16)
}

// description returns the entry message (ie. `Error("...")`), followed by the entry error (ie. `WithError(...)`) if it's present
func (*hook) description(entry *logrus.Entry) string {
	description := entry.Message
	if errValue, ok := entry.Data["error"].(error); ok {
		description += "\n" + errValue.Error()
	}
	return description
}

// teams returns the list of default teams declared in the hook configuration
func (h *hook) teams(entry *logrus.Entry) []alertsv2.TeamRecipient {
	teams := []alertsv2.TeamRecipient{}
	for _, team := range h.config.DefaultTeams {
		teams = append(teams, &team)
	}
	return teams
}

// tags returns the list of default tags declared in the hook configuration, completed with the list of tags in the `ogh:tags` field if it's present
func (h *hook) tags(entry *logrus.Entry) []string {
	tags := h.config.DefaultTags
	if tagsOverride, ok := entry.Data[OverrideTags].([]string); ok {
		tags = append(tags, tagsOverride...)
	}
	return tags
}

// details returns the entry fields, excepts those prefixed with the `ogh:` configuration prefix
func (*hook) details(entry *logrus.Entry) map[string]string {
	details := map[string]string{}
	for key, value := range entry.Data {
		// ignore keys starting with the configuration override prefix
		if strings.HasPrefix(key, OverridePrefix) {
			continue
		}
		details[key] = fmt.Sprintf("%v", value)
	}
	return details
}

// entity returns:
// - the content of the `ogh:entity` field if it's present
// - or the default entity declared in the hook configuration
func (h *hook) entity(entry *logrus.Entry) string {
	if entityOverride, ok := entry.Data[OverrideEntity].(string); ok {
		return entityOverride
	}
	return h.config.DefaultEntity
}

// source returns:
// - the content of the `ogh:source` field if it's present
// - or the default source declared in the hook configuration
func (h *hook) source(entry *logrus.Entry) string {
	if sourceOverride, ok := entry.Data[OverrideSource].(string); ok {
		return sourceOverride
	}
	return h.config.DefaultSource
}

// priority returns:
// - the content of the `ogh:priority` field if it's present and valid
// - or the default priority declared in the hook configuration
func (h *hook) priority(entry *logrus.Entry) alertsv2.Priority {
	if priorityOverride, ok := entry.Data[OverridePriority].(alertsv2.Priority); ok && isValidPriority(priorityOverride) {
		return priorityOverride
	}
	return h.config.DefaultPriority
}

// isValidPriority is a missing helper from the OpsGenie SDK
// It checks that a priority is valid
func isValidPriority(priority alertsv2.Priority) bool {
	return priority == alertsv2.P1 ||
		priority == alertsv2.P2 ||
		priority == alertsv2.P3 ||
		priority == alertsv2.P4 ||
		priority == alertsv2.P5
}

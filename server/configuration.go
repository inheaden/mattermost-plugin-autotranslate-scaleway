package main

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

const (
	defaultScalewayBaseURL     = "https://api.scaleway.ai/v1"
	defaultScalewayModel       = "mistral-small-3.2-24b-instruct-2506"
	defaultScalewayPrompt      = "You are a translation engine. Preserve meaning, tone, markdown, mentions, URLs, and line breaks. Return only valid JSON."
	defaultScalewayTemperature = float64(0)
	defaultCacheTTLSeconds     = 604800
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
type configuration struct {
	ScalewaySecretKey    string
	ScalewayProjectID    string
	ScalewayModel        string
	ScalewayBaseURL      string
	ScalewaySystemPrompt string
	ScalewayTemperature  float64
	ScalewayMaxTokens    int
	EnableCache          bool
	CacheTTLSeconds      int

	// disable plugin
	disabled bool
}

// Clone deep copies the configuration. Your implementation may only require a shallow copy if
// your configuration has no reference types.
func (c *configuration) Clone() *configuration {
	return &configuration{
		ScalewaySecretKey:    c.ScalewaySecretKey,
		ScalewayProjectID:    c.ScalewayProjectID,
		ScalewayModel:        c.ScalewayModel,
		ScalewayBaseURL:      c.ScalewayBaseURL,
		ScalewaySystemPrompt: c.ScalewaySystemPrompt,
		ScalewayTemperature:  c.ScalewayTemperature,
		ScalewayMaxTokens:    c.ScalewayMaxTokens,
		EnableCache:          c.EnableCache,
		CacheTTLSeconds:      c.CacheTTLSeconds,
		disabled:             c.disabled,
	}
}

func (c *configuration) getScalewayBaseURL() string {
	if c.ScalewayBaseURL != "" {
		return strings.TrimRight(c.ScalewayBaseURL, "/")
	}

	if c.ScalewayProjectID != "" {
		return fmt.Sprintf("https://api.scaleway.ai/%s/v1", c.ScalewayProjectID)
	}

	return defaultScalewayBaseURL
}

func (c *configuration) getScalewayModel() string {
	if c.ScalewayModel == "" {
		return defaultScalewayModel
	}

	return c.ScalewayModel
}

func (c *configuration) getScalewaySystemPrompt() string {
	if c.ScalewaySystemPrompt == "" {
		return defaultScalewayPrompt
	}

	return c.ScalewaySystemPrompt
}

func (c *configuration) getScalewayTemperature() float64 {
	return c.ScalewayTemperature
}

func (c *configuration) isCacheEnabled() bool {
	return c.EnableCache
}

func (c *configuration) getCacheTTLSeconds() int {
	if c.CacheTTLSeconds <= 0 {
		return defaultCacheTTLSeconds
	}

	return c.CacheTTLSeconds
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	configuration := p.getConfiguration().Clone()

	// Load the public configuration fields from the Mattermost server configuration.
	if loadConfigErr := p.API.LoadPluginConfiguration(configuration); loadConfigErr != nil {
		return errors.Wrap(loadConfigErr, "failed to load plugin configuration")
	}

	p.setConfiguration(configuration)

	return nil
}

// setEnabled wraps setConfiguration to configure if the plugin is enabled.
func (p *Plugin) setEnabled(enabled bool) {
	var configuration = p.getConfiguration().Clone()
	configuration.disabled = !enabled

	p.setConfiguration(configuration)
}

// IsValid validates plugin configuration
func (p *Plugin) IsValid() error {
	configuration := p.getConfiguration()
	if configuration.ScalewaySecretKey == "" {
		return fmt.Errorf("Must have Scaleway Secret Key")
	}

	if configuration.getScalewayModel() == "" {
		return fmt.Errorf("Must have Scaleway Model")
	}

	if configuration.getScalewayBaseURL() == "" {
		return fmt.Errorf("Must have Scaleway Base URL")
	}

	return nil
}

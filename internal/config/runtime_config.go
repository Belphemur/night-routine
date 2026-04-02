package config

import (
	"github.com/belphemur/night-routine/internal/constants"
	"golang.org/x/oauth2"
)

// ConfigStoreInterface defines the interface for configuration storage.
// Implementations decide where data comes from — database or static file config.
// This is the single source of truth for all configuration in handlers and services.
type ConfigStoreInterface interface {
	GetParents() (parentA, parentB string, err error)
	GetAvailability(parent string) ([]string, error)
	GetSchedule() (updateFrequency string, lookAheadDays, pastEventThresholdDays int, statsOrder constants.StatsOrder, err error)
	// GetOAuthConfig returns the OAuth2 configuration (static, from environment / file config).
	GetOAuthConfig() *oauth2.Config
}

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validConfiguration = &ApplicationConfiguration{
	MainGoal: "test",
}

func TestApplicationConfigurationValidatesValidConfiguration(t *testing.T) {
	assert.Nil(t, validConfiguration.Validate())
}

func TestApplicationConfigurationDoesNotValidateMissingMainGoal(t *testing.T) {
	copy := *validConfiguration
	copy.MainGoal = ""
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Main goal is not set", copy.Validate().Error())
}

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validConfiguration = &ApplicationConfiguration{
	Goals: map[string]GoalConfiguration{
		"test": GoalConfiguration{
			Image: "alpine:3.2",
		},
	},
	MainGoal: "test",
}

func TestApplicationConfigurationValidatesValidConfiguration(t *testing.T) {
	assert.Nil(t, validConfiguration.Validate())
}

func TestApplicationConfigurationChecksForNotSetMainGoal(t *testing.T) {
	copy := *validConfiguration
	copy.MainGoal = ""
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Main goal is not set", copy.Validate().Error())
}

func TestApplicationConfigurationChecksMainGoalNotExisting(t *testing.T) {
	copy := *validConfiguration
	copy.MainGoal = "wrong"
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Main goal 'wrong' is not defined", copy.Validate().Error())
}

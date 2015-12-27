package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validConfiguration = &ApplicationConfiguration{
	Goals: map[string]*GoalConfiguration{
		"test": &GoalConfiguration{
			Image: "alpine:3.2",
		},
	},
	MainGoal: "test",
}

func TestApplicationConfigurationValidatesValidConfiguration(t *testing.T) {
	assert.Nil(t, validConfiguration.Validate())
}

func TestApplicationConfigurationChecksForNotSetMainGoal(t *testing.T) {
	copy := validConfiguration.Clone()
	copy.MainGoal = ""
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Main goal is not set", copy.Validate().Error())
}

func TestApplicationConfigurationChecksMainGoalNotExisting(t *testing.T) {
	copy := validConfiguration.Clone()
	copy.MainGoal = "wrong"
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Main goal 'wrong' is not defined", copy.Validate().Error())
}

func TestApplicationConfigurationChecksGoalNameNotValid(t *testing.T) {
	copy := validConfiguration.Clone()
	copy.Goals["this+|s|invalid"] = &GoalConfiguration{}
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Goal 'this+|s|invalid' has invalid name", copy.Validate().Error())
}

func TestApplicationConfigurationChecksValidGoalImageName(t *testing.T) {
	copy := validConfiguration.Clone()
	goal := copy.Goals["test"]
	goal.Image = "!"
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Goal 'test' has invalid image name", copy.Validate().Error())
}

func TestApplicationConfigurationChecksRunAfterGoalsExisting(t *testing.T) {
	copy := validConfiguration.Clone()
	goal := copy.Goals["test"]

	goal.RunAfter = []string{"test2"}
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Goal 'test' should run after goal 'test2' that does not exist", copy.Validate().Error())
}

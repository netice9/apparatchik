package core

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
		"otherTest": &GoalConfiguration{
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
	require.Equal(t, "Main goal \"wrong\" is not defined", copy.Validate().Error())
}

func TestApplicationConfigurationChecksGoalNameNotValid(t *testing.T) {
	copy := validConfiguration.Clone()
	copy.Goals["this+|s|invalid"] = &GoalConfiguration{}
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Goal \"this+|s|invalid\" has invalid name", copy.Validate().Error())
}

func TestApplicationConfigurationChecksValidGoalImageName(t *testing.T) {
	copy := validConfiguration.Clone()
	goal := copy.Goals["test"]
	goal.Image = "!"
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Goal \"test\" has invalid image name", copy.Validate().Error())
}

func TestApplicationConfigurationChecksRunAfterGoalsExisting(t *testing.T) {
	copy := validConfiguration.Clone()
	goal := copy.Goals["test"]

	goal.RunAfter = []string{"test2"}
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Goal \"test\" should run after goal \"test2\" that does not exist", copy.Validate().Error())
}

func TestApplicationConfigurationChecksLinkedGoalsExisting(t *testing.T) {
	copy := validConfiguration.Clone()
	goal := copy.Goals["test"]

	goal.Links = []string{"test2"}
	require.NotNil(t, copy.Validate())
	require.Equal(t, "Goal \"test\" links goal \"test2\" that does not exist", copy.Validate().Error())
}

func TestApplicationConfigurationWithCircularRunAfterDependency(t *testing.T) {
	copy := validConfiguration.Clone()
	goal := copy.Goals["otherTest"]

	goal.RunAfter = []string{"otherTest"}

	require.NotNil(t, copy.Validate())
	require.Equal(t, "Goal \"otherTest\" has a circular dependency \"otherTest\".", copy.Validate().Error())
}

func TestLinkedContainers(t *testing.T) {
	goal := GoalConfiguration{Links: []string{"c1", "c2:alias"}}
	require.Equal(t, []LinkedContainer{LinkedContainer{"c1", "c1"}, LinkedContainer{"c2", "alias"}}, goal.LinkedContainers())
}

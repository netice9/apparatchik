Feature: execute a simple task

Goal statuses:
  fetching_image
  failed
  terminated
  starting
  running
  created
  not_running
  waiting_for_dependencies
  error: explanation


Scenario: executing a task that won't fail
  Given I create an application with one task that will execute succesfully
  When I wait for the task to finish
  Then the exit code of the task should be 0
  And I should be able to retreive the logs of the task

Scenario: executing a task that depends on another task - happy path
  Given I create an application with one task depending on another task
  When I wait for the second task to finish
  Then the exit code of the first task should be 0
  And the exit code of the second task should be 0

Scenario: executing a task that depends on another task - first task failing
  Given I create an application with one task depending on another task with first task failing
  When I wait for the first task to fail
  Then the exit code of the first task should be 1
  And the second task should be waiting for successful execution of the first task

Scenario: deleting an existing application
  Given I create an application with one task that will execute succesfully
  And I wait for the task to finish
  When I delete the application
  Then the application should not exist anymore

#TODO: test for run_after pointing to itself
#TODO: test for run_after pointing to a task and not a service
#TODO: test for main goal not set
#TODO: test for main goal not existing
#TODO: test for image not specified
#TODO: test for logs of non existing goal


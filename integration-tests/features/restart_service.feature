Feature: restart service

Scenario: Stopping service when upstream service terminates
  Given I have a service depending on a service that will terminate
  When the first service terminates
  Then the second service should fail

Scenario: Restarting a single service
  Given I have a service that fails after a timeout and is marked to restart
  When the service has failed
  Then the service should be started again

Scenario: Restarting dependent service
  When I have a service depending on a service that will terminate and both services are marked to restart
  Then the second service should be restarted

Scenario: Restarting dependent task
  When I have a task depending on a service that will terminate with service marked to restart
  Then the task should be executed multiple times

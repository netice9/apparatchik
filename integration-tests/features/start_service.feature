Feature: start service

Scenario: starting service - happy path
  When I create an application with one service
  Then the service should be running

Scenario: starting service that links another service
  When I create an application with a service linked to another service
  Then both services should be running
  And second service should be linked to the first service

Scenario: starting service that links another service
  When I create an application with a service linked to another service with a link alias
  Then both services should be running
  And second service should be linked to the first service using the alias

Scenario: starting service - happy path
  When I create an application with one service
  Then the service should be running

Scenario: starting service that links another service
  When I create an application with a service linked to another service
  Then both services should be running
  And second service should be linked to the first service

Scenario: starting service that links another service
  When I create an application with a service linked to another service with a link alias
  Then both services should be running
  And second service should be linked to the first service using the alias

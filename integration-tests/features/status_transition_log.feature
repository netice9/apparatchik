Feature: status transition log

Scenario: status transition log for a started goal
  Given I create an application with one service
  When the service is running
  Then I should be able to inspect the goal transition log

Scenario: status transition log for a not existing goal
  Given I create an application with one service
  And the service is running
  When I inspect transition log an non-existing goal
  Then the service should respond with 404 status code

Scenario: status transition log for a non existing application
  Given there are no applications
  When I inspect transition log of a non-existing application
  Then the service should respond with 404 status code

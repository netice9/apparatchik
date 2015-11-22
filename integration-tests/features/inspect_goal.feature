Feature: inspect goal

Scenario: inspect goal of a service
  Given I create an application with one service
  When the service is running
  Then I should be able to inspect the goal container of the service

Scenario: inspect goal of an non-existing task
  Given I create an application with one service
  When I inspect a goal of an non-existing task
  Then the service should respond with 404 status code

Scenario: inspect goal of an non-existing application
  Given there are no applications
  When I inspect a goal of a non-existing application
  Then the service should respond with 404 status code

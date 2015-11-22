Feature: goal stats

  Scenario: stats of a service
    Given I create an application with one service that uses some cpu
    When I wait for the service to start
    Then the service should have some cpu and memory stats

  Scenario: current stats of a service
    Given I create an application with one service that uses some cpu
    When I wait for the service to start
    Then the service should have current stats

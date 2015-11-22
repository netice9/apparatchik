Feature: named container

  Scenario: standalone named container
    Given there is one application with a named container
    And the service is running
    Then a container with the service's name should exist

  Scenario: service linked to named container
    When I start an application with a service linked to another service in the same application that has a named container
    Then the second service should be running

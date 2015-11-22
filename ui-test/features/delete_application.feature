Feature: delete application

  Scenario: delete existing application
    Given there is one application run by apparatchik
    When I delete the application
    Then I should see empty list of applications

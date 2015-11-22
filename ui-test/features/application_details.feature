Feature: application details

  Scenario: show application details
    Given there is one application run by apparatchik
    When I click on details of that application
    Then I should see the name of the application
    Then I should see the name of the application's main goal
    And I should see application's goals

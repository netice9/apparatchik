Feature: goal output

  Scenario:
    Given there is one application run by apparatchik
    When I navigate to output of the application's goal
    Then I should see the output

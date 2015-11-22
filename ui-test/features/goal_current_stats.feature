Feature: goal current stats

  Scenario: current status of a running goal
    Given there is one application run by apparatchik
    When I navigate to application goal's current stats
    Then I should see the goal's current stats

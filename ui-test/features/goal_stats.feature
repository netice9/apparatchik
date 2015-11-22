Feature: goal stats

  Scenario:
    Given there is one application run by apparatchik
    When I navigate to application goal's stats
    Then I should see the goal's stats

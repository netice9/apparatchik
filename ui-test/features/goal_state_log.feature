Feature: goal state log

  Scenario:
    Given there is one application run by apparatchik
    When I navigate to application goal's state log
    Then I should see the goal state log

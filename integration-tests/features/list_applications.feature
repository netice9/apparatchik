Feature: list applications

Scenario: no applications
  Given there are no applications
  When I list the applications
  Then I should get an empty list

Scenario: one application
  Given there is one application
  When I list the applications
  Then I should get a list with only that application
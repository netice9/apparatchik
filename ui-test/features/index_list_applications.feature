Feature: index will list applications

Scenario: no applications
  When there are no applications running by apparatchik
  When I visit the index page
  Then I should see empty list of applications

Scenario: no applications
  When there is one application run by apparatchik
  When I visit the index page
  Then I should see the running application

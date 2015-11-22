Feature: upload application

  Scenario: upload new application
    When I click on Create Application
    And I fill in a new name for the application
    And I fill in a valid application description file
    And I upload the application
    Then the new application should be running
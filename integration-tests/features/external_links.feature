Feature: external_links

  Scenario: linking containers outside of the application
    Given there is one application with a named container
    And there is another application linking to the named container
    Then the second application should be running and be linked to the named container

@smoke
Feature: Staging auth smoke tests

  Scenario: User can sign in with email on desktop
    When I visit the staging login page
    And I sign in with email "smoke1@test.com"
    Then I should be on the staging matches page
    And I should see "smoke1@test.com" in the staging header

  Scenario: Second account can sign in with email on desktop
    When I visit the staging login page
    And I sign in with email "smoke2@test.com"
    Then I should be on the staging matches page
    And I should see "smoke2@test.com" in the staging header

  Scenario: User can sign in with email on iPhone 15
    Given I am on an iPhone 15 viewport
    When I visit the staging login page
    And I sign in with email "smoke1@test.com"
    Then I should be on the staging matches page
    And I should see "smoke1@test.com" in the staging header

  Scenario: User can sign in with email on Galaxy S24
    Given I am on a Galaxy S24 viewport
    When I visit the staging login page
    And I sign in with email "smoke1@test.com"
    Then I should be on the staging matches page
    And I should see "smoke1@test.com" in the staging header

  Scenario: Google sign-in initiates redirect on desktop
    When I visit the staging login page
    And I click the Google sign-in button
    Then the page should navigate toward Google for authentication

  Scenario: Google sign-in initiates redirect on iPhone 15
    Given I am on an iPhone 15 viewport
    When I visit the staging login page
    And I click the Google sign-in button
    Then the page should navigate toward Google for authentication

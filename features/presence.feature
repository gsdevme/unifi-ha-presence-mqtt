Feature: Publish UniFi device presence to MQTT with Home Assistant autodiscovery

  The service authenticates to the (mock) UniFi controller, reads the active
  clients, applies the presence debounce and republishes each tracked device to
  MQTT with HA autodiscovery.

  Scenario: Authenticate and publish discovery on startup
    Given the tracked devices are "phone1,phone2"
    And "phone1" is active on the controller
    When the service starts up
    Then the presence discovery for "phone1" is published retained
    And the connectivity discovery for "phone1" is published retained
    And the presence discovery for "phone2" is published retained
    And availability "online" is published retained

  Scenario: A present device reports home immediately
    Given the tracked devices are "phone1"
    And "phone1" is active on the controller
    When the service starts up
    And a poll runs
    Then the state for "phone1" reports present true
    And the readiness endpoint reports ready

  Scenario: An absent device reports not_home only after the failure threshold
    Given the tracked devices are "phone1"
    And the failure threshold is 2
    And "phone1" is active on the controller
    When the service starts up
    And a poll runs
    Then the state for "phone1" reports present true
    When "phone1" leaves the controller
    And a poll runs
    Then the state for "phone1" reports present true
    When a poll runs
    Then the state for "phone1" reports present false

  Scenario: Re-authenticate when the controller session expires
    Given the tracked devices are "phone1"
    And "phone1" is active on the controller
    When the service starts up
    And a poll runs
    And the controller session expires
    And a poll runs
    Then the controller received more than one login
    And the state for "phone1" reports present true

  Scenario: Graceful shutdown publishes offline
    Given the tracked devices are "phone1"
    When the service starts up
    And the service shuts down
    Then availability "offline" is published retained

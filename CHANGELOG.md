# Changelog

## [2.0.0](https://github.com/gsdevme/unifi-ha-presence-mqtt/compare/v1.2.1...v2.0.0) (2026-08-04)


### ⚠ BREAKING CHANGES

* **unifi:** UNIFI_USERNAME/UNIFI_PASSWORD are removed; set UNIFI_API_KEY to a local Integration API key instead. TRACK_DEVICES now takes Name/Name=Pattern entries matched as substrings against the client's name or MAC.

### Features

* **config:** add fail-fast env config with redacted String() ([5316558](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/53165580802f8d65cbbc437461b309a4d8761b05))
* **homeassistant:** add presence discovery + state payload builder ([5e00e97](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/5e00e97a317ddfb796f29054c82bc50b834d261b))
* **mock,cmd:** add UniFi controller mock and wire the serve/mock commands ([d6af083](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/d6af083428849a40d290f4f22bcab1755d803583))
* **mqtt,publisher:** add autopaho MQTT5 client and presence publisher ([9643ed3](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/9643ed311c0198af894e582537b9e4238e5bbeae))
* **presence:** add tracker with asymmetric presence debounce ([cab2b9e](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/cab2b9e13b851b0448fba6bb6de026afc6b32d09))
* scaffold spec-driven rebuild in hyundai-bluelink-mqtt style ([43f6ab2](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/43f6ab25b4d8272e78fe1d360794aea705c63363))
* **scheduler,server:** add presence poll loop and health/status server ([eebb40c](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/eebb40c0dda5d5f3456161595cf44b7f5e15211c))
* **unifi:** bake in a self-contained UniFi controller client ([2ffbfea](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/2ffbfeae058b16501649f4c532c9eda4481cb163))
* **unifi:** migrate to the official Network Integration API with name/MAC matching ([760bf9f](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/760bf9f75ff6a8ce9b894214b409ebf41726aaad))


### Bug Fixes

* **cmd:** give the MQTT manager its own lifetime so offline is published on shutdown ([e250c05](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/e250c05d0ed06daeea9e959532d363e09909e017))

## Changelog

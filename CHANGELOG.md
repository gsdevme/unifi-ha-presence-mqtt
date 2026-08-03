# Changelog

## 1.0.0 (2026-08-03)


### ⚠ BREAKING CHANGES

* **unifi:** UNIFI_USERNAME/UNIFI_PASSWORD are removed; set UNIFI_API_KEY to a local Integration API key instead. TRACK_DEVICES now takes Name/Name=Pattern entries matched as substrings against the client's name or MAC.

### Features

* add a very rough outline of the integration ([2c3295e](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/2c3295edad1eb55534a5db9f8a199c3794227352))
* add default environment variables ([0941d26](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/0941d261148a19292c533c8e8f145272a9df323a))
* add FailureThreshold to smooth out blips/disconnects ([4d3774f](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/4d3774f9cf038219d381f66357e6748e1197a232))
* add linux arm image ([ddbd781](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/ddbd781f48a3019af696aa879f80e84a709bb0c7))
* **config:** add fail-fast env config with redacted String() ([5316558](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/53165580802f8d65cbbc437461b309a4d8761b05))
* **homeassistant:** add presence discovery + state payload builder ([5e00e97](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/5e00e97a317ddfb796f29054c82bc50b834d261b))
* implement a primitive retry for an auth error ([8fad542](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/8fad542af8314962f67c389bee117bc6c46290f1))
* implement primitive polling of the api ([ce15718](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/ce15718db26c48e2f3b6e1b91cf1ba24ce53f8c6))
* **mock,cmd:** add UniFi controller mock and wire the serve/mock commands ([d6af083](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/d6af083428849a40d290f4f22bcab1755d803583))
* move configuration to ENVs ([8aa5068](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/8aa5068b6b6c55ea87589711ae5f472b78b3c75b))
* **mqtt,publisher:** add autopaho MQTT5 client and presence publisher ([9643ed3](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/9643ed311c0198af894e582537b9e4238e5bbeae))
* **presence:** add tracker with asymmetric presence debounce ([cab2b9e](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/cab2b9e13b851b0448fba6bb6de026afc6b32d09))
* publish container to dockerhub ([d03b336](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/d03b336864daa9c07b9f20e4c2f4377915751cb3))
* scaffold spec-driven rebuild in hyundai-bluelink-mqtt style ([43f6ab2](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/43f6ab25b4d8272e78fe1d360794aea705c63363))
* **scheduler,server:** add presence poll loop and health/status server ([eebb40c](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/eebb40c0dda5d5f3456161595cf44b7f5e15211c))
* **unifi:** bake in a self-contained UniFi controller client ([2ffbfea](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/2ffbfeae058b16501649f4c532c9eda4481cb163))
* **unifi:** migrate to the official Network Integration API with name/MAC matching ([760bf9f](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/760bf9f75ff6a8ce9b894214b409ebf41726aaad))
* update deps ([1830166](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/183016638ac8cfc3ee78d5afc6d224f81e277024))
* use latest pkg ([f49b0c8](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/f49b0c85ccd2d3835f323ea1f44cbba8fa76988e))


### Bug Fixes

* add github token env ([51c3634](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/51c3634d48ea0d2133eef57d31b798010b507e84))
* allow manual dispatch of workflow ([945e5f7](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/945e5f75248dc9a329b3fa40531905be76db424a))
* **cmd:** give the MQTT manager its own lifetime so offline is published on shutdown ([e250c05](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/e250c05d0ed06daeea9e959532d363e09909e017))
* detect the type of error rather than pointer address ([9d29346](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/9d2934627fc9b4c49e9156bce2eb73d999026246))
* dont wrap the error to ensure error.is() can assert ([cb0d6f6](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/cb0d6f6398772bb4fea69a18daa495dac0246aa7))
* ensure broker tcp is open and implement timeouts ([0cd7954](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/0cd7954cba0acaa510d35e6e5e7b6c6aad2b5d67))
* increase FailureThreshold to 5 ([0bb9b34](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/0bb9b34f76753437d1c319402fc3ded5d959fa67))


### Performance Improvements

* once auth token is created use it for the polling ([55fe8af](https://github.com/gsdevme/unifi-ha-presence-mqtt/commit/55fe8afc42e3596a06f86f7e3cb8070fa61b07e5))

## Changelog

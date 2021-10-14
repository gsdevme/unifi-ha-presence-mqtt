FROM scratch

ENV MQTT_DNS="mqtt://127.0.0.1:1883"
ENV TRACK_DEVICES=""
ENV UNIFI_HOST="https://192.168.1.1"
ENV UNIFI_PASSWORD="password"
ENV UNIFI_USERNAME=admin

ENTRYPOINT ["/unifi-ha-presence-mqtt"]
COPY unifi-ha-presence-mqtt /
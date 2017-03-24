FROM debian:jessie
COPY mqtt_blackbox_exporter /bin/mqtt_blackbox_exporter
ENTRYPOINT ["/bin/mqtt_blackbox_exporter"]
CMD ["-config.file /config.yaml"]

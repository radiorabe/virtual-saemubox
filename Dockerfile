FROM scratch

COPY virtual-saemubox /

COPY etc/passwd /etc/passwd

USER 65534

ENTRYPOINT ["/virtual-saemubox"]

FROM scratch

COPY emptypage /

COPY etc/passwd /etc/passwd

USER 65534

ENTRYPOINT ["/virtual-saemubox"]

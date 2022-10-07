FROM gcr.io/distroless/base-debian11
ENTRYPOINT ["/balanced"]
COPY balanced /

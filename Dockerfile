FROM gcr.io/distroless/base-debian12
ENTRYPOINT ["/balanced"]
COPY balanced /

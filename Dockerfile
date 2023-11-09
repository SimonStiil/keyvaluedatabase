FROM scratch
WORKDIR /app
COPY kvdb /usr/bin/
LABEL org.opencontainers.image.source https://github.com/SimonStiil/keyvaluedatabase
LABEL org.opencontainers.image.description "A Key Value database usable as a webhook server"
LABEL org.opencontainers.image.licenses GPL-2.0-only
ENTRYPOINT ["kvdb"]
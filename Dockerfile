FROM scratch
WORKDIR /app
COPY kvdb /usr/bin/
LABEL org.opencontainers.image.source https://github.com/SimonStiil/keyvaluedatabase
ENTRYPOINT ["kvdb"]
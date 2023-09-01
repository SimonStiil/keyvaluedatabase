FROM scratch
WORKDIR /app
COPY kvdb /usr/bin/
ENTRYPOINT ["kvdb"]
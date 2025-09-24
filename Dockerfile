FROM scratch
ARG TARGETARCH
WORKDIR /app
COPY keyvaluedatabase-${TARGETARCH} /usr/bin/keyvaluedatabase
ENTRYPOINT ["keyvaluedatabase"]
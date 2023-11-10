source docker.env
echo -n "FROM ${DOCKER_BASE}
WORKDIR ${DOCKER_WORKDIR}
COPY ${DOCKER_APPLICATION} ${DOCKER_DESTINATION}
LABEL org.opencontainers.image.source ${DOCKER_SOURCE}
LABEL org.opencontainers.image.licenses ${DOCKER_LICENSE}
ENTRYPOINT [\"${DOCKER_APPLICATION}\"]" >Dockerfile
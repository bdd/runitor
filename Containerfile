ARG BUILDER_IMG=docker.io/alpine:latest
FROM --platform=${BUILDPLATFORM} ${BUILDER_IMG} AS build
ARG RELEASE
ARG RELBUILD
ARG TARGETOS
ARG TARGETARCH
RUN apk add bash coreutils curl jq openssh-keygen
COPY scripts/dlrel scripts/verify /usr/local/bin/
RUN DOWNLOAD_DIR=/tmp/rel dlrel ${RELBUILD} ${TARGETOS} ${TARGETARCH}

ARG RUNTIME_IMG
ARG TARGETPLATFORM
FROM --platform=${TARGETPLATFORM} ${RUNTIME_IMG}
ARG RELEASE
ARG RELBUILD
ARG TARGETOS
ARG TARGETARCH
LABEL \
  org.opencontainers.image.title="runitor" \
  org.opencontainers.image.description="Runitor is a command runner with healthchecks.io integration." \
  org.opencontainers.image.url="https://bdd.fi/x/runitor" \
  org.opencontainers.image.source="https://github.com/bdd/runitor" \
  org.opencontainers.image.authors="Berk D. Demir <https://bdd.fi>" \
  org.opencontainers.image.version=${RELEASE}
COPY --from=build --chmod=0755 /tmp/rel/runitor-${RELBUILD}-${TARGETOS}-${TARGETARCH} /usr/local/bin/runitor

# Unlike Alpine, Debian and Ubuntu container images do not ship with trust
# anchors needed to verify TLS certificates.
#
# A GH Issue from Nov 2017 (still open as of Mar 2023):
# https://github.com/debuerreotype/docker-debian-artifacts/issues/15
#
# This RUN step of installing `ca-certificates` necessitates binfmt_misc
# registrations for target architectures.
RUN if [ -f /etc/debian_version ]; then \
  export DEBIAN_FRONTEND=noninteractive && \
  apt-get update && \
  apt-get install -y ca-certificates && \
  rm -rf /var/lib/apt/lists/*; \
fi

ENTRYPOINT ["/usr/local/bin/runitor"]
CMD ["-help"]

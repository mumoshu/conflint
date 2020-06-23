FROM golang:1.14 as build

LABEL maintainer="Yusuke KUOKA <https://github.com/mumoshu/conflint/issues>" \
      org.opencontainers.image.title="conflint" \
      org.opencontainers.image.description="Enterprise-grade repository and secrets management for Flux CD" \
      org.opencontainers.image.url="https://github.com/mumoshu/conflint" \
      org.opencontainers.image.source="git@github.com:mumoshu/conflint" \
      org.opencontainers.image.vendor="mumoshu" \
      org.label-schema.schema-version="1.0" \
      org.label-schema.name="conflint" \
      org.label-schema.description="Unified config linter" \
      org.label-schema.url="https://github.com/mumoshu/conflint" \
      org.label-schema.vcs-url="git@github.com:mumoshu/conflint" \
      org.label-schema.vendor="mumoshu"

RUN apt-get update -y \
 && apt-get install -y curl

RUN curl -sfL https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh| sh -s -- -b /usr/local/bin v0.10.0

RUN curl -LO https://github.com/instrumenta/conftest/releases/download/v0.19.0/conftest_0.19.0_Linux_x86_64.tar.gz \
 && tar xzf conftest_0.19.0_Linux_x86_64.tar.gz \
 && mv conftest /usr/local/bin

RUN curl -LO https://github.com/instrumenta/kubeval/releases/download/0.15.0/kubeval-linux-amd64.tar.gz \
 && tar xzf kubeval-linux-amd64.tar.gz \
 && mv kubeval /usr/local/bin

WORKDIR /build

COPY ./conflint /build

FROM ubuntu:20.04 as runtime

COPY --from=build /usr/local/bin/kubeval /usr/local/bin/kubeval
COPY --from=build /usr/local/bin/conftest /usr/local/bin/conftest
COPY --from=build /usr/local/bin/reviewdog /usr/local/bin/reviewdog
COPY --from=build /build/conflint /usr/local/bin/conflint

ENV PATH=/bin:/usr/bin:/usr/local/bin

RUN conftest --version \
 && kubeval --version \
 && reviewdog --version \
 && conflint -h

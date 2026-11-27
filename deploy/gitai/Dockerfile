# gitai (feder-cr/git-ai) server image.
# Builds gitaly + bundled Git + graft CLI from a pinned upstream tag.
# The build is slow (it compiles three bundled Git versions), so CI builds it once per release.
ARG GITAI_REF=v0.4.2

FROM golang:1.26-bookworm AS build
ARG GITAI_REF
RUN apt-get update && apt-get install -y --no-install-recommends \
      build-essential meson ninja-build pkg-config cmake \
      libssl-dev libcurl4-openssl-dev zlib1g-dev libpcre2-dev \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
RUN git clone --depth 1 --branch ${GITAI_REF} https://github.com/feder-cr/git-ai.git .
RUN make build && mkdir -p /out && \
    for b in gitaly gitaly-hooks gitaly-ssh graft; do cp _build/bin/$b /out/; done

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates libcurl4 libpcre2-8-0 zlib1g openssl git \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/ /opt/gitai/bin/
COPY entrypoint.sh /opt/gitai/entrypoint.sh
ENV PATH=/opt/gitai/bin:$PATH
VOLUME /var/lib/gitai
EXPOSE 8075 9236
ENTRYPOINT ["/opt/gitai/entrypoint.sh"]

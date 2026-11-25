ARG PG_MAJOR=18
ARG PGX_ULID_VERSION=0.2.2

FROM postgres:${PG_MAJOR}-bookworm AS base

FROM base AS extension
ARG PG_MAJOR
ARG PGX_ULID_VERSION
ARG TARGETARCH

ADD https://github.com/pksunkara/pgx_ulid/releases/download/v${PGX_ULID_VERSION}/pgx_ulid-v${PGX_ULID_VERSION}-pg${PG_MAJOR}-${TARGETARCH}-linux-gnu.deb \
    /tmp/pgx_ulid.deb

FROM base
ARG PG_MAJOR

COPY --from=extension /tmp/pgx_ulid.deb /tmp/pgx_ulid.deb

RUN apt-get update \
    && apt-get install -y --no-install-recommends /tmp/pgx_ulid.deb \
    && rm -rf /var/lib/apt/lists/* /tmp/pgx_ulid.deb

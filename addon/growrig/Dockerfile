# Thin Home Assistant add-on image: drop the prebuilt, arch-matched Grow Core
# binary onto the HA base image. All heavy lifting (Go compile, web build,
# catalogue embedding) happens ahead of time via `make addon` on the build
# host, so on-device install is just a copy — fast even on a Raspberry Pi.
ARG BUILD_FROM
FROM ${BUILD_FROM}

# BUILD_ARCH is supplied by the Supervisor (aarch64 | amd64 | armv7) and selects
# the matching binary produced by `make addon` into ./bin/growcore.<arch>.
ARG BUILD_ARCH

COPY rootfs /
COPY bin/growcore.${BUILD_ARCH} /usr/bin/growcore

RUN apk add --no-cache ffmpeg \
    && chmod +x /usr/bin/growcore /run.sh

CMD [ "/run.sh" ]

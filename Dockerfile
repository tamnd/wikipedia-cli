# Consumed by GoReleaser: it copies the already cross-compiled binary out of the
# build context rather than compiling, so the image build is fast and uses the
# same static binary every other artifact ships.
#
# GoReleaser builds one multi-platform image with buildx and stages each
# platform's binary under a $TARGETPLATFORM directory (e.g. linux/amd64/) in the
# build context, so the COPY line selects the right one through the automatic
# TARGETPLATFORM build arg.
FROM alpine:3.21

ARG TARGETPLATFORM

# ca-certificates for HTTPS to the Wikimedia hosts; tzdata for sane timestamps.
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 10001 wiki \
 && mkdir -p /data \
 && chown wiki:wiki /data

COPY $TARGETPLATFORM/wiki /usr/bin/wiki

USER wiki
WORKDIR /data

# All state lives under /data; mount a volume to keep the cache and downloads:
#
#   docker run -v ~/data/wiki:/data ghcr.io/tamnd/wiki read "Alan Turing"
ENV WIKI_DATA_DIR=/data
VOLUME ["/data"]

ENTRYPOINT ["/usr/bin/wiki"]

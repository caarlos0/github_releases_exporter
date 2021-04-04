FROM alpine
EXPOSE 9222
WORKDIR /
COPY github_releases_exporter*.apk /tmp
RUN apk add --allow-untrusted /tmp/github_releases_exporter_*.apk
ENTRYPOINT ["/usr/local/bin/github_releases_exporter"]

FROM gcr.io/distroless/base
EXPOSE 9222
WORKDIR /
COPY releases.yml .
COPY github_releases_exporter .
ENTRYPOINT ["./github_releases_exporter"]

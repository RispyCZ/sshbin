ARG BUILDER_IMAGE=caddy:builder RUNTIME_IMAGE=gcr.io/distroless/static-debian13

FROM ${BUILDER_IMAGE} as build
RUN xcaddy build --with github.com/mholt/caddy-l4

FROM ${RUNTIME_IMAGE}
COPY --from=build /usr/bin/caddy /usr/bin/caddy
ENTRYPOINT ["caddy"]
CMD ["run", "-c", "/etc/caddy/Caddyfile"]
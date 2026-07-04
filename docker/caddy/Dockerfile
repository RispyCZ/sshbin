ARG BUILDER_IMAGE=""
ARG RUNTIME_IMAGE=""

FROM ${BUILDER_IMAGE} as build

RUN xcaddy build --with github.com/mholt/caddy-l4

FROM ${RUNTIME_IMAGE}

COPY --from=build /usr/bin/caddy /usr/bin/caddy

ENTRYPOINT ["caddy"]
CMD ["run", "-c", "/etc/caddy/Caddyfile"]
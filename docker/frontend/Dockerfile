# use the official Bun image
# see all versions at https://hub.docker.com/r/oven/bun/tags
FROM oven/bun:1.4.3 AS base
WORKDIR /usr/src/app

# install dependencies into temp directory
# this will cache them and speed up future builds
FROM base AS install
RUN mkdir -p /temp/dev
COPY package.json bun.lock /temp/dev/
RUN cd /temp/dev && bun install --frozen-lockfile

# copy node_modules from temp directory
# then copy all (non-ignored) project files into the image
FROM base AS prerelease
COPY --from=install /temp/dev/node_modules node_modules
COPY . .

# [optional] tests & build
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN bun run build

# test stage: used by compose.test.yaml (`just test`), never part of the final image.
# Runs lint, typecheck, unit tests and the production build when the container starts.
FROM base AS test
# Vitest needs a real Node.js (jsdom does not work when Bun stands in for node).
# Keep the major in line with the Node version Bun reports (process.version) and @types/node.
COPY --from=node:26-slim /usr/local/bin/node /usr/local/bin/node
# Node 26 links against libatomic, which the Bun image does not ship.
RUN apt-get update && apt-get install -y --no-install-recommends libatomic1 && rm -rf /var/lib/apt/lists/* \
    && node --version
COPY --from=install /temp/dev/node_modules node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
CMD ["sh", "-c", "bun run lint && bun run typecheck && bun run test && bun run build"]

# the standalone output already contains the traced production dependencies
# (no separate `bun install --production` stage needed)
FROM base AS release

COPY --from=prerelease --chown=bun:bun /usr/src/app/public ./public
COPY --from=prerelease --chown=bun:bun /usr/src/app/.next/standalone .
COPY --from=prerelease --chown=bun:bun /usr/src/app/.next/static .next/static

RUN chown bun:bun /usr/src/app

# Install wget for healthcheck (Debian-based bun image)
USER root
RUN apt-get update && apt-get install -y --no-install-recommends wget curl ca-certificates && rm -rf /var/lib/apt/lists/*

# Set environment variables from build arguments
ENV NEXT_TELEMETRY_DISABLED=1
ENV HOSTNAME=0.0.0.0

# Healthcheck and run the app
USER bun
EXPOSE 3000/tcp
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 CMD wget -qO- http://localhost:3000/api/health || exit 1
ENTRYPOINT ["bun", "run", "server.js"]

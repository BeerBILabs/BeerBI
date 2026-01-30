# use the official Bun image
# see all versions at https://hub.docker.com/r/oven/bun/tags
FROM oven/bun:1 AS base
WORKDIR /usr/src/app

# install dependencies into temp directory
# this will cache them and speed up future builds
FROM base AS install
RUN mkdir -p /temp/dev
COPY package.json bun.lock /temp/dev/
RUN cd /temp/dev && bun install --frozen-lockfile

# install with --production (exclude devDependencies)
RUN mkdir -p /temp/prod
COPY package.json bun.lock /temp/prod/
RUN cd /temp/prod && bun install --frozen-lockfile --production

# copy node_modules from temp directory
# then copy all (non-ignored) project files into the image
FROM base AS prerelease
COPY --from=install /temp/dev/node_modules node_modules
COPY . .

# [optional] tests & build
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV NEXT_PRIVATE_STANDALONE=true

RUN bun run build

# copy production dependencies and source code into final image
FROM base AS release

COPY --from=install --chown=bun:bun /temp/prod/node_modules node_modules
COPY --from=prerelease --chown=bun:bun /usr/src/app/public ./public
COPY --from=prerelease --chown=bun:bun /usr/src/app/.next/standalone .
COPY --from=prerelease --chown=bun:bun /usr/src/app/.next/static .next/static

RUN chown bun:bun /usr/src/app

# Install wget for healthcheck (Debian-based bun image)
USER root
RUN apt-get update && apt-get install -y --no-install-recommends wget curl ca-certificates && rm -rf /var/lib/apt/lists/*

# Set environment variables from build arguments
ENV NEXT_TELEMETRY_DISABLED=1

# Healthcheck and run the app
USER bun
EXPOSE 3000/tcp
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 CMD wget -qO- http://localhost:3000/api/health || exit 1
ENTRYPOINT ["bun", "run", "server.js"]

# This Dockerfile contains the image of redis.
# TODO: This will probably contain the binaries of each component of the project
FROM redis:7.2-alpine

LABEL org.opencontainers.image.title="distq-redis"
LABEL org.opencontainers.image.description="Redis container for DistQ development"

EXPOSE 6379

CMD ["redis-server", "--appendonly", "yes"]

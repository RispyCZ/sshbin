#!/bin/bash

DOCKER_COMPOSE_CMD="docker compose -f docker-compose-prod.yaml"
echo "Stopping currently deployed containers (If they exist)"
${DOCKER_COMPOSE_CMD} down
echo "Build images"
${DOCKER_COMPOSE_CMD} build
echo "Starting containers"
${DOCKER_COMPOSE_CMD} up -d --wait --force-recreate --remove-orphans -y

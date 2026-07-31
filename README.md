# Gobbler Overview

**Gobbler Agent** is a server management component of the Gobbler telemetry suite. More speficaly, it is a native REST server application installed on a Linux host machine/VM to create and manager one or more instances of Gobbler running in Docker container(s). 

| Component | Repository | Role |
|---|---|---|
| **gobbler** | [kozwoj/gobbler](https://github.com/kozwoj/gobbler) | Server — accepts, validates, buffers, and flushes telemetry items to storage; also exposes a GQL query endpoint (`POST /gobbler/query`) over stored data |
| **gobbler-agent** | *this repo* | A Docker proxy for creating, listing, checking status and deleting Gobbler servers running in Docker conatainers. 

Gobbler Agent exposes the following REST endpoints (see docs/REST-endpoints.md for details) 

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/agent/instance` | Create + start + configure a new Gobbler instance |
| `GET` | `/agent/instances` | List all instances with URLs, volume mappings, and status |
| `GET` | `/agent/instances/{name}` | Get a single instance's details |
| `DELETE` | `/agent/instances/{name}` | Stop and remove a Gobbler container instance |
| `GET` | `/agent/status` | Agent status and state check => Docker status check|

## Host preparation

A Linux machine/VM must be prepared for hosting Gobbler instances in the followiing way
- Docker must be installed on the host and configured to expose Docker Angine REST API (see detaile below)
- Gobbler Docker image must be generated for the host's architectur, saved, and then moved to the host and loaded to Docker as gobble:latest (see detials below) 
- /gobbler directory must be created at th root, which will be then used by the Gobble servers for storing data when they run in `file` mode 
- Gobbler Agent has to be installed and started on the host - by default it will be assigned port 9000. 

## Building the agent

## Configuring Docker Angine

## Creating Gobbler and loading Gobbler image
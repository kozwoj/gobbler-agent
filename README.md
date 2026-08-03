# Gobbler Overview

**Gobbler Agent** is a server management component of the Gobbler telemetry suite. More specifically, it is a native REST server application installed on a Linux host machine/VM to create and manage one or more instances of Gobbler running in Docker container(s). 

| Component | Repository | Role |
|---|---|---|
| **gobbler** | [kozwoj/gobbler](https://github.com/kozwoj/gobbler) | Server — accepts, validates, buffers, and flushes telemetry items to storage; also exposes a GQL query endpoint (`POST /gobbler/query`) over stored data |
| **gobbler-agent** | *this repo* | A Docker proxy for creating, listing, checking status and deleting Gobbler servers running in Docker conatainers. 

Gobbler Agent exposes the following REST endpoints (see docs/REST-endpoints.md for details) 

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/agent/instance` | Create + start + configure a new Gobbler instance |
| `GET` | `/agent/instances` | List all instances with their URLs, volume mappings, and status |
| `GET` | `/agent/instances/{name}` | Get a single instance's details |
| `DELETE` | `/agent/instances/{name}` | Stop and remove a Gobbler container instance |
| `GET` | `/agent/status` | Agent status and state check => Docker status check|

## Host preparation

A Linux machine/VM must be prepared for hosting Gobbler instances in the following way
- Docker must be installed on the host and configured to expose Docker Engine REST API (see details below)
- Gobbler Docker image must be generated for the host's architecture, saved, and then moved to the host and loaded to Docker as gobble:latest (see details below) 
- `/gobbler` directory must be created at th root, which will be then used by the Gobble servers for storing data when they run in `file` mode 
- Gobbler Agent has to be installed and started on the host - by default it will be assigned port 9000. 

## Volume mappings in file mode

If Gobbler is configured to run in `file` mode the ingested data is stored on the host, not in the container. For that the Agent enforces the following convention: the configuration property `outputDir` must equal `"/gobbler/{instanceName}"`, where {instanceName} is the value of the `instanceName` configuration property.  

When starting a container the Agent 
- maps the `outputDir` to exactly the same directory on the host, hence the requirement to create `/gobbler` on the host, and
- makes the container name the same as `instanceName` 

This convention makes it easy to find the data stored by each running Gobbler instance, as it is in a subdirectories of root `/gobbler` directory named after the host instance name.  

## Building the Agent

Build the agent from source on a Linux machine of the same architecture as the intended hosts.

```
git clone https://github.com/kozwoj/gobbler-agent
cd gobbler-agent
go build -o gobbler-agent .
```

## Configuring Docker Engine

The objective is to expose Docker's REST API (the Docker Engine API) on the local port `127.0.0.1:2375`, so it can be called by a __locally running gobbler agent__. The Agent itself is called remotely, but only through its own exposed endpoints — it acts as the sole intermediary between the outside world and Docker.

The initial `/etc/docker/daemon.json` usually has the Docker API bound to `tcp://0.0.0.0:2375` — reachable from __any network interface, with no encryption and no authentication__. That is a critical security issue (OWASP broken access control vulnerability).

Since in our scenario the only intended caller is the Gobbler Agent running on the same host, there was no need to expose Docker's API to the network at all:
- The Gobbler Agent connects locally, so binding to `127.0.0.1` is sufficient
- No TLS/auth setup is needed because the traffic never leaves the box
- The attack surface is minimized — Docker's API (which is effectively root access) is only reachable from processes already on the machine, not from the network.
- The Gobbler Agent's own externally-exposed endpoints become the sole trust boundary, so they need to be tightly authenticated/validated, since they now act as a proxy to root-level power.

Configuration requires the following steps. 

1. In `/etc/docker/daemon.json` changed `hosts` to bind only to localhost, alongside the existing Unix socket:
   ```json
   {
     "hosts": [
       "unix:///var/run/docker.sock",
       "tcp://127.0.0.1:2375"
     ]
   }
   ```

2.  The default unit starts `dockerd` with `-H fd://` (socket activation), which conflicts with the `hosts` array in `daemon.json` (dockerd refuses to start with `-H` set in both places). To fixed that create a systemd override:
   ```bash
   sudo mkdir -p /etc/systemd/system/docker.service.d
   sudo tee /etc/systemd/system/docker.service.d/override.conf > /dev/null <<'EOF'
   [Service]
   ExecStart=
   ExecStart=/usr/bin/dockerd --containerd=/run/containerd/containerd.sock
   EOF
   ```

3. If the `docker.socket` is masked, which blocks `docker.service` from starting even after the fixes above (the service still depends on the socket unit) resolved it as follows:
   ```bash
   sudo systemctl unmask docker.socket
   sudo systemctl daemon-reload
   sudo systemctl restart docker
   sudo systemctl enable docker   # start on boot
   ```

4. Verify the API

```bash
sudo systemctl status docker --no-pager
sudo ss -tlnp | grep 2375
curl http://127.0.0.1:2375/version
```

## Creating Gobbler portable image

The portable image is created on a development machine using the \gobbler\Dockerfile. It is important that the image is created on a machine with host's architecture, that is, linux/x86-64 or linux/arm64. Change to the gobbler root directory and run:
```bash
docker build -t gobbler:latest .
docker save -o gobbler.tar gobbler:latest
```

## Putting it all together

- On the host create a directory /gobbler 
- Mover `gobbler-agent` executable to that directory 
- Move Gobbler portable image `gobbler.tar` to that directory 
- change to that directory
  ```bash
  cd /gobbler
  ```
- load the portable gobbler image to Docker 
  ```bash
  sudo docker load -i gobbler.tar
  ```
- start the Agent 
  ```bash 
  sudo sh -c 'nohup ./docker-agent > agent.log 2>&1 &'
  ```

- stop the Agent 
  ```bash
  pkill docker-agent
  ```

See `gobbler-agent\docs\docker_REST.http` file for examples of how to manually test Gobbler running in a Docker container on a local or remote host. 
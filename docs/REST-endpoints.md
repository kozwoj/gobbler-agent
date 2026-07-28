# Gobbler-Agent REST Commands Reference

This document is a reference for all Gobbler-Agent REST API endpoints.

By convention we will start with assumption that the Agent's port is `http://{host}:9000`. The Gobbler instances will be then assigned consecutive ports 9001, 90002, ...

One command (`POST /agent/instance`) takes JSON object as input (request body). Two commands (`GET /agent/instances/{name}` and `DELETE /agent/instances/{name}`) pass server instance name in the URL. All commands return JSON as the response.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/agent/instance` | Create + start + configure a new Gobbler instance |
| `GET` | `/agent/instances` | List all instances with URLs, volume mappings, and status |
| `GET` | `/agent/instances/{name}` | Get a single instance's details |
| `DELETE` | `/agent/instances/{name}` | Stop and remove a Gobbler container instance |
| `GET` | `/agent/status` | Agent status and state check => Docker status check|

**`POST /agent/instance`** 

**Request** is a full Gobbler config with the following schema
```json
{
  "title": "gobblerConfigurationSchema",
  "description": "Gobbler server instance name, storage and configuration",
  "type": "object",
  "properties": {
    "mode": {"type": "string", "optional": false},
    "outputDir": {"type": "string", "optional": true},
    "accountName": {"type": "string", "optional": true},
    "accountKey": {"type": "string", "optional": true},
    "writerQueueSize": {"type": "integer", "optional": false},
    "writerBatchSize": {"type": "integer", "optional": false},
    "loggerEndpoint": {"type": "string", "optional": true},
    "loggerTypes": {"type": "array", "items": {"type": "string"}, "optional": true},
    "loggerBatchSize": {"type": "integer", "optional": true},
    "loggerFlushInterval": {"type": "string", "optional": true},
    "instanceName": {"type": "string", "optional": false}
  }
}
```
**Responses:**

| Status | Body | Condition |
|---|---|---|
| 200 | `{ "instanceName": "gobbler-west-1", "url": "http://192.168.1.50:9001", "hostPort": 9001, "containerId": "a1b2c3d4e5f6", "status": "running" }` | Server created |
| 400 | `{"error": "..."}` | Malformed JSON or validation failure |
| 409 | `{"error": "..."}` | Server with that name is already running |
| 500 | `{"error": "..."}` | Internal service error e.g. Docker not reachable |


**`GET /agent/instances`** 

**Responses:**
```json
[
    { "instanceName": "gobbler-west-1", "url": "http://192.168.1.50:9001", "hostPort": 9001, "containerId": "a1b2c3d4e5f6", "status": "running" },
    { "instanceName": "gobbler-east-1", "url": "http://192.168.1.50:9002", "hostPort": 9002, "containerId": "b2c3d4e5f6a1", "status": "stopped" }
]
```
| Status | Body | Condition |
|---|---|---|
| 200 | `[{ "instanceName": "gobbler-west-1", "url": "http://192.168.1.50:9001", "hostPort": 9001, "containerId": "a1b2c3d4e5f6", "status": "running" }, ...]` | Server list |
| 500 | `{"error": "..."}` | Internal service error e.g. Docker not reachable |


**`GET /agent/instances/{name}`** 

**Responses:**
```json
{ "instanceName": "gobbler-west-1", "url": "http://192.168.1.50:9001", "hostPort": 9001, "containerId": "a1b2c3d4e5f6", "status": "running" }
```

| Status | Body | Condition |
|---|---|---|
| 200 | `{ "instanceName": "gobbler-west-1", "url": "http://192.168.1.50:9001", "hostPort": 9001, "containerId": "a1b2c3d4e5f6", "status": "running" }` | Server port and status |
| 404 | `{"error": "..."}` | Server with that name does not exist |
| 500 | `{"error": "..."}` | Internal service error e.g. Docker not reachable |


**`DELETE /agent/instances/{name}`** 

**Responses:**

| Status | Body | Condition |
|---|---|---|
| 204 | `{}` |Server successfully removed |
| 409 | `{"error": "..."}` | Server with that name does not exist |
| 500 | `{"error": "..."}` | Internal service error e.g. Docker not reachable |




**`GET /agent/status`** 

**Responses:** - the list of returned properties is still TBD

```json
{
  "ID": "7TRN:IPZB:QYBB:VPBQ:UMPP:KARE:6ZNR:XE6T:7EWV:PKF4:ZOJD:TPYS",
  "Containers": 14,
  "ContainersRunning": 3,
  "ContainersPaused": 1,
  "ContainersStopped": 10,
  "...": "..."
}
```

| Status | Body | Condition |
|---|---|---|
| 204 | `{... selected properties of Docker endpoint GET /info ...}` |Server removed |
| 500 | `{"error": "..."}` | Internal service error e.g. Docker not reachable |

```json
{
  "ID": "7TRN:IPZB:QYBB:VPBQ:UMPP:KARE:6ZNR:XE6T:7EWV:PKF4:ZOJD:TPYS",
  "Containers": 14,
  "ContainersRunning": 3,
  "ContainersPaused": 1,
  "ContainersStopped": 10,
  "...": "..."
}
```
# Realtime Chat System (Go)

A **production-grade, real-time chat backend** built in **Go**, designed to support **10k+ concurrent WebSocket clients**, **strict message ordering**, **anonymous participation**, and **first-class observability**.

This project focuses on **correctness, scalability, and debuggability** rather than feature sprawl. It demonstrates how to design a **horizontally scalable WebSocket system** using a **distributed monolith** runtime architecture and **hexagonal architecture** code architecture, without prematurely splitting into microservices.

---

## What This System Is

This is a **real-time WebSocket chat server** that provides:

- Authenticated user access (OTP via Twilio)
- Anonymous chat participation via per-conversation sender IDs
- Room-based conversations
- Guaranteed message ordering per conversation
- Durable message persistence
- Presence tracking (Redis + PostgreSQL)
- Horizontal scalability
- Full observability (logs, metrics, traces)

The system is **stateless at the application layer** and externalizes all state to Redis and PostgreSQL.

---

## Core Design Principles

- **Runtime Distributed monolith**
  - Single deployable service
  - Scales horizontally behind a load balancer
- **Event-driven**
  - Redis Streams for message ingestion
- **Externally stateful**
  - Redis → presence, streams
  - PostgreSQL → users, conversations, participants, messages
- **Hexagonal architecture**
  - Business logic isolated from infrastructure
  - Deterministic and unit-testable core

---

## High-Level Architecture


```
                                            Clients
                                            │
                                            │ WebSocket
                                            ▼
                                    +-----------------------+
                                    | Chat Server (Go)      |
                                    |-----------------------|
                                    | ChatService           |
                                    | WebSocket Adapter     |   <- horizontally scaled
                                    | Redis Pub/Sub Adapter |
                                    | PostgreSQL Adapter    |
                                    +-----------------------+
                                    │             │
                                    ▼             ▼
                                    Redis       PostgreSQL
```

---

### Code Architecture (Hexagonal)

The system follows **hexagonal architecture**, isolating business logic from infrastructure.

```
internal/
  ├── app/          
  ├── core/             
  ├── plugins/           
  ├── config/

app/
  ├── registry/          
  ├── server/             
  ├── worker/           

core/
  ├── contracts/          
  ├── domain/             
  ├── services/           

plugins/
  ├── postgres/          
  ├── redis/             
  ├── twilio/           

config/
```

---

### Why a Distributed Monolith?

- WebSockets benefit from locality
- Lower latency than service-to-service hops
- Easier debugging and deployment
- Horizontally scalable
- Can be split later **without rewriting core logic**

---

## Authentication & Identity Model

### User Identity

- Users authenticate via **OTP (Twilio Verify)**
- Identity is **phone-number based**
- Stored durably in PostgreSQL

```go
type User struct {
    ID        string // phone number
    CreatedAt time.Time
}
```
---

## Anonymous Participation Model

Users are **never exposed directly** in conversations.

Each conversation assigns a **participant (sender) ID**:

```go
type Participant struct {
    ID             uuid.UUID // public sender_id
    ConversationID uuid.UUID
    UserID         string
    JoinedAt       time.Time
    LastSeenAt     time.Time
    LeftAt         *time.Time
}
```

This enables:

* True sender anonymity
* Multiple identities per user across conversations
* Rejoin without identity leakage

---

## Session Rejoin Semantics

When a user joins a conversation:

* If they rejoin within a configurable **rejoin window** (e.g. 5 minutes):

  * The same `participant_id` is reused
* Otherwise:

  * A new anonymous participant is created

This guarantees:

* Stable identity across brief disconnects
* Clean identity rotation after longer gaps

---

## Conversation Model

All chats are modeled uniformly as **conversations**.

```go
type Conversation struct {
    ID        uuid.UUID
    CreatedAt time.Time
}
```

* No storage-level distinction between “room” or “direct”
* Single message pipeline
* Simplified authorization and scaling

---

## WebSocket Lifecycle

1. User authenticates via OTP
2. Client calls:
  - `POST /auth/register` with payload `{"phone": "+91XXXXXXXXXX"}`, 
  - `POST /auth/verify` with payload `{"phone": "+91XXXXXXXXXX", "code": "XXXXXX"}`
  - `ws://localhost:8080/ws?conv_id=XXXX` with Header `Authorization: Bearer Token`
3. Server:
   * Authenticates user
   * Ensures participant session (transactional)
   * Updates presence
4. WebSocket upgrade occurs **after commit**
5. Client is registered with in-memory hub

This ordering prevents:

* Ghost sessions
* Presence leaks
* Orphaned connections

---

## Message Protocol

### Client → Server

```json
{
  "type": "message.send",
  "client_msg_id": "uuid",
  "payload": "hello world"
}
```

### Server Acknowledgements (Important)

The server sends **two distinct acknowledgements** to the sender.

#### 1. `server_received`

Sent when the server **accepts the message into the pipeline**
(queued into Redis Streams).

```json
{
  "type": "ack",
  "client_msg_id": "uuid",
  "status": "server_received",
  "timestamp": "time",
}
```

Guarantees:

* WebSocket connection is alive
* Message accepted for processing
* Not yet persisted or delivered

#### 2. `sent_ack`

Sent when the message is **fully processed**:

* Persisted in PostgreSQL
* Assigned a sequence number
* Published to conversation subscribers

```json
{
  "type": "ack",
  "client_msg_id": "uuid",
  "status": "persisted",
  "seq": 3,
  "timestamp": "time",
}
```

Guarantees:

* Durable persistence
* Global ordering
* Successful fan-out

This dual-ACK model prevents:

* Client-side message loss
* False delivery assumptions
* Ambiguous retry behavior

---

## Message Flow (End-to-End)

1. Client sends message over WebSocket
2. Server enqueues message into **Redis Streams**
3. Redis Stream worker:

   * Reads via consumer group
   * Starts DB transaction
   * Atomically increments per-conversation sequence
   * Persists message
   * Passes message to registry
4. Registry delivers message to end users
   * Registry broadcasts message into conversation and sends `persisted` ack to sender
   * Acknowledges Redis stream entry
   * Deletes Redis stream entry

---

## Ordering Guarantee

Ordering is enforced by a **per-conversation sequence row**.

```sql
UPDATE conversation_sequences
SET last_seq = last_seq + 1
WHERE conversation_id = $1
RETURNING last_seq;
```

* One sequence per conversation
* Atomic inside a transaction
* Independent of WebSocket node count

---

## Presence Model

### Fast Path (Redis)

* Heartbeat every ~30s
* TTL-based keys

```
presence:{conversation_id} TIMESTAMP {sender_id}
```

### Durable Path (PostgreSQL)

* `last_seen_at` updated periodically
* Final update on disconnect

This balances:

* Accuracy
* Database load
* Crash safety

---

## Observability (First-Class)

### Logs

* Structured
* Correlated
* Context-rich (user, participant, conversation)

### Metrics

* Active WebSocket connections
* Messages/sec
* Worker throughput
* Error rates

### Tracing

* WebSocket join
* Message enqueue
* Redis worker processing
* Database transactions

Stack:

* OpenTelemetry
* Prometheus
* Loki
* Tempo
* Grafana

---


## Scalability Characteristics

* Horizontally scalable WebSocket servers
* Redis Streams absorb bursts
* Stateless application layer
* Safe for 10k+ concurrent clients per cluster
* Explicit backpressure handling

---

## Tech Stack

* **Go** – concurrency & performance
* **WebSockets** – real-time communication
* **Redis** – Pub/Sub & presence
* **PostgreSQL** – durable storage
* **k6** – load testing
* **Prometheus / OpenTelemetry** – observability

---

## What’s Implemented

* OTP authentication
* Conversation lifecycle
* Anonymous participants
* WebSocket hub
* Redis Streams worker
* Message persistence
* Presence tracking
* Observability stack

---

## What’s Intentionally Out of Scope

* End-to-end encryption
* Read receipts
* Typing indicators
* Message search APIs
* Frontend clients

---

## Status

🚧 Actively evolving

Focus areas:

* Architectural correctness
* Ordering guarantees
* Failure handling
* Observability clarity

Feature completeness is intentionally secondary.

---

## License

[MIT](./LICENSE)

---
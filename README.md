
# Live Location Map

A small real-time location tracking app: open `/` to see an OpenStreetMap view, connect over WebSocket, and watch all users’ live positions update on the map.

Built to handle **high concurrency** and **heavy Redis write load** safely.

---

## What it does

- Serves an **OpenStreetMap UI** at `/`
- Opens a **WebSocket** connection and streams location updates
- Stores locations in **Redis GEO** (geohash-based structure)
- Broadcasts all active users’ latest locations to connected clients
- Handles disconnects cleanly (no goroutine leaks)

---

## Stack

- Go `1.25`
- Gorilla WebSocket
- Redis (GEO + geohashing)
- Lua scripts (faster chained Redis ops)
- Writer queue (smooth Redis writes under load)
- Load testing CLI (simulate thousands of clients)

---

## Quick start

### 1) Run Redis
```bash
docker run -p 6379:6379 redis:7
```

### 2) Run Server
```
go run ./app/server
```
### 3) Open LiveTrace 
```
navigate to http://localhost:8080/
```

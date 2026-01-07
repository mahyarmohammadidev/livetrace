function getOrCreateUserId() {
    let id = localStorage.getItem("userId");
    if (!id) {
        id = crypto.randomUUID();
        localStorage.setItem("userId", id);
    }
    return id;
}

const userId = getOrCreateUserId();
const statusEl = document.getElementById("status");

// Map
const map = L.map("map").setView([51.505, -0.09], 13);

L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
    maxZoom: 19,
    attribution: '&copy; OpenStreetMap contributors',
}).addTo(map);

const markers = new Map();
let centeredOnce = false;

function updateMarker(uid, lat, lng) {
    const isMe = uid === userId;
    let marker = markers.get(uid);

    if (!marker) {
        marker = L.marker([lat, lng]).addTo(map);
        marker.bindPopup(isMe ? "You" : uid);
        markers.set(uid, marker);
    } else {
        marker.setLatLng([lat, lng]);
    }

    if (isMe && !centeredOnce) {
        centeredOnce = true;
        map.setView([lat, lng], 15);
    }
}

let ws;
function connectWS() {
    const protocol = location.protocol === "https:" ? "wss" : "ws";
    const url = `${protocol}://${location.host}/ws?userId=${encodeURIComponent(userId)}`;

    ws = new WebSocket(url);

    ws.onopen = () => {
        statusEl.textContent = "🟢 Connected";
    };

    ws.onclose = () => {
        statusEl.textContent = "🔴 Disconnected (retrying)";
        setTimeout(connectWS, 1000);
    };

    ws.onerror = () => {
        statusEl.textContent = "⚠️ WS Error";
    };

    ws.onmessage = (event) => {
        // server may batch messages separated by newline
        const lines = event.data.split("\n");
        for (const line of lines) {
            if (!line.trim()) continue;

            try {
                const msg = JSON.parse(line);
                if (msg.type === "location") {
                    updateMarker(msg.userId, msg.lat, msg.lng);
                }
            } catch {}
        }
    };
}

connectWS();

if (!navigator.geolocation) {
    alert("Geolocation not supported in this browser.");
} else {
    navigator.geolocation.watchPosition(
        (pos) => {
            const lat = pos.coords.latitude;
            const lng = pos.coords.longitude;
            const accuracy = pos.coords.accuracy || 0;
            const ts = Math.floor(Date.now() / 1000);

            // Update immediately locally
            updateMarker(userId, lat, lng);

            // Send to server
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: "location",
                    userId,
                    lat,
                    lng,
                    accuracy,
                    ts
                }));
            }
        },
        (err) => {
            console.log("Geolocation error:", err);
            const codes = {
                1: "PERMISSION_DENIED",
                2: "POSITION_UNAVAILABLE",
                3: "TIMEOUT",
            };
            const reason = codes[err.code] || "UNKNOWN_ERROR";
            statusEl.textContent = `⚠️ Location error: ${reason} - ${err.message}`;
        },
        {
            enableHighAccuracy: true,
            maximumAge: 5000,
            timeout: 15000,
        }
    );
}

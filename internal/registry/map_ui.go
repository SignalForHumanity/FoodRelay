package registry

import (
	"net/http"
)

// MapUI handles GET /map — a Leaflet/OpenStreetMap page showing all active nodes.
// Fetches /v1/nodes (same origin) and renders a marker + coverage circle per node.
func (h *Handler) MapUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(mapHTML)) //nolint:errcheck
}

const mapHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8"/>
  <title>FoodRelay Nodes</title>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"/>
  <link rel="stylesheet" href="https://unpkg.com/leaflet.markercluster@1.5.3/dist/MarkerCluster.css"/>
  <link rel="stylesheet" href="https://unpkg.com/leaflet.markercluster@1.5.3/dist/MarkerCluster.Default.css"/>
  <style>
    html, body, #map { height: 100%; margin: 0; }
    .popup { font-family: system-ui, sans-serif; min-width: 180px; }
    .popup h3 { margin: 0 0 6px 0; font-size: 15px; }
    .popup .row { margin: 3px 0; font-size: 13px; }
    .popup .hint { margin-top: 10px; font-size: 12px; color: #555; }
    .pill { display:inline-block; padding:2px 8px; border-radius:999px; font-size:11px; font-weight:600; }
    .pill-active { background:#d1fae5; color:#065f46; }
    .pill-stale  { background:#fee2e2; color:#991b1b; }
  </style>
</head>
<body>
<div id="map"></div>

<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<script src="https://unpkg.com/leaflet.markercluster@1.5.3/dist/leaflet.markercluster.js"></script>
<script>
  const map = L.map('map').setView([20, 0], 2);

  L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
  }).addTo(map);

  const cluster = L.markerClusterGroup();
  map.addLayer(cluster);

  function fmtAge(s) {
    if (s < 60)        return s + 's ago';
    if (s < 3600)      return Math.floor(s/60) + 'm ago';
    if (s < 172800)    return Math.floor(s/3600) + 'h ago';
    return Math.floor(s/86400) + 'd ago';
  }

  fetch('/v1/nodes')
    .then(r => r.json())
    .then(nodes => {
      const nowSec = Math.floor(Date.now() / 1000);

      nodes.forEach(n => {
        if (!n.lat || !n.lon) return; // skip nodes with no geo

        const lastSeenSec = n.last_seen ? Math.floor(new Date(n.last_seen).getTime() / 1000) : 0;
        const age    = lastSeenSec ? (nowSec - lastSeenSec) : 999999;
        const active = age <= 900; // 15 min

        const statusPill = active
          ? '<span class="pill pill-active">active</span>'
          : '<span class="pill pill-stale">stale</span>';

        const radiusKm = n.coverage_radius_km || 0;
        const radiusRow = radiusKm
          ? '<div class="row">\uD83D\uDCCF Radius: <b>' + radiusKm + ' km</b></div>'
          : '';

        const popup =
          '<div class="popup">' +
            '<h3>' + escHtml(n.node_id) + '</h3>' +
            '<div class="row">\uD83D\uDCDE <b>' + escHtml(n.public_phone) + '</b></div>' +
            radiusRow +
            '<div class="row" style="margin-top:6px;">' +
              (lastSeenSec ? 'Last seen: ' + fmtAge(age) + ' ' : 'Never seen ') +
              statusPill +
            '</div>' +
            '<div class="hint">Text <b>FOOD</b> to ' + escHtml(n.public_phone) + ' to get started.</div>' +
          '</div>';

        const marker = L.marker([n.lat, n.lon]).bindPopup(popup);
        cluster.addLayer(marker);

        if (radiusKm) {
          L.circle([n.lat, n.lon], {
            radius: radiusKm * 1000,
            color: active ? '#059669' : '#dc2626',
            weight: 1,
            fillOpacity: 0.06
          }).addTo(map);
        }
      });

      // If we got nodes with valid coords, fit the map to them
      const validCoords = nodes.filter(n => n.lat && n.lon).map(n => [n.lat, n.lon]);
      if (validCoords.length > 0) {
        map.fitBounds(validCoords, { padding: [40, 40], maxZoom: 10 });
      }
    })
    .catch(err => {
      console.error('Failed to load nodes:', err);
    });

  function escHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }
</script>
</body>
</html>`

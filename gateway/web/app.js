/* ============================================================
   ShelterNav — Web Application
   ============================================================ */

'use strict';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const OKC_CENTER = { lat: 35.4676, lon: -97.5164 };
const API_TIMEOUT = 2000;
const SEARCH_DEBOUNCE = 200;

const STATUS = { CLOSED: 0, OPEN: 1, FULL: 2 };
const TYPE = { EMERGENCY: 0, OVERNIGHT: 1, LONG_TERM: 2 };

const STATUS_LABELS = { [STATUS.CLOSED]: 'Closed', [STATUS.OPEN]: 'Open', [STATUS.FULL]: 'Full' };
const TYPE_LABELS = { [TYPE.EMERGENCY]: 'Emergency', [TYPE.OVERNIGHT]: 'Overnight', [TYPE.LONG_TERM]: 'Long-term' };

const STATUS_COLORS = {
    [STATUS.CLOSED]: '#F87171',
    [STATUS.OPEN]:   '#34D399',
    [STATUS.FULL]:   '#FBBF24',
};

// ---------------------------------------------------------------------------
// Mock Data
// ---------------------------------------------------------------------------

const MOCK_SHELTERS = [
    {
        id: 1, name: 'Hope Community Shelter', lat: 35.4732, lon: -97.5185,
        type: TYPE.EMERGENCY, capacity: 50, occupancy: 34, status: STATUS.OPEN,
        address: '123 N Robinson Ave, OKC', distance_m: 420,
    },
    {
        id: 2, name: 'Riverside Safe Haven', lat: 35.4598, lon: -97.5063,
        type: TYPE.OVERNIGHT, capacity: 50, occupancy: 47, status: STATUS.FULL,
        address: '456 SE River Blvd, OKC', distance_m: 1240,
    },
    {
        id: 3, name: 'Downtown Emergency Center', lat: 35.4710, lon: -97.5230,
        type: TYPE.EMERGENCY, capacity: 120, occupancy: 78, status: STATUS.OPEN,
        address: '789 W Main St, OKC', distance_m: 650,
    },
    {
        id: 4, name: 'Midtown Rest House', lat: 35.4805, lon: -97.5140,
        type: TYPE.LONG_TERM, capacity: 30, occupancy: 30, status: STATUS.FULL,
        address: '321 NW 10th St, OKC', distance_m: 1800,
    },
    {
        id: 5, name: 'Southside Family Shelter', lat: 35.4520, lon: -97.5295,
        type: TYPE.OVERNIGHT, capacity: 80, occupancy: 42, status: STATUS.OPEN,
        address: '555 SW 29th St, OKC', distance_m: 2300,
    },
    {
        id: 6, name: 'Veterans Haven OKC', lat: 35.4680, lon: -97.4980,
        type: TYPE.LONG_TERM, capacity: 40, occupancy: 38, status: STATUS.OPEN,
        address: '900 NE 4th St, OKC', distance_m: 1650,
    },
    {
        id: 7, name: 'Grace Refuge Center', lat: 35.4765, lon: -97.5340,
        type: TYPE.EMERGENCY, capacity: 60, occupancy: 60, status: STATUS.FULL,
        address: '234 NW Classen Blvd, OKC', distance_m: 1100,
    },
    {
        id: 8, name: 'New Day Recovery Shelter', lat: 35.4620, lon: -97.5400,
        type: TYPE.LONG_TERM, capacity: 25, occupancy: 12, status: STATUS.OPEN,
        address: '678 SW 15th St, OKC', distance_m: 2800,
    },
    {
        id: 9, name: 'Capitol Hill Community Home', lat: 35.4440, lon: -97.5120,
        type: TYPE.OVERNIGHT, capacity: 45, occupancy: 0, status: STATUS.CLOSED,
        address: '1200 S Walker Ave, OKC', distance_m: 3100,
    },
    {
        id: 10, name: 'Bricktown Relief Station', lat: 35.4660, lon: -97.5080,
        type: TYPE.EMERGENCY, capacity: 90, occupancy: 65, status: STATUS.OPEN,
        address: '100 E Sheridan Ave, OKC', distance_m: 900,
    },
];

const MOCK_ROUTE = {
    path: [
        { lat: 35.4676, lon: -97.5164 },
        { lat: 35.4678, lon: -97.5170 },
        { lat: 35.4682, lon: -97.5175 },
        { lat: 35.4688, lon: -97.5178 },
        { lat: 35.4695, lon: -97.5180 },
        { lat: 35.4700, lon: -97.5182 },
        { lat: 35.4705, lon: -97.5183 },
        { lat: 35.4708, lon: -97.5184 },
        { lat: 35.4712, lon: -97.5185 },
        { lat: 35.4715, lon: -97.5186 },
        { lat: 35.4718, lon: -97.5185 },
        { lat: 35.4720, lon: -97.5184 },
        { lat: 35.4722, lon: -97.5185 },
        { lat: 35.4725, lon: -97.5185 },
        { lat: 35.4728, lon: -97.5185 },
        { lat: 35.4730, lon: -97.5185 },
        { lat: 35.4732, lon: -97.5185 },
    ],
    total_distance_m: 420,
    estimated_seconds: 360,
    maneuvers: [
        { point: { lat: 35.4676, lon: -97.5164 }, instruction: 'Head north on N Robinson Ave', distance_m: 150 },
        { point: { lat: 35.4700, lon: -97.5182 }, instruction: 'Turn left on NW 3rd St', distance_m: 120 },
        { point: { lat: 35.4715, lon: -97.5186 }, instruction: 'Continue straight', distance_m: 100 },
        { point: { lat: 35.4732, lon: -97.5185 }, instruction: 'Arrive at Hope Community Shelter', distance_m: 50 },
    ],
    query_ms: 2.5,
};

// ---------------------------------------------------------------------------
// API Client
// ---------------------------------------------------------------------------

class ApiClient {
    constructor(baseUrl) {
        this.baseUrl = baseUrl || '';
        this.online = false;
    }

    async _fetch(path, params, timeout) {
        const url = new URL(path, this.baseUrl || window.location.origin);
        for (const [k, v] of Object.entries(params || {})) {
            if (v !== undefined && v !== null) url.searchParams.set(k, String(v));
        }

        const controller = new AbortController();
        const timer = setTimeout(() => controller.abort(), timeout || API_TIMEOUT);

        try {
            const res = await fetch(url.toString(), { signal: controller.signal });
            clearTimeout(timer);
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            return await res.json();
        } catch (e) {
            clearTimeout(timer);
            throw e;
        }
    }

    async checkHealth() {
        try {
            const data = await this._fetch('/healthz', null, API_TIMEOUT);
            this.online = data && data.status === 'ok';
        } catch {
            this.online = false;
        }
        return this.online;
    }

    async findNearest(lat, lon, radius, limit) {
        if (!this.online) return { shelters: MOCK_SHELTERS, query_ms: 0 };
        try {
            return await this._fetch('/v1/shelters/nearest', { lat, lon, radius, limit });
        } catch {
            return { shelters: MOCK_SHELTERS, query_ms: 0 };
        }
    }

    async getRoute(startLat, startLon, endLat, endLon) {
        if (!this.online) return MOCK_ROUTE;
        try {
            return await this._fetch('/v1/route', {
                start_lat: startLat,
                start_lon: startLon,
                end_lat: endLat,
                end_lon: endLon,
            });
        } catch {
            return MOCK_ROUTE;
        }
    }
}

// ---------------------------------------------------------------------------
// Map Manager
// ---------------------------------------------------------------------------

class MapManager {
    constructor(containerId) {
        this.map = null;
        this.containerId = containerId;
        this.userMarker = null;
        this.userLat = null;
        this.userLon = null;
        this.onShelterClick = null;
        this.onClusterClick = null;
    }

    init(center) {
        this.map = new maplibregl.Map({
            container: this.containerId,
            style: {
                version: 8,
                sources: {
                    osm: {
                        type: 'raster',
                        tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
                        tileSize: 256,
                        attribution: '&copy; OpenStreetMap contributors',
                    },
                },
                layers: [
                    {
                        id: 'background',
                        type: 'background',
                        paint: { 'background-color': '#0F1117' },
                    },
                    {
                        id: 'osm-tiles',
                        type: 'raster',
                        source: 'osm',
                        paint: {
                            'raster-brightness-max': 0.3,
                            'raster-saturation': -0.8,
                            'raster-contrast': 0.1,
                        },
                    },
                ],
            },
            center: [center.lon, center.lat],
            zoom: 13,
            maxZoom: 18,
            minZoom: 4,
            attributionControl: false,
        });

        this.map.addControl(new maplibregl.AttributionControl({ compact: true }), 'bottom-left');

        return new Promise((resolve) => {
            this.map.on('load', () => resolve());
        });
    }

    addShelterSource(shelters) {
        const features = shelters.map((s) => ({
            type: 'Feature',
            properties: {
                id: s.id,
                name: s.name,
                status: s.status,
                type: s.type,
                capacity: s.capacity,
                occupancy: s.occupancy,
                address: s.address,
                distance_m: s.distance_m,
            },
            geometry: { type: 'Point', coordinates: [s.lon, s.lat] },
        }));

        const geojson = { type: 'FeatureCollection', features };

        if (this.map.getSource('shelters')) {
            this.map.getSource('shelters').setData(geojson);
            return;
        }

        this.map.addSource('shelters', {
            type: 'geojson',
            data: geojson,
            cluster: true,
            clusterMaxZoom: 14,
            clusterRadius: 50,
        });

        // Cluster circles
        this.map.addLayer({
            id: 'clusters',
            type: 'circle',
            source: 'shelters',
            filter: ['has', 'point_count'],
            paint: {
                'circle-color': '#242836',
                'circle-radius': ['step', ['get', 'point_count'], 20, 5, 26, 10, 32],
                'circle-stroke-width': 2,
                'circle-stroke-color': '#60A5FA',
                'circle-opacity': 0.9,
            },
        });

        // Cluster count labels
        this.map.addLayer({
            id: 'cluster-count',
            type: 'symbol',
            source: 'shelters',
            filter: ['has', 'point_count'],
            layout: {
                'text-field': '{point_count_abbreviated}',
                'text-size': 13,
                'text-font': ['Open Sans Bold', 'Arial Unicode MS Bold'],
            },
            paint: { 'text-color': '#F1F5F9' },
        });

        // Individual shelter pins
        this.map.addLayer({
            id: 'shelter-pins',
            type: 'circle',
            source: 'shelters',
            filter: ['!', ['has', 'point_count']],
            paint: {
                'circle-color': [
                    'match', ['get', 'status'],
                    STATUS.OPEN, '#34D399',
                    STATUS.FULL, '#FBBF24',
                    STATUS.CLOSED, '#F87171',
                    '#94A3B8',
                ],
                'circle-radius': 8,
                'circle-stroke-width': 2.5,
                'circle-stroke-color': '#ffffff',
                'circle-opacity': 0.95,
            },
        });

        // Click: shelter pin
        this.map.on('click', 'shelter-pins', (e) => {
            if (!e.features || !e.features.length) return;
            const props = e.features[0].properties;
            const coords = e.features[0].geometry.coordinates;
            if (this.onShelterClick) this.onShelterClick(props, coords);
        });

        // Click: cluster -> zoom in
        this.map.on('click', 'clusters', (e) => {
            const features = this.map.queryRenderedFeatures(e.point, { layers: ['clusters'] });
            if (!features.length) return;
            const clusterId = features[0].properties.cluster_id;
            this.map.getSource('shelters').getClusterExpansionZoom(clusterId, (err, zoom) => {
                if (err) return;
                this.map.flyTo({
                    center: features[0].geometry.coordinates,
                    zoom: zoom,
                    duration: 500,
                });
            });
        });

        // Cursor changes
        this.map.on('mouseenter', 'shelter-pins', () => {
            this.map.getCanvas().style.cursor = 'pointer';
        });
        this.map.on('mouseleave', 'shelter-pins', () => {
            this.map.getCanvas().style.cursor = '';
        });
        this.map.on('mouseenter', 'clusters', () => {
            this.map.getCanvas().style.cursor = 'pointer';
        });
        this.map.on('mouseleave', 'clusters', () => {
            this.map.getCanvas().style.cursor = '';
        });
    }

    setUserLocation(lat, lon) {
        this.userLat = lat;
        this.userLon = lon;

        if (this.userMarker) {
            this.userMarker.setLngLat([lon, lat]);
            return;
        }

        const el = document.createElement('div');
        el.className = 'user-marker';
        el.innerHTML = '<div class="user-marker-pulse"></div><div class="user-marker-dot"></div>';

        this.userMarker = new maplibregl.Marker({ element: el, anchor: 'center' })
            .setLngLat([lon, lat])
            .addTo(this.map);
    }

    flyTo(lat, lon, zoom) {
        this.map.flyTo({
            center: [lon, lat],
            zoom: zoom || 15,
            duration: 600,
            essential: true,
        });
    }

    flyToUser() {
        if (this.userLat !== null && this.userLon !== null) {
            this.flyTo(this.userLat, this.userLon, 15);
        }
    }

    drawRoute(routeData) {
        const coords = routeData.path.map((p) => [p.lon, p.lat]);

        if (this.map.getSource('route')) {
            this.map.getSource('route').setData({
                type: 'Feature',
                geometry: { type: 'LineString', coordinates: coords },
            });
        } else {
            this.map.addSource('route', {
                type: 'geojson',
                data: {
                    type: 'Feature',
                    geometry: { type: 'LineString', coordinates: coords },
                },
            });

            // Route glow (behind)
            this.map.addLayer({
                id: 'route-glow',
                type: 'line',
                source: 'route',
                layout: { 'line-join': 'round', 'line-cap': 'round' },
                paint: {
                    'line-color': '#60A5FA',
                    'line-width': 10,
                    'line-opacity': 0.2,
                    'line-blur': 6,
                },
            }, 'shelter-pins');

            // Route line
            this.map.addLayer({
                id: 'route-line',
                type: 'line',
                source: 'route',
                layout: { 'line-join': 'round', 'line-cap': 'round' },
                paint: {
                    'line-color': '#60A5FA',
                    'line-width': 4,
                    'line-opacity': 0.9,
                },
            }, 'shelter-pins');
        }

        // Fit route bounds
        const bounds = coords.reduce(
            (b, c) => b.extend(c),
            new maplibregl.LngLatBounds(coords[0], coords[0])
        );

        this.map.fitBounds(bounds, {
            padding: { top: 120, bottom: 200, left: 60, right: 60 },
            pitch: 45,
            duration: 800,
        });
    }

    clearRoute() {
        if (this.map.getLayer('route-line')) this.map.removeLayer('route-line');
        if (this.map.getLayer('route-glow')) this.map.removeLayer('route-glow');
        if (this.map.getSource('route')) this.map.removeSource('route');
        this.map.setPitch(0);
    }

    highlightShelter(shelterId) {
        this.map.setPaintProperty('shelter-pins', 'circle-stroke-color', [
            'case',
            ['==', ['get', 'id'], shelterId],
            '#60A5FA',
            '#ffffff',
        ]);
        this.map.setPaintProperty('shelter-pins', 'circle-radius', [
            'case',
            ['==', ['get', 'id'], shelterId],
            11,
            8,
        ]);
    }

    clearHighlight() {
        if (this.map.getLayer('shelter-pins')) {
            this.map.setPaintProperty('shelter-pins', 'circle-stroke-color', '#ffffff');
            this.map.setPaintProperty('shelter-pins', 'circle-radius', 8);
        }
    }
}

// ---------------------------------------------------------------------------
// Panel Controller
// ---------------------------------------------------------------------------

class PanelController {
    constructor() {
        this.panel = document.getElementById('shelter-panel');
        this.handle = document.getElementById('panel-drag-handle');
        this.list = document.getElementById('shelter-list');
        this.summary = document.getElementById('panel-summary');
        this.fabContainer = document.getElementById('fab-container');
        this.state = 'peek'; // peek | half | full
        this.shelters = [];
        this.selectedId = null;
        this.onNavigate = null;
        this.onCardClick = null;

        this._initDrag();
    }

    _initDrag() {
        let startY = 0;
        let startTransform = 0;
        let dragging = false;

        const getTranslateY = () => {
            const style = getComputedStyle(this.panel);
            const matrix = new DOMMatrix(style.transform);
            return matrix.m42;
        };

        const onStart = (e) => {
            // Don't drag on desktop
            if (window.innerWidth >= 1024) return;

            dragging = true;
            startY = e.touches ? e.touches[0].clientY : e.clientY;
            startTransform = getTranslateY();
            this.panel.style.transition = 'none';
        };

        const onMove = (e) => {
            if (!dragging) return;
            const clientY = e.touches ? e.touches[0].clientY : e.clientY;
            const dy = clientY - startY;
            const newY = Math.max(0, startTransform + dy);
            this.panel.style.transform = `translateY(${newY}px)`;
        };

        const onEnd = (e) => {
            if (!dragging) return;
            dragging = false;
            this.panel.style.transition = '';

            const clientY = e.changedTouches ? e.changedTouches[0].clientY : e.clientY;
            const dy = clientY - startY;
            const vh = window.innerHeight;

            // Determine snap state based on drag direction and distance
            if (dy < -80) {
                // Swiped up
                this.setState(this.state === 'peek' ? 'half' : 'full');
            } else if (dy > 80) {
                // Swiped down
                this.setState(this.state === 'full' ? 'half' : 'peek');
            } else {
                // Snap back to current
                this.setState(this.state);
            }
        };

        this.handle.addEventListener('touchstart', onStart, { passive: true });
        this.handle.addEventListener('mousedown', onStart);
        window.addEventListener('touchmove', onMove, { passive: false });
        window.addEventListener('mousemove', onMove);
        window.addEventListener('touchend', onEnd);
        window.addEventListener('mouseup', onEnd);
    }

    setState(state) {
        this.state = state;
        this.panel.classList.remove('state-half', 'state-full');
        this.panel.style.transform = '';

        if (state === 'half') {
            this.panel.classList.add('state-half');
        } else if (state === 'full') {
            this.panel.classList.add('state-full');
        }

        // Adjust FAB position on mobile
        if (window.innerWidth < 768) {
            if (state === 'peek') {
                this.fabContainer.style.bottom = 'calc(var(--panel-peek) + 24px)';
            } else if (state === 'half') {
                this.fabContainer.style.bottom = 'calc(var(--panel-half) + 16px)';
            } else {
                this.fabContainer.style.bottom = 'calc(var(--panel-full) + 16px)';
            }
        }
    }

    renderShelters(shelters) {
        this.shelters = shelters;
        this.list.innerHTML = '';

        if (!shelters || shelters.length === 0) {
            this._renderEmpty();
            return;
        }

        const openCount = shelters.filter((s) => s.status === STATUS.OPEN).length;
        const closestDist = Math.min(...shelters.map((s) => s.distance_m));
        this.summary.textContent = `${openCount} open shelter${openCount !== 1 ? 's' : ''} nearby \u00B7 Closest ${formatDistance(closestDist)}`;

        shelters.forEach((shelter) => {
            const card = this._createCard(shelter);
            this.list.appendChild(card);
        });
    }

    _renderEmpty() {
        this.summary.textContent = 'No shelters found';

        const empty = document.createElement('div');
        empty.className = 'empty-state';

        const icon = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        icon.setAttribute('width', '48');
        icon.setAttribute('height', '48');
        icon.setAttribute('viewBox', '0 0 24 24');
        icon.setAttribute('fill', 'none');
        icon.setAttribute('stroke', 'currentColor');
        icon.setAttribute('stroke-width', '1.5');
        icon.classList.add('empty-state-icon');
        const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        path.setAttribute('d', 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-4 0a1 1 0 01-1-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 01-1 1');
        icon.appendChild(path);

        const title = document.createElement('div');
        title.className = 'empty-state-title';
        title.textContent = 'No shelters found';

        const desc = document.createElement('div');
        desc.className = 'empty-state-desc';
        desc.textContent = 'Try expanding your search area or adjusting filters.';

        empty.appendChild(icon);
        empty.appendChild(title);
        empty.appendChild(desc);
        this.list.appendChild(empty);
    }

    _createCard(shelter) {
        const card = document.createElement('div');
        card.className = 'shelter-card';
        card.dataset.id = shelter.id;

        if (shelter.id === this.selectedId) card.classList.add('selected');

        // Header row
        const header = document.createElement('div');
        header.className = 'card-header';

        const titleGroup = document.createElement('div');
        titleGroup.className = 'card-title-group';

        const dot = document.createElement('div');
        dot.className = 'status-indicator';
        if (shelter.status === STATUS.OPEN) dot.classList.add('status-open');
        else if (shelter.status === STATUS.FULL) dot.classList.add('status-full');
        else dot.classList.add('status-closed');

        const name = document.createElement('div');
        name.className = 'card-name';
        name.textContent = shelter.name;

        titleGroup.appendChild(dot);
        titleGroup.appendChild(name);

        const distance = document.createElement('div');
        distance.className = 'card-distance';
        distance.textContent = formatDistance(shelter.distance_m);

        header.appendChild(titleGroup);
        header.appendChild(distance);

        // Meta row
        const meta = document.createElement('div');
        meta.className = 'card-meta';

        const addr = document.createElement('span');
        addr.className = 'card-address';
        addr.textContent = shelter.address;

        const badge = document.createElement('span');
        badge.className = 'card-type-badge';
        if (shelter.type === TYPE.EMERGENCY) badge.classList.add('type-emergency');
        else if (shelter.type === TYPE.OVERNIGHT) badge.classList.add('type-overnight');
        else badge.classList.add('type-longterm');
        badge.textContent = TYPE_LABELS[shelter.type] || 'Shelter';

        meta.appendChild(addr);
        meta.appendChild(badge);

        // Capacity
        const capSection = document.createElement('div');
        capSection.className = 'card-capacity';

        const capHeader = document.createElement('div');
        capHeader.className = 'capacity-header';

        const capLabel = document.createElement('span');
        capLabel.className = 'capacity-label';
        capLabel.textContent = `${shelter.occupancy}/${shelter.capacity} beds`;

        const pct = shelter.capacity > 0 ? Math.round((shelter.occupancy / shelter.capacity) * 100) : 0;

        const capPct = document.createElement('span');
        capPct.className = 'capacity-pct';
        capPct.textContent = `${pct}%`;
        capPct.style.color = pct >= 90 ? '#F87171' : pct >= 70 ? '#FBBF24' : '#34D399';

        capHeader.appendChild(capLabel);
        capHeader.appendChild(capPct);

        const bar = document.createElement('div');
        bar.className = 'capacity-bar';

        const fill = document.createElement('div');
        fill.className = 'capacity-fill';
        if (pct >= 90) fill.classList.add('cap-high');
        else if (pct >= 70) fill.classList.add('cap-med');
        else fill.classList.add('cap-low');

        // Animate width on next frame
        fill.style.width = '0%';
        requestAnimationFrame(() => {
            requestAnimationFrame(() => {
                fill.style.width = `${pct}%`;
            });
        });

        bar.appendChild(fill);
        capSection.appendChild(capHeader);
        capSection.appendChild(bar);

        // Actions
        const actions = document.createElement('div');
        actions.className = 'card-actions';

        const navBtn = document.createElement('button');
        navBtn.className = 'btn-navigate';
        if (shelter.status === STATUS.CLOSED) navBtn.classList.add('btn-disabled');

        const navIcon = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        navIcon.setAttribute('width', '14');
        navIcon.setAttribute('height', '14');
        navIcon.setAttribute('viewBox', '0 0 24 24');
        navIcon.setAttribute('fill', 'currentColor');
        const navPath = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        navPath.setAttribute('d', 'M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z');
        navIcon.appendChild(navPath);

        const navText = document.createTextNode('Navigate');
        navBtn.appendChild(navIcon);
        navBtn.appendChild(navText);

        navBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (shelter.status === STATUS.CLOSED) return;
            if (this.onNavigate) this.onNavigate(shelter);
        });

        const statusBtn = document.createElement('button');
        statusBtn.className = 'btn-secondary';
        statusBtn.textContent = STATUS_LABELS[shelter.status] || 'Unknown';

        actions.appendChild(navBtn);
        actions.appendChild(statusBtn);

        // Assemble card
        card.appendChild(header);
        card.appendChild(meta);
        card.appendChild(capSection);
        card.appendChild(actions);

        card.addEventListener('click', () => {
            if (this.onCardClick) this.onCardClick(shelter);
        });

        return card;
    }

    selectCard(id) {
        this.selectedId = id;
        const cards = this.list.querySelectorAll('.shelter-card');
        cards.forEach((c) => {
            if (parseInt(c.dataset.id) === id) {
                c.classList.add('selected');
                c.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
            } else {
                c.classList.remove('selected');
            }
        });
    }
}

// ---------------------------------------------------------------------------
// Navigation Controller
// ---------------------------------------------------------------------------

class NavigationController {
    constructor(mapManager) {
        this.mapManager = mapManager;
        this.overlay = document.getElementById('nav-overlay');
        this.etaEl = document.getElementById('nav-eta');
        this.maneuverEl = document.getElementById('nav-maneuver');
        this.maneuverText = document.getElementById('maneuver-text');
        this.maneuverDist = document.getElementById('maneuver-distance');
        this.destName = document.getElementById('nav-dest-name');
        this.destInfo = document.getElementById('nav-dest-info');
        this.backBtn = document.getElementById('nav-back');
        this.active = false;
        this.onExit = null;

        this.backBtn.addEventListener('click', () => this.exit());
    }

    start(shelter, routeData) {
        this.active = true;
        this.overlay.classList.remove('hidden');

        // ETA
        const minutes = Math.ceil(routeData.estimated_seconds / 60);
        this.etaEl.textContent = `ETA ${minutes} min`;

        // First maneuver
        if (routeData.maneuvers && routeData.maneuvers.length > 0) {
            this.maneuverEl.classList.remove('hidden');
            const m = routeData.maneuvers[0];
            this.maneuverText.textContent = m.instruction;
            this.maneuverDist.textContent = `in ${formatDistance(m.distance_m)}`;
        }

        // Destination
        this.destName.textContent = shelter.name;
        const statusLabel = STATUS_LABELS[shelter.status] || '';
        this.destInfo.textContent = `${statusLabel} \u00B7 ${formatDistance(shelter.distance_m)} \u00B7 ${minutes} min`;

        // Draw route on map
        this.mapManager.drawRoute(routeData);
    }

    exit() {
        this.active = false;
        this.overlay.classList.add('hidden');
        this.maneuverEl.classList.add('hidden');
        this.mapManager.clearRoute();
        this.mapManager.clearHighlight();
        if (this.onExit) this.onExit();
    }
}

// ---------------------------------------------------------------------------
// Search Handler
// ---------------------------------------------------------------------------

class SearchHandler {
    constructor() {
        this.input = document.getElementById('search-input');
        this.clearBtn = document.getElementById('search-clear');
        this.allShelters = [];
        this.onResults = null;
        this.debounceTimer = null;

        this.input.addEventListener('input', () => this._onInput());
        this.input.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                this._filter();
            }
        });
        this.clearBtn.addEventListener('click', () => this.clear());
    }

    _onInput() {
        const val = this.input.value.trim();
        this.clearBtn.classList.toggle('hidden', val.length === 0);

        clearTimeout(this.debounceTimer);
        this.debounceTimer = setTimeout(() => this._filter(), SEARCH_DEBOUNCE);
    }

    _filter() {
        const query = this.input.value.trim().toLowerCase();
        if (!query) {
            if (this.onResults) this.onResults(this.allShelters);
            return;
        }

        const results = this.allShelters.filter(
            (s) =>
                s.name.toLowerCase().includes(query) ||
                s.address.toLowerCase().includes(query)
        );

        if (this.onResults) this.onResults(results);
    }

    setData(shelters) {
        this.allShelters = shelters;
    }

    clear() {
        this.input.value = '';
        this.clearBtn.classList.add('hidden');
        if (this.onResults) this.onResults(this.allShelters);
    }
}

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

function formatDistance(meters) {
    if (meters < 1000) return `${Math.round(meters)} m`;
    return `${(meters / 1000).toFixed(1)} km`;
}

// ---------------------------------------------------------------------------
// App Init
// ---------------------------------------------------------------------------

(async function init() {
    const api = new ApiClient();
    const mapMgr = new MapManager('map');
    const panel = new PanelController();
    const nav = new NavigationController(mapMgr);
    const search = new SearchHandler();

    // Step 1: Health check
    const online = await api.checkHealth();
    const statusBanner = document.getElementById('connection-status');
    if (!online) {
        statusBanner.classList.remove('hidden');
    }

    // Step 2: Init map centered on OKC
    await mapMgr.init(OKC_CENTER);

    // Step 3: Request geolocation
    if ('geolocation' in navigator) {
        navigator.geolocation.getCurrentPosition(
            (pos) => {
                const lat = pos.coords.latitude;
                const lon = pos.coords.longitude;
                mapMgr.setUserLocation(lat, lon);
                mapMgr.flyTo(lat, lon, 14);

                // Re-query nearest with real location
                api.findNearest(lat, lon, 5000, 10).then((data) => {
                    if (data && data.shelters && data.shelters.length > 0) {
                        // Recalculate distances with user location
                        data.shelters.forEach((s) => {
                            if (!s.distance_m || s.distance_m === 0) {
                                s.distance_m = haversineDistance(lat, lon, s.lat, s.lon);
                            }
                        });
                        data.shelters.sort((a, b) => a.distance_m - b.distance_m);
                        search.setData(data.shelters);
                        panel.renderShelters(data.shelters);
                        mapMgr.addShelterSource(data.shelters);
                    }
                });
            },
            () => {
                // Geolocation denied — use default OKC position
                mapMgr.setUserLocation(OKC_CENTER.lat, OKC_CENTER.lon);
            },
            { enableHighAccuracy: true, timeout: 5000 }
        );
    }

    // Step 4: Load initial shelters
    const initialData = await api.findNearest(OKC_CENTER.lat, OKC_CENTER.lon, 5000, 10);
    let shelters = initialData.shelters || [];
    shelters.sort((a, b) => a.distance_m - b.distance_m);

    search.setData(shelters);
    panel.renderShelters(shelters);
    mapMgr.addShelterSource(shelters);

    // Step 5: Panel to peek
    panel.setState('peek');

    // ---- Wire up interactions ----

    // Shelter pin click on map
    mapMgr.onShelterClick = (props, coords) => {
        const id = typeof props.id === 'string' ? parseInt(props.id) : props.id;
        panel.selectCard(id);
        mapMgr.highlightShelter(id);
        mapMgr.flyTo(coords[1], coords[0], 15);
        if (panel.state === 'peek') panel.setState('half');
    };

    // Shelter card click in panel
    panel.onCardClick = (shelter) => {
        panel.selectCard(shelter.id);
        mapMgr.highlightShelter(shelter.id);
        mapMgr.flyTo(shelter.lat, shelter.lon, 15);
    };

    // Navigate button
    panel.onNavigate = async (shelter) => {
        const startLat = mapMgr.userLat || OKC_CENTER.lat;
        const startLon = mapMgr.userLon || OKC_CENTER.lon;
        const routeData = await api.getRoute(startLat, startLon, shelter.lat, shelter.lon);
        nav.start(shelter, routeData);
        panel.setState('peek');
    };

    // Exit navigation
    nav.onExit = () => {
        panel.setState('peek');
    };

    // Search
    search.onResults = (results) => {
        panel.renderShelters(results);
        mapMgr.addShelterSource(results);
    };

    // FAB: locate me
    document.getElementById('fab-locate').addEventListener('click', () => {
        mapMgr.flyToUser();
    });

    // FAB: find nearest
    document.getElementById('fab-nearest').addEventListener('click', async () => {
        const lat = mapMgr.userLat || OKC_CENTER.lat;
        const lon = mapMgr.userLon || OKC_CENTER.lon;
        const data = await api.findNearest(lat, lon, 5000, 10);
        let results = data.shelters || [];
        results.sort((a, b) => a.distance_m - b.distance_m);
        search.setData(results);
        search.clear();
        panel.renderShelters(results);
        mapMgr.addShelterSource(results);
        panel.setState('half');

        // Fly to nearest open shelter
        const nearest = results.find((s) => s.status === STATUS.OPEN);
        if (nearest) {
            mapMgr.flyTo(nearest.lat, nearest.lon, 15);
            mapMgr.highlightShelter(nearest.id);
            panel.selectCard(nearest.id);
        }
    });
})();

// Simple Haversine for client-side distance calc
function haversineDistance(lat1, lon1, lat2, lon2) {
    const R = 6371000;
    const toRad = (d) => (d * Math.PI) / 180;
    const dLat = toRad(lat2 - lat1);
    const dLon = toRad(lon2 - lon1);
    const a =
        Math.sin(dLat / 2) * Math.sin(dLat / 2) +
        Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon / 2) * Math.sin(dLon / 2);
    return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

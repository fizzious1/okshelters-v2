CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE shelters (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    location    GEOGRAPHY(POINT, 4326) NOT NULL,
    address     TEXT,
    type        SMALLINT NOT NULL,  -- 0=emergency, 1=overnight, 2=long-term
    capacity    SMALLINT NOT NULL,
    occupancy   SMALLINT DEFAULT 0,
    status      SMALLINT DEFAULT 1, -- 0=closed, 1=open, 2=full
    phone       TEXT,
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_shelters_geo ON shelters USING GIST(location);
CREATE INDEX idx_shelters_status ON shelters (status) WHERE status = 1;

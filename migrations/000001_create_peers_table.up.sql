-- The `peers` table keeps track of all peers ever seen
CREATE TABLE peers
(
    -- The unique identifier in the scope of this database
    id              BIGINT GENERATED ALWAYS AS IDENTITY,
    -- The peerID in form of QmFoo or 12D3Bar
    multi_hash      TEXT        NOT NULL,
    -- The agent version of the peer, e.g., kubo/0.15.0
    agent_version   TEXT,
    -- An array of supported protocols, e.g., bitswap/1.0.0
    protocols       TEXT[],
    -- An array of advertised multi addresses at which that peer is reachable
    multi_addresses TEXT[] NOT NULL,
    -- An array of extracted ip_addresses from the multi_address array
    ip_addresses    TEXT[] NOT NULL,
    -- An array of countries that the IP addresses could be associated with
    countries       TEXT[] NOT NULL,
    -- An array of continents that the IP addresses could be associated with
    continents      TEXT[] NOT NULL,
    -- An array of autonomous system numbers that the IP addresses could be associated with
    asns            INT[] NOT NULL,
    -- Type of the target, e.g., gateway or pinning service
    target_type     TEXT        NOT NULL,
    -- Name of the target, e.g., ipfs.io
    target_name     TEXT        NOT NULL,
    -- The timestamp at which this peer has last contacted the antares host
    last_seen_at    TIMESTAMPTZ NOT NULL,
    -- The timestamp at which any of the fields were updated the last time
    updated_at      TIMESTAMPTZ NOT NULL,
    -- The timestamp at which this row was inserted into the database
    created_at      TIMESTAMPTZ NOT NULL,

    -- Ensure the peer is only once in the database
    CONSTRAINT uq_peers_multi_hash UNIQUE (multi_hash, target_name),

    PRIMARY KEY (id)
);

CREATE SCHEMA IF NOT EXISTS fleet;
CREATE TABLE IF NOT EXISTS fleet.servers(id text PRIMARY KEY,data jsonb NOT NULL,last_seen timestamptz,token_hash text UNIQUE);
CREATE TABLE IF NOT EXISTS fleet.commands(id text PRIMARY KEY,server_id text NOT NULL REFERENCES fleet.servers(id),data jsonb NOT NULL,status text NOT NULL DEFAULT 'Queued');

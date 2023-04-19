CREATE SCHEMA IF NOT EXISTS public;

CREATE TABLE IF NOT EXISTS tokens (
    ID varchar(255) PRIMARY KEY,
    DELETED boolean NOT NULL DEFAULT false,
    VERSION int,
    CREATED_AT timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    LAST_UPDATED timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    NAME varchar,
    DESCRIPTION varchar,
    CONTRACT_ADDRESS varchar(255),
    MEDIA jsonb,
    CHAIN int,
    OWNER_ADDRESS varchar(255),
    TOKEN_URI varchar,
    TOKEN_TYPE varchar,
    TOKEN_ID varchar,
    QUANTITY varchar,
    OWNERSHIP_HISTORY jsonb [],
    TOKEN_METADATA jsonb,
    EXTERNAL_URL varchar,
    BLOCK_NUMBER bigint,
    IS_SPAM boolean
);

CREATE UNIQUE INDEX IF NOT EXISTS erc1155_idx ON tokens (TOKEN_ID, CONTRACT_ADDRESS, OWNER_ADDRESS) WHERE TOKEN_TYPE = 'ERC-1155';

CREATE UNIQUE INDEX IF NOT EXISTS erc721_idx ON tokens (TOKEN_ID, CONTRACT_ADDRESS) WHERE TOKEN_TYPE = 'ERC-721';

CREATE INDEX IF NOT EXISTS token_id_contract_address_idx ON tokens (TOKEN_ID, CONTRACT_ADDRESS);

CREATE INDEX IF NOT EXISTS owner_address_idx ON tokens (OWNER_ADDRESS);

CREATE INDEX IF NOT EXISTS contract_address_idx ON tokens (CONTRACT_ADDRESS);

CREATE INDEX IF NOT EXISTS block_number_idx ON tokens (BLOCK_NUMBER);

CREATE TABLE IF NOT EXISTS contracts (
    ID varchar(255) PRIMARY KEY,
    DELETED boolean NOT NULL DEFAULT false,
    VERSION int,
    CREATED_AT timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    LAST_UPDATED timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHAIN int,
    NAME varchar,
    SYMBOL varchar,
    ADDRESS varchar(255),
    OWNER_ADDRESS varchar(255),
    LATEST_BLOCK bigint
);

CREATE UNIQUE INDEX IF NOT EXISTS address_idx ON contracts (ADDRESS);

alter table contracts add column owner_address character varying(255);

ALTER TABLE
    users
ADD
    COLUMN LAST_UPDATED timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
ADD
    COLUMN CREATED_AT timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE
    galleries
ADD
    COLUMN LAST_UPDATED timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
ADD
    COLUMN CREATED_AT timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE
    nfts
ADD
    COLUMN LAST_UPDATED timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
ADD
    COLUMN CREATED_AT timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
ADD
    COLUMN TOKEN_COLLECTION_NAME varchar,
ADD
    COLUMN COLLECTORS_NOTE varchar;

ALTER TABLE
    collections
ADD
    COLUMN LAST_UPDATED timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
ADD
    COLUMN CREATED_AT timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE
    nonces
ADD
    COLUMN LAST_UPDATED timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
ADD
    COLUMN CREATED_AT timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
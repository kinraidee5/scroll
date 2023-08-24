-- +goose Up
-- +goose StatementBegin

create table l2_block
(
-- block
    number                  BIGINT          NOT NULL,
    hash                    VARCHAR         NOT NULL,
    parent_hash             VARCHAR         NOT NULL,
    header                  TEXT            NOT NULL,
    transactions            TEXT            NOT NULL,
    withdraw_root           VARCHAR         NOT NULL,
    state_root              VARCHAR         NOT NULL,
    tx_num                  INTEGER         NOT NULL,
    gas_used                BIGINT          NOT NULL,
    block_timestamp         NUMERIC         NOT NULL,
    row_consumption         TEXT,

-- chunk
    chunk_hash              VARCHAR         DEFAULT NULL,

-- metadata
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMP(0)    DEFAULT NULL
);

create unique index l2_block_hash_uindex
on l2_block (hash) where deleted_at IS NULL;

create unique index l2_block_number_uindex
on l2_block (number) where deleted_at IS NULL;

create index l2_block_chunk_hash_index
on l2_block (chunk_hash) where deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists l2_block;
-- +goose StatementEnd

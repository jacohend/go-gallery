// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.17.2
// source: token_gallery.sql

package coredb

import (
	"context"
	"time"

	"github.com/jackc/pgtype"
)

const upsertTokens = `-- name: UpsertTokens :many
insert into tokens
(
  id
  , deleted
  , version
  , created_at
  , last_updated
  , name
  , description
  , collectors_note
  , media
  , token_type
  , token_id
  , quantity
  , ownership_history
  , token_metadata
  , external_url
  , block_number
  , owner_user_id
  , owned_by_wallets
  , chain
  , contract
  , is_user_marked_spam
  , is_provider_marked_spam
  , last_synced
  , token_uri
) (
  select
    id
    , deleted 
    , version
    , created_at
    , last_updated
    , name
    , description
    , collectors_note
    , media
    , token_type
    , token_id
    , quantity
    , ownership_history[ownership_history_start_idx::int:ownership_history_end_idx::int]
    , token_metadata
    , external_url
    , block_number
    , owner_user_id
    , owned_by_wallets[owned_by_wallets_start_idx::int:owned_by_wallets_end_idx::int]
    , chain
    , contract
    , is_user_marked_spam
    , is_provider_marked_spam
    , last_synced
    , token_uri
  from (
    select
      unnest($1::varchar[]) as id
      , unnest($2::boolean[]) as deleted
      , unnest($3::int[]) as version
      , unnest($4::timestamptz[]) as created_at
      , unnest($5::timestamptz[]) as last_updated
      , unnest($6::varchar[]) as name
      , unnest($7::varchar[]) as description
      , unnest($8::varchar[]) as collectors_note
      , unnest($9::jsonb[]) as media
      , unnest($10::varchar[]) as token_type
      , unnest($11::varchar[]) as token_id
      , unnest($12::varchar[]) as quantity
      , $13::jsonb[] as ownership_history
      , unnest($14::int[]) as ownership_history_start_idx
      , unnest($15::int[]) as ownership_history_end_idx
      , unnest($16::jsonb[]) as token_metadata
      , unnest($17::varchar[]) as external_url
      , unnest($18::bigint[]) as block_number
      , unnest($19::varchar[]) as owner_user_id
      , $20::varchar[] as owned_by_wallets
      , unnest($21::int[]) as owned_by_wallets_start_idx
      , unnest($22::int[]) as owned_by_wallets_end_idx
      , unnest($23::int[]) as chain
      , unnest($24::varchar[]) as contract
      , unnest($25::bool[]) as is_user_marked_spam
      , unnest($26::bool[]) as is_provider_marked_spam
      , unnest($27::timestamptz[]) as last_synced
      , unnest($28::varchar[]) as token_uri
  ) bulk_upsert
)
on conflict (token_id, contract, chain, owner_user_id) where deleted = false
do update set
  media = excluded.media
  , token_type = excluded.token_type
  , chain = excluded.chain
  , name = excluded.name
  , description = excluded.description
  , token_uri = excluded.token_uri
  , quantity = excluded.quantity
  , owner_user_id = excluded.owner_user_id
  , owned_by_wallets = excluded.owned_by_wallets
  , ownership_history = tokens.ownership_history || excluded.ownership_history
  , token_metadata = excluded.token_metadata
  , external_url = excluded.external_url
  , block_number = excluded.block_number
  , version = excluded.version
  , last_updated = excluded.last_updated
  , is_user_marked_spam = tokens.is_user_marked_spam
  , is_provider_marked_spam = excluded.is_provider_marked_spam
  , last_synced = greatest(excluded.last_synced,tokens.last_synced)
returning id, deleted, version, created_at, last_updated, name, description, collectors_note, media, token_uri, token_type, token_id, quantity, ownership_history, token_metadata, external_url, block_number, owner_user_id, owned_by_wallets, chain, contract, is_user_marked_spam, is_provider_marked_spam, last_synced
`

type UpsertTokensParams struct {
	ID                       []string
	Deleted                  []bool
	Version                  []int32
	CreatedAt                []time.Time
	LastUpdated              []time.Time
	Name                     []string
	Description              []string
	CollectorsNote           []string
	Media                    []pgtype.JSONB
	TokenType                []string
	TokenID                  []string
	Quantity                 []string
	OwnershipHistory         []pgtype.JSONB
	OwnershipHistoryStartIdx []int32
	OwnershipHistoryEndIdx   []int32
	TokenMetadata            []pgtype.JSONB
	ExternalUrl              []string
	BlockNumber              []int64
	OwnerUserID              []string
	OwnedByWallets           []string
	OwnedByWalletsStartIdx   []int32
	OwnedByWalletsEndIdx     []int32
	Chain                    []int32
	Contract                 []string
	IsUserMarkedSpam         []bool
	IsProviderMarkedSpam     []bool
	LastSynced               []time.Time
	TokenUri                 []string
}

func (q *Queries) UpsertTokens(ctx context.Context, arg UpsertTokensParams) ([]Token, error) {
	rows, err := q.db.Query(ctx, upsertTokens,
		arg.ID,
		arg.Deleted,
		arg.Version,
		arg.CreatedAt,
		arg.LastUpdated,
		arg.Name,
		arg.Description,
		arg.CollectorsNote,
		arg.Media,
		arg.TokenType,
		arg.TokenID,
		arg.Quantity,
		arg.OwnershipHistory,
		arg.OwnershipHistoryStartIdx,
		arg.OwnershipHistoryEndIdx,
		arg.TokenMetadata,
		arg.ExternalUrl,
		arg.BlockNumber,
		arg.OwnerUserID,
		arg.OwnedByWallets,
		arg.OwnedByWalletsStartIdx,
		arg.OwnedByWalletsEndIdx,
		arg.Chain,
		arg.Contract,
		arg.IsUserMarkedSpam,
		arg.IsProviderMarkedSpam,
		arg.LastSynced,
		arg.TokenUri,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Token
	for rows.Next() {
		var i Token
		if err := rows.Scan(
			&i.ID,
			&i.Deleted,
			&i.Version,
			&i.CreatedAt,
			&i.LastUpdated,
			&i.Name,
			&i.Description,
			&i.CollectorsNote,
			&i.Media,
			&i.TokenUri,
			&i.TokenType,
			&i.TokenID,
			&i.Quantity,
			&i.OwnershipHistory,
			&i.TokenMetadata,
			&i.ExternalUrl,
			&i.BlockNumber,
			&i.OwnerUserID,
			&i.OwnedByWallets,
			&i.Chain,
			&i.Contract,
			&i.IsUserMarkedSpam,
			&i.IsProviderMarkedSpam,
			&i.LastSynced,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

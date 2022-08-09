// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.13.0

package sqlc

import (
	"database/sql"
	"time"

	"github.com/jackc/pgtype"
	"github.com/mikeydub/go-gallery/service/persist"
)

type Access struct {
	ID                  persist.DBID
	Deleted             bool
	Version             sql.NullInt32
	CreatedAt           time.Time
	LastUpdated         time.Time
	UserID              sql.NullString
	MostRecentBlock     sql.NullInt64
	RequiredTokensOwned pgtype.JSONB
	IsAdmin             sql.NullBool
}

type Backup struct {
	ID          persist.DBID
	Deleted     bool
	Version     sql.NullInt32
	CreatedAt   time.Time
	LastUpdated time.Time
	GalleryID   sql.NullString
	Gallery     pgtype.JSONB
}

type Collection struct {
	ID             persist.DBID
	Deleted        bool
	OwnerUserID    persist.DBID
	Nfts           persist.DBIDList
	Version        sql.NullInt32
	LastUpdated    time.Time
	CreatedAt      time.Time
	Hidden         bool
	CollectorsNote sql.NullString
	Name           sql.NullString
	Layout         persist.TokenLayout
	TokenSettings  map[persist.DBID]persist.CollectionTokenSettings
}

type CollectionEvent struct {
	ID           persist.DBID
	UserID       sql.NullString
	CollectionID sql.NullString
	Version      sql.NullInt32
	EventCode    sql.NullInt32
	CreatedAt    time.Time
	LastUpdated  time.Time
	Data         pgtype.JSONB
	Sent         sql.NullBool
}

type Contract struct {
	ID             persist.DBID
	Deleted        bool
	Version        sql.NullInt32
	CreatedAt      time.Time
	LastUpdated    time.Time
	Name           sql.NullString
	Symbol         sql.NullString
	Address        persist.Address
	CreatorAddress persist.Address
	Chain          sql.NullInt32
}

type EarlyAccess struct {
	Address string
}

type Event struct {
	ID             persist.DBID
	Version        int32
	ActorID        persist.DBID
	ResourceTypeID persist.ResourceType
	SubjectID      persist.DBID
	UserID         persist.DBID
	TokenID        persist.DBID
	CollectionID   persist.DBID
	Action         persist.Action
	Data           persist.EventData
	Deleted        bool
	LastUpdated    time.Time
	CreatedAt      time.Time
}

type Feature struct {
	ID                  persist.DBID
	Deleted             bool
	Version             sql.NullInt32
	LastUpdated         time.Time
	CreatedAt           time.Time
	RequiredToken       sql.NullString
	RequiredAmount      sql.NullInt64
	TokenType           sql.NullString
	Name                sql.NullString
	IsEnabled           sql.NullBool
	AdminOnly           sql.NullBool
	ForceEnabledUserIds []string
}

type FeedBlocklist struct {
	ID          persist.DBID
	UserID      persist.DBID
	Action      persist.Action
	LastUpdated time.Time
	CreatedAt   time.Time
	Deleted     bool
}

type FeedEvent struct {
	ID          persist.DBID
	Version     int32
	OwnerID     persist.DBID
	Action      persist.Action
	Data        persist.FeedEventData
	EventTime   time.Time
	EventIds    persist.DBIDList
	Deleted     bool
	LastUpdated time.Time
	CreatedAt   time.Time
}

type Follow struct {
	ID          persist.DBID
	Follower    persist.DBID
	Followee    persist.DBID
	Deleted     bool
	CreatedAt   time.Time
	LastUpdated time.Time
}

type Gallery struct {
	ID          persist.DBID
	Deleted     bool
	LastUpdated time.Time
	CreatedAt   time.Time
	Version     sql.NullInt32
	OwnerUserID persist.DBID
	Collections persist.DBIDList
}

type LoginAttempt struct {
	ID                 persist.DBID
	Deleted            bool
	Version            sql.NullInt32
	CreatedAt          time.Time
	LastUpdated        time.Time
	Address            sql.NullString
	RequestHostAddress sql.NullString
	UserExists         sql.NullBool
	Signature          sql.NullString
	SignatureValid     sql.NullBool
	RequestHeaders     pgtype.JSONB
	NonceValue         sql.NullString
}

type Membership struct {
	ID          persist.DBID
	Deleted     bool
	Version     sql.NullInt32
	CreatedAt   time.Time
	LastUpdated time.Time
	TokenID     sql.NullString
	Name        sql.NullString
	AssetUrl    sql.NullString
	Owners      persist.TokenHolderList
}

type NftEvent struct {
	ID          persist.DBID
	UserID      sql.NullString
	NftID       sql.NullString
	Version     sql.NullInt32
	EventCode   sql.NullInt32
	CreatedAt   time.Time
	LastUpdated time.Time
	Data        pgtype.JSONB
	Sent        sql.NullBool
}

type Nonce struct {
	ID          persist.DBID
	Deleted     bool
	Version     sql.NullInt32
	LastUpdated time.Time
	CreatedAt   time.Time
	UserID      sql.NullString
	Address     sql.NullString
	Value       sql.NullString
	Chain       sql.NullInt32
}

type Token struct {
	ID               persist.DBID
	Deleted          bool
	Version          sql.NullInt32
	CreatedAt        time.Time
	LastUpdated      time.Time
	Name             sql.NullString
	Description      sql.NullString
	CollectorsNote   sql.NullString
	Media            pgtype.JSONB
	TokenUri         sql.NullString
	TokenType        sql.NullString
	TokenID          sql.NullString
	Quantity         sql.NullString
	OwnershipHistory []pgtype.JSONB
	TokenMetadata    pgtype.JSONB
	ExternalUrl      sql.NullString
	BlockNumber      sql.NullInt64
	OwnerUserID      persist.DBID
	OwnedByWallets   persist.DBIDList
	Chain            sql.NullInt32
	Contract         persist.DBID
}

type User struct {
	ID                 persist.DBID
	Deleted            bool
	Version            sql.NullInt32
	LastUpdated        time.Time
	CreatedAt          time.Time
	Username           sql.NullString
	UsernameIdempotent sql.NullString
	Wallets            persist.WalletList
	Bio                sql.NullString
}

type UserEvent struct {
	ID          persist.DBID
	UserID      sql.NullString
	Version     sql.NullInt32
	EventCode   sql.NullInt32
	CreatedAt   time.Time
	LastUpdated time.Time
	Data        pgtype.JSONB
	Sent        sql.NullBool
}

type Wallet struct {
	ID          persist.DBID
	CreatedAt   time.Time
	LastUpdated time.Time
	Deleted     bool
	Version     sql.NullInt32
	Address     persist.Address
	WalletType  persist.WalletType
	Chain       sql.NullInt32
}

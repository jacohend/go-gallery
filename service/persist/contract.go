package persist

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// Contract represents an ethereum contract in the database
type Contract struct {
	Version      NullInt32       `json:"version"` // schema version for this model
	ID           DBID            `json:"id" binding:"required"`
	CreationTime CreationTime    `json:"created_at"`
	Deleted      NullBool        `json:"-"`
	LastUpdated  LastUpdatedTime `json:"last_updated"`

	Chain Chain `json:"chain"`

	Address        EthereumAddress `json:"address"`
	Symbol         NullString      `json:"symbol"`
	Name           NullString      `json:"name"`
	OwnerAddress   EthereumAddress `json:"owner_address"`
	CreatorAddress EthereumAddress `json:"creator_address"`

	LatestBlock      BlockNumber      `json:"latest_block"`
	ContractURI      TokenURI         `json:"contract_uri"`
	ContractMetadata ContractMetadata `json:"contract_metadata"`
}

type ContractMetadata map[string]interface{}

// ContractUpdateInput is the input for updating contract metadata fields
type ContractUpdateInput struct {
	Symbol         NullString      `json:"symbol"`
	Name           NullString      `json:"name"`
	OwnerAddress   EthereumAddress `json:"owner_address"`
	CreatorAddress EthereumAddress `json:"creator_address"`

	LatestBlock BlockNumber `json:"latest_block"`
}

// ContractRepository represents a repository for interacting with persisted contracts
type ContractRepository interface {
	GetByAddress(context.Context, EthereumAddress) (Contract, error)
	GetMetadataByAddress(context.Context, EthereumAddress) (Contract, error)
	UpdateByAddress(context.Context, EthereumAddress, ContractUpdateInput) error
	UpsertByAddress(context.Context, EthereumAddress, Contract) error
	GetContractsOwnedByAddress(context.Context, EthereumAddress) ([]Contract, error)
	BulkUpsert(context.Context, []Contract) error
	UpdateMetadataByAddress(context.Context, EthereumAddress, Contract) error
}

// ErrContractNotFoundByAddress is an error type for when a contract is not found by address
type ErrContractNotFoundByAddress struct {
	Address EthereumAddress
}

type ErrContractNotFoundByID struct {
	ID DBID
}

func (e ErrContractNotFoundByAddress) Error() string {
	return fmt.Sprintf("contract not found by address: %s", e.Address)
}

func (e ErrContractNotFoundByID) Error() string {
	return fmt.Sprintf("contract not found by ID: %s", e.ID)
}

// Scan implements the database/sql Scanner interface for the TokenMetadata type
func (m *ContractMetadata) Scan(src interface{}) error {
	if src == nil {
		*m = ContractMetadata{}
		return nil
	}
	return json.Unmarshal(src.([]uint8), m)
}

// Value implements the database/sql/driver Valuer interface for the TokenMetadata type
func (m ContractMetadata) Value() (driver.Value, error) {
	return m.MarshallJSON()
}

// MarshallJSON implements the json.Marshaller interface for the TokenMetadata type
func (m ContractMetadata) MarshallJSON() ([]byte, error) {
	val, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	cleaned := strings.ToValidUTF8(string(val), "")
	// Replace literal '\\u0000' with empty string (marshal to JSON escapes each backslash)
	cleaned = strings.ReplaceAll(cleaned, "\\\\u0000", "")
	// Replace unicode NULL char (u+0000) i.e. '\u0000' with empty string
	cleaned = strings.ReplaceAll(cleaned, "\\u0000", "")
	return []byte(cleaned), nil
}

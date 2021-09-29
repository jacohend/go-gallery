package persist

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mikeydub/go-gallery/runtime"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	nftColName           = "nfts"
	nftCollectionColName = "nft_collections"
)

// NftDB represents an nft in the database
type NftDB struct {
	Version      int64              `bson:"version"              json:"version"` // schema version for this model
	ID           DBID               `bson:"_id"                  json:"id" binding:"required"`
	CreationTime primitive.DateTime `bson:"created_at"        json:"created_at"`
	Deleted      bool               `bson:"deleted" json:"-"`

	CollectorsNote string `bson:"collectors_note" json:"collectors_note"`
	OwnerUserID    DBID   `bson:"owner_user_id" json:"user_id"`

	OwnershipHistory *OwnershipHistory `bson:"ownership_history,only_get" json:"ownership_history"`

	Name             string   `bson:"name"                 json:"name"`
	Description      string   `bson:"description"          json:"description"`
	ExternalURL      string   `bson:"external_url"         json:"external_url"`
	TokenMetadataURL string   `bson:"token_metadata_url" json:"token_metadata_url"`
	CreatorAddress   string   `bson:"creator_address"      json:"creator_address"`
	CreatorName      string   `bson:"creator_name" json:"creator_name"`
	OwnerAddress     string   `bson:"owner_address" json:"owner_address"`
	Contract         Contract `bson:"contract"     json:"asset_contract"`

	OpenSeaID int `bson:"opensea_id"       json:"opensea_id"`
	// OPEN_SEA_TOKEN_ID
	// https://api.opensea.io/api/v1/asset/0xa7d8d9ef8d8ce8992df33d8b8cf4aebabd5bd270/26000331
	// (/asset/:contract_address/:token_id)
	OpenSeaTokenID string `bson:"opensea_token_id" json:"opensea_token_id"`

	// IMAGES - OPENSEA
	ImageURL             string `bson:"image_url"           json:"image_url"`
	ImageThumbnailURL    string `bson:"image_thumbnail_url" json:"image_thumbnail_url"`
	ImagePreviewURL      string `bson:"image_preview_url"   json:"image_preview_url"`
	ImageOriginalURL     string `bson:"image_original_url" json:"image_original_url"`
	AnimationURL         string `bson:"animation_url" json:"animation_url"`
	AnimationOriginalURL string `bson:"animation_original_url" json:"animation_original_url"`

	AcquisitionDateStr string `bson:"acquisition_date" json:"acquisition_date"`
}

// Nft represents an nft throughout the application
type Nft struct {
	Version      int64              `bson:"version"              json:"version"` // schema version for this model
	ID           DBID               `bson:"_id"                  json:"id" binding:"required"`
	CreationTime primitive.DateTime `bson:"created_at"        json:"created_at"`
	Deleted      bool               `bson:"deleted" json:"-"`

	CollectorsNote string `bson:"collectors_note" json:"collectors_note"`
	OwnerUserID    DBID   `bson:"owner_user_id" json:"user_id"`
	OwnerUsername  string `bson:"owner_username" json:"owner_username"`

	OwnershipHistory *OwnershipHistory `bson:"ownership_history,only_get" json:"ownership_history"`

	Name             string   `bson:"name"                 json:"name"`
	Description      string   `bson:"description"          json:"description"`
	ExternalURL      string   `bson:"external_url"         json:"external_url"`
	TokenMetadataURL string   `bson:"token_metadata_url" json:"token_metadata_url"`
	CreatorAddress   string   `bson:"creator_address"      json:"creator_address"`
	CreatorName      string   `bson:"creator_name" json:"creator_name"`
	OwnerAddress     string   `bson:"owner_address" json:"owner_address"`
	Contract         Contract `bson:"contract"     json:"asset_contract"`

	OpenSeaID int `bson:"opensea_id"       json:"opensea_id"`
	// OPEN_SEA_TOKEN_ID
	// https://api.opensea.io/api/v1/asset/0xa7d8d9ef8d8ce8992df33d8b8cf4aebabd5bd270/26000331
	// (/asset/:contract_address/:token_id)
	OpenSeaTokenID string `bson:"opensea_token_id" json:"opensea_token_id"`

	// IMAGES - OPENSEA
	ImageURL             string `bson:"image_url"           json:"image_url"`
	ImageThumbnailURL    string `bson:"image_thumbnail_url" json:"image_thumbnail_url"`
	ImagePreviewURL      string `bson:"image_preview_url"   json:"image_preview_url"`
	ImageOriginalURL     string `bson:"image_original_url" json:"image_original_url"`
	AnimationURL         string `bson:"animation_url" json:"animation_url"`
	AnimationOriginalURL string `bson:"animation_original_url" json:"animation_original_url"`

	AcquisitionDateStr string `bson:"acquisition_date" json:"acquisition_date"`
}

// CollectionNft represents and NFT in a collection of NFTs
type CollectionNft struct {
	ID           DBID               `bson:"_id"                  json:"id" binding:"required"`
	CreationTime primitive.DateTime `bson:"created_at"        json:"created_at"`

	OwnerUserID DBID `bson:"owner_user_id" json:"user_id"`

	Name string `bson:"name"                 json:"name"`

	Contract ContractCollectionNft `bson:"contract"     json:"asset_contract"`

	// IMAGES - OPENSEA
	ImageURL          string `bson:"image_url"           json:"image_url"`
	ImageThumbnailURL string `bson:"image_thumbnail_url" json:"image_thumbnail_url"`
	ImagePreviewURL   string `bson:"image_preview_url"   json:"image_preview_url"`
}

// Contract represents a smart contract's information for a given NFT
type Contract struct {
	ContractAddress      string `bson:"contract_address"     json:"address"`
	ContractName         string `bson:"contract_name" json:"name"`
	ContractImage        string `bson:"contract_image_url" json:"image_url"`
	ContractDescription  string `bson:"contract_description" json:"description"`
	ContractExternalLink string `bson:"contract_external_link" json:"external_link"`
	ContractSchemaName   string `bson:"contract_schema_name" json:"schema_name"`
	ContractSymbol       string `bson:"contract_symbol" json:"symbol"`
	ContractTotalSupply  string `bson:"contract_total_supply" json:"total_supply"`
}

// ContractCollectionNft represents a contract within a collection nft
type ContractCollectionNft struct {
	ContractName  string `bson:"contract_name" json:"name"`
	ContractImage string `bson:"contract_image_url" json:"image_url"`
}

// UpdateNFTInfoInput represents a MongoDB input to update the user defined info
// associated with a given NFT in the DB
type UpdateNFTInfoInput struct {
	CollectorsNote string `bson:"collectors_note"`
}

// NftCreateBulk is a helper function to create multiple nfts in one call and returns
// the ids of each nft created
func NftCreateBulk(pCtx context.Context, pNfts []*NftDB,
	pRuntime *runtime.Runtime) ([]DBID, error) {

	mp := newStorage(0, nftColName, pRuntime)

	nfts := make([]interface{}, len(pNfts))

	for i, v := range pNfts {
		nfts[i] = v
	}

	ids, err := mp.insertMany(pCtx, nfts)

	if err != nil {
		return nil, err
	}
	return ids, nil
}

// NftCreate inserts an NFT into the database
func NftCreate(pCtx context.Context, pNFT *NftDB,
	pRuntime *runtime.Runtime) (DBID, error) {

	mp := newStorage(0, nftColName, pRuntime)

	return mp.insert(pCtx, pNFT)
}

// NftGetByUserID finds an nft by its owner user id
func NftGetByUserID(pCtx context.Context, pUserID DBID,
	pRuntime *runtime.Runtime) ([]*Nft, error) {
	opts := options.Aggregate()
	if deadline, ok := pCtx.Deadline(); ok {
		dur := time.Until(deadline)
		opts.SetMaxTime(dur)
	}
	mp := newStorage(0, nftColName, pRuntime)
	result := []*Nft{}

	if err := mp.aggregate(pCtx, newNFTPipeline(bson.M{"owner_user_id": pUserID}), &result, opts); err != nil {
		return nil, err
	}

	return result, nil
}

// NftGetByID finds an nft by its id
func NftGetByID(pCtx context.Context, pID DBID, pRuntime *runtime.Runtime) ([]*Nft, error) {

	opts := options.Aggregate()
	if deadline, ok := pCtx.Deadline(); ok {
		dur := time.Until(deadline)
		opts.SetMaxTime(dur)
	}

	mp := newStorage(0, nftColName, pRuntime)
	result := []*Nft{}

	if err := mp.aggregate(pCtx, newNFTPipeline(bson.M{"_id": pID}), &result, opts); err != nil {
		return nil, err
	}

	return result, nil
}

// NftGetByContractData finds an nft by its contract data
func NftGetByContractData(pCtx context.Context, pTokenID, pContractAddress string,
	pRuntime *runtime.Runtime) ([]*Nft, error) {
	opts := options.Aggregate()
	if deadline, ok := pCtx.Deadline(); ok {
		dur := time.Until(deadline)
		opts.SetMaxTime(dur)
	}
	mp := newStorage(0, nftColName, pRuntime)
	result := []*Nft{}

	if err := mp.aggregate(pCtx, newNFTPipeline(bson.M{"opensea_token_id": pTokenID, "contract.contract_address": pContractAddress}), &result, opts); err != nil {
		return nil, err
	}

	return result, nil
}

// NftGetByOpenseaID finds an nft by its opensea ID
func NftGetByOpenseaID(pCtx context.Context, pOpenseaID int,
	pRuntime *runtime.Runtime) ([]*Nft, error) {
	opts := options.Aggregate()
	if deadline, ok := pCtx.Deadline(); ok {
		dur := time.Until(deadline)
		opts.SetMaxTime(dur)
	}
	mp := newStorage(0, nftColName, pRuntime)
	result := []*Nft{}

	if err := mp.aggregate(pCtx, newNFTPipeline(bson.M{"opensea_id": pOpenseaID}), &result, opts); err != nil {
		return nil, err
	}

	return result, nil
}

// NftUpdateByID updates an nft by its id, also ensuring that the NFT is owned
// by a given authorized user
// pUpdate is a struct that has bson tags representing the fields to be updated
func NftUpdateByID(pCtx context.Context, pID DBID, pUserID DBID, pUpdate interface{}, pRuntime *runtime.Runtime) error {

	mp := newStorage(0, nftColName, pRuntime)

	return mp.update(pCtx, bson.M{"_id": pID, "owner_user_id": pUserID}, pUpdate)
}

// NftBulkUpsert will create a bulk operation on the database to upsert many nfts for a given wallet address
// This function's primary purpose is to be used when syncing a user's NFTs from an external provider
func NftBulkUpsert(pCtx context.Context, pNfts []*NftDB, pRuntime *runtime.Runtime) ([]DBID, error) {

	mp := newStorage(0, nftColName, pRuntime)

	ids := make(chan DBID)
	errs := make(chan error)

	for _, v := range pNfts {

		go func(nft *NftDB) {
			id, err := mp.upsert(pCtx, bson.M{"opensea_id": nft.OpenSeaID}, nft)
			if err != nil {
				errs <- err
			}
			ids <- id
		}(v)
	}

	result := make([]DBID, len(pNfts))
	for i := 0; i < len(pNfts); i++ {
		select {
		case id := <-ids:
			result[i] = id
		case err := <-errs:
			return nil, err
		}
	}

	return result, nil

}

// NftOpenseaCacheSet adds a set of nfts to the opensea cache under a given wallet address
func NftOpenseaCacheSet(pCtx context.Context, pWalletAddresses []string, pNfts []*Nft, pRuntime *runtime.Runtime) error {

	mp := newStorage(0, nftColName, pRuntime).withRedis(OpenseaAssetsRDB, pRuntime)
	defer mp.cacheClose()
	for i, v := range pWalletAddresses {
		pWalletAddresses[i] = strings.ToLower(v)
	}

	toCache, err := json.Marshal(pNfts)
	if err != nil {
		return err
	}
	return mp.cacheSet(pCtx, fmt.Sprint(pWalletAddresses), toCache, openseaAssetsTTL)
}

// NftOpenseaCacheGet gets a set of nfts from the opensea cache under a given wallet address
func NftOpenseaCacheGet(pCtx context.Context, pWalletAddresses []string, pRuntime *runtime.Runtime) ([]*Nft, error) {

	mp := newStorage(0, nftColName, pRuntime).withRedis(OpenseaAssetsRDB, pRuntime)
	defer mp.cacheClose()
	for i, v := range pWalletAddresses {
		pWalletAddresses[i] = strings.ToLower(v)
	}

	result, err := mp.cacheGet(pCtx, fmt.Sprint(pWalletAddresses))
	if err != nil {
		return nil, err
	}

	nfts := []*Nft{}
	if err := json.Unmarshal([]byte(result), &nfts); err != nil {
		return nil, err
	}
	return nfts, nil
}

func findDifference(nfts []*NftDB, dbNfts []*NftDB) ([]DBID, error) {
	currOpenseaIds := map[int]bool{}

	for _, v := range nfts {
		currOpenseaIds[v.OpenSeaID] = true
	}

	diff := []DBID{}
	for _, v := range dbNfts {
		if !currOpenseaIds[v.OpenSeaID] {
			diff = append(diff, v.ID)
		}
	}

	return diff, nil
}

func newNFTPipeline(matchFilter bson.M) mongo.Pipeline {

	return mongo.Pipeline{
		{{Key: "$match", Value: matchFilter}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "history",
			"localField":   "_id",
			"foreignField": "nft_id",
			"as":           "ownership_history",
		}}},
		{{Key: "$set", Value: bson.M{"ownership_history": bson.M{"$arrayElemAt": []interface{}{"$ownership_history", 0}}}}},
		{{Key: "$lookup", Value: bson.M{
			"from": "users",
			"let":  bson.M{"userID": "$owner_user_id"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{
						"$eq": []interface{}{"$_id", "$$userID"},
					},
				}}},
			},
			"as": "owner_user",
		}}},
		{{Key: "$set", Value: bson.M{"owner_user": bson.M{"$arrayElemAt": []interface{}{"$owner_user", 0}}}}},
		{{Key: "$set", Value: bson.M{"owner_username": "$owner_user.username"}}},
		{{Key: "$project", Value: bson.M{"owner_user": 0}}},
	}
}

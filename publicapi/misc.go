package publicapi

import (
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	"github.com/mikeydub/go-gallery/db/sqlc"
	"github.com/mikeydub/go-gallery/graphql/dataloader"
	"github.com/mikeydub/go-gallery/service/logger"
	"github.com/mikeydub/go-gallery/service/persist"
	"github.com/spf13/viper"
)

type MiscAPI struct {
	repos         *persist.Repositories
	queries       *sqlc.Queries
	loaders       *dataloader.Loaders
	validator     *validator.Validate
	ethClient     *ethclient.Client
	storageClient *storage.Client
}

func (api MiscAPI) GetGeneralAllowlist(ctx context.Context) ([]persist.Address, error) {
	// Nothing to validate

	bucket := viper.GetString("SNAPSHOT_BUCKET")
	logger.For(ctx).Infof("Proxying snapshot from bucket %s", bucket)

	obj := api.storageClient.Bucket(viper.GetString("SNAPSHOT_BUCKET")).Object("snapshot.json")

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}

	var addresses []persist.Address
	err = json.NewDecoder(r).Decode(&addresses)
	if err != nil {
		return nil, err
	}

	err = r.Close()
	if err != nil {
		return nil, err
	}

	return addresses, nil
}
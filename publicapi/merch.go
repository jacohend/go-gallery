package publicapi

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/mikeydub/go-gallery/contracts"
	"github.com/mikeydub/go-gallery/service/auth"
	"github.com/mikeydub/go-gallery/service/logger"
	"github.com/mikeydub/go-gallery/service/multichain"
	"github.com/mikeydub/go-gallery/service/persist/postgres"
	"github.com/spf13/viper"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"cloud.google.com/go/storage"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	db "github.com/mikeydub/go-gallery/db/gen/coredb"
	"github.com/mikeydub/go-gallery/graphql/dataloader"
	"github.com/mikeydub/go-gallery/graphql/model"
	"github.com/mikeydub/go-gallery/service/persist"
)

const (
	merchTypeTShirt = iota
	merchTypeHat
	merchTypeCard
)

var uriToMerchType = map[string]int{
	"ipfs://QmSWiQSXkxXhaoMJ2m9goR9DVXnyijdozE57jEsAwqLNZY": merchTypeTShirt,
	"ipfs://QmVXF8H7Xcnqr4oQXGEtoMCMnah8d6fBZuQQ5tcv9nL8Po": merchTypeHat,
	"ipfs://QmSPdA9Gg8xAdVxWvUyGkdFKQ8YMVYnGjYcr3cGMcBH1ae": merchTypeCard,
}

type merchAttribute struct {
	TraitType string `json:"trait_type"`
	Value     string `json:"value"`
}

type merchMetadata struct {
	Attributes []merchAttribute `json:"attributes"`
}

type MerchAPI struct {
	repos              *postgres.Repositories
	queries            *db.Queries
	loaders            *dataloader.Loaders
	validator          *validator.Validate
	ethClient          *ethclient.Client
	storageClient      *storage.Client
	multichainProvider *multichain.Provider
	secrets            *secretmanager.Client
}

func (api MerchAPI) GetMerchTokens(ctx context.Context, address persist.Address) ([]*model.MerchToken, error) {

	if err := validateFields(api.validator, validationMap{
		"address": {address, "required"},
	}); err != nil {
		return nil, err
	}

	merchAddress := viper.GetString("MERCH_CONTRACT_ADDRESS")

	tokens, err := api.multichainProvider.GetTokensOfContractForWallet(ctx, persist.Address(merchAddress), persist.NewChainAddress(address, persist.ChainETH), 0, 0)
	if err != nil {
		return nil, err
	}

	merchTokens := make([]*model.MerchToken, len(tokens))

	for i, token := range tokens {
		t := &model.MerchToken{
			TokenID: token.TokenID.String(),
		}
		discountCode, err := api.queries.GetMerchDiscountCodeByTokenID(ctx, token.TokenID)
		if err != nil && err != pgx.ErrNoRows {
			return nil, fmt.Errorf("failed to get discount code for token %v: %w", token.TokenID, err)
		}
		if discountCode.Valid && discountCode.String != "" {
			t.DiscountCode = &discountCode.String
			t.Redeemed = true
		}

		if token.TokenURI != "" {
			otype := uriToMerchType[token.TokenURI.String()]
			switch otype {
			case merchTypeTShirt:
				t.ObjectType = model.MerchTypeTShirt
			case merchTypeHat:
				t.ObjectType = model.MerchTypeHat
			case merchTypeCard:
				t.ObjectType = model.MerchTypeCard
			default:
				return nil, fmt.Errorf("unknown merch type for token %v", token.TokenID)
			}
		} else if token.TokenMetadata != nil && len(token.TokenMetadata) > 0 {
			// TokenURI should exist but since we have been talking about removing this field, I added this extra backup logic
			asBytes, err := json.Marshal(token.TokenMetadata)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal token metadata: %w", err)
			}
			var metadata merchMetadata
			if err := json.Unmarshal(asBytes, &metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal token metadata: %w", err)
			}
			for _, attr := range metadata.Attributes {
				if attr.TraitType == "Object" {
					switch attr.Value {
					case "001":
						t.ObjectType = model.MerchTypeTShirt
					case "002":
						t.ObjectType = model.MerchTypeHat
					case "003":
						t.ObjectType = model.MerchTypeCard
					default:
						return nil, fmt.Errorf("unknown merch type for token %v", token.TokenID)
					}
					break
				}
			}
		}

		merchTokens[i] = t
	}

	return merchTokens, nil
}

func (api MerchAPI) RedeemMerchItems(ctx context.Context, tokenIDs []persist.TokenID, address persist.ChainAddress, sig string, walletType persist.WalletType) ([]*model.MerchToken, error) {

	if err := validateFields(api.validator, validationMap{
		"tokenIDs": {tokenIDs, "required,unique"},
		"address":  {address, "required"},
		"sig":      {sig, "required"},
	}); err != nil {
		return nil, err
	}

	// check if user owns tokens

	merchAddress := viper.GetString("MERCH_CONTRACT_ADDRESS")

	mer, err := contracts.NewMerch(common.HexToAddress(merchAddress), api.ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate a Merch contract: %w", err)
	}

	for _, tokenID := range tokenIDs {
		owner, err := mer.OwnerOf(&bind.CallOpts{Context: ctx}, tokenID.BigInt())
		if err != nil {
			return nil, fmt.Errorf("failed to get owner of token %v: %w", tokenID, err)
		}
		logger.For(ctx).Infof("owner of token %v is %v", tokenID, owner.String())

		owns := strings.EqualFold(owner.String(), address.Address().String())
		if !owns {
			return nil, fmt.Errorf("user does not own token %v", tokenID)
		}
	}

	// verify signature

	// user should have signed the tokenIDs in place of the usual nonce
	valid, err := api.multichainProvider.VerifySignature(ctx, sig, fmt.Sprintf("%v", tokenIDs), persist.NewChainPubKey(persist.PubKey(address.Address()), address.Chain()), walletType)
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, auth.ErrSignatureInvalid
	}

	// redeem tokens on chain

	chainID, err := api.ethClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	if chainID.Cmp(big.NewInt(1)) == 0 && viper.GetString("ENV") == "production" {
		privateKey, err := api.secrets.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
			Name: "backend-eth-private-key",
		})
		if err != nil {
			return nil, err
		}

		key, err := x509.ParseECPrivateKey(privateKey.Payload.Data)
		if err != nil {
			return nil, err
		}

		auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create authorized transactor: %w", err)
		}
		auth.Context = ctx

		asBigs := make([]*big.Int, 0, len(tokenIDs))
		for _, tokenID := range tokenIDs {
			isRedeemed, err := mer.IsRedeemed(&bind.CallOpts{Context: ctx}, tokenID.BigInt())
			if err != nil {
				return nil, fmt.Errorf("failed to check if token %v is redeemed: %w", tokenID, err)
			}
			if !isRedeemed {
				asBigs = append(asBigs, tokenID.BigInt())
			}
		}

		if len(asBigs) > 0 {
			tx, err := mer.RedeemAdmin(auth, asBigs)
			if err != nil {
				return nil, fmt.Errorf("failed to redeem tokens: %w", err)
			}
			logger.For(ctx).Infof("redeemed merch items with tx: %s", tx.Hash())
		}

	}

	// redeem and return codes in DB

	result := make([]*model.MerchToken, 0, len(tokenIDs))
	for _, tokenID := range tokenIDs {

		uri, err := mer.TokenURI(&bind.CallOpts{Context: ctx}, tokenID.BigInt())
		if err != nil {
			return nil, fmt.Errorf("failed to get token URI for token %v: %w", tokenID, err)
		}

		logger.For(ctx).Debugf("token %v has URI %v", tokenID, uri)

		objectType, ok := uriToMerchType[uri]
		if ok {
			// first check if the token ID has already been redeemed, then just return the code
			discountCode, err := api.queries.GetMerchDiscountCodeByTokenID(ctx, tokenID)
			if err != nil {
				if err == pgx.ErrNoRows {
					logger.For(ctx).Debugf("failed to get discount code for token %v: %v", tokenID, err)
					// if not, redeem it
					discountCode, err = api.queries.RedeemMerch(ctx, db.RedeemMerchParams{
						TokenHex:   tokenID,
						ObjectType: int32(objectType),
					})
					if err != nil {
						return nil, fmt.Errorf("failed to redeem token %v: %w", tokenID, err)
					}
				} else {
					return nil, fmt.Errorf("failed to get discount code for token %v: %w", tokenID, err)
				}
			}

			var modelObjectType model.MerchType

			switch objectType {
			case merchTypeCard:
				modelObjectType = model.MerchTypeCard
			case merchTypeHat:
				modelObjectType = model.MerchTypeHat
			case merchTypeTShirt:
				modelObjectType = model.MerchTypeTShirt
			default:
				return nil, fmt.Errorf("unknown merch type %v", objectType)
			}

			t := &model.MerchToken{
				TokenID:    tokenID.String(),
				ObjectType: modelObjectType,
				Redeemed:   true,
			}

			if discountCode.Valid {
				t.DiscountCode = &discountCode.String
			} else {
				return nil, fmt.Errorf("discount code for token %v is null", tokenID)
			}
			result = append(result, t)
		} else {
			logger.For(ctx).Errorf("unknown merch type for %v", uri)
		}

	}

	return result, nil
}
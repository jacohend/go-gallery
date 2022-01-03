package indexer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/mikeydub/go-gallery/service/media"
	"github.com/mikeydub/go-gallery/service/persist"
	"github.com/mikeydub/go-gallery/service/rpc"
	"github.com/mikeydub/go-gallery/service/task"
	"github.com/mikeydub/go-gallery/util"
	"github.com/sirupsen/logrus"
)

var errInvalidUpdateMediaInput = errors.New("must provide either owner_address or token_id and contract_address")

type updateMediaInput struct {
	OwnerAddress    persist.Address `json:"owner_address"`
	TokenID         persist.TokenID `json:"token_id"`
	ContractAddress persist.Address `json:"contract_address"`
}

type tokenUpdateMedia struct {
	TokenDBID       persist.DBID
	TokenID         persist.TokenID
	ContractAddress persist.Address
	Update          persist.TokenUpdateMediaInput
}

func getStatus(i *Indexer, tokenRepository persist.TokenRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c, 10*time.Second)
		defer cancel()
		total, _ := tokenRepository.Count(ctx, persist.CountTypeTotal)
		mostRecent, _ := tokenRepository.MostRecentBlock(ctx)
		noMetadata, _ := tokenRepository.Count(ctx, persist.CountTypeNoMetadata)
		erc721, _ := tokenRepository.Count(ctx, persist.CountTypeERC721)
		erc1155, _ := tokenRepository.Count(ctx, persist.CountTypeERC1155)

		c.JSON(http.StatusOK, gin.H{
			"total_tokens": total,
			"recent_block": i.mostRecentBlock,
			"most_recent":  mostRecent,
			"bad_uris":     i.badURIs,
			"no_metadata":  noMetadata,
			"erc721":       erc721,
			"erc1155":      erc1155,
		})
	}
}

func updateMedia(tq *task.Queue, tokenRepository persist.TokenRepository, ethClient *ethclient.Client, ipfsClient *shell.Shell, storageClient *storage.Client) gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		input := updateMediaInput{}
		if err := ginContext.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(ginContext, http.StatusBadRequest, err)
			return
		}

		var tokens []persist.Token
		var key string
		if input.OwnerAddress != "" {
			t, err := tokenRepository.GetByWallet(ginContext, input.OwnerAddress, -1, -1)
			if err != nil {
				util.ErrResponse(ginContext, http.StatusInternalServerError, err)
				return
			}
			tokens = t
			key = input.OwnerAddress.String()
		} else if input.TokenID != "" && input.ContractAddress != "" {
			t, err := tokenRepository.GetByTokenIdentifiers(ginContext, input.TokenID, input.ContractAddress, 1, 0)
			if err != nil {
				util.ErrResponse(ginContext, http.StatusInternalServerError, err)
				return
			}
			tokens = t
			key = persist.NewTokenIdentifiers(input.ContractAddress, input.TokenID).String()
		} else {
			util.ErrResponse(ginContext, http.StatusBadRequest, errInvalidUpdateMediaInput)
			return
		}
		c := ginContext.Copy()
		task := func() {
			updateChan := make(chan tokenUpdateMedia)
			errChan := make(chan error)
			for _, t := range tokens {
				go func(token persist.Token) {

					uri := token.TokenURI
					metadata := token.TokenMetadata
					med := token.Media
					ctx, cancel := context.WithTimeout(c, 10*time.Second)
					defer cancel()
					if uri == persist.InvalidTokenURI {
						errChan <- nil
						return
					}

					if _, ok := metadata["error"]; ok {
						errChan <- nil
						return
					}

					if med.MediaType == persist.MediaTypeInvalid {
						errChan <- nil
						return
					}

					if uri.Type() == persist.URITypeNone {
						u, err := rpc.GetTokenURI(ctx, token.TokenType, token.ContractAddress, token.TokenID, ethClient)
						if err != nil {
							errChan <- fmt.Errorf("failed to get token URI: %v", err)
							return
						}
						uri = u
					}

					if metadata == nil || len(metadata) == 0 {
						md, err := rpc.GetMetadataFromURI(token.TokenURI, ipfsClient)
						if err != nil {
							errChan <- fmt.Errorf("failed to get metadata for token %s: %v", token.TokenID, err)
							return
						}
						metadata = md
					}

					if med.MediaType == "" && med.MediaURL == "" {
						m, err := media.MakePreviewsForMetadata(ctx, metadata, token.ContractAddress, token.TokenID, uri, ipfsClient, storageClient)
						if err != nil {
							errChan <- fmt.Errorf("failed to make media for token %s: %v", token.TokenID, err)
							return
						}
						med = m
					}

					updateChan <- tokenUpdateMedia{
						TokenDBID:       token.ID,
						TokenID:         token.TokenID,
						ContractAddress: token.ContractAddress,
						Update: persist.TokenUpdateMediaInput{
							TokenURI: uri,
							Metadata: metadata,
							Media:    med,
						},
					}
				}(t)
			}

			for i := 0; i < len(tokens); i++ {
				select {
				case update := <-updateChan:
					if input.OwnerAddress != "" {
						if err := tokenRepository.UpdateByIDUnsafe(c, update.TokenDBID, update.Update); err != nil {
							logrus.WithError(err).Error("failed to update token in database")
							return
						}
					} else if input.ContractAddress != "" && input.TokenID != "" {
						if err := tokenRepository.UpdateByTokenIdentifiersUnsafe(c, update.TokenID, update.ContractAddress, update.Update); err != nil {
							logrus.WithError(err).Error("failed to update token in database")
							return
						}
					}
				case err := <-errChan:
					if err != nil {
						logrus.WithError(err).Error("failed to update media for token")
						return
					}
				}
			}
		}
		tq.QueueTask(key, task)
	}
}
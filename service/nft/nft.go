package nft

import (
	"context"
	"errors"

	"github.com/mikeydub/go-gallery/service/persist"
)

var errInvalidPreviewsInput = errors.New("user_id or username required for previews")

// GetPreviewsForUserInput is the input for receiving at most 3 image previews for the first NFTs displayed in a user's gallery
type GetPreviewsForUserInput struct {
	UserID   persist.DBID `form:"user_id"`
	Username string       `form:"username"`
}

// GetPreviewsForUser returns a slice of 3 preview URLs from a user's collections
func GetPreviewsForUser(pCtx context.Context, galleryRepo persist.GalleryRepository, userRepo persist.UserRepository, u GetPreviewsForUserInput) ([]persist.NullString, error) {
	var galleries []persist.Gallery
	var err error
	if u.UserID != "" {
		galleries, err = galleryRepo.GetByUserID(pCtx, u.UserID)
	} else if u.Username != "" {
		user, err := userRepo.GetByUsername(pCtx, u.Username)
		if err != nil {
			return nil, err
		}
		galleries, err = galleryRepo.GetByUserID(pCtx, user.ID)
	} else {
		return nil, errInvalidPreviewsInput
	}
	if err != nil {
		return nil, err
	}
	result := make([]persist.NullString, 0, 3)

	for _, g := range galleries {
		previews := GetPreviewsFromCollections(g.Collections)
		result = append(result, previews...)
		if len(result) > 2 {
			break
		}
	}
	if len(result) > 3 {
		return result[:3], nil
	}
	return result, nil
}

// GetPreviewsForUserToken returns a slice of 3 preview URLs from a user's collections
func GetPreviewsForUserToken(pCtx context.Context, galleryRepo persist.GalleryTokenRepository, userRepo persist.UserRepository, u GetPreviewsForUserInput) ([]persist.NullString, error) {
	var galleries []persist.GalleryToken
	var err error
	if u.UserID != "" {
		galleries, err = galleryRepo.GetByUserID(pCtx, u.UserID)
	} else if u.Username != "" {
		user, err := userRepo.GetByUsername(pCtx, u.Username)
		if err != nil {
			return nil, err
		}
		galleries, err = galleryRepo.GetByUserID(pCtx, user.ID)
	} else {
		return nil, errInvalidPreviewsInput
	}
	if err != nil {
		return nil, err
	}
	result := make([]persist.NullString, 0, 3)

	for _, g := range galleries {
		previews := GetPreviewsFromCollectionsToken(g.Collections)
		result = append(result, previews...)
		if len(result) > 2 {
			break
		}
	}
	if len(result) > 3 {
		return result[:3], nil
	}
	return result, nil
}

// GetPreviewsFromCollections returns a slice of 3 preview URLs from a slice of CollectionTokens
func GetPreviewsFromCollections(pColls []persist.Collection) []persist.NullString {
	result := make([]persist.NullString, 0, 3)

outer:
	for _, c := range pColls {
		for _, n := range c.NFTs {
			if n.ImageThumbnailURL != "" {
				result = append(result, n.ImageThumbnailURL)
			}
			if len(result) > 2 {
				break outer
			}
		}
		if len(result) > 2 {
			break outer
		}
	}
	return result

}

// GetPreviewsFromCollectionsToken returns a slice of 3 preview URLs from a slice of CollectionTokens
func GetPreviewsFromCollectionsToken(pColls []persist.CollectionToken) []persist.NullString {
	result := make([]persist.NullString, 0, 3)

outer:
	for _, c := range pColls {
		for _, n := range c.NFTs {
			if n.Media.ThumbnailURL != "" {
				result = append(result, n.Media.ThumbnailURL)
			}
			if len(result) > 2 {
				break outer
			}
		}
		if len(result) > 2 {
			break outer
		}
	}
	return result

}
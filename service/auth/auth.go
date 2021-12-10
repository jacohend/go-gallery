package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mikeydub/go-gallery/contracts"
	"github.com/mikeydub/go-gallery/middleware"
	"github.com/mikeydub/go-gallery/service/eth"
	"github.com/mikeydub/go-gallery/service/persist"
	"github.com/sirupsen/logrus"
)

// WalletType is the type of wallet used to sign a message
type WalletType int

const (
	// WalletTypeEOA represents an externally owned account (regular wallet address)
	WalletTypeEOA WalletType = iota
	// WalletTypeGnosis represents a smart contract gnosis safe
	WalletTypeGnosis
)

const noncePrepend = "Gallery uses this cryptographic signature in place of a password, verifying that you are the owner of this Ethereum address: "

var errAddressSignatureMismatch = errors.New("address does not match signature")

var eip1271MagicValue = [4]byte{0x16, 0x26, 0xBA, 0x7E}

// LoginInput is the input to the login pipeline
type LoginInput struct {
	Signature  string          `json:"signature" binding:"signature"`
	Address    persist.Address `json:"address"   binding:"required,eth_addr"` // len=42"` // standard ETH "0x"-prefixed address
	WalletType WalletType      `json:"wallet_type"`
	Nonce      string          `json:"nonce"`
}

// LoginOutput is the output of the login pipeline
type LoginOutput struct {
	SignatureValid bool            `json:"signature_valid"`
	JWTtoken       string          `json:"jwt_token"`
	UserID         persist.DBID    `json:"user_id"`
	Address        persist.Address `json:"address"`
}

// GetPreflightInput is the input to the preflight pipeline
type GetPreflightInput struct {
	Address persist.Address `json:"address" form:"address" binding:"required,eth_addr"` // len=42"` // standard ETH "0x"-prefixed address
}

// GetPreflightOutput is the output of the preflight pipeline
type GetPreflightOutput struct {
	Nonce      string `json:"nonce"`
	UserExists bool   `json:"user_exists"`
}

type errAddressDoesNotOwnRequiredNFT struct {
	address persist.Address
}

func (e errAddressDoesNotOwnRequiredNFT) Error() string {
	return fmt.Sprintf("required tokens not owned by address: %s", e.address)
}

// ErrUserNotFound is returned when a user is not found
type ErrUserNotFound struct {
	UserID   persist.DBID
	Address  persist.Address
	Username string
}

func (e ErrUserNotFound) Error() string {
	return fmt.Sprintf("user not found: address: %s, ID: %s, Username: %s", e.Address, e.UserID, e.Username)
}

// generateNonce generates a random nonce to be signed by a wallet
func generateNonce() string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	nonceInt := seededRand.Int()
	nonceStr := fmt.Sprintf("%d", nonceInt)
	return nonceStr
}

// LoginAndMemorizeAttempt will run the login pipeline and memorize the result
func LoginAndMemorizeAttempt(pCtx context.Context, pInput LoginInput,
	pReq *http.Request, userRepo persist.UserRepository, nonceRepo persist.NonceRepository,
	loginRepo persist.LoginAttemptRepository, ec *ethclient.Client) (LoginOutput, error) {

	output, err := LoginPipeline(pCtx, pInput, userRepo, nonceRepo, ec)
	if err != nil {
		return LoginOutput{}, err
	}

	loginAttempt := persist.UserLoginAttempt{

		Address:        pInput.Address,
		Signature:      pInput.Signature,
		SignatureValid: output.SignatureValid,

		ReqHostAddr: pReq.RemoteAddr,
		ReqHeaders:  map[string][]string(pReq.Header),
	}

	_, err = loginRepo.Create(pCtx, loginAttempt)
	if err != nil {
		return LoginOutput{}, err
	}

	return output, err
}

// LoginPipeline logs in a user by validating their signed nonce
func LoginPipeline(pCtx context.Context, pInput LoginInput, userRepo persist.UserRepository,
	nonceRepo persist.NonceRepository, ec *ethclient.Client) (LoginOutput, error) {

	output := LoginOutput{}

	nonce, userID, err := GetUserWithNonce(pCtx, pInput.Address, userRepo, nonceRepo)
	if err != nil {
		return LoginOutput{}, err
	}

	if pInput.WalletType != WalletTypeEOA {
		if nonce != pInput.Nonce {
			output.SignatureValid = false
			return output, nil
		}
	}

	sigValid, err := VerifySignatureAllMethods(pInput.Signature,
		nonce,
		pInput.Address, pInput.WalletType, ec)
	if err != nil {
		return LoginOutput{}, err
	}

	output.SignatureValid = sigValid
	if !sigValid {
		return output, nil
	}

	output.UserID = userID

	jwtTokenStr, err := middleware.JWTGeneratePipeline(pCtx, userID)
	if err != nil {
		return LoginOutput{}, err
	}

	output.JWTtoken = jwtTokenStr

	err = NonceRotate(pCtx, pInput.Address, userID, nonceRepo)
	if err != nil {
		return LoginOutput{}, err
	}

	return output, nil
}

// VerifySignatureAllMethods will verify a signature using all available methods (eth_sign and personal_sign)
func VerifySignatureAllMethods(pSignatureStr string,
	pNonce string,
	pAddressStr persist.Address, pWalletType WalletType, ec *ethclient.Client) (bool, error) {

	// personal_sign
	validBool, err := VerifySignature(pSignatureStr,
		pNonce,
		pAddressStr, pWalletType,
		true, ec)

	if !validBool || err != nil {
		// eth_sign
		validBool, err = VerifySignature(pSignatureStr,
			pNonce,
			pAddressStr, pWalletType,
			false, ec)
	}

	if err != nil {
		return false, err
	}

	return validBool, nil
}

// VerifySignature will verify a signature using either personal_sign or eth_sign
func VerifySignature(pSignatureStr string,
	pDataStr string,
	pAddress persist.Address, pWalletType WalletType,
	pUseDataHeaderBool bool, ec *ethclient.Client) (bool, error) {

	// eth_sign:
	// - https://goethereumbook.org/signature-verify/
	// - http://man.hubwiz.com/docset/Ethereum.docset/Contents/Resources/Documents/eth_sign.html
	// - sign(keccak256("\x19Ethereum Signed Message:\n" + len(message) + message)))

	nonceWithPrepend := noncePrepend + pDataStr

	var dataStr string
	if pUseDataHeaderBool {
		dataStr = fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(nonceWithPrepend), nonceWithPrepend)
	} else {
		dataStr = nonceWithPrepend
	}

	switch pWalletType {
	case WalletTypeEOA:
		dataHash := crypto.Keccak256Hash([]byte(dataStr))

		sig, err := hexutil.Decode(pSignatureStr)
		if err != nil {
			return false, err
		}
		// Ledger-produced signatures have v = 0 or 1
		if sig[64] == 0 || sig[64] == 1 {
			sig[64] += 27
		}
		v := sig[64]
		if v != 27 && v != 28 {
			return false, errors.New("invalid signature (V is not 27 or 28)")
		}
		sig[64] -= 27

		sigPublicKeyECDSA, err := crypto.SigToPub(dataHash.Bytes(), sig)
		if err != nil {
			return false, err
		}

		pubkeyAddressHexStr := crypto.PubkeyToAddress(*sigPublicKeyECDSA).Hex()
		log.Println("pubkeyAddressHexStr:", pubkeyAddressHexStr)
		log.Println("pAddress:", pAddress)
		if !strings.EqualFold(pubkeyAddressHexStr, pAddress.String()) {
			return false, errAddressSignatureMismatch
		}

		publicKeyBytes := crypto.CompressPubkey(sigPublicKeyECDSA)

		signatureNoRecoverID := sig[:len(sig)-1]

		return crypto.VerifySignature(publicKeyBytes, dataHash.Bytes(), signatureNoRecoverID), nil
	case WalletTypeGnosis:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		sigValidator, err := contracts.NewISignatureValidator(pAddress.Address(), ec)
		if err != nil {
			return false, err
		}

		hashedData := crypto.Keccak256([]byte(dataStr))
		var input [32]byte
		copy(input[:], hashedData)

		result, err := sigValidator.IsValidSignature(&bind.CallOpts{Context: ctx}, input, []byte{})
		if err != nil {
			logrus.WithError(err).Error("IsValidSignature")
			return false, nil
		}

		return result == eip1271MagicValue, nil
	default:
		return false, errors.New("wallet type not supported")
	}

}

// GetPreflight will establish if a user is permitted to preflight a login and generate a nonce to be signed
func GetPreflight(pCtx context.Context, pInput GetPreflightInput, pPreAuthed bool,
	userRepo persist.UserRepository, nonceRepo persist.NonceRepository, ethClient *eth.Client) (*GetPreflightOutput, error) {

	user, err := userRepo.GetByAddress(pCtx, pInput.Address)

	logrus.WithError(err).Error("error retrieving user by address for auth preflight")

	userExistsBool := user.ID != ""

	output := &GetPreflightOutput{
		UserExists: userExistsBool,
	}
	if !userExistsBool {

		if !pPreAuthed {

			hasNFT, err := ethClient.HasNFTs(pCtx, middleware.RequiredNFTs, pInput.Address)
			if err != nil {
				return nil, err
			}
			if !hasNFT {
				return nil, errAddressDoesNotOwnRequiredNFT{pInput.Address}
			}

		}

		nonce := persist.UserNonce{
			Address: pInput.Address,
			Value:   generateNonce(),
		}

		err := nonceRepo.Create(pCtx, nonce)
		if err != nil {
			return nil, err
		}
		output.Nonce = noncePrepend + nonce.Value

	} else {
		nonce, err := nonceRepo.Get(pCtx, pInput.Address)
		if err != nil {
			return nil, err
		}
		output.Nonce = noncePrepend + nonce.Value
	}

	return output, nil
}

// NonceRotate will rotate a nonce for a user
func NonceRotate(pCtx context.Context, pAddress persist.Address, pUserID persist.DBID, nonceRepo persist.NonceRepository) error {

	newNonce := persist.UserNonce{
		Value:   generateNonce(),
		Address: pAddress,
	}

	err := nonceRepo.Create(pCtx, newNonce)
	if err != nil {
		return err
	}
	return nil
}

// GetUserWithNonce returns nonce value string, user id
// will return empty strings and error if no nonce found
// will return empty string if no user found
func GetUserWithNonce(pCtx context.Context, pAddress persist.Address, userRepo persist.UserRepository, nonceRepo persist.NonceRepository) (nonceValue string, userID persist.DBID, err error) {

	nonce, err := nonceRepo.Get(pCtx, pAddress)
	if err != nil {
		return nonceValue, userID, err
	}

	nonceValue = nonce.Value

	user, err := userRepo.GetByAddress(pCtx, pAddress)
	if err != nil {
		return nonceValue, userID, err
	}
	if user.ID != "" {
		userID = user.ID
	} else {
		return nonceValue, userID, ErrUserNotFound{Address: pAddress}
	}

	return nonceValue, userID, nil
}
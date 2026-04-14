package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/chains/evmutil"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/confighelper"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3confighelper"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	ragetypes "github.com/smartcontractkit/libocr/ragep2p/types"
	"golang.org/x/crypto/curve25519"
)

func envInt(name string) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return 0, fmt.Errorf("%s is required", name)
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}

	return parsed, nil
}

func envInt64(name string) (int64, error) {
	value := os.Getenv(name)
	if value == "" {
		return 0, fmt.Errorf("%s is required", name)
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}

	return parsed, nil
}

func oracleIdentities(n int, seed int64) ([]confighelper.OracleIdentityExtra, error) {
	oracles := make([]confighelper.OracleIdentityExtra, 0, n)

	for i := 0; i < n; i++ {
		offSeed := sha256.Sum256([]byte(fmt.Sprintf("offchain|%d|%d", seed, i)))
		offPriv := ed25519.NewKeyFromSeed(offSeed[:])

		cfgSeed := sha256.Sum256([]byte(fmt.Sprintf("cfg|%d|%d", seed, i)))
		cfgPub, err := curve25519.X25519(cfgSeed[:], curve25519.Basepoint)
		if err != nil {
			return nil, err
		}

		privateKeyHex := os.Getenv(fmt.Sprintf("ORACLE%d_PRIVATE_KEY", i))
		if privateKeyHex == "" {
			return nil, fmt.Errorf("ORACLE%d_PRIVATE_KEY is required", i)
		}

		privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
		if err != nil {
			return nil, fmt.Errorf("ORACLE%d_PRIVATE_KEY is invalid: %w", i, err)
		}

		peerID, err := ragetypes.PeerIDFromPrivateKey(offPriv)
		if err != nil {
			return nil, err
		}

		var offchainPublicKey ocrtypes.OffchainPublicKey
		copy(offchainPublicKey[:], offPriv.Public().(ed25519.PublicKey))

		var configEncryptionPublicKey ocrtypes.ConfigEncryptionPublicKey
		copy(configEncryptionPublicKey[:], cfgPub)

		address := crypto.PubkeyToAddress(privateKey.PublicKey)
		oracles = append(oracles, confighelper.OracleIdentityExtra{
			OracleIdentity: confighelper.OracleIdentity{
				OffchainPublicKey: offchainPublicKey,
				OnchainPublicKey:  address.Bytes(),
				PeerID:            peerID.String(),
				TransmitAccount:   ocrtypes.Account(address.Hex()),
			},
			ConfigEncryptionPublicKey: configEncryptionPublicKey,
		})
	}

	return oracles, nil
}

func buildContractConfig(n, f int, seed int64) (ocrtypes.ContractConfig, evmutil.EVMOffchainConfigDigester, error) {
	oracles, err := oracleIdentities(n, seed)
	if err != nil {
		return ocrtypes.ContractConfig{}, evmutil.EVMOffchainConfigDigester{}, err
	}

	transmissionSchedule := []int{1, 2, 4}
	signers, transmitters, fOut, onchain, version, offchain, err := ocr3confighelper.ContractSetConfigArgsDeterministic(
		sha256.Sum256([]byte(fmt.Sprintf("sk|%d", seed))),
		[16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		45*time.Second,
		20*time.Second,
		10*time.Second,
		5*time.Second,
		2*time.Second,
		2*time.Second,
		10*time.Second,
		5,
		transmissionSchedule,
		oracles,
		[]byte("{}"),
		nil,
		20*time.Second,
		45*time.Second,
		20*time.Second,
		90*time.Second,
		f,
		nil,
	)
	if err != nil {
		return ocrtypes.ContractConfig{}, evmutil.EVMOffchainConfigDigester{}, err
	}

	address := common.HexToAddress("0x" + strings.Repeat("11", 20))
	digester := evmutil.EVMOffchainConfigDigester{
		ChainID:         1337,
		ContractAddress: address,
	}
	contractConfig := ocrtypes.ContractConfig{
		ConfigCount:           1,
		Signers:               signers,
		Transmitters:          transmitters,
		F:                     fOut,
		OnchainConfig:         onchain,
		OffchainConfigVersion: version,
		OffchainConfig:        offchain,
	}

	contractConfig.ConfigDigest, err = digester.ConfigDigest(context.Background(), contractConfig)
	if err != nil {
		return ocrtypes.ContractConfig{}, evmutil.EVMOffchainConfigDigester{}, err
	}

	return contractConfig, digester, nil
}

func main() {
	n, err := envInt("DIGEST_NUM_ORACLES")
	if err != nil {
		panic(err)
	}

	f, err := envInt("DIGEST_FAULT_TOLERANCE")
	if err != nil {
		panic(err)
	}

	seed, err := envInt64("DIGEST_OCR_SEED")
	if err != nil {
		panic(err)
	}

	contractConfig, _, err := buildContractConfig(n, f, seed)
	if err != nil {
		panic(err)
	}

	digest := contractConfig.ConfigDigest.Hex()
	if !strings.HasPrefix(digest, "0x") {
		digest = "0x" + digest
	}

	fmt.Printf("CONFIG_DIGEST=%s\n", digest)
}

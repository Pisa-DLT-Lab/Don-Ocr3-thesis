package main 

import(
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/confighelper" 
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	ragetypes "github.com/smartcontractkit/libocr/ragep2p/types"
	"golang.org/x/crypto/curve25519"
)

// =============================================================================
// KEY MANAGEMENT (Hybrid: Deterministic + Hardhat)
// =============================================================================

// ecdsaOnchainKeyring implements the OCR3 "onchain keyring" interface:
// it is responsible for signing/verifying report bytes for the on-chain verifier.

type ecdsaOnchainKeyring struct{ 
	priv *ecdsa.PrivateKey 
}

// PublicKey returns the onchain public key used to identify this oracle to the
// on-chain verifier. Here that's the Ethereum address derived from the ECDSA pubkey.
func (k *ecdsaOnchainKeyring) PublicKey() ocrtypes.OnchainPublicKey {
	addr := crypto.PubkeyToAddress(k.priv.PublicKey)
	return addr.Bytes()
}

// reportSigHash defines WHAT exactly is signed for a report.
//
// In OCR3 the contract verifies signatures over (configDigest, seqNr, reportBytes).
// The exact hashing scheme is verifier-dependent; this experiment uses a simple
// keccak256 over concatenation, which is enough to simulate signature collection.
func reportSigHash(configDigest ocrtypes.ConfigDigest, seqNr uint64, report ocrtypes.Report) common.Hash {
	var seq [8]byte
	binary.BigEndian.PutUint64(seq[:], seqNr)
	payload := make([]byte, 0, len(configDigest)+len(seq)+len(report))
	payload = append(payload, configDigest[:]...)
	payload = append(payload, seq[:]...)
	payload = append(payload, report...)
	return crypto.Keccak256Hash(payload)
}

// Sign produces an on-chain signature for the given report at (configDigest, seqNr).
func (k *ecdsaOnchainKeyring) Sign(configDigest ocrtypes.ConfigDigest, seqNr uint64, reportWithInfo ocr3types.ReportWithInfo[struct{}]) ([]byte, error) {
	h := reportSigHash(configDigest, seqNr, reportWithInfo.Report)
	return crypto.Sign(h.Bytes(), k.priv)
}

// Verify checks that sig is a valid signature (by pubkey) over (configDigest, seqNr, report).
func (k *ecdsaOnchainKeyring) Verify(pubkey ocrtypes.OnchainPublicKey, configDigest ocrtypes.ConfigDigest, seqNr uint64, reportWithInfo ocr3types.ReportWithInfo[struct{}], sig []byte) bool {
	if len(pubkey) != 20 {
		return false
	}
	if len(sig) != 65 {
		return false
	}
	h := reportSigHash(configDigest, seqNr, reportWithInfo.Report)
	recovered, err := crypto.SigToPub(h.Bytes(), sig)
	if err != nil {
		return false
	}
	addr := crypto.PubkeyToAddress(*recovered).Bytes()
	return bytes.Equal(addr, pubkey)
}

// MaxSignatureLength tells libocr the maximum size (bytes) of signatures this
// onchain keyring produces. For secp256k1 signatures in geth format this is 65.
func (k *ecdsaOnchainKeyring) MaxSignatureLength() int { return 65 }

type offchainKeyring struct {
	offPriv ed25519.PrivateKey
	cfgPriv [32]byte
	cfgPub  [32]byte
}

// newOffchainKeyringDeterministic derives an offchain+config keypair from (seed, oracleID).
//
// WARNING: this is for experimentation only. Deterministic keys are insecure in production.
// We do it here so every run produces the same identities/configDigest.
func newOffchainKeyringDeterministic(seed int64, oracleID int) (*offchainKeyring, error) {
	offSeed := sha256.Sum256([]byte(fmt.Sprintf("offchain|%d|%d", seed, oracleID)))
	offPriv := ed25519.NewKeyFromSeed(offSeed[:])

	cfgSeed := sha256.Sum256([]byte(fmt.Sprintf("cfg|%d|%d", seed, oracleID)))
	var cfgPriv [32]byte
	copy(cfgPriv[:], cfgSeed[:])

	cfgPub, err := curve25519.X25519(cfgPriv[:], curve25519.Basepoint)
	if err != nil {
		return nil, err
	}
	var cfgPubArr [32]byte
	copy(cfgPubArr[:], cfgPub)
	return &offchainKeyring{offPriv: offPriv, cfgPriv: cfgPriv, cfgPub: cfgPubArr}, nil
}

// OffchainSign signs protocol messages (NOT reports). OCR3 uses Ed25519 for
// offchain message authentication.
func (k *offchainKeyring) OffchainSign(msg []byte) ([]byte, error) {
	return ed25519.Sign(k.offPriv, msg), nil
}

// ConfigDiffieHellman derives a shared secret used to encrypt offchain config
// values between oracles (X25519).
func (k *offchainKeyring) ConfigDiffieHellman(point [32]byte) ([32]byte, error) {
	out, err := curve25519.X25519(k.cfgPriv[:], point[:])
	var r [32]byte
	copy(r[:], out)
	return r, err
}

// OffchainPublicKey returns the Ed25519 public key used for offchain message verification.
func (k *offchainKeyring) OffchainPublicKey() ocrtypes.OffchainPublicKey {
	var pub ocrtypes.OffchainPublicKey
	copy(pub[:], k.offPriv.Public().(ed25519.PublicKey))
	return pub
}

// ConfigEncryptionPublicKey returns the oracle's config encryption public key.
func (k *offchainKeyring) ConfigEncryptionPublicKey() ocrtypes.ConfigEncryptionPublicKey {
	return k.cfgPub
}

// newOnchainKeyringDeterministic derives an ECDSA keypair (secp256k1) from (seed, oracleID).
//
// We try multiple nonces because not all 32-byte seeds are valid secp256k1 private keys.
// WARNING: deterministic keys are insecure in production.
func newOnchainKeyringDeterministic(seed int64, oracleID int) (*ecdsaOnchainKeyring, ocrtypes.Account, error) {
	// 1. Build the name of the env variable (es. ORACLE0_PRIVATE_KEY)
	envVarName := fmt.Sprintf("ORACLE%d_PRIVATE_KEY", oracleID)

	// 2. Read the value from .env
	privKeyHex := os.Getenv(envVarName)
	if privKeyHex == "" {
		// In the docker-compose is missing the 'env_file' or the variable 
		return nil, "", fmt.Errorf("Missing variable %s in the .env", envVarName)
	}

	// 3. Parsing and trimming of 0x prefix
	privKeyHex = strings.TrimPrefix(privKeyHex, "0x")

	priv, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return nil, "", fmt.Errorf("Key parsing error %s: %v", envVarName, err)
	}

	kr := &ecdsaOnchainKeyring{priv: priv}
	addr := crypto.PubkeyToAddress(priv.PublicKey)

	return kr, ocrtypes.Account(addr.Hex()), nil
}


// =============================================================================
// DERIVE THE CONFIGURATION OF THE NODES USING THE KEYS
// =============================================================================
// derivedNode bundles everything we need to run an oracle process.
// We derive it deterministically so N separate containers all agree on identities.
type derivedNode struct { 
	offKR 		*offchainKeyring
	onKR 		*ecdsaOnchainKeyring
	fromAcct 	ocrtypes.Account
	peerID 		string 
}

// deriveAllNodes deterministically derives N oracle identities and returns:
// - local helper structs (keys + peerID) for each oracle
// - libocr-compatible OracleIdentityExtra entries for config generation
func deriveAllNodes(n int, seed int64) ([]derivedNode, []confighelper.OracleIdentityExtra, error) {
	nodes := make([]derivedNode, 0, n)
	oracles := make([]confighelper.OracleIdentityExtra, 0, n)
	for i := 0; i < n; i++ {
		off, err := newOffchainKeyringDeterministic(seed, i)
		if err != nil {
			return nil, nil, err
		}

		on, from, err := newOnchainKeyringDeterministic(seed, i)
		if err != nil {
            return nil, nil, fmt.Errorf("Error in deriveAllNodes %d: %v", i, err)
        }

		pid, err := ragetypes.PeerIDFromPrivateKey(off.offPriv)
		if err != nil {
			return nil, nil, err
		}

		nodes = append(nodes, derivedNode{off, on, from, pid.String()})
		oracles = append(oracles, confighelper.OracleIdentityExtra{
			OracleIdentity: confighelper.OracleIdentity{
				OffchainPublicKey: off.OffchainPublicKey(), 
				OnchainPublicKey: on.PublicKey(),
				PeerID: pid.String(), 
				TransmitAccount: from,
			}, 
			ConfigEncryptionPublicKey: off.ConfigEncryptionPublicKey(),
		})
	}
	return nodes, oracles, nil
}
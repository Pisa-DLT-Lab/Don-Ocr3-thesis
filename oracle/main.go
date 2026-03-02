package main

import (
	"context"
//	"crypto/ecdsa"
//	"crypto/ed25519"
//	"crypto/sha256"
//	"encoding/json"
//	"errors"
	"flag"
	"fmt"
	"log"
//	"math/big"
	"math/rand"
//	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

//	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
//	gethtypes "github.com/ethereum/go-ethereum/core/types"
//	"github.com/ethereum/go-ethereum/crypto"
//	"github.com/ethereum/go-ethereum/ethclient"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/networking"
	"github.com/smartcontractkit/libocr/offchainreporting2plus"
//	"github.com/smartcontractkit/libocr/offchainreporting2plus/chains/evmutil"
//	"github.com/smartcontractkit/libocr/offchainreporting2plus/confighelper" 
//	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3confighelper"
//	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2plus/types"
//	"github.com/smartcontractkit/libocr/quorumhelper"
	ragetypes "github.com/smartcontractkit/libocr/ragep2p/types"
)

// =============================================================================
// RUNNERS (Node Bootstrapping & P2P Networking)
// =============================================================================

// runBootstrap initializes the Bootstrap Node.
// This node does not participate in the OCR3 consensus (it doesn't propose or sign data).
// It handles the Oracles discovery of peer by communicating with them and giving the other peer informations
func runBootstrap(ctx context.Context, n, f int, seed int64, listen, announce string) error {
	cc, _, err := buildContractConfig(n, f, seed)
	if err != nil { return err }
	
	_, oracles, err := deriveAllNodes(n, seed)
	if err != nil {	return err }
	
	peerIDs := make([]string, 0, n)
	for _, o := range oracles { 
		peerIDs = append(peerIDs, o.PeerID) 
	}
	
	// DNS Resolution: useful for Docker
	announceIP, err := resolveHostnamePortToIP(announce)
	if err != nil {
		return fmt.Errorf("Resolve bootstrap_announce %q: %w", announce, err)
	}

	priv := deriveBootstrapPriv(seed)
	pid, err := ragetypes.PeerIDFromPrivateKey(priv)
	if err != nil {
		return err
	}

	// Initialize the networking layer (libp2p)
	peer, err := networking.NewPeer(networking.PeerConfig{
		PrivKey: 				priv, 
		Logger: 				quietLogger{log.New(os.Stdout, "", 0)}, 
		V2ListenAddresses: 		[]string{listen}, 
		V2AnnounceAddresses: 	[]string{announceIP},
		V2DeltaReconcile:		10 * time.Second,
		V2EndpointConfig: networking.EndpointConfigV2{
			IncomingMessageBufferSize: 100,
			OutgoingMessageBufferSize: 50,
		},
		MetricsRegisterer:		prometheus.NewRegistry(),
	})
	if err != nil { return err }

	bootstrapper, err := peer.OCR2BootstrapperFactory().NewBootstrapper(cc.ConfigDigest, peerIDs, nil, f)
	if err != nil {
		return err
	}
	if err := bootstrapper.Start(); err != nil {
		return err
	}
	defer func() { _ = bootstrapper.Close() }()

	fmt.Printf("Bootstrap peerId=%s listen=%s announce=%s configDigest=%s\n", pid.String(), listen, announceIP, cc.ConfigDigest.Hex())
	<-ctx.Done()
	return nil
}

// runOracle assembles and starts an OCR3 Oracle node.
// It connects the Off-chain (IPFS, AI computation) with 
// the On-Chain components (Smart contract),
func runOracle(ctx context.Context, id, n, f int, seed int64, bootAddr, listen, announce string) error {
	// ==========================================
	// 1. Environment Configuration & SECRETS
	// ==========================================	
	privKeyHex := os.Getenv("PRIVATE_KEY")
	if privKeyHex == "" {
		log.Fatal("Critical Error: PRIVATE KEY variable not found. Check the docker-compose.yml")
	}

	// Remove eventual 0x prefix
    privKeyHex = strings.TrimPrefix(privKeyHex, "0x")

	// Retrieval of the smart contract addresses from .env
	verifierAddressHex := os.Getenv("VERIFIER_ADDRESS")
	queueAddressHex := os.Getenv("QUEUE_ADDRESS") 
	ipfsUrl := getEnvironment("IPFS_API_URL", "http://ipfs:5001")

	if verifierAddressHex == "" || queueAddressHex == "" {
		log.Fatal("Critical: Contract addresses missing in .env")
	}

	// Change into Address type
	verifierAddress := common.HexToAddress(verifierAddressHex)
	rpc := getEnvironment("CHAIN_RPC", "http://localhost:8545")

	// ==========================================
	// 2. Off-Chain Infrastructure Boot
	// ==========================================	
	// Setup IPFS Shell
	ipfsShell := shell.NewShell(ipfsUrl)
	
	// Start the listener asynchronous Event Listener
	// This runs in a separate thread and populates the JobCache.
	go startChainListener(ctx, rpc, queueAddressHex, ipfsShell)

	
	// ==========================================
	// 3. P2P Networking Setup
	// ==========================================	
	announceIP, err := resolveHostnamePortToIP(announce)
	if err != nil { return err }
	bootIP, err := resolveHostnamePortToIP(bootAddr)
	if err != nil { return err }


	// Configdigest configuration
	cc, digester, err := buildContractConfig(n, f, seed)
	if err != nil { return err }

	nodes, _, err := deriveAllNodes(n, seed)
	if err != nil {
		return err
    }
	me := nodes[id]

	// Instantiate the P2P libp2p node for this oracle
	peer, err := networking.NewPeer(networking.PeerConfig{
		PrivKey: 				me.offKR.offPriv, 
		Logger: 				quietLogger{log.New(os.Stdout, "", 0)},
		V2ListenAddresses: 		[]string{listen}, 
		V2AnnounceAddresses: 	[]string{announceIP}, // Use of announceIP
		V2EndpointConfig: networking.EndpointConfigV2{
			IncomingMessageBufferSize: 100,
			OutgoingMessageBufferSize: 50,
		},                      
        V2DeltaReconcile:            10 * time.Second,           
        MetricsRegisterer:           prometheus.NewRegistry(),   
	})

	if err != nil { return err }
	
	bootPriv := deriveBootstrapPriv(seed)
	bootPID, _ := ragetypes.PeerIDFromPrivateKey(bootPriv)

	// ==========================================
	// 4. OCR3 Node Assembly
	// Injection of dependencies into the libocr core.
	// ==========================================
	args := offchainreporting2plus.OCR3OracleArgs[struct{}]{
		BinaryNetworkEndpointFactory: peer.OCR2BinaryNetworkEndpointFactory(),
		// Connect to the Bootstrap node for peer discovery		
		V2Bootstrappers: []commontypes.BootstrapperLocator{{PeerID: bootPID.String(), Addrs: []string{bootIP}}},
		ContractConfigTracker: staticTracker{cc},
		// Injecting our custom on-chain interaction logic
		ContractTransmitter: &ethTransmitter{
			oracleID: 		id, 
			rpcUrl: 		rpc, 
			privKeyHex: 	privKeyHex, // .env key
			contractAddr: 	verifierAddress, // .env smart contract address
			jobCache: 		JobCache, // Sharing the memory space with the Listener
		},
		Database: &memDB3{},
		// Tuning the protocol limits to handle large array submissions
		LocalConfig: ocrtypes.LocalConfig{
			DevelopmentMode: 					ocrtypes.EnableDangerousDevelopmentMode,
			BlockchainTimeout: 					15*time.Second, 
			ContractConfigConfirmations:		1, 
			ContractConfigTrackerPollInterval: 	5*time.Second, 
			ContractConfigLoadTimeout: 			10*time.Second,
			ContractTransmitterTransmitTimeout: 30*time.Second,  // Modified later for big array processing
			DatabaseTimeout: 					10*time.Second,
		},
		Logger: 				quietLogger{log.New(os.Stdout, "", 0)},
		MetricsRegisterer: 		prometheus.NewRegistry(),
		MonitoringEndpoint: 	noopMonitoring{},
		OffchainConfigDigester: digester,
		OffchainKeyring: 		me.offKR, 
		OnchainKeyring: 		me.onKR,
		// Injecting our custom application logic 
		ReportingPluginFactory: attributionPluginFactory{
			oracleID:	id, // passing the ID of the oracle
			numOracles: n,
		},
	}

	oracle, err := offchainreporting2plus.NewOracle(args)
	if err != nil { panic(err) }

	// Starts the internal state machine of OCR3
	oracle.Start()
	fmt.Printf("ORACLE %d STARTED\n", id)

	// Block the main thread until graceful shutdown
	<-ctx.Done()
	return nil
}

func main() {
	// Seed initialization for random generator (vital for the Random Backoff sleep times)
	rand.Seed(time.Now().UnixNano()) 

	// CLI flags parsing for node configuration
	mode := flag.String("mode", "oracle", "bootstrap|oracle")
	n := flag.Int("n", 4, "")
	f := flag.Int("f", 1, "")
	id := flag.Int("oracle_id", 0, "")
	seed := flag.Int64("seed", 1, "")
	bootL := flag.String("bootstrap_listen", "", "")
	bootA := flag.String("bootstrap_announce", "", "")
	bootAddr := flag.String("bootstrap_addr", "", "")
	p2pL := flag.String("p2p_listen", "", "")
	p2pA := flag.String("p2p_announce", "", "")
	flag.Parse()
	
	// OS Signal trapping for graceful shutdown (SIGTERM, SIGINT)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Node Role routing
	if *mode == "bootstrap" {
		runBootstrap(ctx, *n, *f, *seed, *bootL, *bootA)
	} else {
		runOracle(ctx, *id, *n, *f, *seed, *bootAddr, *p2pL, *p2pA)
	}
}
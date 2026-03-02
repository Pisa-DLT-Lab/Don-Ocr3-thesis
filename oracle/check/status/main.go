package main

import (
	"fmt"
	//"log"
	"math/big"
	"os"
	//"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv" 

	// ⚠️ VERIFICA CHE QUESTO IMPORT SIA CORRETTO COL TUO GO.MOD
	"OCR3-thesis/contracts" 
)

func main() {
    _ = godotenv.Load()
    
    if len(os.Args) < 3 {
        fmt.Println("Uso: go run check/check-status.go <JOB_ID> <CONTRACT_ADDRESS>")
        return
    }
    
    jobIdString := os.Args[1]
    contractAddrHex := os.Args[2] // Ora legge l'indirizzo dal comando!

    rpcURL := "http://127.0.0.1:8545"
    client, _ := ethclient.Dial(rpcURL)

    address := common.HexToAddress(contractAddrHex)
    jobID := new(big.Int)
    jobID.SetString(jobIdString, 10)

    verifier, _ := contracts.NewOracleVerifier(address, client)
    isCompleted, err := verifier.IsCompleted(nil, jobID)
    
    if err != nil {
        fmt.Printf("❌ Errore su %s: %v\n", contractAddrHex, err)
    } else {
        fmt.Printf("✅ Successo su %s! Risultato: %v\n", contractAddrHex, isCompleted)
    }
}
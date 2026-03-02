package main

import (
	"context"
	//"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	//"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
    client, err := ethclient.Dial("http://127.0.0.1:8545")
    if err != nil {
        log.Fatal(err)
    }

    jobID := big.NewInt(0)
    if len(os.Args) > 1 {
        jobID.SetString(os.Args[1], 10)
    }

    fmt.Printf("🔍 Interrogo job ID: %s\n", jobID.String())

    addr := common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3")

    idBytes := make([]byte, 32)
    jobID.FillBytes(idBytes)
    data := append([]byte{0x7a, 0x41, 0x98, 0x4b}, idBytes...)

    msg := ethereum.CallMsg{
        To:   &addr,
        Data: data,
    }

    res, err := client.CallContract(context.Background(), msg, nil)
    if err != nil {
        fmt.Printf("❌ Errore: %v\n", err)
    } else {
        isCompleted := len(res) > 0 && res[31] == 1
        fmt.Printf("✅ Risultato raw: %x\n", res)
        fmt.Printf("✅ isCompleted: %v\n", isCompleted)
    }
}
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"unsafe"
)

const (
	MaxBlockSize            = 1000000 // MAX_BLOCK_SIZE
	MaxCoinValue            = 21e6    // Maximum number of bitcoins
	CoinbaseMaturity        = 100     // Coinbase maturity
	SignatureOperationLimit = 20000   // Signature operation limit
	MinTransactionSize      = 100     // Minimum transaction size in bytes
	MinTransactionFee       = 1000    // Minimum transaction fee
	// MempoolPath             = `C:\Users\himan\Desktop\SOB\code-challenge-2024-himanshu5133\mempool` //path of mempool folder
	MempoolPath           = "mempool"
)

// Block represents a block containing transactions
type Block struct {
	Size             uint64
	Header           BlockHeader
	TransactionCount uint64
	Transactions     []Transaction
}

// BlockHeader represents the header of a block
type BlockHeader struct {
	Version          uint32
	PreviousBlockHash [32]byte
	MerkleRoot       [32]byte
	Timestamp        uint32
	DifficultyTarget string
	Nonce            uint32
}

// Transaction represents a Bitcoin transaction
type Transaction struct {
	Version uint32   `json:"version"`
	Locktime uint32  `json:"locktime"`
	Vin     []TxInput `json:"vin"`
	Vout    []TxOutput `json:"vout"`
}

type TxInput struct {
	Txid       string   `json:"txid"`
	Vout       int      `json:"vout"`
	ScriptSig  string   `json:"scriptsig"`
	Witness    []string `json:"witness"`
	IsCoinbase bool     `json:"is_coinbase"`
	Sequence   uint32   `json:"sequence"`
	PrevOut    Prevout  `json:"prevout"`
}

type Prevout struct {
	ScriptPubKey     string `json:"scriptpubkey"`
	ScriptPubKeyASM  string `json:"scriptpubkey_asm"`
	ScriptPubKeyType string `json:"scriptpubkey_type"`
	ScriptPubKeyAddr string `json:"scriptpubkey_address"`
	Value            int    `json:"value"`
}

type TxOutput struct {
	ScriptPubKey     string `json:"scriptpubkey"`
	ScriptPubKeyASM  string `json:"scriptpubkey_asm"`
	ScriptPubKeyType string `json:"scriptpubkey_type"`
	ScriptPubKeyAddr string `json:"scriptpubkey_address"`
	Value            int    `json:"value"`
}

// LoadTransactionsFromFolder loads transactions from JSON files in a folder
func LoadTransactionsFromFolder(folderPath string) ([]Transaction, error) {
	var transactions []Transaction

	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			data, err := ioutil.ReadFile(folderPath + "/" + file.Name())
			if err != nil {
				return nil, err
			}

			var tx Transaction
			if err := json.Unmarshal(data, &tx); err != nil {
				return nil, err
			}
			transactions = append(transactions, tx)
		}
	}

	return transactions, nil
}

// SerializeBlockHeader serializes the block header
func SerializeBlockHeader(header BlockHeader) []byte {
	var serializedHeader []byte

	// Serialize each field of the block header
	serializedHeader = append(serializedHeader, serializeUint32(header.Version)...)
	serializedHeader = append(serializedHeader, header.PreviousBlockHash[:]...)
	serializedHeader = append(serializedHeader, header.MerkleRoot[:]...)
	serializedHeader = append(serializedHeader, serializeUint32(header.Timestamp)...)
	serializedHeader = append(serializedHeader, []byte(header.DifficultyTarget)...)
	serializedHeader = append(serializedHeader, serializeUint32(header.Nonce)...)

	return serializedHeader
}

// HashBlockHeader hashes the serialized block header twice using SHA256
func HashBlockHeader(serializedHeader []byte) [32]byte {
	hash := sha256.Sum256(serializedHeader)
	hash = sha256.Sum256(hash[:])
	return hash
}

// serializeUint32 serializes a uint32 value into a little-endian byte slice
func serializeUint32(value uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, value)
	return buf
}

// WriteBlockToOutputFile writes the block data to the output file
func WriteBlockToOutputFile(block Block, hash [32]byte) error {
	file, err := os.Create("output.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	// Write block header
	blockHeaderBytes := SerializeBlockHeader(block.Header)
	blockHeaderHash := sha256.Sum256(blockHeaderBytes)
	blockHeaderHash = sha256.Sum256(blockHeaderHash[:])
	if _, err := file.WriteString(hex.EncodeToString(blockHeaderHash[:]) + "\n"); err != nil {
		return err
	}

	// Write serialized coinbase transaction
	serializedCoinbaseTx, err := json.Marshal(block.Transactions[0])
	if err != nil {
		return err
	}
	if _, err := file.Write(serializedCoinbaseTx); err != nil {
		return err
	}
	if _, err := file.WriteString("\n"); err != nil {
		return err
	}

	// Write transaction IDs
	for i, tx := range block.Transactions {
		if i == 0 {
			continue // Skip coinbase transaction
		}
		if _, err := file.WriteString(tx.Vin[0].Txid + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// ValidateTransaction verifies that a transaction meets the specified criteria
func ValidateTransaction(tx Transaction) bool {
	var input = 0
	var output = 0
	for _, vin := range tx.Vin {
		input += vin.PrevOut.Value
	}
	for _, vout := range tx.Vout {
		output += vout.Value
	}
	if input > output {
		return true
	}
	return false
}

// CreateCoinbaseTransaction creates a coinbase transaction
func CreateCoinbaseTransaction() Transaction {
	coinbaseTx := Transaction{
		Version:  1,
		Locktime: 0,
		Vin: []TxInput{
			{
				Txid:       "",
				Vout:       -1,
				ScriptSig:  "",
				Witness:    nil,
				IsCoinbase: true,
				Sequence:   0xFFFFFFFF,
				PrevOut: Prevout{
					ScriptPubKey:     "",
					ScriptPubKeyASM:  "",
					ScriptPubKeyType: "",
					ScriptPubKeyAddr: "",
					Value:            0,
				},
			},
		},
		Vout: []TxOutput{
			{
				ScriptPubKey:     "",
				ScriptPubKeyASM:  "",
				ScriptPubKeyType: "",
				ScriptPubKeyAddr: "",
				Value:            0,
			},
		},
	}
	return coinbaseTx
}

func main() {
	// Load transactions from the mempool folder
	transactions, err := LoadTransactionsFromFolder(MempoolPath)
	if err != nil {
		fmt.Println("Error loading transactions:", err)
		return
	}
	fmt.Println("Number of transactions in mempool:", len(transactions))

	// Validate each transaction and create a list of valid transactions
	var validTransactions []Transaction
	for _, tx := range transactions {
		if ValidateTransaction(tx) {
			validTransactions = append(validTransactions, tx)
		} else {
			fmt.Printf("Invalid transaction %s\n", tx.Vin[0].Txid)
		}
	}
	fmt.Println("Number of valid transactions:", len(validTransactions))

	// Create a coinbase transaction
	coinbaseTx := CreateCoinbaseTransaction()

	// Ensure that the coinbase transaction is the first transaction in the block
	blockTransactions := append([]Transaction{coinbaseTx}, validTransactions...)

	// Create a block
	block := Block{
		Size:             0, // Calculate block size later
		Header:           BlockHeader{},
		TransactionCount: uint64(len(blockTransactions)),
		Transactions:     blockTransactions,
	}

	// Set block header fields (dummy values for demonstration)
	block.Header.Version = 1
	block.Header.Timestamp = uint32(time.Now().Unix())
	block.Header.DifficultyTarget = "0000ffff00000000000000000000000000000000000000000000000000000000"
	block.Header.Nonce = 0 // Dummy nonce

	// Calculate block size (excluding block size field itself)
	blockSize := uint64(len(SerializeBlockHeader(block.Header)) + 8) // 8 bytes for transaction counter
	for _, tx := range block.Transactions {
		blockSize += uint64(unsafe.Sizeof(tx)) // Add size of each transaction
	}
	block.Size = blockSize

	// Serialize block header
	serializedHeader := SerializeBlockHeader(block.Header)

	// Hash the block header twice
	blockHash := HashBlockHeader(serializedHeader)

	// Write the block data to the output file
	if err := WriteBlockToOutputFile(block, blockHash); err != nil {
		fmt.Println("Error writing block to output file:", err)
		return
	}

	fmt.Println("Block data written to output.txt")
}

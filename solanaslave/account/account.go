package account

import (
	"context"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"log"

	"github.com/gagliardetto/solana-go/rpc"
	"math/big"
	"sync"
)

type Account struct {
	id         int
	privateKey solana.PrivateKey
	key        string
	address    solana.PublicKey
	//nonce      uint64
	balance *big.Int
	mutex   sync.Mutex
}

//func (self.)

func (self *Account) TransferNewValueTransferTx(c *rpc.Client, wsClient *ws.Client, to *Account, value *uint64) error {
	//ctx := context.Background() //context.WithTimeout(context.Background(), 100*time.Second)
	self.mutex.Lock()
	defer self.mutex.Unlock()

	//nonce := self.GetNonce(c)
	recent, err := c.GetRecentBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("Failed to get blockhash: %v", err)
	} else {
		log.Printf("block hash : %v", recent.Value.Blockhash)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(
				*value, //in lamports
				self.address,
				to.address,
			).Build(),
		},
		recent.Value.Blockhash,
		solana.TransactionPayer(self.address),
	)
	if err != nil {
		panic(err)
	}

	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if self.address.Equals(key) {
				return &self.privateKey
			}
			return nil
		},
	)
	if err != nil {
		log.Fatalf("Failed to sign tx: %v", err)
	}
	//spew.Dump(tx)
	// Pretty print the transaction:
	//tx.EncodeTree(text.NewTreeEncoder(os.Stdout, "Transfer SOL"))

	// Send transaction, and wait for confirmation:
	sig, err := confirm.SendAndConfirmTransaction(
		context.TODO(),
		c,
		wsClient,
		tx,
	)

	if err != nil {
		fmt.Printf("Account(%v) : Failed to sendTransaction: %v\n", self.address.String(), err)
		//if err.Error()
		return err
	}
	log.Printf("signature: %v", sig)
	//spew.Dump(sig)

	return nil
}

func (self *Account) GetAddress() solana.PublicKey {
	return self.address
}

func NewAccount(id int) *Account {
	acc := solana.NewWallet()
	tAcc := Account{
		id,
		acc.PrivateKey,
		acc.PrivateKey.String(),
		acc.PublicKey(),
		big.NewInt(0),
		sync.Mutex{},
	}
	return &tAcc
}

func GetAccountFromKey(id int, key string) *Account {
	//acc, err := solana.WalletFromPrivateKeyBase58(key)
	//if err != nil{
	//	log.Fatalf("Key(%v): Failed to create wallet", key, err)
	//}
	privKey, err := solana.PrivateKeyFromBase58(key)
	if err != nil {
		log.Fatalf("Key(%v): Failed to create wallet", key, err)
	}

	tAcc := Account{
		id,
		privKey,
		key,
		privKey.PublicKey(),
		big.NewInt(0),
		sync.Mutex{},
	}
	return &tAcc
}

package newValueTransferTC

import (
	"context"
	"github.com/Yeonju-Kim/solana-load-test/solanaslave/account"
	"github.com/Yeonju-Kim/solana-load-test/solanaslave/clipool"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/myzhan/boomer"
	"math/rand"

	"github.com/gagliardetto/solana-go/rpc"
	"log"
	"math/big"
)

const Name = "solana-"

var (
	accGrp   []*account.Account
	cliPool  clipool.Solana_ClientPool
	nAcc     int
	endPoint string

	transferedValue *big.Int
	expectedFee     *big.Int

	fromAccount     *account.Account
	prevBalanceFrom *big.Int

	toAccount     *account.Account
	prevBalanceTo *big.Int
)

func Run() {
	cliPrev, cliWsPrev := cliPool.Alloc() // .(*rpc.Client)
	cli, cliWs := cliPrev.(*rpc.Client), cliWsPrev.(*ws.Client)

	from := accGrp[rand.Int()%nAcc]
	to := accGrp[rand.Int()%nAcc]
	value := uint64(rand.Int() % 3)
	start := boomer.Now()
	err := from.TransferNewValueTransferTx(cli, cliWs, to, &value)
	elapsed := boomer.Now() - start

	if err == nil {
		boomer.Events.Publish("request_success", "http", "transferNewValueTransferTx"+" to "+endPoint, elapsed, int64(10))
		cliPool.Free(cli, cliWs)
	} else {
		boomer.Events.Publish("request_failure", "http", "transferNewValueTransferTx"+" to "+endPoint, elapsed, err.Error())
	}

}

func Init(accs []*account.Account, endpoint string, endpointWs string) {

	endPoint = endpoint

	//for json rpc
	cliCreate := func() interface{} {
		c := rpc.New(endPoint)
		return c
	}

	//for websocket
	cliCreateWs := func() interface{} {
		c, err := ws.Connect(context.Background(), endpointWs)
		if err != nil {
			log.Fatalf("Failed to connect RPC: #{err}")
		}
		return c
	}
	cliPool.Init(20, 300, cliCreate, cliCreateWs)

	for _, acc := range accs {
		accGrp = append(accGrp, acc)
	}

	nAcc = len(accGrp)

}

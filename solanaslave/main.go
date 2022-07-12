package main

import (
	"context"
	"fmt"
	"github.com/Yeonju-Kim/solana-load-test/solanaslave/newValueTransferTC"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/myzhan/boomer"
	"os"
	"runtime"
	"syscall"
	"time"

	"flag"
	"github.com/Yeonju-Kim/solana-load-test/solanaslave/account"
	"github.com/gagliardetto/solana-go/rpc"
	"log"
	"math/big"
	"strings"
)

var (
	coinbasePrivatekey = ""
	gCli               *rpc.Client
	gCliWs             *ws.Client
	gEndpoint          string
	gEndpointWs        string

	coinbase    *account.Account
	newCoinbase *account.Account

	//nUserForUnsigned    = 5 //number of virtual user account for unsigned tx
	//accGrpForUnsignedTx []*solana_account.Account

	nUserForSigned    = 5
	accGrpForSignedTx []*account.Account
	//
	//nUserForNewAccounts  = 5
	//accGrpForNewAccounts []*solana_account.Account

	activeUserPercent = 100

	//SmartContractAccount *account.Account

	tcStr     string
	tcStrList []string

	chargeValue *uint64

	gasPrice *big.Int
	baseFee  *big.Int
)

type SolanaExtendedTask struct {
	Name       string
	Weight     int
	Fn         func()
	Init       func(accs []*account.Account, endpoint string, endpointWs string)
	AccGrp     []*account.Account
	EndPoint   string
	EndPointWs string
}

func initTCList_solana() (taskSet []*SolanaExtendedTask) {
	taskSet = append(taskSet, &SolanaExtendedTask{
		Name:       "newValueTransferTx",
		Weight:     10,
		Fn:         newValueTransferTC.Run,
		Init:       newValueTransferTC.Init,
		AccGrp:     accGrpForSignedTx,
		EndPoint:   gEndpoint,
		EndPointWs: gEndpointWs,
	})
	return taskSet
}

func Create(endpoint string) *rpc.Client {
	c := rpc.New(endpoint)
	return c
}

func CreateWs(endpoint string) *ws.Client {
	c, err := ws.Connect(context.Background(), endpoint)
	//rpc.MainNetBeta_WS
	if err != nil {
		log.Fatalf("Failed to connect WS : %v", err)
	}
	return c
}

func inTheTCList(tcName string) bool {
	for _, tc := range tcStrList {
		if tcName == tc {
			return true
		}
	}
	return false
}

func initArgs(tcNames string) {
	chargeSOLAmount := uint64(100000)
	gEndpointPtr := flag.String("endpoint", "http://3.38.216.155:8899", "Target EndPoint")
	gEndpointWsPtr := flag.String("endpointWs", "ws://3.38.216.155:8900", "Target Endpoint for Ws connection")
	activeUserPercentPtr := flag.Int("activepercent", activeUserPercent, "percent of active accounts")
	keyPtr := flag.String("key", "", "privatekey of coinbase")
	chargeSOLAmountPtr := flag.Uint64("charge", chargeSOLAmount, "charging amount for each test account in SOL")
	nUserForSignedPtr := flag.Int("vusigned", nUserForSigned, "num of test account for signed Tx TC")

	flag.StringVar(&tcStr, "tc", tcNames, "tasks which user want to run, multiple tasks are separated by comma.")
	flag.Parse()

	if *keyPtr == "" {
		log.Fatal("key argument is not defined. You should set the key for coinbase.\n example) klaytc -key='2ef07640fd8d3f568c23185799ee92e0154bf08ccfe5c509466d1d40baca3430'")
	}

	if tcStr != "" {
		tcStrList = strings.Split(tcStr, ",")
	}

	gEndpoint = *gEndpointPtr
	gEndpointWs = *gEndpointWsPtr
	nUserForSigned = *nUserForSignedPtr
	activeUserPercent = *activeUserPercentPtr
	coinbasePrivatekey = *keyPtr
	chargeSOLAmount = *chargeSOLAmountPtr
	chargeValue = new(uint64)
	// SOL to lamports
	*chargeValue = solana.LAMPORTS_PER_SOL * chargeSOLAmount

	fmt.Println("Arguments are set like the following:")
	fmt.Printf("- Target EndPoint = %v\n", gEndpoint)
	fmt.Printf("- Target WS EndPoint = %v\n", gEndpointWs)
	fmt.Printf("- nUserForSigned = %v\n", nUserForSigned)
	//fmt.Printf("- nUserForUnsigned = %v\n", nUserForUnsigned)
	fmt.Printf("- activeUserPercent = %v\n", activeUserPercent)
	fmt.Printf("- coinbasePrivatekey = %v\n", coinbasePrivatekey)
	fmt.Printf("- charging SOL Amount = %v\n", chargeSOLAmount)
	fmt.Printf("- tc = %v\n", tcStr)
}

func estimateRemainingTime(accGrp map[solana.PublicKey]*account.Account, numChargedAcc, lastFailedNum int) (int, int) {
	if lastFailedNum > 0 {
		TPS := (numChargedAcc - lastFailedNum)
		lastFailedNum = numChargedAcc

		if TPS <= 5 {
			log.Printf("Retry to charge test account ##{numChargedAcc}. But it is too slow. #{TPS} TPS\n")
		} else {
			remainTime := (len(accGrp) - numChargedAcc) / TPS
			remainHour := remainTime / 3600
			remainMinute := (remainTime % 3600) / 60
			log.Printf("Retry to charge test account #%d. Estimated remaining time: %d hours %d mins later\n", numChargedAcc, remainHour, remainMinute)

		}
	} else {
		lastFailedNum = numChargedAcc
		log.Printf("Retry to charge test account ##{numChargedAcc}. \n")
	}
	time.Sleep(5 * time.Second)
	return numChargedAcc, lastFailedNum
}

func chargeSOLToTestAccounts(accGrp map[solana.PublicKey]*account.Account) {
	log.Printf("Start charging SOL to test accounts %v", len(accGrp))
	numChargedAcc := 0
	lastFailedNum := 0

	for _, acc := range accGrp {
		for {
			err := newCoinbase.TransferNewValueTransferTx(gCli, gCliWs, acc, chargeValue)
			if err == nil {
				break
			}
			numChargedAcc, lastFailedNum = estimateRemainingTime(accGrp, numChargedAcc, lastFailedNum)
		}
		numChargedAcc++
	}

	log.Printf("Finished charging SOL to %d test account(s), Total %d transactions are sent.\n", len(accGrp), numChargedAcc)
}

func prepareAccounts() {
	totalChargeValue := new(uint64)
	*totalChargeValue = (*chargeValue) * uint64(nUserForSigned+1)

	coinbase = account.GetAccountFromKey(0, coinbasePrivatekey)
	newCoinbase = account.NewAccount(0) //  solana.NewWallet()
	if *chargeValue != 0 {
		for {
			err := coinbase.TransferNewValueTransferTx(gCli, gCliWs, newCoinbase, totalChargeValue)
			if err != nil {
				log.Printf("%v: charge newCoinbase fail: %v\n", os.Getpid(), err)
				time.Sleep(1000 * time.Millisecond)
				continue
			}

			log.Printf("%v : charge newCoinbase: %v\n", os.Getpid(), newCoinbase.GetAddress().String()) //, hash.String())

			getReceipt := false
			for i := 0; i < 5; i++ {
				time.Sleep(2000 * time.Millisecond)
				ctx := context.Background()

				out, err := gCli.GetBalance(ctx, newCoinbase.GetAddress(), rpc.CommitmentFinalized)
				if err == nil {
					if out.Value > 0 { //if bigger than zero, break
						getReceipt = true
						log.Printf("%v : charge newCoinbase success: %v, balance=%v peb\n", os.Getpid(),
							newCoinbase.GetAddress().String(), out.Value)
						break
					}
					log.Printf("%v : charge newCoinbase waiting: %v\n", os.Getpid(), newCoinbase.GetAddress().String())
				} else {
					log.Printf("%v : check balance err: %v\n", os.Getpid(), err)
				}
			}
			if getReceipt {
				break
			}
		}
	}

	for i := 0; i < nUserForSigned; i++ {
		accGrpForSignedTx = append(accGrpForSignedTx, account.NewAccount(i))
	}
}

func setRLimit(resourceType int, val uint64) error {
	if runtime.GOOS == "darwin" {
		return nil
	}

	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(resourceType, &rLimit)
	if err != nil {
		return err
	}
	rLimit.Cur = val
	err = syscall.Setrlimit(resourceType, &rLimit)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	taskSet := initTCList_solana()

	var tcNames string
	for i, task := range taskSet {
		if i != 0 {
			tcNames += ","
		}
		tcNames += task.Name
	}

	initArgs(tcNames)

	gCli = Create(gEndpoint)
	gCliWs = CreateWs(gEndpointWs)

	prepareAccounts()

	taskSet = initTCList_solana()

	var filteredTask []*SolanaExtendedTask
	println("Adding tasks")
	for _, task := range taskSet {
		if task.Name == "" {
			continue
		} else {
			flag := false
			for _, name := range tcStrList {
				if name == task.Name {
					flag = true
					break
				}
			}
			if flag {
				filteredTask = append(filteredTask, task)
				println("=> " + task.Name + " task is added.")
			}
		}
	}

	//log.Printf("%v", len(filteredTask))

	//Charge Accounts
	accGrp := make(map[solana.PublicKey]*account.Account)
	for _, task := range filteredTask {
		log.Printf("added to map accGrp %v", len(task.AccGrp))

		for _, acc := range task.AccGrp {
			_, exist := accGrp[acc.GetAddress()]
			//log.Printf("added to map accGrp %v", acc.GetAddress().String())

			if !exist {
				accGrp[acc.GetAddress()] = acc
			}
		}
	}
	//log.Printf("%v\n", newCoinbase.GetAddress().String())

	chargeSOLToTestAccounts(accGrp)

	// After charging accounts, cur the slice to the desired length, calculated by ActiveAccountPercent.
	for _, task := range filteredTask {
		if activeUserPercent > 100 {
			log.Fatalf("ActiveAccountPercent should be less than or equal to 100, but it is %v", activeUserPercent)
		}
		numActiveAccounts := len(task.AccGrp) * activeUserPercent / 100

		if numActiveAccounts == 0 {
			numActiveAccounts = 1
		}
		task.AccGrp = task.AccGrp[:numActiveAccounts]

	}
	if len(filteredTask) == 0 {
		log.Fatal("No Tc is set. Please set TcList. \nExample argument) -tc='" + tcNames + "'")
	}

	println("Initializinig tasks")
	var filteredBoomerTask []*boomer.Task

	for _, task := range filteredTask {
		task.Init(task.AccGrp, task.EndPoint, task.EndPointWs)
		filteredBoomerTask = append(filteredBoomerTask, &boomer.Task{task.Weight, task.Fn, task.Name})
		println("=> " + task.Name + " task is initialized.")
	}

	setRLimit(syscall.RLIMIT_NOFILE, 1024*400)

	// Locust Slave Run
	boomer.Run(filteredBoomerTask...)
	//boomer.Run(cpuHeavyTx)
}

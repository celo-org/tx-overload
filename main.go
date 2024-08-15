package main

import (
	"context"
	"math/big"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
)

var logger log.Logger

type TxModeType string

const (
	Random TxModeType = "random"
	Erc20  TxModeType = "erc20"
	// Mix      TxModeType = "mix"
)

type TxOverload struct {
	Distrbutor      *Distributor
	BytesPerSecond  int
	StartTime       time.Time
	NumDistributors int
	BlockTimeMs     int
	TxMode          TxModeType
}

func (t *TxOverload) generateTxCandidate() (txmgr.TxCandidate, error) {
	switch t.TxMode {
	case Random:
		return t.generateRandomTxCandidate()
	case Erc20:
		return t.generateErc20TxCandidate()
		// case Mix:
		// 	return t.generateMixTxCandidate()
	}
	return txmgr.TxCandidate{}, nil
}

func (t *TxOverload) generateRandomTxCandidate() (txmgr.TxCandidate, error) {
	var to common.Address

	data := make([]byte, t.BytesPerSecond)
	//dur := time.Since(t.StartTime)
	_, err := rand.Read(data)
	if err != nil {
		return txmgr.TxCandidate{}, err
	}

	intrinsicGas, err := core.IntrinsicGas(data, nil, false, true, true, false)
	if err != nil {
		return txmgr.TxCandidate{}, err
	}
	return txmgr.TxCandidate{
		To:       &to,
		TxData:   data,
		GasLimit: intrinsicGas,
	}, nil
}

func (t *TxOverload) generateErc20TxCandidate() (txmgr.TxCandidate, error) {
	amount := big.NewInt(0) // send 0 to don't mind about balance
	// cUSD default address
	tokenAddress := common.HexToAddress("0x765DE816845861e75A25fCA122bb6898B8B1282a")
	toAddressHex := make([]byte, 20)
	_, err := rand.Read(toAddressHex)
	toAddress := common.BytesToAddress(toAddressHex)
	paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)
	// Get the transfer function signature
	// transferFnSignature := []byte("transfer(address,uint256)") // do not include spaces in the string
	// hash := sha3.NewLegacyKeccak256()
	// hash.Write(transferFnSignature)
	// methodID := hash.Sum(nil)[:4]
	// fmt.Println(hexutil.Encode(methodID)) // 0xa9059cbb
	methodID := common.Hex2Bytes("a9059cbb")

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)

	intrinsicGas, err := core.IntrinsicGas(data, nil, false, true, true, false)
	if err != nil {
		return txmgr.TxCandidate{}, err
	}
	return txmgr.TxCandidate{
		To:       &tokenAddress,
		TxData:   data,
		GasLimit: intrinsicGas,
	}, nil
}

func (t *TxOverload) Start() {
	ctx := context.Background()
	t.Distrbutor.Start()

	var blockTimeMs = t.BlockTimeMs
	tickRate := time.Duration(blockTimeMs/t.NumDistributors) * time.Millisecond
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	var backoff = tickRate
	var backingOff bool
	backoffFn := func(err error) {
		const maxBackoff = time.Second * 2
		switch {
		case err == nil && backingOff:
			backoff = tickRate
			backingOff = false
			ticker.Reset(tickRate)
		case err == ErrQueueFull:
			backingOff = true
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			ticker.Reset(backoff)
			logger.Debug("backoff", "duration", backoff)
		}
	}

	for {
		select {
		case <-ticker.C:
			if t.StartTime.IsZero() {
				t.StartTime = time.Now()
			}
			candidate, err := t.generateTxCandidate()
			if err != nil {
				logger.Warn("unable to generate tx candidate", "err", err)
				continue
			}
			err = t.Distrbutor.Send(ctx, candidate)
			backoffFn(err)
		case <-ctx.Done():
			return
		}
	}
}

func Main(cliCtx *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	if err := logCfg.Check(); err != nil {
		return err
	}
	logger = oplog.NewLogger(logCfg)
	txmgrCfg := txmgr.ReadCLIConfig(cliCtx)
	txmgrCfg.L1RPCURL = cliCtx.GlobalString(EthRpcFlag.Name) // should be named L2RPCURL but this will work just as well
	if err := txmgrCfg.Check(); err != nil {
		return err
	}

	numDistributors := cliCtx.GlobalInt(NumDistributors.Name)
	distributors = keys[:numDistributors]

	blockTimeMs := cliCtx.GlobalInt(BlockTimeFlag.Name)

	metricsCfg := opmetrics.ReadCLIConfig(cliCtx)
	m := NewMetrics()
	if metricsCfg.Enabled {
		logger.Info("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
		go func() {
			if err := m.Serve(context.Background(), metricsCfg.ListenAddr, metricsCfg.ListenPort); err != nil {
				logger.Error("error starting metrics server", err)
			}
		}()
	}

	distributor, err := NewDistributor(txmgrCfg, logger, m)
	if err != nil {
		return err
	}

	t := &TxOverload{
		Distrbutor:      distributor,
		TxMode:          TxModeType(cliCtx.GlobalString(TxModeFlag.Name)),
		BytesPerSecond:  cliCtx.GlobalInt(DataRateFlag.Name),
		NumDistributors: numDistributors,
		BlockTimeMs:     blockTimeMs,
	}
	go t.Start()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}...)
	<-interrupt

	return nil
}

func main() {
	oplog.SetupDefaults()
	app := cli.NewApp()
	app.Name = "tx-overload"
	app.Flags = flags
	app.Action = func(ctx *cli.Context) error {
		return Main(ctx)
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

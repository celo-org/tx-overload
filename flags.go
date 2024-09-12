package main

import (
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/urfave/cli"
)

const envVarPrefix = "TX_OVERLOAD"

var (
	EthRpcFlag = cli.StringFlag{
		Name:     "eth-rpc",
		Required: true,
		EnvVar:   opservice.PrefixEnvVar(envVarPrefix, "ETH_RPC"),
	}
	TxModeFlag = cli.StringFlag{
		Name:   "tx-mode",
		Usage:  "type of transaction. Valid values are 'random' and 'erc20'.",
		Value:  "random",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "TX_MODE"),
	}
	DataRateFlag = cli.Int64Flag{
		Name:   "data-rate",
		Usage:  "data rate in bytes per second.",
		Value:  5000,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "DATA_RATE"),
	}
	NumDistributors = cli.Int64Flag{
		Name:   "num-distributors",
		Value:  20,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "NUM_DISTRIBUTORS"),
	}
	StartingIndex = cli.Int64Flag{
		Name:   "starting-index",
		Value:  0,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "STARTING_INDEX"),
	}
	BlockTimeFlag = cli.Int64Flag{
		Name:   "block-time-ms",
		Value:  1000,
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "BLOCK_TIME"),
	}
)

func init() {
	flags = append(flags, EthRpcFlag, TxModeFlag, DataRateFlag, NumDistributors, StartingIndex, BlockTimeFlag)
	flags = append(flags, oplog.CLIFlags(envVarPrefix)...)
	flags = append(flags, txmgr.CLIFlags(envVarPrefix)...)
	flags = append(flags, opmetrics.CLIFlags(envVarPrefix)...)
}

var flags []cli.Flag

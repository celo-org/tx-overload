# Transaction Overload

Generate load on Optimism Bedrock using transactions with random calldata.

## Usage

```go
go build 

./tx-overload \
    --eth-rpc http://localhost:8545 \
    --private-key <private_key> \
    --num-distributors 10 \
    --sender-selection round-robin \
    --data-rate 1000 \
```

`--sender-selection` defaults to `random`. Use `round-robin` to rotate across
configured sender accounts in order. It can also be set with
`TX_OVERLOAD_SENDER_SELECTION`.

More options are avaiable:
```
./tx-overload --help
```

## License

All files within this repository are licensed under the [MIT License](https://github.com/ethereum-optimism/tx-overload/blob/master/LICENSE).

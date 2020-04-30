package transport

import (
	"fmt"
	"ledger-sats-stack/pkg/types"
	"ledger-sats-stack/pkg/utils"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

// Wire is a copper wire
type Wire struct {
	*rpcclient.Client
}

func (w Wire) getBlockByHash(hash *chainhash.Hash) (*BlockContainer, error) {
	rawBlock, err := w.GetBlockVerbose(hash)
	if err != nil {
		return nil, err
	}

	block := new(BlockContainer)
	block.init(rawBlock)
	return block, nil
}

func (w Wire) getBlockHashByReference(blockRef string) (*chainhash.Hash, error) {
	switch {
	case blockRef == "current":
		return w.GetBestBlockHash()

	case strings.HasPrefix(blockRef, "0x"):
		// 256-bit hex string with 0x prefix
		return chainhash.NewHashFromStr(strings.TrimLeft(blockRef, "0x"))
	case len(blockRef) == 64:
		// 256-bit hex string WITHOUT 0x prefix
		return chainhash.NewHashFromStr(blockRef)
	default:
		{
			// Either an int64 block height, or garbage input
			blockHeight, err := strconv.ParseInt(blockRef, 10, 64)

			switch err {
			case nil:
				return w.GetBlockHash(blockHeight)

			default:
				return nil, fmt.Errorf("Invalid block '%s'", blockRef)
			}
		}

	}
}

func (w Wire) buildUtxoMap(vin []btcjson.Vin) (utxoMapType, error) {
	utxoMap := make(utxoMapType)

	for _, inputRaw := range vin {
		if inputRaw.IsCoinBase() {
			continue
		}

		txn, err := w.getTransactionByHash(inputRaw.Txid)
		if err != nil {
			return nil, err
		}
		utxoRaw := txn.Vout[inputRaw.Vout]

		utxo := func(addresses []string) types.UTXO {
			switch len(addresses) {
			case 0:
				// TODO: Document when this happens
				return types.UTXO{
					Value:   utils.ParseSatoshi(utxoRaw.Value), // !FIXME: Can panic
					Address: "",                                // Will be omitted by the JSON serializer
				}
			case 1:
				return types.UTXO{
					Value:   utils.ParseSatoshi(utxoRaw.Value),
					Address: addresses[0], // ?XXX: Investigate why we do this
				}
			default:
				// TODO: Log an error
				return types.UTXO{
					Value:   utils.ParseSatoshi(utxoRaw.Value), // !FIXME: Can panic
					Address: "",                                // Will be omitted by the JSON serializer
				}
			}
		}(utxoRaw.ScriptPubKey.Addresses)

		utxoMap[inputRaw.Txid] = make(utxoVoutMapType)
		utxoMap[inputRaw.Txid][inputRaw.Vout] = utxo
	}

	return utxoMap, nil
}

// getTransactionByHash gets the transaction with the given hash.
// Supports transaction hashes with or without 0x prefix.
func (w Wire) getTransactionByHash(txHash string) (*btcjson.TxRawResult, error) {
	txHashRaw, err := chainhash.NewHashFromStr(strings.TrimLeft(txHash, "0x"))
	if err != nil {
		return nil, err
	}

	txRaw, err := w.GetRawTransactionVerbose(txHashRaw)
	if err != nil {
		return nil, err
	}
	return txRaw, nil
}
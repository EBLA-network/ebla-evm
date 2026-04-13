package vm

import "github.com/EBLA-network/ebla-evm/common"

type LogRecord struct {
	Address common.Address
	Topics  []common.Hash
	Data    []byte
}

package trie

import "github.com/EBLA-network/ebla-evm/common"

type Schema interface {
	ValueStorageToHashEncoding(enc_storage []byte) (enc_hash []byte)
	MaxValueSizeToStoreInTrie() int
}
type Input interface {
	GetValue(*common.Hash, func([]byte))
	GetNode(*common.Hash, func([]byte))
}
type IO interface {
	Input
	PutValue(*common.Hash, []byte)
	PutNode(*common.Hash, []byte)
}

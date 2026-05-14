package trie

import (
	"github.com/EBLA-network/ebla-evm/common"
	"github.com/EBLA-network/ebla-evm/ebla/util/keccak256"
	"github.com/EBLA-network/ebla-evm/rlp"
)

type hash_encoder struct {
	encoder  rlp.Encoder
	disabled bool
}

func (self *hash_encoder) ResizeReset(string_buf_cap, list_buf_cap int) {
	self.encoder.ResizeReset(string_buf_cap, list_buf_cap)
}

func (self *hash_encoder) Reset() {
	self.encoder.Reset()
	self.disabled = false
}

func (self *hash_encoder) Toggle() {
	self.disabled = !self.disabled
}

func (self *hash_encoder) AppendString(str []byte) {
	if !self.disabled {
		self.encoder.AppendString(str)
	}
}

func (self *hash_encoder) ListStart() int {
	if !self.disabled {
		return self.encoder.ListStart()
	}
	return -1
}

func (self *hash_encoder) ListEnd(list_pos int, is_root bool, out **node_hash) {
	if self.disabled {
		return
	}
	self.encoder.ListEnd(list_pos)
	if self.encoder.ListSize(list_pos) < common.HashLength && !is_root {
		return
	}
	hasher := keccak256.GetHasherFromPool()
	self.encoder.Flush(list_pos, hasher.Write)
	hash := hasher.Hash()
	*out = (*node_hash)(hash)
	keccak256.ReturnHasherToPool(hasher)
	if !is_root {
		self.encoder.RevertToListStart(list_pos)
		self.encoder.AppendString(hash[:])
	}
}

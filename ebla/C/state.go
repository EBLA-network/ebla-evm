package main

//#include "common.h"
//#include "state.h"
//#include <rocksdb/c.h>
import "C"
import (
	"math/big"
	"sync"
	"unsafe"

	"github.com/EBLA-network/ebla-evm/ebla/state/chain_config"
	"github.com/EBLA-network/ebla-evm/ebla/state/rewards_stats"
	"github.com/holiman/uint256"

	"github.com/EBLA-network/ebla-evm/ebla/state/state_db_rocksdb"

	"github.com/EBLA-network/ebla-evm/common"
	"github.com/EBLA-network/ebla-evm/core/types"
	"github.com/EBLA-network/ebla-evm/core/vm"
	"github.com/EBLA-network/ebla-evm/ebla/state"
	"github.com/EBLA-network/ebla-evm/ebla/util"
	"github.com/EBLA-network/ebla-evm/ebla/util/asserts"
	"github.com/EBLA-network/ebla-evm/ebla/util/bin"
)

type state_API struct {
	state.API
	db             state_db_rocksdb.DB
	get_blk_hash_C C.ebla_evm_GetBlockHash
}

func (self *state_API) blk_hash(num types.BlockNum) *big.Int {
	hash_c, err := C.ebla_evm_GetBlockHashApply(self.get_blk_hash_C, C.uint64_t(num))
	util.PanicIfNotNil(err)
	return new(big.Int).SetBytes(bin.AnyBytes2(unsafe.Pointer(&hash_c.Val), common.HashLength))
}

//export ebla_evm_state_api_new
func ebla_evm_state_api_new(
	params_enc C.ebla_evm_Bytes,
	cb_err C.ebla_evm_BytesCallback,
) C.ebla_evm_state_API_ptr {
	defer handle_err(cb_err)
	var params struct {
		GetBlockHash uintptr
		ChainConfig  *chain_config.ChainConfig
		Opts         state.APIOpts
		OptsDB       state_db_rocksdb.Opts
	}
	dec_rlp(params_enc, &params)
	self := new(state_API)
	self.db.Init(params.OptsDB)
	self.get_blk_hash_C = *(*C.ebla_evm_GetBlockHash)(unsafe.Pointer(params.GetBlockHash))
	self.Init(&self.db, self.blk_hash, params.ChainConfig, params.Opts)

	defer util.LockUnlock(&state_API_alloc_mu)()
	lastpos := len(state_API_available_ptrs) - 1
	asserts.Holds(lastpos >= 0)
	ptr := state_API_available_ptrs[lastpos]
	asserts.Holds(state_API_instances[ptr] == nil)
	state_API_available_ptrs = state_API_available_ptrs[:lastpos]
	state_API_instances[ptr] = self
	return C.ebla_evm_state_API_ptr(ptr)
}

//export ebla_evm_state_api_free
func ebla_evm_state_api_free(
	ptr C.ebla_evm_state_API_ptr,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	self := state_API_instances[ptr]
	self.Close()
	self.db.Close()
	defer util.LockUnlock(&state_API_alloc_mu)()
	state_API_instances[ptr], state_API_available_ptrs = nil, append(state_API_available_ptrs, state_API_ptr(ptr))
}

//export ebla_evm_state_api_get_last_committed_state_descriptor
func ebla_evm_state_api_get_last_committed_state_descriptor(
	ptr C.ebla_evm_state_API_ptr,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	ret := state_API_instances[ptr].GetCommittedStateDescriptor()
	enc_rlp(&ret, cb)
}

//export ebla_evm_state_api_update_state_config
func ebla_evm_state_api_update_state_config(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		ChainConfig chain_config.ChainConfig
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[ptr]
	self.UpdateConfig(&params.ChainConfig)
}

//export ebla_evm_state_api_get_account
func ebla_evm_state_api_get_account(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
	}
	dec_rlp(params_enc, &params)
	state_API_instances[ptr].ReadBlock(params.BlkNum).GetRawAccount(&params.Addr, func(bytes []byte) {
		call_bytes_cb(bytes, cb)
	})
}

//export ebla_evm_state_api_get_account_storage
func ebla_evm_state_api_get_account_storage(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
		Key    common.Hash
	}
	dec_rlp(params_enc, &params)
	state_API_instances[ptr].ReadBlock(params.BlkNum).GetAccountStorage(&params.Addr, &params.Key, func(bytes []byte) {
		call_bytes_cb(bytes, cb)
	})
}

//export ebla_evm_state_api_get_code_by_address
func ebla_evm_state_api_get_code_by_address(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
	}
	dec_rlp(params_enc, &params)
	ret := state_API_instances[ptr].ReadBlock(params.BlkNum).GetCodeByAddress(&params.Addr)
	call_bytes_cb(ret, cb)
}

//export ebla_evm_state_api_dry_run_transaction
func ebla_evm_state_api_dry_run_transaction(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Blk    vm.BlockInfo
		Trx    vm.Transaction
	}
	dec_rlp(params_enc, &params)
	ret := state_API_instances[ptr].DryRunTransaction(&vm.Block{params.BlkNum, params.Blk}, &params.Trx)
	enc_rlp(&ret, cb)
}

//export ebla_evm_state_api_trace_transactions
func ebla_evm_state_api_trace_transactions(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum    types.BlockNum
		Blk       vm.BlockInfo
		StateTrxs []vm.Transaction
		Trxs      []vm.Transaction
		Params    *vm.TracingConfig `rlp:"nil"`
	}
	dec_rlp(params_enc, &params)
	ret := state_API_instances[ptr].Trace(&vm.Block{params.BlkNum, params.Blk}, &params.StateTrxs, &params.Trxs, params.Params)
	enc_rlp(&ret, cb)
}

//export ebla_evm_state_api_execute_transactions
func ebla_evm_state_api_execute_transactions(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		Blk vm.BlockInfo
		Txs []vm.Transaction
	}
	dec_rlp(params_enc, &params)

	var retval struct {
		ExecutionResults []vm.ExecutionResult
	}
	self := state_API_instances[ptr]
	st := self.GetStateTransition()

	st.BeginBlock(&params.Blk)

	for i := range params.Txs {
		tx := &params.Txs[i]
		txResult := st.ExecuteTransaction(tx)
		retval.ExecutionResults = append(retval.ExecutionResults, txResult)
	}

	enc_rlp(&retval, cb)
}

//export ebla_evm_state_api_distribute_rewards
func ebla_evm_state_api_distribute_rewards(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {

	defer handle_err(cb_err)
	var params struct {
		Rewards_stats []rewards_stats.RewardsStats
	}
	dec_rlp(params_enc, &params)

	var retval struct {
		StateRoot   common.Hash
		TotalReward *big.Int
	}
	self := state_API_instances[ptr]
	st := self.GetStateTransition()

	totalReward := uint256.NewInt(0)
	for i := range params.Rewards_stats {
		reward := st.DistributeRewards(&params.Rewards_stats[i])
		if reward != nil {
			totalReward.Add(totalReward, reward)
		}
	}

	st.EndBlock()
	if totalReward != nil {
		retval.TotalReward = totalReward.ToBig()
	}
	retval.StateRoot = st.PrepareCommit()

	enc_rlp(&retval, cb)
}

//export ebla_evm_state_api_transition_state_commit
func ebla_evm_state_api_transition_state_commit(
	ptr C.ebla_evm_state_API_ptr,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	state_API_instances[ptr].GetStateTransition().Commit()
}

//export ebla_evm_state_api_dpos_is_eligible
func ebla_evm_state_api_dpos_is_eligible(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb_err C.ebla_evm_BytesCallback,
) bool {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
	}
	dec_rlp(params_enc, &params)

	// If validator is jailed, return false
	if state_API_instances[ptr].SlashingReader(params.BlkNum).IsJailed(params.BlkNum, &params.Addr) {
		return false
	}

	return state_API_instances[ptr].DPOSDelayedReader(params.BlkNum).IsEligible(&params.Addr)
}

//export ebla_evm_state_api_dpos_get_staking_balance
func ebla_evm_state_api_dpos_get_staking_balance(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
	}
	dec_rlp(params_enc, &params)
	call_bytes_cb(state_API_instances[ptr].DPOSDelayedReader(params.BlkNum).GetStakingBalance(&params.Addr).Bytes(), cb)
}

//export ebla_evm_state_api_dpos_get_vrf_key
func ebla_evm_state_api_dpos_get_vrf_key(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
	}
	dec_rlp(params_enc, &params)
	call_bytes_cb(state_API_instances[ptr].DPOSDelayedReader(params.BlkNum).GetVrfKey(&params.Addr), cb)
}

//export ebla_evm_state_api_dpos_total_amount_delegated
func ebla_evm_state_api_dpos_total_amount_delegated(
	ptr C.ebla_evm_state_API_ptr,
	blk_n uint64,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	call_bytes_cb(state_API_instances[ptr].DPOSReader(blk_n).TotalAmountDelegated().Bytes(), cb)
}

//export ebla_evm_state_api_dpos_eligible_vote_count
func ebla_evm_state_api_dpos_eligible_vote_count(
	ptr C.ebla_evm_state_API_ptr,
	blk_n uint64,
	cb_err C.ebla_evm_BytesCallback,
) uint64 {
	defer handle_err(cb_err)
	return state_API_instances[ptr].DPOSDelayedReader(blk_n).TotalEligibleVoteCount()
}

//export ebla_evm_state_api_dpos_get_eligible_vote_count
func ebla_evm_state_api_dpos_get_eligible_vote_count(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb_err C.ebla_evm_BytesCallback,
) uint64 {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
	}
	dec_rlp(params_enc, &params)
	return state_API_instances[ptr].DPOSDelayedReader(params.BlkNum).GetEligibleVoteCount(&params.Addr)
}

//export ebla_evm_state_api_db_snapshot
func ebla_evm_state_api_db_snapshot(
	ptr C.ebla_evm_state_API_ptr,
	dir string,
	log_size_for_flush uint64,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	db := &state_API_instances[ptr].db
	util.PanicIfNotNil(db.Snapshot(dir, log_size_for_flush))
}

//export ebla_evm_state_api_prune
func ebla_evm_state_api_prune(
	ptr C.ebla_evm_state_API_ptr,
	params_enc C.ebla_evm_Bytes,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		StateRootToKeep []common.Hash
		BlkNum          types.BlockNum
	}
	dec_rlp(params_enc, &params)
	state_API_instances[ptr].db.Prune(params.StateRootToKeep, params.BlkNum)
}

//export ebla_evm_state_api_validators_stakes
func ebla_evm_state_api_validators_stakes(
	ptr C.ebla_evm_state_API_ptr,
	blk_n uint64,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	ret := state_API_instances[ptr].DPOSReader(blk_n).GetValidatorsTotalStakes()
	enc_rlp(&ret, cb)
}

//export ebla_evm_state_api_validators_vote_counts
func ebla_evm_state_api_validators_vote_counts(
	ptr C.ebla_evm_state_API_ptr,
	blk_n uint64,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	ret := state_API_instances[ptr].DPOSDelayedReader(blk_n).GetValidatorsVoteCounts()
	enc_rlp(&ret, cb)
}

//export ebla_evm_state_api_dpos_yield
func ebla_evm_state_api_dpos_yield(
	ptr C.ebla_evm_state_API_ptr,
	blk_n uint64,
	cb_err C.ebla_evm_BytesCallback,
) uint64 {
	defer handle_err(cb_err)
	return state_API_instances[ptr].DPOSDelayedReader(blk_n).GetYield()
}

//export ebla_evm_state_api_dpos_total_supply
func ebla_evm_state_api_dpos_total_supply(
	ptr C.ebla_evm_state_API_ptr,
	blk_n uint64,
	cb C.ebla_evm_BytesCallback,
	cb_err C.ebla_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	call_bytes_cb(state_API_instances[ptr].DPOSReader(blk_n).GetTotalSupply().Bytes(), cb)
}

type state_API_ptr = byte

const state_API_max_instances = ^state_API_ptr(0)

var state_API_alloc_mu sync.Mutex
var state_API_instances [state_API_max_instances]*state_API
var state_API_available_ptrs = func() (ret []state_API_ptr) {
	ret = make([]state_API_ptr, state_API_max_instances)
	for i := state_API_ptr(0); i < state_API_max_instances; i++ {
		ret[i] = i
	}
	return
}()

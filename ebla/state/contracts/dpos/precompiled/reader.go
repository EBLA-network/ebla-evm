package dpos

import (
	"math/big"

	"github.com/EBLA-network/ebla-evm/common"
	"github.com/EBLA-network/ebla-evm/rlp"
	"github.com/holiman/uint256"

	chain_config "github.com/EBLA-network/ebla-evm/ebla/state/chain_config"
	slashing "github.com/EBLA-network/ebla-evm/ebla/state/contracts/slashing/precompiled"
	storage "github.com/EBLA-network/ebla-evm/ebla/state/contracts/storage"

	"github.com/EBLA-network/ebla-evm/core/types"

	"github.com/EBLA-network/ebla-evm/ebla/util/bigutil"
	"github.com/EBLA-network/ebla-evm/ebla/util/bin"
)

type Reader struct {
	cfg             *chain_config.ChainConfig
	block_n         types.BlockNum
	storage         *storage.StorageReaderWrapper
	slashing_reader *slashing.Reader
}

type ValidatorStake struct {
	Address    common.Address
	TotalStake *big.Int
}

type ValidatorVoteCount struct {
	Address   common.Address
	VoteCount uint64
}

func (r *Reader) Init(cfg *chain_config.ChainConfig, blk_n types.BlockNum, storage_factory func(types.BlockNum) storage.StorageReader) *Reader {
	r.cfg = cfg
	r.block_n = blk_n

	r.storage = new(storage.StorageReaderWrapper).Init(dpos_contract_address, storage_factory(r.block_n))
	r.slashing_reader = new(slashing.Reader).Init(cfg, blk_n, r, storage_factory)
	return r
}

func (r *Reader) InitDelayedReader(cfg *chain_config.ChainConfig, blk_n types.BlockNum, storage_factory func(types.BlockNum) storage.StorageReader) *Reader {
	r.cfg = cfg

	r.block_n = uint64(0)
	if uint64(r.cfg.DPOS.DelegationDelay) < blk_n {
		r.block_n = blk_n - uint64(r.cfg.DPOS.DelegationDelay)
	}

	r.storage = new(storage.StorageReaderWrapper).Init(dpos_contract_address, storage_factory(r.block_n))
	r.slashing_reader = new(slashing.Reader).Init(cfg, blk_n, r, storage_factory)
	return r
}

func (r Reader) TotalEligibleVoteCount() (ret uint64) {
	r.storage.Get(storage.Stor_k_1(field_eligible_vote_count), func(bytes []byte) {
		ret = bin.DEC_b_endian_compact_64(bytes)
	})
	for _, addr := range r.slashing_reader.GetJailedValidators() {
		ret -= r.GetEligibleVoteCount(&addr)
	}
	return
}

func (r Reader) GetEligibleVoteCount(addr *common.Address) (ret uint64) {
    factor := r.getVotingPowerFactor(addr)
    return voteCountWithFactor(r.GetStakingBalance(addr), r.cfg, r.block_n, factor)
}

func (r Reader) TotalAmountDelegated() (ret *big.Int) {
	ret = big.NewInt(0)
	r.storage.Get(storage.Stor_k_1(field_amount_delegated), func(bytes []byte) {
		ret = bigutil.FromBytes(bytes)
	})
	return
}

func (r Reader) IsEligible(address *common.Address) bool {
    factor := r.getVotingPowerFactor(address)
    if factor == 0 {
        return false
    }
    effective := new(big.Int).Mul(r.GetStakingBalance(address), big.NewInt(int64(factor)))
    effective.Div(effective, big.NewInt(10000))
    return r.cfg.DPOS.EligibilityBalanceThreshold.Cmp(effective) <= 0
}

func (r Reader) GetStakingBalance(addr *common.Address) (ret *big.Int) {
	ret = big.NewInt(0)
	r.storage.Get(storage.Stor_k_1(field_validators, validator_index, addr[:]), func(bytes []byte) {
		validator := new(Validator)
		validator.ValidatorV1 = new(ValidatorV1)

		err := rlp.DecodeBytes(bytes, validator)
		if err != nil {
			// Try to decode into pre-hardfork ValidatorV1 struct first
			err = rlp.DecodeBytes(bytes, validator.ValidatorV1)
			validator.UndelegationsCount = 0
			if err != nil {
				// This should never happen
				panic("Unable to decode validator rlp")
			}
		}

		ret = validator.TotalStake
	})
	return
}

func (r Reader) GetValidatorsTotalStakes() (ret []ValidatorStake) {
	reader := new(storage.AddressesIMapReader)
	reader.Init(r.storage, append(field_validators, validator_list_index...))

	validators, _ := reader.GetAccounts(0, reader.GetCount())

	for _, addr := range validators {
		ret = append(ret, ValidatorStake{Address: addr, TotalStake: r.GetStakingBalance(&addr)})
	}

	return
}

func (r Reader) GetValidatorsVoteCounts() (ret []ValidatorVoteCount) {
	reader := new(storage.AddressesIMapReader)
	reader.Init(r.storage, append(field_validators, validator_list_index...))

	validators, _ := reader.GetAccounts(0, reader.GetCount())

	for _, addr := range validators {
		factor := r.getVotingPowerFactor(&addr)
        ret = append(ret, ValidatorVoteCount{Address: addr, VoteCount: voteCountWithFactor(r.GetStakingBalance(&addr), r.cfg, r.block_n, factor)})
	}

	return
}

func (r Reader) GetVrfKey(addr *common.Address) (ret []byte) {
	r.storage.Get(storage.Stor_k_1(field_validators, validator_vrf_index, addr[:]), func(bytes []byte) {
		ret = bytes
	})
	return
}

func (r Reader) GetYield() uint64 {
	// Yield is saved & updated since Aspen hardfork
	if !r.cfg.Hardforks.IsOnAspenHardforkPartTwo(r.block_n) {
		return 0
	}

	yield := uint64(0)
	r.storage.Get(storage.Stor_k_1(field_yield), func(bytes []byte) {
		rlp.MustDecodeBytes(bytes, &yield)
	})

	// To get percents -> yield / 10000
	// To get fraction -> yield / 1000000 (YieldFractionDecimalPrecision)

	return yield
}

func (r Reader) GetTotalSupply() *big.Int {
	// Total supply is saved & updated since Aspen hardfork
	if !r.cfg.Hardforks.IsOnAspenHardforkPartTwo(r.block_n) {
		return big.NewInt(0)
	}

	total_supply := uint256.NewInt(0)
	r.storage.Get(storage.Stor_k_1(field_total_supply), func(bytes []byte) {
		total_supply.SetBytes(bytes)
	})

	return total_supply.ToBig()
}

// getVotingPowerFactor reads the voting power factor from storage for the Reader.
// Uses sentinel encoding: stored value is factor+1, 0 means not set (returns 10000).
func (r Reader) getVotingPowerFactor(addr *common.Address) uint64 {
    var stored uint64
    r.storage.Get(storage.Stor_k_1(field_voting_power_factor, addr[:]), func(bytes []byte) {
        stored = bin.DEC_b_endian_compact_64(bytes)
    })
    if stored == 0 {
        return 10000
    }
    return stored - 1
}

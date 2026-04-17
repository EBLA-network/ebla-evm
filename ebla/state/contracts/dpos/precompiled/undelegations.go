package dpos

import (
	"math/big"

	contract_storage "github.com/EBLA-network/ebla-evm/ebla/state/contracts/storage"

	"github.com/EBLA-network/ebla-evm/common"
	"github.com/EBLA-network/ebla-evm/core/types"
	"github.com/EBLA-network/ebla-evm/rlp"
)


// Post cornus hardfork - with undelegation Id. Ids are used to support multiple undelegations at the same time
type Undelegation struct {
	Amount *big.Int
	Block  types.BlockNum
	// Undelegation id (unique per delegator address)
	Id uint64
}

type DelegatorUndelegations struct {
	// list of validators addresses, from which delegator undelegated
	Validators *contract_storage.AddressesIMap
	// <validator address -> list of undelegations ids> as each delegator can have multiple undelegations from the same validator at the same time
	// Note 1: used for post corvus hardfork undelegations processing
	// Note 2: Undelegations_ids_map should contain only validators addresses that are also in Validators struct member
	Undelegations_ids_map map[common.Address]*contract_storage.IdsIMap
}

type Undelegations struct {
	storage *contract_storage.StorageWrapper

	undelegations_field                            []byte
	delegator_undelegations_field               []byte
	delegator_undelegations_ids_field           []byte
	delegator_undelegations_last_uniqe_id_field []byte
}

func (self *Undelegations) Init(stor *contract_storage.StorageWrapper, prefix []byte) {
	self.storage = stor

	// Init Delegations storage fields keys - relative to the prefix
	self.undelegations_field = append(prefix, []byte{0}...)
	self.delegator_undelegations_field = append(prefix, []byte{2}...)
	self.delegator_undelegations_ids_field = append(prefix, []byte{3}...)
	self.delegator_undelegations_last_uniqe_id_field = append(prefix, []byte{4}...)
}

// Returns true if for given values there is undelegation in queue
func (self *Undelegations) UndelegationExists(delegator_address *common.Address, validator_address *common.Address, undelegation_id uint64) bool { 	
	return self.undelegationExists(delegator_address, validator_address, undelegation_id)
}

func (self *Undelegations) undelegationExists(delegator_address *common.Address, validator_address *common.Address, undelegation_id uint64) bool {
	_, ids_map := self.GetUndelegationsMaps(delegator_address, validator_address)
	return ids_map.IdExists(undelegation_id)
}

// NEW — returns the full Undelegation object (replaces GetUndelegationBaseObject)
func (self *Undelegations) GetUndelegationBaseObject(delegator_address *common.Address, validator_address *common.Address, undelegation_id uint64) *Undelegation {
    return self.GetUndelegation(delegator_address, validator_address, undelegation_id)
}

func (self *Undelegations) GetUndelegation(delegator_address *common.Address, validator_address *common.Address, undelegation_id uint64) (undelegation *Undelegation) {
	key := self.genUndelegationKey(delegator_address, validator_address, undelegation_id)
	self.storage.Get(key, func(bytes []byte) {
		undelegation = new(Undelegation)
		rlp.MustDecodeBytes(bytes, undelegation)
	})

	return
}

// Returns number of V2 undelegations for specified address
func (self *Undelegations) GetUndelegationsCount(delegator_address *common.Address) uint32 {
	count := uint32(0)

	undelegations_validators, _ := self.GetUndelegationsMaps(delegator_address, nil)
	for _, undelegations_validator := range undelegations_validators.GetAllAccounts() {
		_, undelegations_ids := self.GetUndelegationsMaps(delegator_address, &undelegations_validator)

		count += undelegations_ids.GetCount()
	}

	return count
}


func (self *Undelegations) CreateUndelegation(delegator_address *common.Address, validator_address *common.Address, block types.BlockNum, amount *big.Int) uint64 {
	undelegation := new(Undelegation)
	undelegation.Amount = amount
	undelegation.Block = block
	undelegation.Id = self.genUniqueId(delegator_address)

	self.saveUndelegationObject(self.genUndelegationKey(delegator_address, validator_address, undelegation.Id), rlp.MustEncodeToBytes(undelegation))

	validators_map, ids_map := self.GetUndelegationsMaps(delegator_address, validator_address)
	if ids_map.CreateId(undelegation.Id) == 1 {
		validators_map.CreateAccount(validator_address)
	}

	return undelegation.Id
}

func (self *Undelegations) genUniqueId(delegator_address *common.Address) uint64 {
	key := contract_storage.Stor_k_1(self.delegator_undelegations_last_uniqe_id_field, delegator_address[:])

	unique_id := uint64(0)
	self.storage.Get(key, func(bytes []byte) {
		unique_id = contract_storage.BytesToUint64(bytes)
	})

	unique_id++
	self.storage.Put(key, contract_storage.Uint64ToBytes(unique_id))

	return unique_id
}

// Removes undelegation object from storage
func (self *Undelegations) RemoveUndelegation(delegator_address *common.Address, validator_address *common.Address, undelegation_id uint64) {
		self.removeUndelegation(delegator_address, validator_address, undelegation_id)
}

func (self *Undelegations) removeUndelegation(delegator_address *common.Address, validator_address *common.Address, undelegation_id uint64) {
	self.removeUndelegationObject(self.genUndelegationKey(delegator_address, validator_address, undelegation_id))
	validators_map, ids_map := self.GetUndelegationsMaps(delegator_address, validator_address)
	if ids_map.RemoveId(undelegation_id) == 0 {
		validators_map.RemoveAccount(validator_address)
	}
}

func (self *Undelegations) saveUndelegationObject(key *common.Hash, undelegation_data []byte) {
	self.storage.Put(key, undelegation_data)
}

func (self *Undelegations) removeUndelegationObject(key *common.Hash) {
	self.storage.Put(key, nil)
}


// New EBLA - Returns validator by idx, from which is delegator <delegator_address> currently undelegating v2
func (self *Undelegations) GetUndelegationsValidator(delegator_address *common.Address, validator_idx uint32) (*common.Address, bool) {
	validators_map, _ := self.GetUndelegationsMaps(delegator_address, nil)
	validators, end := validators_map.GetAccounts(validator_idx, 1)
	if len(validators) > 0 {
		return &validators[0], end
	}

	return nil, end
}

// Returns list of undelegations for given address

func (self *Undelegations) GetUndelegationsMaps(delegator_address *common.Address, validator_address *common.Address) (validators_map *contract_storage.AddressesIMap, ids_map *contract_storage.IdsIMap) {
	undelegations_validators := new(DelegatorUndelegations)
	undelegations_validators_prefix := append(self.delegator_undelegations_field, delegator_address[:]...)
	undelegations_validators.Validators = new(contract_storage.AddressesIMap)
	undelegations_validators.Validators.Init(self.storage, undelegations_validators_prefix)

	validators_map = undelegations_validators.Validators

	if validator_address != nil {
		undelegations_ids, ids_found := undelegations_validators.Undelegations_ids_map[*validator_address]
		if !ids_found {
			undelegations_ids_prefix := append(append(self.delegator_undelegations_ids_field, delegator_address[:]...), validator_address[:]...)
			undelegations_ids = new(contract_storage.IdsIMap)
			undelegations_ids.Init(self.storage, undelegations_ids_prefix)
		}

		ids_map = undelegations_ids
	}

	return
}

// Return key to storage where undelegations V2 is stored
func (self *Undelegations) genUndelegationKey(delegator_address *common.Address, validator_address *common.Address, undelegation_id uint64) *common.Hash {
	// Post-cornus hf undelegation key is created from delegator address, validator address & and undelegation id
	return contract_storage.Stor_k_1(self.undelegations_field, delegator_address[:], validator_address[:], contract_storage.Uint64ToBytes(undelegation_id))
}

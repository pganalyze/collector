package state

type OidToIdxMap map[Oid](map[Oid]int32)

func MakeOidToIdxMap() OidToIdxMap {
	return make(map[Oid](map[Oid]int32))
}

func (m OidToIdxMap) Put(dbOid, objOid Oid, idx int32) {
	if _, ok := m[dbOid]; !ok {
		m[dbOid] = make(map[Oid]int32)
	}
	m[dbOid][objOid] = idx
}

func (m OidToIdxMap) Get(dbOid, objOid Oid) int32 {
	if _, ok := m[dbOid]; !ok {
		return -1
	}
	idx, ok := m[dbOid][objOid]
	if !ok {
		return -1
	}
	return idx
}

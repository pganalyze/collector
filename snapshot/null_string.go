package snapshot

func (ns *NullString) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	ns.Valid = true
	return convertAssign(&ns.Value, value)
}

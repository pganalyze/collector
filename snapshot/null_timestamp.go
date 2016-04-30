package snapshot

import "time"

func (ns *NullTimestamp) Scan(value interface{}) error {
	var time time.Time
	if value == nil {
		return nil
	}
	err := convertAssign(&time, value)
	if err != nil {
		return err
	}
	ns.Valid = true
	ns.Value = time.Unix()
	return nil
}

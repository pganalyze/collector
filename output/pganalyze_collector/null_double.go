package pganalyze_collector

func (ns *NullDouble) Scan(value interface{}) error {
  if value == nil {
    return nil
  }
  ns.Valid = true
  return convertAssign(&ns.Value, value)
}

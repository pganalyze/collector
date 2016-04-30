//go:generate msgp

package snapshot

type Setting struct {
	Name         string         `msg:"name"`
	CurrentValue NullableString `msg:"current_value"`
	Unit         NullableString `msg:"unit"`
	BootValue    NullableString `msg:"boot_value"`
	ResetValue   NullableString `msg:"reset_value"`
	Source       NullableString `msg:"source"`
	SourceFile   NullableString `msg:"sourcefile"`
	SourceLine   NullableString `msg:"sourceline"`
}

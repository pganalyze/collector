package settings

type Setting struct {
	Name        string
	Validate    func(newVal string) error
	Recommended string
	IsSupported func(value string) bool
}

package ledger

// Validatable — контракт проверки данных перед записью.
type Validatable interface {
	Validate() error
}

// CheckValid вызывает Validate у любого значения, которое реализует контракт.
func CheckValid(v Validatable) error {
	return v.Validate()
}

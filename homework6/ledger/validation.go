package ledger

// Validatable is the contract for checking that a value can be stored.
type Validatable interface {
	Validate() error
}

// CheckValid runs validation for any type that implements Validatable.
func CheckValid(v Validatable) error {
	return v.Validate()
}

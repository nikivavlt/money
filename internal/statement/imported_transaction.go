package statement

type Source string

const (
	Revolut  Source = "revolut"
	Swedbank Source = "swedbank"
)

type importedTransaction struct {
	source Source

	accountText     string
	occurredAtText  string
	completedAtText string

	amountText    string
	feeText       string
	currencyText  string
	directionText string

	rawDescription   string
	counterpartyText string

	externalID string
	typeText   string
	stateText  string
}

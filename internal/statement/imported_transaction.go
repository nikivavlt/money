package statement

type statementSource string

const (
	sourceRevolut  statementSource = "revolut"
	sourceSwedbank statementSource = "swedbank"
)

type importedTransaction struct {
	source statementSource

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

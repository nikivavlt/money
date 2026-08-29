package statement

import (
	"encoding/csv"
	"fmt"
	"io"
	"slices"
)

var swedbankRequiredHeaders = []string{
	"Account No",
	"Date",
	"Beneficiary",
	"Details",
	"Amount",
	"Currency",
	"D/K",
	"Record ID",
	"Code",
}

type swedbankRow struct {
	accountNumberText    string
	dateText             string
	beneficiaryText      string
	detailsText          string
	amountText           string
	currencyText         string
	directionText        string
	externalID           string
	codeText             string
	referenceNumberText  string
	documentNumberText   string
	payerCodeText        string
	clientCodeText       string
	originatorText       string
	beneficiaryPartyText string
}

func swedbankColumnIndexes(header []string) (map[string]int, error) {
	indexes, err := indexCSVColumns(header)
	if err != nil {
		return nil, fmt.Errorf(
			"index Swedbank CSV header: %w",
			err,
		)
	}

	if err := requireCSVColumns(
		indexes,
		swedbankRequiredHeaders,
	); err != nil {
		return nil, fmt.Errorf(
			"index Swedbank CSV header: %w",
			err,
		)
	}

	return indexes, nil
}

func swedbankRowFromRecord(
	record []string,
	indexes map[string]int,
) (swedbankRow, error) {
	if err := requireCSVRecordFields(
		record,
		indexes,
		swedbankRequiredHeaders,
	); err != nil {
		return swedbankRow{}, fmt.Errorf(
			"read Swedbank record: %w",
			err,
		)
	}

	return swedbankRow{
		accountNumberText: record[indexes["Account No"]],
		dateText:          record[indexes["Date"]],
		beneficiaryText:   record[indexes["Beneficiary"]],
		detailsText:       record[indexes["Details"]],
		amountText:        record[indexes["Amount"]],
		currencyText:      record[indexes["Currency"]],
		directionText:     record[indexes["D/K"]],
		externalID:        record[indexes["Record ID"]],
		codeText:          record[indexes["Code"]],

		referenceNumberText: optionalCSVField(
			record,
			indexes,
			"Reference No",
		),
		documentNumberText: optionalCSVField(
			record,
			indexes,
			"Doc. No",
		),
		payerCodeText: optionalCSVField(
			record,
			indexes,
			"Code in payer IS",
		),
		clientCodeText: optionalCSVField(
			record,
			indexes,
			"Client code",
		),
		originatorText: optionalCSVField(
			record,
			indexes,
			"Originator",
		),
		beneficiaryPartyText: optionalCSVField(
			record,
			indexes,
			"Beneficiary party",
		),
	}, nil
}

func readSwedbankRows(input io.Reader) ([]swedbankRow, error) {
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1

	header, err := readCSVHeaderRecord(reader)
	if err != nil {
		return nil, fmt.Errorf("read Swedbank CSV: %w", err)
	}

	rows, _, err := readSwedbankRowsAfterHeader(reader, header)
	return rows, err
}

func (r swedbankRow) toImportedTransaction() importedTransaction {
	return importedTransaction{
		source:           Swedbank,
		accountText:      r.accountNumberText,
		occurredAtText:   r.dateText,
		completedAtText:  "",
		amountText:       r.amountText,
		feeText:          "",
		currencyText:     r.currencyText,
		directionText:    r.directionText,
		rawDescription:   r.detailsText,
		counterpartyText: r.beneficiaryText,
		externalID:       r.externalID,
		typeText:         r.codeText,
		stateText:        "",
	}
}

func importSwedbankTransactions(
	input io.Reader,
) ([]importedTransaction, error) {
	rows, err := readSwedbankRows(input)
	if err != nil {
		return nil, fmt.Errorf(
			"import Swedbank transactions: %w",
			err,
		)
	}

	transactions := make(
		[]importedTransaction,
		len(rows),
	)

	for index, row := range rows {
		transactions[index] = row.toImportedTransaction()
	}

	return transactions, nil
}

func swedbankRowsToImportedTransactions(
	rows []swedbankRow,
) []importedTransaction {
	transactions := make(
		[]importedTransaction,
		len(rows),
	)

	for index, row := range rows {
		transactions[index] = row.toImportedTransaction()
	}

	return transactions
}

func readSwedbankRowsAfterHeader(reader *csv.Reader, header []string) ([]swedbankRow, [][]string, error) {
	if reader == nil {
		return nil, nil, fmt.Errorf("read Swedbank CSV: nil reader")
	}

	reader.FieldsPerRecord = -1

	indexes, err := swedbankColumnIndexes(header)
	if err != nil {
		return nil, nil, fmt.Errorf("read Swedbank CSV: %w", err)
	}

	var (
		rows       []swedbankRow
		rawRecords [][]string
	)

	for recordNumber := 2; ; recordNumber++ {
		record, err := reader.Read()
		if err == io.EOF {
			return rows, rawRecords, nil
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read Swedbank CSV record %d: %w", recordNumber, err)
		}

		row, err := swedbankRowFromRecord(record, indexes)
		if err != nil {
			return nil, nil, fmt.Errorf("parse Swedbank CSV record %d: %w", recordNumber, err)
		}

		rows = append(rows, row)
		rawRecords = append(rawRecords, slices.Clone(record))
	}
}

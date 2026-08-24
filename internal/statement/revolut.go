package statement

import (
	"encoding/csv"
	"fmt"
	"io"
)

var revolutRequiredHeaders = []string{
	"Type",
	"Product",
	"Started Date",
	"Completed Date",
	"Description",
	"Amount",
	"Fee",
	"Currency",
	"State",
}

type revolutRow struct {
	transactionType   string
	product           string
	startedDateText   string
	completedDateText string
	rawDescription    string
	amountText        string
	feeText           string
	currencyText      string
	stateText         string
	balanceText       string
}

func revolutColumnIndexes(header []string) (map[string]int, error) {
	indexes, err := indexCSVColumns(header)
	if err != nil {
		return nil, fmt.Errorf(
			"index Revolut CSV header: %w",
			err,
		)
	}

	if err := requireCSVColumns(
		indexes,
		revolutRequiredHeaders,
	); err != nil {
		return nil, fmt.Errorf(
			"index Revolut CSV header: %w",
			err,
		)
	}

	return indexes, nil
}

func revolutRowFromRecord(
	record []string,
	indexes map[string]int,
) (revolutRow, error) {
	if err := requireCSVRecordFields(
		record,
		indexes,
		revolutRequiredHeaders,
	); err != nil {
		return revolutRow{}, fmt.Errorf(
			"read Revolut record: %w",
			err,
		)
	}

	return revolutRow{
		transactionType:   record[indexes["Type"]],
		product:           record[indexes["Product"]],
		startedDateText:   record[indexes["Started Date"]],
		completedDateText: record[indexes["Completed Date"]],
		rawDescription:    record[indexes["Description"]],
		amountText:        record[indexes["Amount"]],
		feeText:           record[indexes["Fee"]],
		currencyText:      record[indexes["Currency"]],
		stateText:         record[indexes["State"]],
		balanceText: optionalCSVField(
			record,
			indexes,
			"Balance",
		),
	}, nil
}

func readRevolutRows(input io.Reader) ([]revolutRow, error) {
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1

	header, err := readCSVHeaderRecord(reader)
	if err != nil {
		return nil, fmt.Errorf("read Revolut CSV: %w", err)
	}

	return readRevolutRowsAfterHeader(reader, header)
}

func (r revolutRow) toImportedTransaction() importedTransaction {
	return importedTransaction{
		source:           sourceRevolut,
		accountText:      r.product,
		occurredAtText:   r.startedDateText,
		completedAtText:  r.completedDateText,
		amountText:       r.amountText,
		feeText:          r.feeText,
		currencyText:     r.currencyText,
		directionText:    "",
		rawDescription:   r.rawDescription,
		counterpartyText: "",
		externalID:       "",
		typeText:         r.transactionType,
		stateText:        r.stateText,
	}
}

func importRevolutTransactions(
	input io.Reader,
) ([]importedTransaction, error) {
	rows, err := readRevolutRows(input)
	if err != nil {
		return nil, fmt.Errorf(
			"import Revolut transactions: %w",
			err,
		)
	}

	return revolutRowsToImportedTransactions(rows), nil
}

func revolutRowsToImportedTransactions(
	rows []revolutRow,
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

func readRevolutRowsAfterHeader(
	reader *csv.Reader,
	header []string,
) ([]revolutRow, error) {
	if reader == nil {
		return nil, fmt.Errorf("read Revolut CSV: nil reader")
	}

	reader.FieldsPerRecord = -1

	indexes, err := revolutColumnIndexes(header)
	if err != nil {
		return nil, fmt.Errorf("read Revolut CSV: %w", err)
	}

	var rows []revolutRow

	for recordNumber := 2; ; recordNumber++ {
		record, err := reader.Read()
		if err == io.EOF {
			return rows, nil
		}
		if err != nil {
			return nil, fmt.Errorf(
				"read Revolut CSV record %d: %w",
				recordNumber,
				err,
			)
		}

		row, err := revolutRowFromRecord(record, indexes)
		if err != nil {
			return nil, fmt.Errorf(
				"parse Revolut CSV record %d: %w",
				recordNumber,
				err,
			)
		}

		rows = append(rows, row)
	}
}

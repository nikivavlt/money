package statement

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

func readCSVHeader(input io.Reader) ([]string, error) {
	return readCSVHeaderRecord(csv.NewReader(input))
}

func readCSVHeaderRecord(reader *csv.Reader) ([]string, error) {
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}

	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\uFEFF")
	}

	return header, nil
}

func readCSVHeaderFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open CSV file %q: %w", path, err)
	}
	defer file.Close()

	header, err := readCSVHeader(file)
	if err != nil {
		return nil, fmt.Errorf("inspect CSV file %q: %w", path, err)
	}

	return header, nil
}

func indexCSVColumns(header []string) (map[string]int, error) {
	indexes := make(map[string]int, len(header))

	for index, name := range header {
		if name == "" {
			continue
		}

		if _, exists := indexes[name]; exists {
			return nil, fmt.Errorf(
				"duplicate column %q",
				name,
			)
		}

		indexes[name] = index
	}

	return indexes, nil
}

func requireCSVColumns(
	indexes map[string]int,
	required []string,
) error {
	for _, name := range required {
		if _, exists := indexes[name]; !exists {
			return fmt.Errorf(
				"missing required column %q",
				name,
			)
		}
	}

	return nil
}

func requireCSVRecordFields(
	record []string,
	indexes map[string]int,
	required []string,
) error {
	for _, column := range required {
		index, exists := indexes[column]
		if !exists {
			return fmt.Errorf(
				"column %q was not indexed",
				column,
			)
		}

		if index < 0 || index >= len(record) {
			return fmt.Errorf(
				"required column %q is absent",
				column,
			)
		}
	}

	return nil
}

func optionalCSVField(
	record []string,
	indexes map[string]int,
	column string,
) string {
	index, exists := indexes[column]
	if !exists || index < 0 || index >= len(record) {
		return ""
	}

	return record[index]
}

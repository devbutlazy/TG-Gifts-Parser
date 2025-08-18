package utils

import (
	"fmt"

	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/reader"
)

type Row struct {
	Number   int32  `parquet:"name=number, type=INT32"`
	Model    string `parquet:"name=model, type=BYTE_ARRAY, convertedtype=UTF8"`
	Backdrop string `parquet:"name=backdrop, type=BYTE_ARRAY, convertedtype=UTF8"`
	Symbol   string `parquet:"name=symbol, type=BYTE_ARRAY, convertedtype=UTF8"`
}

func QueryEntriesParquet(parquetPath, model, backdrop, symbol string) ([]int, error) {
	fr, err := local.NewLocalFileReader(parquetPath)
	if err != nil {
		return nil, fmt.Errorf("open parquet: %w", err)
	}
	defer fr.Close()

	pr, err := reader.NewParquetReader(fr, new(Row), 4)
	if err != nil {
		return nil, fmt.Errorf("new parquet reader: %w", err)
	}

	defer pr.ReadStop()

	numRows := int(pr.GetNumRows())
	const batchSize = 1000

	matches := make([]int, 0)

	for offset := 0; offset < numRows; offset += batchSize {
		n := batchSize
		if offset+n > numRows {
			n = numRows - offset
		}

		rows := make([]Row, n)
		if err := pr.Read(&rows); err != nil {
			return nil, fmt.Errorf("read parquet rows: %w", err)
		}

		for _, r := range rows {
			dbModelClean := RemovePercent(r.Model)
			dbBackdropClean := RemovePercent(r.Backdrop)
			dbSymbolClean := RemovePercent(r.Symbol)

			modelMatch := dbModelClean == model
			backdropMatch := backdrop == "" || dbBackdropClean == backdrop
			symbolMatch := symbol == "" || dbSymbolClean == symbol

			if modelMatch && backdropMatch && symbolMatch {
				matches = append(matches, int(r.Number))
			}
		}
	}

	return matches, nil
}

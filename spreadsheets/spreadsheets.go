package spreadsheets

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
)

const UploadsDir = "./uploads/"

var ErrWorksheetNotFound = errors.New("лист не найден")

type ContragentsSumMode int

const (
	SumModeStores ContragentsSumMode = iota
	SumModeStoresAndDelivery
	SumModeAll
)

// Восточка — отдельный лист-контрагент, встречается только в опте кондитерки.
// В отличие от остальных контрагентов, он определяется по имени листа, а не по
// заголовку колонки, и на листе всегда ровно одна колонка с количеством.
const VostochkaContragent = "Восточка"
const vostochkaAmountColumn = 2

func GetContragents(fileId string, applicationType string, manufactureType string) ([]string, error) {
	file, err := excelize.OpenFile(UploadsDir + fileId + ".xlsx")
	if err != nil {
		log.Print(err)
		return nil, err
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Print(err)
		}
	}()

	var contragents []string

	if applicationType == "store" {
		contragents, err = getContragentsFromSheet(file, "Хлеб У", 0)
		if err != nil {
			if !errors.Is(err, ErrWorksheetNotFound) {
				return nil, err
			}

			contragents, err = getContragentsFromSheet(file, "Хлеб 1600", 0)
			if err != nil {
				return nil, err
			}
		}
	} else {
		contragents, err = getContragentsFromSheet(file, "Хлеб", 1)
		if err != nil {
			return nil, err
		}
	}

	if applicationType == "wholesale" && manufactureType == "kond" && hasSheet(file, VostochkaContragent) {
		contragents = append(contragents, VostochkaContragent)
	}

	return contragents, nil
}

func hasSheet(file *excelize.File, name string) bool {
	for _, sheet := range file.GetSheetList() {
		if sheet == name {
			return true
		}
	}

	return false
}

func GetWorksheets(manufactureType string) ([]string, error) {
	var worksheets = map[string][]string{
		"bread": {"Хлеб", "Булки", "Макаронсы"},
		"kond":  {"Десерты", "Торты", "Торты 2", "Нарезные-пир", "Пирожные"},
	}

	res, ok := worksheets[manufactureType]
	if !ok {
		return nil, errors.New("неверный тип прозоводства")
	}

	return res, nil
}

func GetDaytimes(application_type string) ([]string, error) {
	var daytimes = map[string][]string{
		"store":     {"Утро", "День", "Дозавоз", "12:00", "14:00", "16:00", "18:00"},
		"wholesale": {},
	}

	res, ok := daytimes[application_type]

	if !ok {
		return nil, errors.New("неверный тип заявки")
	}

	return res, nil
}

func GetInvoiceNew(invoiceId string, contragent string, daytime *string, worksheets []string, applicationType string) ([][]string, error) {
	file, err := excelize.OpenFile(UploadsDir + invoiceId + ".xlsx")
	if err != nil {
		log.Print(err)
		return nil, err
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Print(err)
		}
	}()

	shift := getShift(applicationType)
	rowStart := 2 + shift
	invoice := make([][]string, 0)
	worksheetInvoices := make([][][]string, len(worksheets))

	var wg sync.WaitGroup
	wg.Add(len(worksheets))

	var once sync.Once
	var firstErr error

	for worksheetIndex, worksheet := range worksheets {
		worksheetIndex := worksheetIndex
		worksheet := worksheet
		go func() {
			defer wg.Done()

			sheetName := withDaytimeSuffix(worksheet, daytime)

			rows, err := file.GetRows(sheetName)
			if err != nil {
				once.Do(func() {
					firstErr = mapWorksheetError(err, sheetName)
				})
				return
			}

			if len(rows) == 0 {
				return
			}

			contragentCol := -1
			header := []string{}
			if shift >= 0 && shift < len(rows) {
				header = rows[shift]
			}

			if sheetName == VostochkaContragent {
				contragentCol = vostochkaAmountColumn
			} else {
				for colIndex := 2; colIndex < len(header); colIndex++ {
					if header[colIndex] == contragent {
						contragentCol = colIndex
						break
					}
				}
			}

			if contragentCol == -1 {
				once.Do(func() {
					firstErr = errors.New("контрагент не найден")
				})
				return
			}

			worksheetInvoice := make([][]string, 0)
			for rowIndex := rowStart; rowIndex < len(rows); rowIndex++ {
				product := getCellValue(rows, rowIndex, 0)
				if product == "Ноль когда закончили" {
					break
				}

				article := getCellValue(rows, rowIndex, 1)
				amount := getCellValue(rows, rowIndex, contragentCol)
				if len(product) == 0 || len(article) == 0 || len(amount) == 0 {
					continue
				}

				worksheetInvoice = append(worksheetInvoice, []string{product, article, amount})
			}

			worksheetInvoices[worksheetIndex] = worksheetInvoice
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	for _, worksheetInvoice := range worksheetInvoices {
		invoice = append(invoice, worksheetInvoice...)
	}

	return invoice, nil
}

func GetInvoiceAllContragentsByMode(invoiceId string, daytime *string, worksheets []string, applicationType string, mode ContragentsSumMode) ([][]string, error) {
	file, err := excelize.OpenFile(UploadsDir + invoiceId + ".xlsx")
	if err != nil {
		log.Print(err)
		return nil, err
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Print(err)
		}
	}()

	shift := getShift(applicationType)
	rowStart := 2 + shift
	invoice := make([][]string, 0)
	worksheetInvoices := make([][][]string, len(worksheets))

	var wg sync.WaitGroup
	wg.Add(len(worksheets))

	var once sync.Once
	var firstErr error

	for worksheetIndex, worksheet := range worksheets {
		worksheetIndex := worksheetIndex
		worksheet := worksheet
		go func() {
			defer wg.Done()

			sheetName := withDaytimeSuffix(worksheet, daytime)
			rows, err := file.GetRows(sheetName)
			if err != nil {
				once.Do(func() {
					firstErr = mapWorksheetError(err, sheetName)
				})
				return
			}

			if len(rows) == 0 {
				return
			}

			header := []string{}
			if shift >= 0 && shift < len(rows) {
				header = rows[shift]
			}

			storeCols, deliveryCol, optCols := splitContragentColumns(header)
			if len(storeCols) == 0 {
				once.Do(func() {
					firstErr = errors.New("контрагенты не найдены")
				})
				return
			}

			sumCols := append([]int{}, storeCols...)
			if mode == SumModeStoresAndDelivery || mode == SumModeAll {
				if deliveryCol != -1 {
					sumCols = append(sumCols, deliveryCol)
				}
			}
			if mode == SumModeAll {
				sumCols = append(sumCols, optCols...)
			}

			worksheetInvoice := make([][]string, 0)
			for rowIndex := rowStart; rowIndex < len(rows); rowIndex++ {
				product := getCellValue(rows, rowIndex, 0)
				if product == "Ноль когда закончили" {
					break
				}

				article := getCellValue(rows, rowIndex, 1)
				if len(product) == 0 || len(article) == 0 {
					continue
				}

				sum := 0.0
				hasAmount := false
				for _, colIndex := range sumCols {
					amountRaw := getCellValue(rows, rowIndex, colIndex)
					parsedAmount, ok, err := parseAmount(amountRaw)
					if err != nil {
						once.Do(func() {
							firstErr = err
						})
						return
					}
					if !ok {
						continue
					}

					hasAmount = true
					sum += parsedAmount
				}

				if !hasAmount {
					continue
				}

				worksheetInvoice = append(worksheetInvoice, []string{product, article, formatAmount(sum)})
			}

			worksheetInvoices[worksheetIndex] = worksheetInvoice
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	for _, worksheetInvoice := range worksheetInvoices {
		invoice = append(invoice, worksheetInvoice...)
	}

	return invoice, nil
}

func splitContragentColumns(header []string) (storeCols []int, deliveryCol int, optCols []int) {
	deliveryCol = -1

	for colIndex := 2; colIndex < len(header); colIndex++ {
		cell := strings.TrimSpace(header[colIndex])
		if cell == "" {
			continue
		}

		if cell == "Доставка" {
			deliveryCol = colIndex
			continue
		}

		if deliveryCol == -1 {
			storeCols = append(storeCols, colIndex)
		} else {
			optCols = append(optCols, colIndex)
		}
	}

	return storeCols, deliveryCol, optCols
}

func getCellValue(rows [][]string, rowIndex int, colIndex int) string {
	if rowIndex < 0 || rowIndex >= len(rows) {
		return ""
	}

	row := rows[rowIndex]
	if colIndex < 0 || colIndex >= len(row) {
		return ""
	}

	return row[colIndex]
}

func parseAmount(raw string) (float64, bool, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(raw), ",", ".")
	if normalized == "" {
		return 0, false, nil
	}

	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, false, errors.New("некорректное количество в заявке")
	}

	return value, true, nil
}

func formatAmount(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}

	return strconv.FormatFloat(value, 'f', -1, 64)
}

func getShift(applicationType string) int {
	switch applicationType {
	case "wholesale":
		return 1
	default:
		return 0
	}
}

func withDaytimeSuffix(worksheet string, daytime *string) string {
	if daytime == nil || *daytime == "" {
		return worksheet
	}

	switch *daytime {
	case "Утро":
		return worksheet + " У"
	case "День":
		return worksheet + " Д"
	case "Дозавоз":
		return worksheet + " Доз."
	case "12:00":
		return worksheet + " 1200"
	case "14:00":
		return worksheet + " 1400"
	case "16:00":
		return worksheet + " 1600"
	case "18:00":
		return worksheet + " 1800"
	default:
		return worksheet + " " + *daytime
	}
}

func getContragentsFromSheet(file *excelize.File, sheetName string, rowNumber int) ([]string, error) {
	rows, err := file.GetRows(sheetName)
	if err != nil {
		return nil, mapWorksheetError(err, sheetName)
	}

	if rowNumber < 0 || rowNumber >= len(rows) {
		return nil, errors.New("некорректная структура файла: отсутствует строка с контрагентами")
	}

	header := rows[rowNumber]
	if len(header) <= 2 {
		return nil, errors.New("некорректная структура файла: нет контрагентов")
	}

	// Строка данных сразу под строкой с датами (rowNumber+1), которая идёт под заголовком.
	firstDataRow := rowNumber + 2

	contragents := make([]string, 0, len(header)-2)
	for colIndex := 2; colIndex < len(header); colIndex++ {
		cell := strings.TrimSpace(header[colIndex])
		if cell == "" {
			continue
		}

		// Итоговые колонки (например "Опт Г"/"ОПТ Д") считаются формулой SUM
		// по остальным контрагентам, а не вводятся вручную — это не контрагент.
		if isComputedColumn(file, sheetName, colIndex, firstDataRow) {
			continue
		}

		contragents = append(contragents, cell)
		if cell == "Доставка" {
			break
		}
	}

	if len(contragents) == 0 {
		return nil, errors.New("контрагенты не найдены")
	}

	return contragents, nil
}

func isComputedColumn(file *excelize.File, sheetName string, colIndex int, dataRow int) bool {
	cellRef, err := excelize.CoordinatesToCellName(colIndex+1, dataRow+1)
	if err != nil {
		return false
	}

	formula, err := file.GetCellFormula(sheetName, cellRef)
	if err != nil {
		return false
	}

	return formula != ""
}

func mapWorksheetError(err error, sheetName string) error {
	if err == nil {
		return nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
		return fmt.Errorf("%w: %s", ErrWorksheetNotFound, sheetName)
	}

	return err
}

func DeleteFile(fileID string) error {
	err := os.Remove(UploadsDir + fileID + ".xlsx")

	if err != nil {
		return errors.New("ошибка удаления файла")
	}

	return nil
}

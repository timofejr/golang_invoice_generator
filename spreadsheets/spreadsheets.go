package spreadsheets

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
)

const UploadsDir = "./uploads/"

func GetContragents(fileId string, applicationType string) ([]string, error) {
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

	var sheetName string
	var rowNumber int

	if applicationType == "store" {
		sheetName = "Хлеб У"
		rowNumber = 0
	} else {
		sheetName = "Хлеб"
		rowNumber = 1
	}

	rows, err := file.GetRows(sheetName)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	contragents := make([]string, 0)

	for _, cell := range rows[rowNumber][2 : len(rows[1])-4] {
		contragents = append(contragents, cell)

		if cell == "Доставка" {
			break
		}
	}

	return contragents, nil
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
		"store":     {"Утро", "День", "Дозавоз"},
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
					firstErr = err
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
			for colIndex := 2; colIndex < len(header); colIndex++ {
				if header[colIndex] == contragent {
					contragentCol = colIndex
					break
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

func GetInvoiceAllContragents(invoiceId string, daytime *string, worksheets []string, applicationType string) ([][]string, error) {
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
					firstErr = err
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

			contragentCols := getAllContragentColumns(header)
			if len(contragentCols) == 0 {
				once.Do(func() {
					firstErr = errors.New("контрагенты не найдены")
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
				if len(product) == 0 || len(article) == 0 {
					continue
				}

				sum := 0.0
				hasAmount := false
				for _, colIndex := range contragentCols {
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

func getAllContragentColumns(header []string) []int {
	columns := make([]int, 0)

	for colIndex := 2; colIndex < len(header); colIndex++ {
		cell := strings.TrimSpace(header[colIndex])
		if cell == "" {
			continue
		}

		columns = append(columns, colIndex)
		if cell == "Доставка" {
			break
		}
	}

	return columns
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
	default:
		return worksheet
	}
}

func DeleteFile(fileID string) error {
	err := os.Remove(UploadsDir + fileID + ".xlsx")

	if err != nil {
		return errors.New("ошибка удаления файла")
	}

	return nil
}

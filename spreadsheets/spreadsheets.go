package spreadsheets

import (
	"errors"
	"log"
	"os"

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

	for _, worksheet := range worksheets {
		sheetName := withDaytimeSuffix(worksheet, daytime)

		cols, err := file.GetCols(sheetName)
		if err != nil {
			return nil, err
		}

		if len(cols) < 3 {
			continue
		}

		contragentCol := -1
		for colIndex := 2; colIndex < len(cols); colIndex++ {
			col := cols[colIndex]
			if shift < len(col) && col[shift] == contragent {
				contragentCol = colIndex
				break
			}
		}

		if contragentCol == -1 {
			return nil, errors.New("контрагент не найден")
		}

		productsCol := cols[0]
		articlesCol := cols[1]
		amountsCol := cols[contragentCol]

		rowsLimit := min(len(productsCol), len(articlesCol), len(amountsCol))
		for rowIndex := rowStart; rowIndex < rowsLimit; rowIndex++ {
			product := productsCol[rowIndex]
			if product == "Ноль когда закончили" {
				break
			}

			article := articlesCol[rowIndex]
			amount := amountsCol[rowIndex]
			if len(product) == 0 || len(article) == 0 || len(amount) == 0 {
				continue
			}

			invoice = append(invoice, []string{product, article, amount})
		}
	}

	return invoice, nil
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
package page_constructor

import "fmt"

func GetMainPage() []byte {
	return []byte(getMainTemplate())
}

func GetDetailedCarPage(page string) []byte {
	template := getDetailedCarTemplate()
	result := fmt.Sprintf(template, page)
	return []byte(result)
}

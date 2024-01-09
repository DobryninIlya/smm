package page_constructor

func GetMainPage() []byte {
	return []byte(getMainTemplate())
}

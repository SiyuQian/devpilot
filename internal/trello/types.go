package trello

type Board struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type List struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Card struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	IDList   string `json:"idList"`
	ShortURL string `json:"shortUrl"`
}

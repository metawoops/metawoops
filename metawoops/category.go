package metawoops

import (
	"encoding/json"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/appengine/urlfetch"
)

type category struct {
	ID        int64
	Name      string
	Color     string
	TextColor string `json:"text_color"`
}

type categoriesResponse struct {
	CategoryList struct {
		Categories []category
	} `json:"category_list"`
}

func loadCategories(ctx context.Context, site string) []category {
	client := urlfetch.Client(ctx)
	resp, err := client.Get(site + "/categories.json")
	if err != nil {
		log.Println(err)
		return nil
	}
	defer resp.Body.Close()

	var res categoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Println(err)
		return nil
	}

	return res.CategoryList.Categories
}

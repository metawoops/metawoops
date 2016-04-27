package metawoops

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"sort"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine/urlfetch"
)

type topic struct {
	ID            int64
	Title         string
	Updated       time.Time
	Deleted       time.Time
	CategoryName  string
	CategoryColor string
}

type post struct {
	ID         int64
	Name       string
	Username   string
	Created    time.Time `json:"created_at"`
	Cooked     string    `datastore:",noindex"`
	Raw        string    `datastore:",noindex"`
	Number     int       `json:"post_number"`
	TopicID    int64     `json:"topic_id"`
	TopicTitle string    `json:"topic_title"`
	Hidden     bool
	ActionCode string `json:"action_code"`
	CategoryID int64  `json:"category_id"`

	Deleted time.Time `json:"-"`
}

func (p post) CookedHTML() template.HTML {
	return template.HTML(p.Cooked)
}

type postsResponse struct {
	Latest []post `json:"latest_posts"`
}

func loadLatestPosts(ctx context.Context, site string) []post {
	client := urlfetch.Client(ctx)
	resp, err := client.Get(site + "/posts.json")
	if err != nil {
		log.Println(err)
		return nil
	}
	defer resp.Body.Close()

	var res postsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Println(err)
		return nil
	}

	sort.Sort(postList(res.Latest))

	return res.Latest
}

type topicResponse struct {
	PostStream struct {
		Posts  []post
		Stream []int
	} `json:"post_stream"`
}

func loadTopic(ctx context.Context, site string, id int64) []post {
	client := urlfetch.Client(ctx)
	page := 1
	var posts []post
	for {
		resp, err := client.Get(fmt.Sprintf("%s/t/%d.json?page=%d", site, id, page))
		if err != nil {
			log.Println(err)
			break
		}
		defer resp.Body.Close()

		var res topicResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			log.Println(err)
			break
		}

		posts = append(posts, res.PostStream.Posts...)
		if len(posts) == len(res.PostStream.Stream) {
			break
		}
		page++
	}
	return posts
}

var topicDeleted = errors.New("topic deleted")

func loadTopicStream(ctx context.Context, site string, id int64) ([]int, error) {
	client := urlfetch.Client(ctx)
	resp, err := client.Get(fmt.Sprintf("%s/t/%d.json", site, id))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 410 || resp.StatusCode == 403 || resp.StatusCode == 404 {
		return nil, topicDeleted
	}
	if resp.StatusCode > 299 {
		return nil, errors.New(resp.Status)
	}

	var res topicResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Println(err)
		return nil, err
	}

	return res.PostStream.Stream, nil
}

type postList []post

func (l postList) Len() int {
	return len(l)
}
func (l postList) Swap(a, b int) {
	l[a], l[b] = l[b], l[a]
}
func (l postList) Less(a, b int) bool {
	return l[a].Created.Before(l[b].Created)
}

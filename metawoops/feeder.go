package metawoops

import (
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

func init() {
	http.Handle("/feed", &feeder{
		site:       "https://meta.discourse.org",
		categories: make(map[int64]category),
	})
}

type feeder struct {
	mut     sync.Mutex
	running bool

	site string

	categories map[int64]category
	lastPostID int64
}

func (f *feeder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	f.mut.Lock()
	if f.running {
		http.Error(w, "already running", http.StatusConflict)
		f.mut.Unlock()
		return
	}
	f.running = true
	f.mut.Unlock()

	defer func() {
		f.mut.Lock()
		f.running = false
		f.mut.Unlock()
	}()

	ctx := appengine.NewContext(req)

	maxPost := f.lastPostID
	posts := loadLatestPosts(ctx, f.site)

	for _, p := range posts {
		if p.ID <= f.lastPostID {
			continue
		}
		if p.ID > maxPost {
			maxPost = p.ID
		}

		topicKey := datastore.NewKey(ctx, "topic", "", p.TopicID, nil)

		var t topic
		topicExisted := true
		err := datastore.Get(ctx, topicKey, &t)
		if err != nil {
			// Didn't exist
			topicExisted = false
			t = topic{
				ID:    p.TopicID,
				Title: p.TopicTitle,
			}
		}

		t.Updated = p.Created
		if t.CategoryName == "" {
			cat := f.category(ctx, p.CategoryID)
			t.CategoryName = cat.Name
			t.CategoryColor = cat.Color
		}

		if _, err := datastore.Put(ctx, topicKey, &t); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !topicExisted {
			log.Println("New topic", t)
			// A new topic. We should refresh it.
			ps := loadTopic(ctx, f.site, p.TopicID)
			for _, p := range ps {
				postKey := datastore.NewKey(ctx, "post", "", p.ID, topicKey)
				if _, err := datastore.Put(ctx, postKey, &p); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				log.Println("Saved post in new topic", p.ID)
			}
		} else {
			// Existing topic, just fill in the posts
			postKey := datastore.NewKey(ctx, "post", "", p.ID, topicKey)
			if _, err := datastore.Put(ctx, postKey, &p); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Println("Saved post in old topic", p.ID)
		}
	}

	if maxPost > f.lastPostID {
		log.Println("New max post ID", maxPost)
		f.lastPostID = maxPost
	}
}

func (f *feeder) category(ctx context.Context, id int64) category {
	cat, ok := f.categories[id]
	if ok {
		return cat
	}

	for _, cat := range loadCategories(ctx, f.site) {
		f.categories[cat.ID] = cat
	}
	return f.categories[id]
}

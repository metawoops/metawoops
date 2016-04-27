package metawoops

import (
	"log"
	"net/http"
	"path"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

func init() {
	http.Handle("/delete/", &deletes{
		site: "https://meta.discourse.org",
	})
}

type deletes struct {
	site string
}

func (d *deletes) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	intv, err := time.ParseDuration(path.Base(req.URL.Path))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	earliest := time.Now().Add(-2 * intv)
	latest := time.Now().Add(-intv)
	ctx := appengine.NewContext(req)
	it := datastore.NewQuery("topic").Filter("Updated >", earliest).Filter("Updated <", latest).KeysOnly().Run(ctx)
	var x interface{}
	for key, err := it.Next(&x); err == nil; key, err = it.Next(&x) {
		id := key.IntID()
		_, err := loadTopicStream(ctx, d.site, id)
		if err == topicDeleted {
			log.Println("Topic", id, "has been deleted")

			var t topic
			if err := datastore.Get(ctx, key, &t); err != nil {
				log.Println(err)
				continue
			}

			t.Deleted = time.Now()
			if _, err := datastore.Put(ctx, key, &t); err != nil {
				log.Println(err)
				continue
			}
		} else if err != nil {
			log.Println(err)
			continue
		}
	}
}

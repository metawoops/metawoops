package metawoops

import (
	"bytes"
	"html/template"
	"net/http"
	"path"
	"strconv"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

var hdr = `<html>
<head>
<!-- Latest compiled and minified CSS -->
<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css" integrity="sha384-1q8mTJOASx8j1Au+a5WDVnPi2lkFfwwEAa8hDDdjZlpLegxhjVME1fgjWPGmkzs7" crossorigin="anonymous">
<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap-theme.min.css" integrity="sha384-fLW2N01lMqjakBkx3l/M9EahuwpSfeNvV63J5ezn3uZzapT0u7EYsXMjQV+0En5r" crossorigin="anonymous">
<style type="text/css">
img.emoji {
    width: 1.25em;
    height: 1.25em;
}
blockquote {
	font-size: 16px;
}
html, body, p {
	font-size: 16px;
}
</style>
</head>
<body>
<div class="container">
<div class="row">
<div class="col-md-12">
`

var footer = `</div>
</div>
</div>
</body>
</html>
`

var tpl = template.Must(template.New("index.html").Parse(hdr + `
<h3><a href="/">Topics</a></h3>
<hr/>
<h1>{{.title}}
<small>({{.id}})</small></h1>
<span class="badge" style="background-color: #{{$.color}};">{{$.category}}</span>
{{range $post := .posts}}
<hr/>
<div>
<h3><small>#{{$post.Number}}</small> @{{$post.Username}} <small>{{$post.Name}}</small></h3>
<p>
	<span class="text-muted">{{$post.Created}} ({{$post.ID}})</span>
	{{if not $post.Deleted.IsZero}}<span class="label label-danger">Deleted at {{$post.Deleted}}</span>{{end}}
</p>
{{$post.CookedHTML}}
{{if $post.ActionCode}}
	<p><span class="label label-warning">{{$post.ActionCode}}</span></p>
{{end}}
</div>
{{end}}
<hr/>
{{if not .deleted.IsZero}}
<div class="alert alert-danger">Topic deleted at {{.deleted}}.</div>
{{end}}
` + footer))

var ltpl = template.Must(template.New("index.html").Parse(hdr + `
{{define "topicList"}}
<ul>
	{{range $topic := .}}
	<li><a href="/{{$topic.ID}}">{{$topic.Title}}</a> <span class="badge" style="background-color: #{{$topic.CategoryColor}};">{{$topic.CategoryName}}</span>
	{{if not $topic.Deleted.IsZero}}
		<span class="label label-danger">Deleted at {{$topic.Deleted}}</b>
	{{end}}
	</li>
	{{end}}
</ul>
{{end}}

<h1>Metawoops</h1>
<h2>Latest Deleted Topics</h2>
{{template "topicList" .deletedTopics}}
<h2>Latest Existing Topics</h2>
{{template "topicList" .topics}}
` + footer))

func init() {
	http.Handle("/", new(handler))
}

type handler struct{}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/" {
		h.handleTopicList(w, req)
	} else {
		h.handleTopic(w, req)
	}
}

func (h *handler) handleTopicList(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	var topics []topic
	if _, err := datastore.NewQuery("topic").Order("-Updated").Limit(50).GetAll(ctx, &topics); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var deletedTopics []topic
	if _, err := datastore.NewQuery("topic").Filter("Deleted >", time.Time{}).Order("-Deleted").Limit(50).GetAll(ctx, &deletedTopics); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	err := ltpl.Execute(buf, map[string]interface{}{
		"topics":        topics,
		"deletedTopics": deletedTopics,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf8")
	buf.WriteTo(w)
}

func (h *handler) handleTopic(w http.ResponseWriter, req *http.Request) {
	idStr := path.Base(req.URL.Path)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := appengine.NewContext(req)
	topicKey := datastore.NewKey(ctx, "topic", "", int64(id), nil)
	var t topic
	if err := datastore.Get(ctx, topicKey, &t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var posts []post
	if _, err := datastore.NewQuery("post").Ancestor(topicKey).Order("Number").GetAll(ctx, &posts); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	buf := new(bytes.Buffer)
	err = tpl.Execute(buf, map[string]interface{}{
		"id":       id,
		"title":    t.Title,
		"posts":    posts,
		"category": t.CategoryName,
		"color":    t.CategoryColor,
		"deleted":  t.Deleted,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf8")
	buf.WriteTo(w)
}

package ocn_receiver

import (
	"bytes"
	"encoding/json"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
	"io/ioutil"
	"net/http"
	"time"
)

func init() {
	http.HandleFunc("/notify", handler)
}

type OCNMessage struct {
	Kind           string
	Id             string
	SelfLink       string
	Name           string
	Bucket         string
	Generation     string
	Metageneration string
	ContentType    string
	Updated        time.Time
	StrageClass    string
	Size           string
	Md5Hash        string
	MediaLink      string
	Owner          ACL
	Crc32c         string
	Etag           string
}

type ACL struct {
	Entity   string
	EntityId string
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	for k, v := range r.Header {
		log.Infof(ctx, "%s:%s", k, v)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf(ctx, "ERROR request body read: %s", err)
		log.Errorf(ctx, "ERROR task queue add: %s", err)
		w.WriteHeader(500)
		return
	}
	log.Infof(ctx, string(body))

	if r.Header.Get("X-Goog-Resource-State") == "sync" {
		w.WriteHeader(200)
		return
	}

	var m OCNMessage
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&m)
	if err != nil {
		log.Errorf(ctx, "ERROR json decode: %s", err)
		log.Errorf(ctx, "ERROR task queue add: %s", err)
		w.WriteHeader(500)
		return
	}

	if r.Header.Get("X-Goog-Resource-State") == "exists" {
		t := &taskqueue.Task{
			Payload: body,
			Method:  "PULL",
		}
		_, err = taskqueue.Add(ctx, t, "pull-queue")
		if err != nil {
			log.Errorf(ctx, "ERROR task queue add: %s", err)
			w.WriteHeader(500)
			return
		}
	}

	w.WriteHeader(200)
	w.Write([]byte("done!"))
}

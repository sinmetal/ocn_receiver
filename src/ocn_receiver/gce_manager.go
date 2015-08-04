package ocn_receiver

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"google.golang.org/api/compute/v1"
	"google.golang.org/appengine"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/urlfetch"
)

const INSTANCE_NAME = "conimg"

func init() {
	http.HandleFunc("/api/1/gcemanager", handlerGceManager)
}

func handlerGceManager(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	client := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(ctx, compute.ComputeScope),
			Base:   &urlfetch.Transport{Context: ctx},
		},
	}
	s, err := compute.New(client)
	if err != nil {
		log.Errorf(ctx, "ERROR compute.New: %s", err)
		w.WriteHeader(500)
		return
	}
	is := compute.NewInstancesService(s)
	ilc := is.List("cp300demo1", "us-central1-b")
	il, err := ilc.Do()
	if err != nil {
		log.Errorf(ctx, "ERROR instances list: %s", err)
		w.WriteHeader(500)
		return
	}
	count := 0
	for _, item := range il.Items {
		log.Infof(ctx, "id = %s, name = %s, creationTimestamp = %s", item.Id, item.Name, item.CreationTimestamp)
		if strings.HasPrefix(item.Name, INSTANCE_NAME) {
			count++
		}
	}
	threshold := 50
	if count > threshold {
		log.Infof(ctx, "Create a new instance is canceled.")
		w.WriteHeader(200)
		return
	}

	qs, err := taskqueue.QueueStats(ctx, []string{"pull-queue"})
	if err != nil {
		log.Errorf(ctx, "ERROR get queue stats: %s", err)
		w.WriteHeader(500)
		return
	}
	log.Infof(ctx, "task count = %d", qs[0].Tasks)
	if qs[0].Tasks < 1 {
		log.Infof(ctx, "gce-manager purge.")
		err = taskqueue.Purge(ctx, "gce-manager")
		if err != nil {
			log.Warningf(ctx, "missing gce-manager purge. err = %s", err)
		}
	}
	if count > qs[0].Tasks {
		log.Infof(ctx, "instance count %d > task count %d", count, qs[0].Tasks)
		w.WriteHeader(200)
		return
	}

	threshold = int(math.Min(float64(threshold), float64(qs[0].Tasks)))
	igs := compute.NewInstanceGroupManagersService(s)
	ope, err := igs.Resize("cp300demo1", "us-central1-b", "preemptibility-group", int64(threshold)).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR resize instance group: %s", err)
		w.WriteHeader(500)
		return
	}
	log.Infof(ctx, "resize instance group. ope.name = %s, ope.targetLink = %s, ope.Status = %s, size = %d", ope.Name, ope.TargetLink, ope.Status, threshold)

	w.WriteHeader(200)
	fmt.Fprint(w, threshold)
}

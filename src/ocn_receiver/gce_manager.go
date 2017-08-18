package ocn_receiver

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/appengine"

	"golang.org/x/net/context"
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

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	client := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(ctx, compute.ComputeScope),
			Base:   &urlfetch.Transport{Context: ctxWithTimeout},
		},
	}
	defer cancel()

	s, err := compute.New(client)
	if err != nil {
		log.Errorf(ctx, "ERROR compute.New: %s", err)
		w.WriteHeader(500)
		return
	}
	is := compute.NewInstancesService(s)
	ilc := is.List(appengine.AppID(ctx), "us-central1-b")
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
	threshold := 1000
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
	sizeParam := r.FormValue("instance-group-size")
	if len(sizeParam) > 0 {
		size, err := strconv.Atoi(sizeParam)
		if err != nil {
			log.Warningf(ctx, "invalid instance-group-size. %v", err)
		} else {
			log.Infof(ctx, "set instance-group-size! %s", size)
			threshold = size
		}
	}

	ope, err := resizeInstanceGroup(ctx, s, threshold)
	if err != nil {
		log.Errorf(ctx, "ERROR resize instance group: %s", err)
		w.WriteHeader(500)
		return
	}
	log.Infof(ctx, "resize instance group. ope.name = %s, ope.targetLink = %s, ope.Status = %s, size = %d", ope.Name, ope.TargetLink, ope.Status, threshold)

	w.WriteHeader(200)
	fmt.Fprint(w, threshold)
}

func resizeInstanceGroup(ctx context.Context, service *compute.Service, threshold int) (*compute.Operation, error) {
	var ope *compute.Operation
	var err error
	count := 0
	for {
		igs := compute.NewInstanceGroupManagersService(service)
		ope, err = igs.Resize(appengine.AppID(ctx), "us-central1-b", "preemptibility-group", int64(threshold)).Do()
		if err != nil {
			count++
			log.Infof(ctx, "retry count = %d", count)

			if count > 3 {
				return ope, err
			}

			if uerr, ok := err.(*url.Error); ok {
				log.Warningf(ctx, "err is URL Error %s, Compute Engine Instance Group Resize Error. try count = %d", uerr.Error(), count)
				time.Sleep(time.Duration(rand.Int31n(8000)) * time.Millisecond)
				continue
			}

			if appengine.IsTimeoutError(err) {
				log.Warningf(ctx, "appengine.IsTimeoutError %s, Compute Engine Instance Group Resize Timeout. try count = %s", err.Error(), count)
				time.Sleep(time.Duration(rand.Int31n(8000)) * time.Millisecond)
				continue
			}

			return ope, err
		}

		log.Infof(ctx, "%v", ope)
		return ope, err
	}
	return ope, err
}

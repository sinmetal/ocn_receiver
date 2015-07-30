package ocn_receiver

import (
	"fmt"
	"net/http"
	"strings"
	"time"
    "math"

	"google.golang.org/api/compute/v1"
	"google.golang.org/appengine"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/urlfetch"

	"github.com/pborman/uuid"
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
    threshold := 3
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
	if count > qs[0].Tasks {
		log.Infof(ctx, "instance count %d > task count %d", count, qs[0].Tasks)
		w.WriteHeader(200)
		return
	}

    threshold = math.MinInt32(threshold, qs[0].Tasks)
	names := make([]string, 0)
	for i := count; i < threshold; i++ {
		name, err := createInstance(ctx, is)
		if err != nil {
			time.Sleep(3 * time.Second)
		}
		names = append(names, name)
	}

	w.WriteHeader(200)
	fmt.Fprint(w, names)
}

func createInstance(ctx context.Context, is *compute.InstancesService) (string, error) {
	name := INSTANCE_NAME + "-" + uuid.New()
	log.Infof(ctx, "instance name = %s", name)

	newIns := &compute.Instance{
		Name:        name,
		Zone:        "https://www.googleapis.com/compute/v1/projects/cp300demo1/zones/us-central1-b",
		MachineType: "https://www.googleapis.com/compute/v1/projects/cp300demo1/zones/us-central1-b/machineTypes/n1-standard-1",
		Disks: []*compute.AttachedDisk{
			&compute.AttachedDisk{
				AutoDelete: true,
				Boot:       true,
				DeviceName: name,
				Mode:       "READ_WRITE",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: "https://www.googleapis.com/compute/v1/projects/cp300demo1/global/images/cp300-06-image",
					DiskType:    "https://www.googleapis.com/compute/v1/projects/cp300demo1/zones/us-central1-b/diskTypes/pd-standard",
					DiskSizeGb:  10,
				},
			},
		},
		CanIpForward: false,
		NetworkInterfaces: []*compute.NetworkInterface{
			&compute.NetworkInterface{
				Network: "https://www.googleapis.com/compute/v1/projects/cp300demo1/global/networks/default",
				AccessConfigs: []*compute.AccessConfig{
					&compute.AccessConfig{
						Name: "External NAT",
						Type: "ONE_TO_ONE_NAT",
					},
				},
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			&compute.ServiceAccount{
				Email: "default",
				Scopes: []string{
					compute.ComputeScope,
					compute.DevstorageFullControlScope,
					"https://www.googleapis.com/auth/taskqueue",
					"https://www.googleapis.com/auth/logging.write",
				},
			},
		},
		Scheduling: &compute.Scheduling{
			AutomaticRestart:  false,
			OnHostMaintenance: "TERMINATE",
			Preemptible:       true,
		},
	}
	ope, err := is.Insert("cp300demo1", "us-central1-b", newIns).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR insert instance: %s", err)
		return "", err
	}
	log.Infof(ctx, "create instance ope.name = %s, ope.targetLink = %s, ope.Status = %s", ope.Name, ope.TargetLink, ope.Status)

	return name, nil
}

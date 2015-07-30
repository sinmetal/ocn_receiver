package ocn_receiver

import (
	"net/http"

	"google.golang.org/api/compute/v1"
	"google.golang.org/appengine"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

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
	for _, item := range il.Items {
		log.Infof(ctx, "id = %s, name = %s", item.Id, item.Name)
	}

	ds := compute.NewDisksService(s)
	d := &compute.Disk{
		Name:           "hoge",
		SourceSnapshot: "https://www.googleapis.com/compute/v1/projects/cp300demo1/global/snapshots/fluentd",
		SizeGb:         10,
		Type:           "https://www.googleapis.com/compute/v1/projects/cp300demo1/zones/us-central1-b/diskTypes/pd-standard",
		Zone:           "https://www.googleapis.com/compute/v1/projects/cp300demo1/zones/us-central1-b",
	}
	ope, err := ds.Insert("cp300demo1", "us-central1-b", d).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR insert disk: %s", err)
		w.WriteHeader(500)
		return
	}
	log.Infof(ctx, "create disk. ope.name = %s, ope.selfLink = %s", ope.Name, ope.SelfLink)

	newIns := &compute.Instance{
		Name:        "hoge",
		Zone:        "https://www.googleapis.com/compute/v1/projects/cp300demo1/zones/us-central1-b",
		MachineType: "https://www.googleapis.com/compute/v1/projects/cp300demo1/zones/us-central1-b/machineTypes/n1-standard-1",
		Disks: []*compute.AttachedDisk{
			&compute.AttachedDisk{
				AutoDelete: true,
				Boot:       true,
				DeviceName: "hoge",
				Mode:       "READ_WRITE",
				Source:     "https://www.googleapis.com/compute/v1/projects/cp300demo1/zones/us-central1-b/disks/hoge",
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
	ope, err = is.Insert("cp300demo1", "us-central1-b", newIns).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR insert instance: %s", err)
		w.WriteHeader(500)
		return
	}
	log.Infof(ctx, "create instance ope.name = %s, ope.selfLink = %s", ope.Name, ope.SelfLink)

	w.WriteHeader(200)
	w.Write([]byte("done!"))
}

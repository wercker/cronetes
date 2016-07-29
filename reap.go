//   Copyright 2016 Wercker Holding BV
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package main

import (
	"fmt"
	"time"

	"gopkg.in/urfave/cli.v1"

	log "github.com/Sirupsen/logrus"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/labels"
)

var defaultMaxAge, _ = time.ParseDuration("24h")

var reapCommand = cli.Command{
	Name:   "reap",
	Usage:  "Reap old cron Jobs",
	Action: reapAction,
	Flags: []cli.Flag{
		cli.BoolTFlag{
			Name:  "no-delete-dependents",
			Usage: "Do not delete dependent pods that are associated with jobs",
		},
		cli.DurationFlag{
			Name:  "age",
			Usage: "Age after which the job and pods will be removed",
			Value: defaultMaxAge,
		},
	},
}

type ReapOption struct {
	*GlobalOptions

	DeleteDependents bool
	Age              time.Duration
}

func ParseReapOptions(c *cli.Context) *ReapOption {
	globalOptions := ParseGlobalOptions(c)

	return &ReapOption{
		GlobalOptions:    globalOptions,
		DeleteDependents: c.Bool("no-delete-dependents"),
		Age:              c.Duration("age"),
	}
}

func reapAction(c *cli.Context) error {
	log.Info("Reaping old cron Jobs")

	o := ParseReapOptions(c)

	setLogLevel(o.Debug)

	log.Debug("Creating kube client")

	client, err := createClient(o.GlobalOptions)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to create client: %s", err)
		return cli.NewExitError(errMsg, 2)
	}

	log.Debug("Listing Jobs created with cronetes")

	listOptions := api.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(map[string]string{
			"cronetes": "true",
		})),
	}
	jobs, err := client.Batch().Jobs(o.KubeNamespace).List(listOptions)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to list Jobs: %s", err)
		return cli.NewExitError(errMsg, 1)
	}

	log.WithField("jobs", len(jobs.Items)).
		Info("Fetched Jobs from Kubernetes")

	minAge := time.Now().Add(-o.Age)
	for _, job := range jobs.Items {
		if job.GetCreationTimestamp().Time.After(minAge) {
			log.WithFields(log.Fields{
				"jobName":           job.GetName(),
				"creationTimestamp": job.GetCreationTimestamp(),
			}).Debug("Skipping deleting Job")
			continue
		}

		if o.DeleteDependents {
			selector, err := unversioned.LabelSelectorAsSelector(job.Spec.Selector)
			if err != nil {
				errMsg := fmt.Sprintf("Unable to create LabelSelector from Job selector")
				return cli.NewExitError(errMsg, 3)
			}

			podListOptions := api.ListOptions{
				LabelSelector: selector,
			}
			pods, err := client.Pods(o.KubeNamespace).List(podListOptions)

			for _, pod := range pods.Items {
				log.WithFields(log.Fields{
					"jobName": job.GetName(),
					"podName": pod.GetName(),
				}).Info("Deleting dependent Pod")

				err = client.Pods(o.KubeNamespace).Delete(pod.GetName(), nil)
				if err != nil {
					errMsg := fmt.Sprintf("Unable to delete dependent Pod %s: %s", pod.GetName(), err)
					return cli.NewExitError(errMsg, 3)
				}
			}
		}

		log.WithFields(log.Fields{
			"jobName": job.GetName(),
		}).Info("Deleting Job")

		err = client.Batch().Jobs(o.KubeNamespace).Delete(job.GetName(), nil)
		if err != nil {
			errMsg := fmt.Sprintf("Unable to delete Job %s: %s", job.GetName(), err)
			return cli.NewExitError(errMsg, 3)
		}
	}

	log.Info("Reaping complete")

	return nil
}

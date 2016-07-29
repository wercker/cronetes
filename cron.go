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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/urfave/cli.v1"

	log "github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
	"github.com/robfig/cron"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/conversion"
)

var cronCommand = cli.Command{
	Name:  "cron",
	Usage: "Start a cron server",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "config",
			Usage:  "Path to the configuration file",
			EnvVar: "CRONETES_CONFIG",
		},
		cli.BoolTFlag{
			Name:  "no-random-slug",
			Usage: "Do not add a random string to the Job name",
		},
	},
	Action: cronAction,
}

type CronOptions struct {
	*GlobalOptions
	ConfigPath string
	RandomSlug bool
}

func ParseCronOptions(c *cli.Context) *CronOptions {
	globalOptions := ParseGlobalOptions(c)

	return &CronOptions{
		GlobalOptions: globalOptions,
		ConfigPath:    c.String("config"),
		RandomSlug:    c.Bool("no-random-slug"),
	}
}

func cronAction(c *cli.Context) error {
	log.Info("Starting server")

	o := ParseCronOptions(c)

	setLogLevel(o.Debug)

	log.Debug("Parsing config")

	cronItems, err := getCronItems(o)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to get cron items: %s", err)
		return cli.NewExitError(errMsg, 2)
	}

	log.Debug("Creating kube client")

	client, err := createClient(o.GlobalOptions)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to create client: %s", err)
		return cli.NewExitError(errMsg, 2)
	}

	log.Debug("Creating cron scheduler")

	scheduler := cron.New()

	for _, cronItem := range cronItems {
		log.WithFields(log.Fields{
			"jobName":  cronItem.Job.GetName(),
			"schedule": cronItem.Schedule,
		}).Debug("Creating cron handler")
		scheduler.AddFunc(cronItem.Schedule, createCronFunc(cronItem, client, o))
	}

	log.Debug("Starting cron scheduler")

	scheduler.Start()
	defer scheduler.Stop()

	errc := make(chan error)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	log.Info("Server started")
	return <-errc
}

func createCronFunc(ci *CronItem, client *unversioned.Client, o *CronOptions) func() {
	cloner := conversion.NewCloner()
	return func() {
		logger := log.WithFields(log.Fields{
			"jobName": ci.Job.GetName(),
		})
		logger.Info("Launching Job")

		cc, err := cloner.DeepCopy(ci.Job)
		if err != nil {
			logger.WithError(err).Error("Unable to clone job")
			return
		}

		copy := cc.(*batch.Job)
		if o.RandomSlug {
			copy.SetName(fmt.Sprintf("%s-%s", copy.GetName(), getRandomSlug()))
		}

		labels := copy.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels["cronetes"] = "true"
		copy.SetLabels(labels)

		_, err = client.Batch().Jobs(o.KubeNamespace).Create(copy)
		if err != nil {
			logger.WithError(err).Error("Unable to create job")
			return
		}

		logger.WithField("launchedJobName", copy.GetName()).Info("Launched Job")
	}
}

func getRandomSlug() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func getCronItems(o *CronOptions) ([]*CronItem, error) {
	r, err := getReader(o.ConfigPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var result []*CronItem
	err = yaml.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func getReader(path string) (io.ReadCloser, error) {
	if path != "" {
		return os.Open(path)
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return os.Stdin, nil
	}

	return nil, errors.New("No config supplied, or config piped in")
}

type CronItem struct {
	Schedule string     `json:"schedule"`
	Job      *batch.Job `json:"job"`
}

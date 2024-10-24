/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"sigs.k8s.io/prow/pkg/logrusutil"

	boskosClient "sigs.k8s.io/boskos/client"
	"sigs.k8s.io/boskos/common"
	"sigs.k8s.io/boskos/internal/ibmcloud-janitor/resources"
)

var (
	boskosURL              = flag.String("boskos-url", "", "Boskos URL")
	rTypes                 common.CommaSeparatedStrings
	username               = flag.String("username", "", "Username used to access the Boskos server")
	passwordFile           = flag.String("password-file", "", "The path to password file used to access the Boskos server")
	logLevel               = flag.String("log-level", "info", fmt.Sprintf("Log level is one of %v.", logrus.AllLevels))
	debug                  = flag.Bool("debug", false, "Setting it to true allows logs for PowerVS client")
	ignoreAPIKey           = flag.Bool("ignore-api-key", false, "Setting it to true will skip clean up and rotation of API keys")
	checkPowervsWorkspaces = flag.Bool("check-pvs-workspace-state", false, "Setting it to true will check the PowerVS workspaces for planned maintenace")
	adiditionTime          = flag.Duration("additional-time", 4*time.Hour, "The additional time added to maintenance widnow for handling PowerVS workspaces.")
)

const (
	sleepTime            = time.Minute * 5
	resourceReleaseError = "cannot release resource"
)

func init() {
	flag.Var(&rTypes, "resource-type", "comma-separated list of resources need to be cleaned up")
}

//nolint:nestif
func run(boskos *boskosClient.Client) error {
	for {
		for _, resourceType := range rTypes {
			if res, err := boskos.Acquire(resourceType, common.Dirty, common.Cleaning); errors.Cause(err) == boskosClient.ErrNotFound {
				logrus.WithField("resource type", resourceType).Info("no dirty resource acquired")
				time.Sleep(sleepTime)
				continue
			} else if err != nil {
				return errors.Wrap(err, "Failed to retrieve a dirty resource from Boskos")
			} else {
				options := &resources.CleanupOptions{
					Resource:               res,
					Debug:                  *debug,
					IgnoreAPIKey:           *ignoreAPIKey,
					CheckPowervsWorkspaces: *checkPowervsWorkspaces,
					AdditionalTime:         *adiditionTime,
				}
				if err := resources.CleanAll(options); err != nil {
					fmt.Println(err.Error())
					if strings.Contains(err.Error(), resourceReleaseError) {
						logrus.WithField("name", res.Name).Info("Skip releasing resource as data center is scheduled for planned maintenance, resource will remain dirty")
						continue
					}
					return errors.Wrapf(err, "Failed to clean resource %q", res.Name)
				}
				if err := boskos.UpdateOne(res.Name, common.Cleaning, res.UserData); err != nil {
					return errors.Wrapf(err, "Failed to update resource %q", res.Name)
				}
				if err := boskos.ReleaseOne(res.Name, common.Free); err != nil {
					return errors.Wrapf(err, "Failed to release resoures %q", res.Name)
				}
				logrus.WithField("name", res.Name).Info("Released resource")
			}
		}

	}
}

func main() {
	logrusutil.ComponentInit()
	flag.Parse()

	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logrus.WithError(err).Fatal("invalid log level specified")
	}
	logrus.SetLevel(level)

	if len(rTypes) == 0 {
		logrus.Info("--resource-type is empty! Setting it to the defaults: powervs-service and vpc-service")
		rTypes = []string{"powervs-service", "vpc-service"}
	}

	boskos, err := boskosClient.NewClient("IBMCloudJanitor", *boskosURL, *username, *passwordFile)
	if err != nil {
		logrus.WithError(err).Fatal("unable to create a Boskos client")
	}
	if err := run(boskos); err != nil {
		logrus.WithError(err).Error("Janitor failure")
	}
}

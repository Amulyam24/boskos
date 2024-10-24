/*
Copyright 2024 The Kubernetes Authors.

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

package resources

import (
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"sigs.k8s.io/boskos/common/ibmcloud"
)

type PowerVSWorkspace struct{}

const (
	powerVS string = "PowerVS"
)

func (n PowerVSWorkspace) cleanup(options *CleanupOptions) error {
	powervsData, err := ibmcloud.GetPowerVSResourceData(options.Resource)
	if err != nil {
		return errors.Wrap(err, "failed to get the resource data")
	}

	sess := session.New()
	currentTime := time.Now().UTC()

	objectMask := `mask[endDate,startDate,statusCode[keyName],notificationOccurrenceEventType[keyName],subject]`
	objectFilter := filter.Build(
		filter.Path("notificationOccurrenceEventType.keyName").Eq("PLANNED"),
		filter.Path("statusCode.keyName").Eq("PUBLISHED"),
		filter.Path("subject").Contains(powerVS),
		filter.Path("subject").Contains(powervsData.Zone),
		filter.Path("startDate").DateBetween(currentTime.Add(time.Duration(-7*24)*time.Hour).String(), currentTime.Add(time.Duration(7*24)*time.Hour).String()))
	notificationService := services.GetNotificationOccurrenceEventService(sess)
	events, err := notificationService.Mask(objectMask).Filter(objectFilter).GetAllObjects()
	if err != nil {
		return err
	}

	for _, event := range events {
		if currentTime.After(event.StartDate.Add(-time.Hour*4)) && currentTime.Before(event.StartDate.Add(time.Hour*4)) {
			return errors.New("cannot release resource as data center is scheduled for planned maintenance")
		}
	}

	logrus.WithField("name", options.Resource.Name).Info("PowerVS workspace maintenace check is completed and resource can be released")

	return nil
}

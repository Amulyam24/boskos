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
	"fmt"

	"github.com/IBM/ibm-cos-sdk-go/aws"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials"
	"github.com/IBM/ibm-cos-sdk-go/aws/session"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/boskos/common/ibmcloud"
)

const (
	accesskey  = ""
	secretkey  = ""
	bucketName = ""
	//authEndpoint    = "https://iam.cloud.ibm.com/identity/token"
	//serviceEndpoint = "https://s3-api.us-geo.objectstorage.softlayer.net"
)

type PowerVSWorkspace struct{}

func (n PowerVSWorkspace) cleanup(options *CleanupOptions) error {
	powervsData, err := ibmcloud.GetPowerVSResourceData(options.Resource)
	if err != nil {
		return errors.Wrap(err, "failed to get the resource data")
	}

	endpoint := fmt.Sprintf("https://s3.%s.cloud-object-storage.appdomain.cloud", powervsData.Region)

	conf := aws.NewConfig().
		WithRegion(powervsData.Region).
		WithEndpoint(endpoint).
		WithCredentials(credentials.NewStaticCredentials(accesskey, secretkey, ""))

	sess := session.Must(session.NewSession()) // Creating a new session
	cosClient := s3.New(sess, conf)

	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
		Prefix: aws.String("cos/"),
	}

	objectList, err := cosClient.ListObjects(input)
	if err != nil {

	}

	for _, object := range objectList.Contents {
		input := &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(*object.Key),
		}
		content, err := cosClient.GetObject(input)
		if err != nil {
			return err
		}
		fmt.Print(content)

		//content.ContentEncoding
	}

	// 1. create cos client -- input:
	// 2. List objects in bucket starting with prefix cos/ -- input: bucket name
	// 3. Download the objects one by one
	// 4. Parse over the start and end time and decide

	logrus.WithField("name", options.Resource.Name).Info("PowerVS workspace maintenace check is completed and resource can be released")

	return nil
}

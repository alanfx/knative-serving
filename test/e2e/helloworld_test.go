// +build e2e

/*
Copyright 2018 The Knative Authors
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

package e2e

import (
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

const (
	helloWorldExpectedOutput = "Hello World! How about some tasty noodles?"
)

func isHelloWorldExpectedOutput() func(body string) (bool, error) {
	return func(body string) (bool, error) {
		return strings.TrimRight(body, "\n") == helloWorldExpectedOutput, nil
	}
}

func TestHelloWorld(t *testing.T) {
	clients := Setup(t)

	//add test case specific name to its own logger
	logger := test.Logger.Named("TestHelloWorld")

	var imagePath string
	imagePath = strings.Join([]string{test.Flags.DockerRepo, "helloworld"}, "/")

	logger.Infof("Creating a new Route and Configuration")
	names, err := CreateRouteAndConfig(clients, logger, imagePath)
	if err != nil {
		t.Fatalf("Failed to create Route and Configuration: %v", err)
	}
	test.CleanupOnInterrupt(func() { TearDown(clients, names) }, logger)
	defer TearDown(clients, names)

	logger.Infof("When the Revision can have traffic routed to it, the Route is marked as Ready.")
	err = test.WaitForRouteState(clients.Routes, names.Route, func(r *v1alpha1.Route) (bool, error) {
		if cond := r.Status.GetCondition(v1alpha1.RouteConditionReady); cond == nil {
			return false, nil
		} else {
			return cond.Status == corev1.ConditionTrue, nil
		}
	})
	if err != nil {
		t.Fatalf("The Route %s was not marked as Ready to serve traffic: %v", names.Route, err)
	}

	route, err := clients.Routes.Get(names.Route, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error fetching Route %s: %v", names.Route, err)
	}
	domain := route.Status.Domain
	err = test.WaitForEndpointState(clients.Kube, logger, test.Flags.ResolvableDomain, domain, NamespaceName, names.Route, isHelloWorldExpectedOutput())
	if err != nil {
		t.Fatalf("The endpoint for Route %s at domain %s didn't serve the expected text \"%s\": %v", names.Route, domain, helloWorldExpectedOutput, err)
	}
}
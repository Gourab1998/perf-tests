/*
Copyright 2021 The Kubernetes Authors.

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

package provider

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	prom "k8s.io/perf-tests/clusterloader2/pkg/prometheus/clients"
)

type GardenerProvider struct {
	features       Features
	SeedKubeConfig string
	ShootNamespace string
}

type ShootPrometheusCreds struct {
	URL      string
	UserName string
	Pssword  string
}

func NewGardenerProvider(config map[string]string) Provider {

	supportEnablePrometheusServer := true
	if config[GardenerSeedKubeConfigKey] == "" && config[GardenerShootNamespace] == "" {
		klog.Warningf("no seed kubeconfig path and shoot namespace specified. SupportEnablePrometheusServer will be false.")
		supportEnablePrometheusServer = false
	}
	return &GardenerProvider{
		features: Features{
			SupportProbe:                        true,
			SupportSSHToMaster:                  false,
			SupportImagePreload:                 true,
			SupportEnablePrometheusServer:       supportEnablePrometheusServer,
			SupportGrabMetricsFromKubelets:      true,
			SupportAccessAPIServerPprofEndpoint: false,
			SupportMetricsServerMetrics:         true,
			SupportResourceUsageMetering:        true,
			ShouldScrapeKubeProxy:               true,
		},
		SeedKubeConfig: config[GardenerSeedKubeConfigKey],
		ShootNamespace: config[GardenerShootNamespace],
	}
}

func (p *GardenerProvider) Name() string {
	return GardenerName
}

func (p *GardenerProvider) Features() *Features {
	return &p.features
}

func (p *GardenerProvider) GetComponentProtocolAndPort(componentName string) (string, int, error) {
	return getComponentProtocolAndPort(componentName)
}

func (p *GardenerProvider) GetConfig() Config {
	return Config{}
}

func (p *GardenerProvider) RunSSHCommand(cmd, host string) (string, string, int, error) {
	// TODO(#1693): To maintain compatibility with the use of SSH to scrape measurements,
	// we can SSH to localhost then run `docker exec -t <masterNodeContainerID> <cmd>`.
	return "", "", 0, fmt.Errorf("kind: ssh not yet implemented")
}

func (p *GardenerProvider) Metadata(client clientset.Interface) (map[string]string, error) {
	return nil, nil
}

func (p *GardenerProvider) GetManagedPrometheusClient() (prom.Client, error) {
	if p.SeedKubeConfig == "" && p.ShootNamespace == "" {
		return nil, ErrNoManagedPrometheus
	}
	seedConfig, err := clientcmd.BuildConfigFromFlags("", p.SeedKubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to build config from seed kubeconfig %v", err)
	}
	//ref: https://github.com/gardener/gardener/blob/master/pkg/apis/core/v1beta1/constants/types_constants.go#L69
	selector := "name=observability-ingress-users"
	seedClientset, err := kubernetes.NewForConfig(seedConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to build client from seed kubeconfig %v", err)
	}
	secretClient := seedClientset.CoreV1().Secrets(p.ShootNamespace)
	ingressClient := seedClientset.NetworkingV1().Ingresses(p.ShootNamespace)
	secretList, err := secretClient.List(context.TODO(), v1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list secrets in shoot namespace: %v", err)
	}
	if len(secretList.Items) > 1 || len(secretList.Items) == 0 {
		return nil, fmt.Errorf("found  %d observability-ingress-users secrets", len(secretList.Items))
	}

	userSecret, err := secretClient.Get(context.TODO(), secretList.Items[0].Name, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get secret: %s", secretList.Items[0].Name)
	}
	username := string(userSecret.Data["username"])
	password := string(userSecret.Data["password"])
	prometheusIngress, err := ingressClient.Get(context.TODO(), "prometheus", v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get prometheus ingress: %v", err)
	}
	prometheusHost := prometheusIngress.Spec.Rules[0].Host
	return prom.NewGardenerManagedPrometheusClient(prometheusHost, username, password)
}

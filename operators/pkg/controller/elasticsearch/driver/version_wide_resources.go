// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package driver

import (
	"bytes"
	"encoding/json"

	"github.com/elastic/cloud-on-k8s/operators/pkg/apis/elasticsearch/v1alpha1"
	"github.com/elastic/cloud-on-k8s/operators/pkg/controller/common/annotation"
	"github.com/elastic/cloud-on-k8s/operators/pkg/controller/common/reconciler"
	"github.com/elastic/cloud-on-k8s/operators/pkg/controller/elasticsearch/configmap"
	"github.com/elastic/cloud-on-k8s/operators/pkg/controller/elasticsearch/label"
	"github.com/elastic/cloud-on-k8s/operators/pkg/controller/elasticsearch/name"
	"github.com/elastic/cloud-on-k8s/operators/pkg/controller/elasticsearch/nodecerts"
	"github.com/elastic/cloud-on-k8s/operators/pkg/controller/elasticsearch/settings"
	"github.com/elastic/cloud-on-k8s/operators/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VersionWideResources are resources that are tied to a version, but no specific pod within that version
type VersionWideResources struct {
	// ExtraFilesSecret contains some extra files we want to have access to in the containers, but had nowhere we wanted
	// it to call home, so they ended up here.
	ExtraFilesSecret corev1.Secret
	// GenericUnecryptedConfigurationFiles contains non-secret files Pods with this version should have access to.
	GenericUnecryptedConfigurationFiles corev1.ConfigMap
}

func reconcileVersionWideResources(
	c k8s.Client,
	scheme *runtime.Scheme,
	es v1alpha1.Elasticsearch,
	trustRelationships []v1alpha1.TrustRelationship,
) (*VersionWideResources, error) {
	expectedConfigMap := configmap.NewConfigMapWithData(k8s.ExtractNamespacedName(&es), settings.DefaultConfigMapData)
	err := configmap.ReconcileConfigMap(c, scheme, es, expectedConfigMap)
	if err != nil {
		return nil, err
	}

	trustRootCfg := nodecerts.NewTrustRootConfig(es.Name, es.Namespace)

	// include the trust restrictions from the trust relationships into the trust restrictions config
	for _, trustRelationship := range trustRelationships {
		trustRootCfg.Include(trustRelationship.Spec.TrustRestrictions)
	}

	trustRootCfgData, err := json.Marshal(&trustRootCfg)
	if err != nil {
		return nil, err
	}

	expectedExtraFilesSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: es.Namespace,
			Name:      name.ExtraFilesSecret(es.Name),
		},
		Data: map[string][]byte{
			nodecerts.TrustRestrictionsFilename: trustRootCfgData,
		},
	}

	var reconciledExtraFilesSecret corev1.Secret

	if err := reconciler.ReconcileResource(reconciler.Params{
		Client:     c,
		Scheme:     scheme,
		Owner:      &es,
		Expected:   &expectedExtraFilesSecret,
		Reconciled: &reconciledExtraFilesSecret,
		NeedsUpdate: func() bool {
			// .Data might be nil in the secret, so make sure to initialize it
			if reconciledExtraFilesSecret.Data == nil {
				reconciledExtraFilesSecret.Data = make(map[string][]byte, 1)
			}
			currentTrustConfig, ok := reconciledExtraFilesSecret.Data[nodecerts.TrustRestrictionsFilename]

			return !ok || !bytes.Equal(currentTrustConfig, trustRootCfgData)
		},
		UpdateReconciled: func() {
			reconciledExtraFilesSecret.Data[nodecerts.TrustRestrictionsFilename] = trustRootCfgData
		},
		PostUpdate: func() {
			annotation.MarkPodsAsUpdated(c,
				client.ListOptions{
					Namespace:     es.Namespace,
					LabelSelector: label.NewLabelSelectorForElasticsearch(es),
				})
		},
	}); err != nil {
		return nil, err
	}

	return &VersionWideResources{
		GenericUnecryptedConfigurationFiles: expectedConfigMap,
		ExtraFilesSecret:                    reconciledExtraFilesSecret,
	}, nil
}

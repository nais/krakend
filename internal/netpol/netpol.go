package netpol

import (
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AllowKrakendIngressNetpol(name, namespace string, labelSelector map[string]string) *v1.NetworkPolicy {
	np := &v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{},
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labelSelector,
			},
			PolicyTypes: []v1.PolicyType{
				v1.PolicyTypeIngress,
			},
			Ingress: []v1.NetworkPolicyIngressRule{
				{
					From: []v1.NetworkPolicyPeer{
						{
							PodSelector:       &metav1.LabelSelector{},
							NamespaceSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
		},
	}
	return np
}

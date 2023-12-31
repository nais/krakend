package netpol

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const ManagedByLabel = "krakend-operator"
const KrakendNameLabel = "krakend"
const DefaultCIDR = "0.0.0.0/0"

// TODO: get IP blocks for our clusters, and make it configurable for tenants
func KrakendNetpol(name string, namespace string, labelSelector map[string]string) *v1.NetworkPolicy {
	np := &v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": ManagedByLabel,
			},
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labelSelector,
			},
			PolicyTypes: []v1.PolicyType{
				v1.PolicyTypeIngress,
				v1.PolicyTypeEgress,
			},
			Ingress: []v1.NetworkPolicyIngressRule{
				{
					From: []v1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
			Egress: []v1.NetworkPolicyEgressRule{
				{
					Ports: []v1.NetworkPolicyPort{
						{
							Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
							Port:     &intstr.IntOrString{IntVal: 443},
						},
					},
					To: []v1.NetworkPolicyPeer{
						{
							IPBlock: &v1.IPBlock{
								CIDR: DefaultCIDR,
								Except: []string{
									"10.6.0.0/15",
									"172.16.0.0/12",
									"192.168.0.0/16",
								},
							},
						},
					},
				},
				{
					To: []v1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{},
							PodSelector:       &metav1.LabelSelector{},
						},
					},
				},
			},
		},
	}
	return np
}

func AppAllowKrakendIngressNetpol(name, namespace string, labelSelector map[string]string) *v1.NetworkPolicy {
	np := &v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": ManagedByLabel,
			},
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
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name": KrakendNameLabel,
								},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": namespace,
								},
							},
						},
					},
				},
			},
		},
	}
	return np
}

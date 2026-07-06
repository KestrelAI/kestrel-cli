package cmd

import (
	"strings"
	"testing"

	"cli/pkg/api"
)

func baseOnboardResponse() *api.OnboardClusterResponse {
	return &api.OnboardClusterResponse{
		AccessToken:  "tok-123",
		ClusterID:    "cid-1",
		ClusterName:  "prod",
		ChartVersion: "1.2.3",
	}
}

func TestBuildOperatorValuesYAMLDefaults(t *testing.T) {
	out := buildOperatorValuesYAML(baseOnboardResponse(), clusterOptions{FlowSource: "cilium", MetricsSource: "none"})

	for _, want := range []string{
		`token: "tok-123"`,
		`id: "cid-1"`,
		`name: "prod"`,
		"disableFlows: false",
		"istio:\n    enabled: false",
		"safeApply:\n    enabled: false",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("values file missing %q:\n%s", want, out)
		}
	}
	for _, notWant := range []string{"otel:", "datadog:", "argocd:", "resources:", "certificates:"} {
		if strings.Contains(out, notWant) {
			t.Errorf("values file should not contain %q by default:\n%s", notWant, out)
		}
	}
}

func TestBuildOperatorValuesYAMLIstio(t *testing.T) {
	out := buildOperatorValuesYAML(baseOnboardResponse(), clusterOptions{FlowSource: "istio", MetricsSource: "none"})
	if !strings.Contains(out, "disableFlows: true") || !strings.Contains(out, "istio:\n    enabled: true") {
		t.Errorf("istio flow source not reflected:\n%s", out)
	}
}

func TestBuildOperatorValuesYAMLMetricsAndArgoCD(t *testing.T) {
	out := buildOperatorValuesYAML(baseOnboardResponse(), clusterOptions{
		FlowSource:       "cilium",
		MetricsSource:    "datadog",
		DatadogNamespace: "dd-agents",
		ArgoCD:           true,
		ArgoCDNamespace:  "gitops",
		SafeApply:        true,
	})
	for _, want := range []string{
		"datadog:\n    enabled: true\n    namespace: \"dd-agents\"",
		"argocd:\n    enabled: true\n    namespace: \"gitops\"",
		"safeApply:\n    enabled: true",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("values file missing %q:\n%s", want, out)
		}
	}

	otel := buildOperatorValuesYAML(baseOnboardResponse(), clusterOptions{FlowSource: "cilium", MetricsSource: "opentelemetry"})
	if !strings.Contains(otel, "otel:\n    enabled: true\n    receiverPort: 4317") {
		t.Errorf("otel metrics source not reflected:\n%s", otel)
	}
}

func TestBuildOperatorValuesYAMLResourceTiers(t *testing.T) {
	cases := []struct {
		count    int
		limitCPU string
		limitMem string
		reqCPU   string
		reqMem   string
	}{
		{100, "500m", "2Gi", "250m", "1Gi"},
		{250, "500m", "2Gi", "250m", "1Gi"},
		{1000, "1000m", "4Gi", "500m", "2Gi"},
		{4000, "2000m", "8Gi", "1000m", "4Gi"},
		{9000, "4000m", "16Gi", "2000m", "8Gi"},
	}
	for _, c := range cases {
		out := buildOperatorValuesYAML(baseOnboardResponse(), clusterOptions{
			FlowSource: "cilium", MetricsSource: "none", WorkloadCount: c.count,
		})
		for _, want := range []string{
			"limits:\n    cpu: " + c.limitCPU + "\n    memory: " + c.limitMem,
			"requests:\n    cpu: " + c.reqCPU + "\n    memory: " + c.reqMem,
		} {
			if !strings.Contains(out, want) {
				t.Errorf("workload count %d: values file missing %q:\n%s", c.count, want, out)
			}
		}
	}
}

func TestBuildOperatorValuesYAMLMTLS(t *testing.T) {
	resp := baseOnboardResponse()
	resp.MTLSCertificates = &api.MTLSCertificates{
		ClientCert: "CERT",
		ClientKey:  "KEY",
		CACert:     "CA",
	}
	out := buildOperatorValuesYAML(resp, clusterOptions{FlowSource: "cilium", MetricsSource: "none"})
	for _, want := range []string{"certificates:", "      CERT", "      KEY", "      CA"} {
		if !strings.Contains(out, want) {
			t.Errorf("values file missing %q:\n%s", want, out)
		}
	}
}

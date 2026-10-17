package provider

import (
	"context"
	"testing"
)

func TestUndeployMLModel_WaitsUntilUndeployed(t *testing.T) {
	fake := newFakeMLModelServer(t, []string{"DEPLOYED", "UNDEPLOYED"})

	if err := undeployMLModel(context.Background(), fake.conf(t), "m1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := fake.undeployCalls.Load(), int32(1); got != want {
		t.Errorf("undeploy calls: got %d, want %d", got, want)
	}
	if got, want := fake.getCalls.Load(), int32(2); got != want {
		t.Errorf("get calls: got %d, want %d", got, want)
	}
}

// A model whose deployment failed runs on no node, and an undeploy request leaves
// it in DEPLOY_FAILED. Treating that state as an error made such a model
// impossible to destroy.
func TestUndeployMLModel_TreatsDeployFailedAsUndeployed(t *testing.T) {
	fake := newFakeMLModelServer(t, []string{"DEPLOY_FAILED"})

	if err := undeployMLModel(context.Background(), fake.conf(t), "m1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := fake.undeployCalls.Load(), int32(1); got != want {
		t.Errorf("undeploy calls: got %d, want %d", got, want)
	}
}

func TestDeleteMLModel_RemovesADeployFailedModel(t *testing.T) {
	fake := newFakeMLModelServer(t, []string{"DEPLOY_FAILED"})
	d := resourceOpensearchMLModel().TestResourceData()
	d.SetId("m1")

	if diags := resourceOpensearchMLModelDelete(context.Background(), d, fake.conf(t)); diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}

	if got, want := fake.undeployCalls.Load(), int32(1); got != want {
		t.Errorf("undeploy calls: got %d, want %d", got, want)
	}
	if got, want := fake.deleteCalls.Load(), int32(1); got != want {
		t.Errorf("delete calls: got %d, want %d", got, want)
	}
}

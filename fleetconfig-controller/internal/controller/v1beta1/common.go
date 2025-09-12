package v1beta1

import (
	"context"
	"fmt"
	"regexp"
	"time"

	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorapi "open-cluster-management.io/api/client/operator/clientset/versioned"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	clusteradm               = "clusteradm"
	requeue                  = 30 * time.Second
	amwExistsError           = "you should manually clean them, uninstall kluster will cause those works out of control."
	addonCleanupTimeout      = 1 * time.Minute
	addonCleanupPollInterval = 2 * time.Second
)

var csrSuffixPattern = regexp.MustCompile(`-[a-zA-Z0-9]{5}$`)

func ret(ctx context.Context, res ctrl.Result, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	if err != nil {
		logger.Error(err, "requeueing due to error")
		return reconcile.Result{}, err
	}
	if res.RequeueAfter > 0 {
		logger.Info("requeueing", "after", res.RequeueAfter)
	} else {
		logger.Info("reconciliation complete; no requeue or error")
	}
	return res, nil

}

// getClusterManager retrieves the ClusterManager resource from the Hub cluster
func getClusterManager(ctx context.Context, operatorC *operatorapi.Clientset) (*operatorv1.ClusterManager, error) {
	cm, err := operatorC.OperatorV1().ClusterManagers().Get(ctx, "cluster-manager", metav1.GetOptions{})
	if err != nil {
		if kerrs.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected error getting cluster-manager: %w", err)
	}
	return cm, nil
}

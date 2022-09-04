package k8sutils

import (
	"context"

	"github.com/go-logr/logr"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dashboardLogger(namespace, name string) logr.Logger {
	reqLogger := log.WithValues("Request.Service.Namespace", namespace, "Request.Dashboard.Name", name)
	return reqLogger
}

func createGrafanaDashBoard(namespace, userName, redisName string, isCluster bool) error {
	logger := dashboardLogger(namespace, redisName)

	var dsb grafanav1alpha1.GrafanaDashboard

	if isCluster {
		dsb = generateGrafanaDashboard(namespace, userName, redisName, true)
	} else {
		dsb = generateGrafanaDashboard(namespace, userName, redisName, false)
	}

	if _, err := generateK8sClient().RESTClient().Post().AbsPath("/apis/integreatly.org/v1alpha1/grafanadashboards").Body(&dsb).DoRaw(context.TODO()); err != nil {
		logger.Error(err, "Failed to create GrafanaDashboard")
	}

	logger.Info("Create GrafanaDashboard Success")
	return nil
}

func createServiceMonitor(namespace, userName, redisName string, isCluster bool) error {

	logger := dashboardLogger(namespace, redisName)

	if isCluster {
		sm_leader := generateServiceMontiorObject(namespace, userName, redisName, true, "leader")
		if _, err := generateK8sClient().RESTClient().Post().AbsPath("/apis/monitoring.coreos.com/v1/servicemonitors").Body(&sm_leader).DoRaw(context.TODO()); err != nil {
			logger.Error(err, "Failed to create ServiceMonitor")
			return err
		}
		sm_follower := generateServiceMontiorObject(namespace, userName, redisName, true, "follower")
		if _, err := generateK8sClient().RESTClient().Post().AbsPath("/apis/monitoring.coreos.com/v1/servicemonitors").Body(&sm_follower).DoRaw(context.TODO()); err != nil {
			logger.Error(err, "Failed to create ServiceMonitor")
			return err
		}
	} else {
		sm := generateServiceMontiorObject(namespace, userName, redisName, false, "")
		if _, err := generateK8sClient().RESTClient().Post().AbsPath("/apis/monitoring.coreos.com/v1/servicemonitors").Body(&sm).DoRaw(context.TODO()); err != nil {
			logger.Error(err, "Failed to create ServiceMonitor")
			return err
		}
	}

	logger.Info("Create ServiceMonitor Success")

	return nil
}

func generateGrafanaDashboard(namespace, userName, redisName string, isCluster bool) grafanav1alpha1.GrafanaDashboard {
	name := redisName
	if isCluster {
		name += "-cluster"
	} else {
		name += "-standalone"
	}

	dsb := grafanav1alpha1.GrafanaDashboard{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				"app": name,
			},
			Annotations: map[string]string{
				"creator": userName,
				"owner":   userName,
				"userId":  userName,
			},
		},
		Spec: grafanav1alpha1.GrafanaDashboardSpec{
			GrafanaCom: &grafanav1alpha1.GrafanaDashboardGrafanaComSource{
				Id: 12776, // https://grafana.com/grafana/dashboards/12776-redis/
			},
		},
	}

	return dsb
}

func generateServiceMontiorObject(namespace, userName, redisName string, isCluster bool, role string) prometheusv1.ServiceMonitor {

	var matchlabel, setupType string
	if isCluster { // leader or follower
		setupType = "cluster"
		matchlabel = redisName + "-" + role
		redisName = matchlabel
	} else {
		setupType = "standalone"
		matchlabel = redisName
		redisName = redisName + "-standalone"
	}

	sm := prometheusv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      redisName,
			Labels: map[string]string{
				"app": matchlabel,
			},
			Annotations: map[string]string{
				"creator": userName,
				"owner":   userName,
			},
		},
		Spec: prometheusv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":              matchlabel,
					"redis_setup_type": setupType,
				},
			},
			Endpoints: []prometheusv1.Endpoint{
				{
					Port: "redis-exporter",
					// Interval: "30s",
					// ScrapeTimeout: "10s",
				},
			},
			NamespaceSelector: prometheusv1.NamespaceSelector{
				MatchNames: []string{
					namespace,
				},
			},
		},
	}

	return sm
}

func deleteGrafanaDashBoard(namespace, redisName string, isCluster bool) error {
	logger := dashboardLogger(namespace, redisName)

	if isCluster {
		if _, err := generateK8sClient().RESTClient().Delete().AbsPath("/apis/monitoring.coreos.com/v1/servicemonitors").Namespace(namespace).Name(redisName + "-cluster").DoRaw(context.TODO()); err != nil {
			logger.Error(err, "Failed to delete GrafanaDahsboard")
			return err
		}
	} else {
		if _, err := generateK8sClient().RESTClient().Delete().AbsPath("/apis/monitoring.coreos.com/v1/servicemonitors").Namespace(namespace).Name(redisName + "-standalone").DoRaw(context.TODO()); err != nil {
			logger.Error(err, "Failed to delete GrafanaDahsboard")
			return err
		}
	}

	logger.Info("Delete GrafanaDashboard Success")

	return nil
}

func deleteServiceMonitor(namespace, redisName string, isCluster bool) error {
	logger := dashboardLogger(namespace, redisName)

	if isCluster {
		if _, err := generateK8sClient().RESTClient().Delete().AbsPath("/apis/monitoring.coreos.com/v1/servicemonitors").Namespace(namespace).Name(redisName + "-leader").DoRaw(context.TODO()); err != nil {
			logger.Error(err, "Failed to delete ServiceMonitor")
			return err
		}
		if _, err := generateK8sClient().RESTClient().Delete().AbsPath("/apis/monitoring.coreos.com/v1/servicemonitors").Namespace(namespace).Name(redisName + "-follower").DoRaw(context.TODO()); err != nil {
			logger.Error(err, "Failed to delete ServiceMonitor")
			return err
		}
	} else {
		if _, err := generateK8sClient().RESTClient().Delete().AbsPath("/apis/monitoring.coreos.com/v1/servicemonitors").Namespace(namespace).Name(redisName).DoRaw(context.TODO()); err != nil {
			logger.Error(err, "Failed to delete ServiceMonitor")
			return err
		}
	}

	logger.Info("Delete ServiceMonitor Success")

	return nil
}

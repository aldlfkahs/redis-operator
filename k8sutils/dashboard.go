package k8sutils

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dashboardLogger(namespace, name string) logr.Logger {
	reqLogger := log.WithValues("Request.Service.Namespace", namespace, "Request.Dashboard.Name", name)
	return reqLogger
}

func CreateGrafanaDashBoard(namespace, userName, redisName string, isCluster bool) error {
	logger := dashboardLogger(namespace, redisName)

	var dsb grafanav1alpha1.GrafanaDashboard
	var body []byte
	var err error

	if isCluster {
		dsb = generateGrafanaDashboard(namespace, userName, redisName, true)
	} else {
		dsb = generateGrafanaDashboard(namespace, userName, redisName, false)
	}

	body, err = json.Marshal(dsb)
	if err != nil {
		return err
	}

	if isCluster {
		_, err = getGrafanaDashboard(namespace, redisName, true)
		if err != nil && !errors.IsNotFound(err) {
			return err
		} else if errors.IsNotFound(err) {
			if _, err := generateK8sClient().RESTClient().Post().AbsPath("/apis/integreatly.org/v1alpha1/namespaces/" + namespace + "/grafanadashboards").Body(body).DoRaw(context.TODO()); err != nil {
				logger.Error(err, "Failed to create GrafanaDashboard")
				return err
			}
			logger.Info("Create GrafanaDashboard Success")
		}
	} else {
		_, err = getGrafanaDashboard(namespace, redisName, false)
		if err != nil && !errors.IsNotFound(err) {
			return err
		} else if errors.IsNotFound(err) {
			if _, err := generateK8sClient().RESTClient().Post().AbsPath("/apis/integreatly.org/v1alpha1/namespaces/" + namespace + "/grafanadashboards").Body(body).DoRaw(context.TODO()); err != nil {
				logger.Error(err, "Failed to create GrafanaDashboard")
				return err
			}
			logger.Info("Create GrafanaDashboard Success")

		}
	}
	logger.Info("GrafanaDashboard is in-sync")
	return nil
}

func CreateServiceMonitor(namespace, userName, redisName string, isCluster bool) error {
	logger := dashboardLogger(namespace, redisName)
	var body []byte
	var err error

	if isCluster {
		_, err = getServiceMonitor(namespace, redisName, true, "leader")
		if err != nil && !errors.IsNotFound(err) {
			return err
		} else if errors.IsNotFound(err) {
			sm_leader := generateServiceMontiorObject(namespace, userName, redisName, true, "leader")
			body, _ = json.Marshal(sm_leader)
			if _, err := generateK8sClient().RESTClient().Post().AbsPath("/apis/monitoring.coreos.com/v1/namespaces/" + namespace + "/servicemonitors").Body(body).DoRaw(context.TODO()); err != nil {
				logger.Error(err, "Failed to create ServiceMonitor")
				return err
			}
			logger.Info("Create ServiceMonitor for leader Success")
		}

		_, err = getServiceMonitor(namespace, redisName, true, "follower")
		if err != nil && !errors.IsNotFound(err) {
			return err
		} else if errors.IsNotFound(err) {
			sm_follower := generateServiceMontiorObject(namespace, userName, redisName, true, "follower")
			body, _ = json.Marshal(sm_follower)
			if _, err := generateK8sClient().RESTClient().Post().AbsPath("/apis/monitoring.coreos.com/v1/namespaces/" + namespace + "/servicemonitors").Body(body).DoRaw(context.TODO()); err != nil {
				logger.Error(err, "Failed to create ServiceMonitor")
				return err
			}
			logger.Info("Create ServiceMonitor for follower Success")
		}
	} else {
		_, err = getServiceMonitor(namespace, redisName, false, "")
		if err != nil && !errors.IsNotFound(err) {
			return err
		} else if errors.IsNotFound(err) {
			sm := generateServiceMontiorObject(namespace, userName, redisName, false, "")
			body, _ = json.Marshal(sm)
			if _, err = generateK8sClient().RESTClient().Post().AbsPath("/apis/monitoring.coreos.com/v1/namespaces/" + namespace + "/servicemonitors").Body(body).DoRaw(context.TODO()); err != nil {
				logger.Error(err, "Failed to create ServiceMonitor")
				return err
			}
			logger.Info("Create ServiceMonitor Success")
		}
	}

	logger.Info("ServiceMonitor is in-sync")
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
		TypeMeta: metav1.TypeMeta{
			APIVersion: "integreatly.org/v1alpha1",
			Kind:       "GrafanaDashboard",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				"app": GRAFANA_APP_NAME,
			},
			Annotations: map[string]string{
				"creator": userName,
				"owner":   userName,
				"userId":  userName,
			},
		},
		Spec: grafanav1alpha1.GrafanaDashboardSpec{
			// GrafanaCom: &grafanav1alpha1.GrafanaDashboardGrafanaComSource{
			// 	Id: 12776, // https://grafana.com/grafana/dashboards/12776-redis/
			// },
			Json: generateGrafanaDashboardJson(redisName, isCluster, namespace),
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
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
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

func getGrafanaDashboard(namespace, redisName string, isCluster bool) (grafanav1alpha1.GrafanaDashboard, error) {
	var dsb grafanav1alpha1.GrafanaDashboard
	if isCluster {
		redisName += "-cluster"
	} else {
		redisName += "-standalone"
	}

	data, err := generateK8sClient().RESTClient().Get().AbsPath("/apis/integreatly.org/v1alpha1/namespaces/" + namespace + "/grafanadashboards").Name(redisName).DoRaw(context.TODO())
	if err != nil {
		return dsb, err
	}

	if err = json.Unmarshal(data, &dsb); err != nil {
		return dsb, err
	}

	return dsb, nil
}

func getServiceMonitor(namespace, redisName string, isCluster bool, role string) (prometheusv1.ServiceMonitor, error) {
	var sm prometheusv1.ServiceMonitor
	if isCluster {
		redisName += "-" + role
	} else {
		redisName += "-standalone"
	}

	data, err := generateK8sClient().RESTClient().Get().AbsPath("/apis/monitoring.coreos.com/v1/namespaces/" + namespace + "/servicemonitors").Name(redisName).DoRaw(context.TODO())
	if err != nil {
		return sm, err
	}

	if err = json.Unmarshal(data, &sm); err != nil {
		return sm, err
	}

	return sm, nil
}

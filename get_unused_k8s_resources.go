package main

import (
	"flag"
	"path/filepath"
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/tools/clientcmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	coreV1 "k8s.io/api/core/v1"

)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	secrets, err := clientset.CoreV1().Secrets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	var secrets_map = make(map[string]*coreV1.Secret)
	for _,secret := range secrets.Items {
		secrets_map[secret.Name] = &secret
	}

	configmaps, err := clientset.CoreV1().ConfigMaps("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	var configmaps_map = make(map[string]*coreV1.ConfigMap)
	for _,configmap := range configmaps.Items {
		configmaps_map[configmap.Name] = &configmap
	}


	for _, pod := range pods.Items {
		volumes := pod.Spec.Volumes
		for _, volume := range volumes {
			if secret := volume.Secret; secret != nil {
				if _, ok := secrets_map[secret.SecretName]; ok {
					delete(secrets_map, secret.SecretName)
				}
			}
			if configmap := volume.ConfigMap; configmap != nil {
				if _, ok := configmaps_map[configmap.Name]; ok {
					delete(configmaps_map, configmap.Name)
				}
			}
		}
		containers := pod.Spec.Containers
		for _, container := range containers {
			for _,envfrom := range container.EnvFrom {
				if configMapRef := envfrom.ConfigMapRef; configMapRef != nil {
					if _, ok := configmaps_map[configMapRef.Name]; ok {
						delete(configmaps_map, configMapRef.Name)
					}
				}
				if secretRef := envfrom.SecretRef; secretRef != nil {
					if _, ok := secrets_map[secretRef.Name]; ok {
						delete(secrets_map, secretRef.Name)
					}
				}
			}
			for _,envvar := range container.Env {
				if envvarRef := envvar.ValueFrom; envvarRef != nil {
					if envvarConfigMapRef := envvarRef.ConfigMapKeyRef; envvarConfigMapRef != nil {
						if _, ok := configmaps_map[envvarConfigMapRef.Name]; ok {
							delete(configmaps_map, envvarConfigMapRef.Name)
						}
					}
					if envvarSecretRef := envvarRef.SecretKeyRef; envvarSecretRef != nil {
						if _, ok := secrets_map[envvarSecretRef.Name]; ok {
							delete(secrets_map, envvarSecretRef.Name)
						}
					}
				}
			}
		}
	}

	ingresses, err := clientset.ExtensionsV1beta1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
	for _,ingress := range ingresses.Items {
		for _,tls := range ingress.Spec.TLS {
			if _, ok := secrets_map[tls.SecretName]; ok {
				delete(secrets_map, tls.SecretName)
			}
		}
	}
	fmt.Println("Secrets not used:")
	fmt.Println(len(secrets_map))
	for name,_ := range secrets_map {
		fmt.Printf("%s \n", name)
	}

	fmt.Println("\nConfigmaps not used:\n")
	fmt.Println(len(configmaps_map))
	for name,_ := range configmaps_map {
		fmt.Printf("%s\n", name)
	}
}
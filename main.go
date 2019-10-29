package main

import (
	"flag"
	"fmt"
	"gopkg.in/inf.v0"
	//apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

func main() {
	var kubeconfig *string
	//var namespace *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	//namespace = flag.String("namespace", apiv1.NamespaceDefault, "Specify namespace")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Kube Config: %s\n", *kubeconfig)

	namespaceList, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	top := DataUsage{
		Name:   ".",
		Dec:    &inf.Dec{},
		Memory: &inf.Dec{},
	}

	for _, ns := range namespaceList.Items {

		namespaceItem := DataUsage{
			Name:   ns.Name,
			Dec:    &inf.Dec{},
			Memory: &inf.Dec{},
		}

		podClient := clientset.CoreV1().Pods(ns.Name)
		list, err := podClient.List(metav1.ListOptions{})
		if err != nil {
			panic(err)
		}

		for _, d := range list.Items {
			pod := DataUsage{
				Name:   d.Name,
				Dec:    &inf.Dec{},
				Memory: &inf.Dec{},
			}
			for _, c := range d.Spec.Containers {
				pod.Inner = append(pod.Inner, DataUsage{
					Name:   c.Name,
					Dec:    c.Resources.Requests.Cpu().AsDec(),
					Memory: c.Resources.Requests.Memory().AsDec(),
				})
				c.Resources.Requests.Memory()
				pod.Dec.Add(pod.Dec, c.Resources.Requests.Cpu().AsDec())
				pod.Memory.Add(pod.Memory, c.Resources.Requests.Memory().AsDec())
			}
			namespaceItem.Dec.Add(namespaceItem.Dec, pod.Dec)
			namespaceItem.Memory.Add(namespaceItem.Memory, pod.Memory)
			namespaceItem.Inner = append(namespaceItem.Inner, pod)
		}
		top.Dec.Add(top.Dec, namespaceItem.Dec)
		top.Memory.Add(top.Memory, namespaceItem.Memory)
		top.Inner = append(top.Inner, namespaceItem)

	}

	var dataUsage []DataUsage
	dataUsage = append(dataUsage, top)

	PrintDu(dataUsage, "")

}

func PrintDu(dataUsage []DataUsage, leftpad string) {
	if len(dataUsage) == 0 {
		return
	}
	for i, du := range dataUsage {
		branch := "├── "
		if i == len(dataUsage)-1 {
			branch = "└── "
		} else {

		}

		mem, _ := du.Memory.Unscaled()
		fmt.Printf("%s%s%s CPU %d MB %s\n", leftpad, branch, du.Dec, mem/1024/1024, du.Name)
		newleftpad := leftpad + "│    "
		if i == len(dataUsage)-1 {
			newleftpad = leftpad + "    "
		}
		PrintDu(du.Inner, newleftpad)
	}
}

type DataUsage struct {
	Name   string
	Dec    *inf.Dec
	Memory *inf.Dec
	Inner  []DataUsage
}

package main

import (
	"flag"
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	"gopkg.in/inf.v0"
	"os"
	"strings"

	//apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

type boxL struct {
	views.BoxLayout
}

var mid *views.Text

var ypos int

func (m *boxL) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyEscape {
			os.Exit(0)
			return true
		}
		//mid.SetText("hello")
	}
	return m.BoxLayout.HandleEvent(ev)
}

func display(dataUsage []DataUsage) {
	depth := 2
	content := PrintDu(dataUsage, "", depth)

	lines := strings.SplitAfter(content, "\n")
	ypos = 0

	screen, err := tcell.NewTerminfoScreen()
	if err != nil {
		panic(err)
	}
	err = screen.Init()
	if err != nil {
		panic(err)
	}

	var app = &views.Application{}
	title := views.NewText()

	mid = views.NewText()

	footer := views.NewText()

	title.SetStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).
		Background(tcell.ColorLightBlue))

	footer.SetStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).
		Background(tcell.ColorLightBlue))

	mid.SetStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).
		Background(tcell.ColorLightGray))

	_, height := screen.Size()
	sublist := lines
	if len(lines) > height {
		sublist := lines[ypos : height+ypos-2]
		subcontent := strings.Join(sublist, "")
		mid.SetText(subcontent)
	} else {
		subcontent := strings.Join(lines, "")
		mid.SetText(subcontent)
	}
	title.SetText(fmt.Sprintf("%d / %d | %s", ypos, len(sublist), depthToString(depth)))
	footer.SetText(fmt.Sprintf("%d / %d | %s", ypos, len(sublist), depthToString(depth)))

	var box = &boxL{}
	box.SetOrientation(views.Vertical)
	box.AddWidget(title, 0)
	box.AddWidget(mid, 1)
	box.AddWidget(footer, 0)

	app.SetRootWidget(box)
	go app.Run()

	fmt.Printf("Started \n\r")

	for {
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape {
				os.Exit(0)
			}
			if ev.Key() == tcell.KeyDown {
				if ypos+height-3 < len(lines) {
					ypos++
				}
			}
			if ev.Key() == tcell.KeyPgDn {
				if ypos+height-3 < len(lines)-height {
					ypos = ypos + height
				} else {
					ypos = len(lines) - height + 2
				}
			}
			if ev.Key() == tcell.KeyUp {
				if ypos > 0 {
					ypos--
				}
			}
			if ev.Key() == tcell.KeyPgUp {
				if ypos > height {
					ypos = ypos - height
				} else {
					ypos = 0
				}
			}
			if ev.Key() == tcell.KeyRight {
				if depth < 4 {
					depth++
				}
				content = PrintDu(dataUsage, "", depth)
				lines = strings.SplitAfter(content, "\n")
			}
			if ev.Key() == tcell.KeyLeft {
				if depth > 2 {
					depth--
				}
				content = PrintDu(dataUsage, "", depth)
				lines = strings.SplitAfter(content, "\n")
			}
			_, height := screen.Size()

			title.SetText(fmt.Sprintf("%d / %d | %s %d", ypos, len(lines), depthToString(depth), depth))
			footer.SetText(fmt.Sprintf("%d / %d | %s %d", ypos, len(lines), depthToString(depth), depth))

			if len(lines) > height {
				sublist := lines[ypos : height+ypos-3]
				subcontent := strings.Join(sublist, "")
				mid.SetText(subcontent)
			} else {
				subcontent := strings.Join(lines, "")
				mid.SetText(subcontent)
			}

			app.Refresh()
		}
	}
}

func depthToString(depth int) string {
	content := "cluster"
	if depth >= 2 {
		content = content + " -> namespace"
	}
	if depth >= 3 {
		content = content + " -> pod"
	}
	if depth >= 4 {
		content = content + " -> container"
	}
	return content
}

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

	fmt.Printf("Kube Config: %s\n\r", *kubeconfig)

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

		for _, p := range list.Items {
			pod := DataUsage{
				Name:   p.Name,
				Dec:    &inf.Dec{},
				Memory: &inf.Dec{},
			}
			for _, c := range p.Spec.Containers {
				pod.Inner = append(pod.Inner, DataUsage{
					Name:   c.Name,
					Dec:    c.Resources.Requests.Cpu().AsDec(),
					Memory: c.Resources.Requests.Memory().AsDec(),
				})
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

	display(dataUsage)

}

func PrintDu(dataUsage []DataUsage, leftpad string, depth int) string {
	if len(dataUsage) == 0 {
		return ""
	}
	if depth == 0 {
		return ""
	}
	content := ""
	for i, du := range dataUsage {
		branch := "├── "
		if i == len(dataUsage)-1 {
			branch = "└── "
		} else {

		}

		mem, _ := du.Memory.Unscaled()
		content = content + fmt.Sprintf("%s%s%s CPU %d MB %s (%d)\n", leftpad, branch, du.Dec, mem/1024/1024, du.Name, len(du.Inner))
		newleftpad := leftpad + "│    "
		if i == len(dataUsage)-1 {
			newleftpad = leftpad + "    "
		}
		content = content + PrintDu(du.Inner, newleftpad, depth-1)
	}
	return content
}

type DataUsage struct {
	Name   string
	Dec    *inf.Dec
	Memory *inf.Dec
	Inner  []DataUsage
	Kind   string
}

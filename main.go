package main

import (
	_ "flag"
	"fmt"
	"github.com/google/go-cmp/cmp"
	ocnetworkv1 "github.com/openshift/api/network/v1"
	ocnetv1 "github.com/openshift/client-go/network/clientset/versioned"
	ocnetv1informers "github.com/openshift/client-go/network/informers/externalversions"
	glog "github.com/sirupsen/logrus"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	_ "path/filepath"
	"time"
)

func main() {
	fmt.Println("Watcher started")
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Print("Error encounter")
		fmt.Print(err)
		os.Exit(1)
	}
	k8sclientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	occlientset, err := ocnetv1.NewForConfig(config)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	fmt.Println(k8sclientset)
	fmt.Println(occlientset)

	fmt.Println("OpenShift cluster targeted")
	fmt.Printf("host %s\n", config.Host)
	networkpolfactory := informers.NewSharedInformerFactory(k8sclientset, 10*time.Second)
	egressnetworkpolfactory := ocnetv1informers.NewSharedInformerFactory(occlientset, 10*time.Second)

	netpolinformer := networkpolfactory.Networking().V1().NetworkPolicies().Informer()
	egressnetpolinformer := egressnetworkpolfactory.Network().V1().EgressNetworkPolicies().Informer()

	stopper := make(chan struct{})
	defer close(stopper)
	defer runtime.HandleCrash()
	netpolinformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onAdd,
		UpdateFunc: onUpdate,
		DeleteFunc: onDelete,
	})

	egressnetpolinformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onAddEgress,
		UpdateFunc: onUpdateEgress,
		DeleteFunc: onDeleteEgress,
	})
	fmt.Println("NetworkPolicy Informer Running")
	go netpolinformer.Run(stopper)
	fmt.Println("EgressNetworkPolicy Informer Running")
	go egressnetpolinformer.Run(stopper)
	<-stopper

	fmt.Printf("%+v\n\n", netpolinformer)
	fmt.Printf("%+v", egressnetpolinformer)
}

func onAdd(obj interface{}) {
	netpol, _ := obj.(*networkv1.NetworkPolicy)

	fmt.Printf("onAdd seen NetPol object: %s on %s\n", netpol.GetName(), netpol.GetNamespace())
}

func onUpdate(oldobj, newobj interface{}) {
	n := newobj.(*networkv1.NetworkPolicy)
	o := oldobj.(*networkv1.NetworkPolicy)
	if n.ResourceVersion != o.ResourceVersion {
		glog.Info(fmt.Printf("onUpdate seen new Netpol object: %s on %s with version %s\n", n.GetName(), n.GetNamespace(), n.ResourceVersion))
		if diff := cmp.Diff(o.Spec, n.Spec); diff != "" {
			fmt.Sprintf("Networkpolicy named %s in %s Project has change. The following are the changes:", o.GetName(), o.GetNamespace())
			// Diff between Annotations
			diffAnnotations := cmp.Diff(o.GetAnnotations(), n.GetAnnotations())
			if diffAnnotations != "" {
				oldannotation := o.GetAnnotations()
				newannotation := n.GetAnnotations()
				glog.Info(fmt.Sprintf("ReasonForChange has changed from %s  to %s\n", oldannotation["ReasonForChange"], newannotation["ReasonForChange"]))
			}
			// Diff between PodSelector
			diffPodSelector := cmp.Diff(o.Spec.PodSelector.MatchLabels, n.Spec.PodSelector.MatchLabels)
			if diffPodSelector != "" {
				glog.Info(fmt.Sprintf("PodSelector has changed from  %+v to %+v\n", o.Spec.PodSelector.MatchLabels, n.Spec.PodSelector.MatchLabels))
			}
			// Diff between Ingress
			diffIngress := cmp.Diff(o.Spec.Ingress, n.Spec.Ingress)
			if diffIngress != "" {
				glog.Info(fmt.Sprintf("Ingress Rules has changed from  %+v to %+v\n", o.Spec.Ingress, n.Spec.Ingress))
			}
			// Diff between Egress
			diffEgress := cmp.Diff(o.Spec.Egress, n.Spec.Egress)
			if diffEgress != "" {
				glog.Info(fmt.Sprintf("Egress Rules has changed from  %+v to %+v\n", o.Spec.Egress, n.Spec.Egress))
			}
			// Diff between PolicyTypes
			diffPolicyTypes := cmp.Diff(o.Spec.PolicyTypes, n.Spec.PolicyTypes)
			if diffPolicyTypes != "" {
				glog.Info(fmt.Sprintf("PolicyTypes has changed from  %+v to %+v\n", o.Spec.PolicyTypes, n.Spec.PolicyTypes))
			}
		}
	}
}

func onDelete(obj interface{}) {
	netpol, _ := obj.(*networkv1.NetworkPolicy)
	fmt.Printf("onDelete object Netpol deleted: %s on %s\n", netpol.GetName(), netpol.GetNamespace())
}

// Egress Informer callback functions
func onAddEgress(obj interface{}) {
	netpol, _ := obj.(*ocnetworkv1.EgressNetworkPolicy)

	fmt.Printf("onAdd seen Egress object: %s on %s\n", netpol.GetName(), netpol.GetNamespace())
}

func onUpdateEgress(oldobj, newobj interface{}) {
	n := newobj.(*ocnetworkv1.EgressNetworkPolicy)
	o := oldobj.(*ocnetworkv1.EgressNetworkPolicy)
	if n.ResourceVersion != o.ResourceVersion {
		glog.Info(fmt.Printf("onUpdate seen changes in  EgressNetworkPolicy object: %s on %s with version %s\n", n.GetName(), n.GetNamespace(), n.ResourceVersion))
		if diff := cmp.Diff(o.Spec, n.Spec); diff != "" {
			glog.Info(fmt.Sprintf("EgressNetworkPolicy named %s in %s Project has change. The following are the changes:", o.GetName(), o.GetNamespace()))
			// Diff between Annotations
			diffAnnotations := cmp.Diff(o.GetAnnotations(), n.GetAnnotations())
			if diffAnnotations != "" {
                                oldannotation := o.GetAnnotations()
				newannotation := n.GetAnnotations()
				glog.Info(fmt.Sprintf("ReasonForChange has changed from %s  to %s\n", oldannotation["ReasonForChange"],  newannotation["ReasonForChange"]))
			}
			// Diff between Egress
			diffEgress := cmp.Diff(o.Spec.Egress, n.Spec.Egress)
			if diffEgress != "" {
				glog.Info(fmt.Sprintf("Egress Rules has changed from:\n"))
                                for _, policyrule := range o.Spec.Egress {
                                    glog.Info(fmt.Sprintf("%s: %s", policyrule.To, policyrule.Type))
                                }
				glog.Info(fmt.Sprintf("Egress Rules has changed to:\n"))
                                for _, newpolicyrule := range n.Spec.Egress {
                                    glog.Info(fmt.Sprintf("%s: %s", newpolicyrule.To, newpolicyrule.Type))
                                }

			}
		}
	}
}

func onDeleteEgress(obj interface{}) {
	netpol, _ := obj.(*ocnetworkv1.EgressNetworkPolicy)
	fmt.Printf("onDelete Egress object deleted: %s\n", netpol.GetName())
	fmt.Printf("onUpdate Egress seen on: %s\n", netpol.GetNamespace())
}

package argocd

import (
	"fmt"
	"io/ioutil"
	"os"

	k8err "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	apixinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	apixv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"

	// Import embedded manifests
	_ "github.com/projectsyn/steward/pkg/manifests"
	"github.com/rakyll/statik/fs"
)

func createArgoCRDs(config *rest.Config) error {
	apixClient, err := apixv1beta1client.NewForConfig(config)
	if err != nil {
		return err
	}

	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	apixinstall.Install(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode

	return fs.Walk(statikFS, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		file, err := statikFS.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		obj, _, err := decode(bytes, nil, nil)
		if err != nil {
			return err
		}
		if crd, ok := obj.(*apixv1beta1.CustomResourceDefinition); ok {
			if _, err = apixClient.CustomResourceDefinitions().Create(crd); err != nil {
				if k8err.IsAlreadyExists(err) {
					klog.Infof("%s CRD already exists, skip", crd.Name)
				} else {
					return err
				}
			} else {
				klog.Infof("%s CRD created", crd.Name)
			}
		} else {
			return fmt.Errorf("Provided manifest is not a valid CRD: %s", path)
		}
		return nil
	})
}

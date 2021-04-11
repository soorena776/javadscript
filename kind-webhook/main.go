package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
	"k8s.io/api/admission/v1"
	authorizationapiv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var (
	tlsCertFilename = "./certs/user.cert"
	tlsKeyFilename  = "./certs/user.key"
	addr            = ":8099"
)

type config struct {
	certFile string
	keyFile  string
	addr     string
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func hook(w http.ResponseWriter, r *http.Request) {
	writer := &writer{w}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writer.WriteError(fmt.Errorf("failed to read request body: %v", err))
		return

	}

	ar := v1.AdmissionReview{}
	if err := json.Unmarshal(body, &ar); err != nil {
		writer.WriteError(fmt.Errorf("failed to decode request body: %v", err))
		return
	}

	fmt.Printf("\nRequest: %v", ar.Request.Kind)

	bb, _ := json.Marshal(ar)
	stringAr := string(bb)
	if strings.Contains(stringAr, `"nodes"`) && strings.Contains(stringAr, `"proxy"`){
		_ = stringAr
	}
	
	fmt.Println(stringAr)

	if ar.Request.Object.Object != nil && false{
		b, _ := yaml.Marshal(ar.Request.Object.Object)

		sar := &authorizationapiv1.SubjectAccessReview{}
		if err := json.Unmarshal(b, &ar); err != nil {
			klog.Errorf("failed to decode request body: %v", err)
		}

		saryaml, _ := yaml.Marshal(sar)
		fmt.Println(string(saryaml))
	}

	// fmt.Println("\nRaw: ", string(ar.Request.Object.Raw))

	var patches []patchOperation
	patches = append(patches, patchOperation{
		Op:    "add",
		Path:  "/metadata/labels",
		Value: map[string]string{"webhook-mutated": "blah-blah"},
	})

	resp := v1.AdmissionResponse{
		Allowed: true,
		UID:     ar.Request.UID,
	}

	// patchBytes, err := json.Marshal(patches)
	// pt := v1.PatchTypeJSONPatch
	// if ar.Request.Operation == v1.Create || ar.Request.Operation == v1.Update {
	// 	resp.Patch = patchBytes
	// 	resp.PatchType = &pt
	// }

	// Write the result back
	if err := writer.WriteResponse(&resp); err != nil {
		klog.Errorf("error writing the response: %v", err)
	}
}

func main() {
	cfg := &config{}
	fl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fl.StringVar(&cfg.certFile, "tls-cert-file", tlsCertFilename, "TLS certificate file")
	fl.StringVar(&cfg.keyFile, "tls-key-file", tlsKeyFilename, "TLS key file")
	fl.StringVar(&cfg.addr, "listen-addr", addr, "The address to start the server")
	fl.Parse(os.Args[1:])

	http.HandleFunc("/hook", hook)
	klog.Infof("Listening on %s", cfg.addr)
	if err := http.ListenAndServeTLS(cfg.addr, cfg.certFile, cfg.keyFile, nil); err != nil {
		klog.Fatalf("error serving webhook: %s", err)
	}
}

type writer struct {
	w http.ResponseWriter
}

func (w *writer) WriteError(err error) error {
	klog.Error(err)
	ar := &v1.AdmissionReview{
		Response: &v1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		},
	}
	return w.Write(ar)
}

func (w *writer) WriteResponse(resp *v1.AdmissionResponse) error {
	ar := &v1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: resp,
	}

	return w.Write(ar)
}

func (w *writer) Write(ar *v1.AdmissionReview) error {
	data, err := json.Marshal(ar)
	if err != nil {
		klog.Error(err)
		return err
	}

	if _, err := w.w.Write(data); err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"gopkg.in/yaml.v2"
	"k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var (
	tlsCert = "./certs/user.cert"
	tlsKey  = "./certs/user.key"
	addr    = ":8099"
)

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

	bb, _ := json.Marshal(ar)
	fmt.Println("user", string(bb))

	if ar.Request.Object.Object != nil {
		b, _ := yaml.Marshal(ar.Request.Object.Object)
		fmt.Println(string(b))
	}

	fmt.Println("\nRaw: ", string(ar.Request.Object.Raw))

	resp := v1.AdmissionResponse{Allowed: true, UID: ar.Request.UID}

	// Write the result back
	if err := writer.WriteResponse(&resp); err != nil {
		klog.Errorf("error writing the response: %v", err)
	}
}

func main() {
	http.HandleFunc("/hook", hook)
	klog.Infof("Listening on %s", addr)
	if err := http.ListenAndServeTLS(addr, tlsCert, tlsKey, nil); err != nil {
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

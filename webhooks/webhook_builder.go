package webhooks

import (
	"encoding/json"
	"fmt"
	"gomodules.xyz/jsonpatch/v2"
	"io/ioutil"
	admregv1 "k8s.io/api/admissionregistration/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/go-logr/logr"

	adminv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	mutate_path_prefix   = "/mutate-"
	validate_path_prefix = "/validate-"
	Cert_mount_path      = "/etc/k8s-webhook-certs/"

	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
	pT     = adminv1.PatchTypeJSONPatch //only thing they supported and golang can't reference a const
)

func init() {
	utilruntime.Must(adminv1.AddToScheme(scheme))
	utilruntime.Must(admregv1.AddToScheme(scheme))
}

// admitFunc is the internal type of function we use for all of our validators and mutators
type admitFunc func(adminv1.AdmissionReview) *adminv1.AdmissionResponse

// httpHandler is the type of function http server takes
type httpHandler func(http.ResponseWriter, *http.Request)

// controller-runtime webhook is not flexible enough, we allow our user to use their own string
func RegisterWebhookWithManager(mgr ctrl.Manager, path string, hook http.Handler) error {
	if handlerExist(mgr, path) {
		return fmt.Errorf("webhook handler for %s already exist", path)
	}
	mgr.GetWebhookServer().Register(path, hook)
	return nil
}

// convertToHttpHandler automatically creates httpHandler functions that
//handle the http portion of a request prior to handing to an admitFunction that we implement
func convertToHttpHandler(admit admitFunc, logger logr.Logger) httpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Body != nil {
			if data, err := ioutil.ReadAll(r.Body); err == nil {
				body = data
			}
		}
		// verify the content type is accurate
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			logger.Error(fmt.Errorf("wrong content type"), "contentType=%s, expect application/json", contentType)
			return
		}

		logger.Info("handling request", "body length", r.ContentLength)

		// The AdmissionReview that was sent to the webhook
		requestedAdmissionReview := adminv1.AdmissionReview{}

		// The AdmissionReview that will be returned
		responseAdmissionReview := adminv1.AdmissionReview{
			TypeMeta: metav1.TypeMeta{
				APIVersion: adminv1.SchemeGroupVersion.String(),
				Kind:       reflect.TypeOf(adminv1.AdmissionReview{}).Name(),
			},
		}

		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
			logger.Error(err, "Failed to deserialize the body", "body", body)
			responseAdmissionReview.Response = toErrAdmissionResponse(err, http.StatusBadRequest)
		} else {
			// pass to admitFunc
			responseAdmissionReview.Response = admit(requestedAdmissionReview)
		}

		// Return the same UID
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID

		logger.Info("sending response",
			"allowed", responseAdmissionReview.Response.Allowed,
			"http code", responseAdmissionReview.Response.Result.Code,
			"status", responseAdmissionReview.Response.Result.Status,
			"message", responseAdmissionReview.Response.Result.Message)

		encoder := json.NewEncoder(w)
		if err := encoder.Encode(responseAdmissionReview); err != nil {
			logger.Error(err, "Failed to write the response")
		}
	}
}

// generate a unique path for each gvk
func generatePath(gvk schema.GroupVersionKind) string {
	return strings.Replace(gvk.Group, ".", "-", -1) + "-" +
		gvk.Version + "-" + strings.ToLower(gvk.Kind)
}

// check if the path is used by another handler
func handlerExist(mgr ctrl.Manager, path string) bool {
	if mgr.GetWebhookServer().WebhookMux == nil {
		return false
	}
	h, p := mgr.GetWebhookServer().WebhookMux.Handler(&http.Request{URL: &url.URL{Path: path}})
	if p == path && h != nil {
		return true
	}
	return false
}

func toErrAdmissionResponse(err error, code int32) *adminv1.AdmissionResponse {
	return &adminv1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Code:    code,
			Status:  metav1.StatusFailure,
			Message: err.Error(),
		},
	}
}

func toErrMutateResponse(err error, code int32) *adminv1.AdmissionResponse {
	return &adminv1.AdmissionResponse{
		Result: &metav1.Status{
			Code:    code,
			Status:  metav1.StatusFailure,
			Message: err.Error(),
		},
	}
}

func patchResponseFromRaw(original, current []byte) ([]jsonpatch.Operation, error) {
	return jsonpatch.CreatePatch(original, current)
}

package resources

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PrerequisiteResult holds the result of the prerequisite check.
type PrerequisiteResult struct {
	Ready      bool
	Message    string
	ConfigHash string
}

// CheckPrerequisites checks if the ConfigMap and Secret exist in the MCP.
// Returns whether they are ready, a message describing the state, and
// a hash of the config data (for triggering pod restarts on config changes).
func CheckPrerequisites(ctx context.Context, c client.Client, ns string) (PrerequisiteResult, error) {
	cm := &corev1.ConfigMap{}
	cmKey := client.ObjectKey{Namespace: ns, Name: ConfigMapName}
	cmExists := true
	if err := c.Get(ctx, cmKey, cm); err != nil {
		if apierrors.IsNotFound(err) {
			cmExists = false
		} else {
			return PrerequisiteResult{}, fmt.Errorf("checking ConfigMap %s: %w", ConfigMapName, err)
		}
	}

	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{Namespace: ns, Name: SecretName}
	secretExists := true
	if err := c.Get(ctx, secretKey, secret); err != nil {
		if apierrors.IsNotFound(err) {
			secretExists = false
		} else {
			return PrerequisiteResult{}, fmt.Errorf("checking Secret %s: %w", SecretName, err)
		}
	}

	if !cmExists && !secretExists {
		return PrerequisiteResult{
			Ready:   false,
			Message: fmt.Sprintf("Waiting for ConfigMap %q and Secret %q in namespace %q", ConfigMapName, SecretName, ns),
		}, nil
	}
	if !cmExists {
		return PrerequisiteResult{
			Ready:   false,
			Message: fmt.Sprintf("Waiting for ConfigMap %q in namespace %q", ConfigMapName, ns),
		}, nil
	}
	if !secretExists {
		return PrerequisiteResult{
			Ready:   false,
			Message: fmt.Sprintf("Waiting for Secret %q in namespace %q", SecretName, ns),
		}, nil
	}

	configHash := computeConfigHash(cm, secret)

	return PrerequisiteResult{
		Ready:      true,
		ConfigHash: configHash,
	}, nil
}

// computeConfigHash computes a SHA-256 hash of the ConfigMap and Secret data to detect changes.
// Keys are sorted to ensure deterministic output regardless of Go map iteration order.
func computeConfigHash(cm *corev1.ConfigMap, secret *corev1.Secret) string {
	h := sha256.New()

	cmKeys := make([]string, 0, len(cm.Data))
	for k := range cm.Data {
		cmKeys = append(cmKeys, k)
	}
	sort.Strings(cmKeys)
	for _, k := range cmKeys {
		h.Write([]byte(k))
		h.Write([]byte(cm.Data[k]))
	}

	secretKeys := make([]string, 0, len(secret.Data))
	for k := range secret.Data {
		secretKeys = append(secretKeys, k)
	}
	sort.Strings(secretKeys)
	for _, k := range secretKeys {
		h.Write([]byte(k))
		h.Write(secret.Data[k])
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

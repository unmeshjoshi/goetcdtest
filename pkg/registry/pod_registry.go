package registry

import (
	"context"
	"fmt"
	"sync"

	"etcdtest/pkg/api"
	"etcdtest/pkg/storage"
)

const podPrefix = "/pods/"

type PodRegistry struct {
	storage storage.Storage
	mutex   sync.RWMutex
}

func NewPodRegistry(storage storage.Storage) *PodRegistry {
	return &PodRegistry{
		storage: storage,
	}
}

func (r *PodRegistry) CreatePod(ctx context.Context, pod *api.Pod) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := podPrefix + pod.Name
	existingPod := &api.Pod{}
	err := r.storage.Get(ctx, key, existingPod)
	if err == nil {
		return fmt.Errorf("pod %s already exists", pod.Name)
	}

	if pod.Status == "" {
		pod.Status = api.PodStatusUnassigned
	}

	// Validate Pod spec
	if err := validatePodSpec(pod.Spec); err != nil {
		return fmt.Errorf("invalid pod spec: %w", err)
	}

	return r.storage.Create(ctx, key, pod)
}

func (r *PodRegistry) GetPod(ctx context.Context, name string) (*api.Pod, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	key := podPrefix + name
	pod := &api.Pod{}
	err := r.storage.Get(ctx, key, pod)
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func (r *PodRegistry) UpdatePod(ctx context.Context, pod *api.Pod) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := podPrefix + pod.Name

	// Validate Pod spec
	if err := validatePodSpec(pod.Spec); err != nil {
		return fmt.Errorf("invalid pod spec: %w", err)
	}

	return r.storage.Update(ctx, key, pod)
}

func (r *PodRegistry) DeletePod(ctx context.Context, name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := podPrefix + name
	return r.storage.Delete(ctx, key)
}

func (r *PodRegistry) ListPods(ctx context.Context) ([]*api.Pod, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var pods []*api.Pod
	err := r.storage.List(ctx, podPrefix, &pods)
	if err != nil {
		return nil, err
	}

	return pods, nil
}

func (r *PodRegistry) ListUnassignedPods(ctx context.Context) ([]*api.Pod, error) {
	pods, err := r.ListPods(ctx)
	if err != nil {
		return nil, err
	}

	unassignedPods := make([]*api.Pod, 0)
	for _, pod := range pods {
		if pod.Status == api.PodStatusUnassigned {
			unassignedPods = append(unassignedPods, pod)
		}
	}

	return unassignedPods, nil
}

func validatePodSpec(spec api.PodSpec) error {
	if spec.Replicas < 1 {
		return fmt.Errorf("replicas must be at least 1")
	}
	if len(spec.Containers) == 0 {
		return fmt.Errorf("at least one container must be specified")
	}
	for _, container := range spec.Containers {
		if container.Image == "" {
			return fmt.Errorf("container image must not be empty")
		}
	}
	return nil
}

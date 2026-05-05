package service

import (
	"context"
	"fmt"
	"strings"

	"mrag-platform/backend/go/internal/model"
)

type DatasetScanAdapter interface {
	Mode() string
	ValidatePath(ctx context.Context, path string, server *model.Server) (*model.DatasetPathValidationResult, error)
	Scan(ctx context.Context, path string, server *model.Server, previewLimit int) (*datasetScanSnapshot, error)
}

type DatasetIndexAdapter interface {
	Mode() string
	StartIndex(ctx context.Context, dataset *model.Dataset, task *model.DatasetIndexTask, server *model.Server) (*datasetIndexTaskUpdate, error)
	SyncIndex(ctx context.Context, dataset *model.Dataset, task *model.DatasetIndexTask, server *model.Server) (*datasetIndexTaskUpdate, error)
}

type DatasetScanAdapterResolver interface {
	Resolve(sourceType string) (DatasetScanAdapter, error)
}

type DatasetIndexAdapterResolver interface {
	Resolve(sourceType string) (DatasetIndexAdapter, error)
}

type runtimeBackedDatasetScanAdapter struct {
	runtime datasetRuntime
}

func (a *runtimeBackedDatasetScanAdapter) Mode() string {
	return a.runtime.Mode()
}

func (a *runtimeBackedDatasetScanAdapter) ValidatePath(ctx context.Context, path string, server *model.Server) (*model.DatasetPathValidationResult, error) {
	return a.runtime.ValidatePath(ctx, path, server)
}

func (a *runtimeBackedDatasetScanAdapter) Scan(ctx context.Context, path string, server *model.Server, previewLimit int) (*datasetScanSnapshot, error) {
	return a.runtime.Scan(ctx, path, server, previewLimit)
}

type runtimeBackedDatasetIndexAdapter struct {
	runtime datasetRuntime
}

func (a *runtimeBackedDatasetIndexAdapter) Mode() string {
	return a.runtime.Mode()
}

func (a *runtimeBackedDatasetIndexAdapter) StartIndex(ctx context.Context, dataset *model.Dataset, task *model.DatasetIndexTask, server *model.Server) (*datasetIndexTaskUpdate, error) {
	return a.runtime.StartIndex(ctx, dataset, task, server)
}

func (a *runtimeBackedDatasetIndexAdapter) SyncIndex(ctx context.Context, dataset *model.Dataset, task *model.DatasetIndexTask, server *model.Server) (*datasetIndexTaskUpdate, error) {
	return a.runtime.SyncIndex(ctx, dataset, task, server)
}

type staticDatasetScanAdapterResolver struct {
	local  DatasetScanAdapter
	remote DatasetScanAdapter
}

func NewDatasetScanAdapterResolver(local DatasetScanAdapter, remote DatasetScanAdapter) DatasetScanAdapterResolver {
	return &staticDatasetScanAdapterResolver{local: local, remote: remote}
}

func (r *staticDatasetScanAdapterResolver) Resolve(sourceType string) (DatasetScanAdapter, error) {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "local":
		return r.local, nil
	case "remote":
		return r.remote, nil
	default:
		return nil, fmt.Errorf("unsupported dataset source type: %s", sourceType)
	}
}

type staticDatasetIndexAdapterResolver struct {
	local  DatasetIndexAdapter
	remote DatasetIndexAdapter
}

func NewDatasetIndexAdapterResolver(local DatasetIndexAdapter, remote DatasetIndexAdapter) DatasetIndexAdapterResolver {
	return &staticDatasetIndexAdapterResolver{local: local, remote: remote}
}

func (r *staticDatasetIndexAdapterResolver) Resolve(sourceType string) (DatasetIndexAdapter, error) {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "local":
		return r.local, nil
	case "remote":
		return r.remote, nil
	default:
		return nil, fmt.Errorf("unsupported dataset source type: %s", sourceType)
	}
}

func NewDatasetScanAdapter(runtime datasetRuntime) DatasetScanAdapter {
	return &runtimeBackedDatasetScanAdapter{runtime: runtime}
}

func NewDatasetIndexAdapter(runtime datasetRuntime) DatasetIndexAdapter {
	return &runtimeBackedDatasetIndexAdapter{runtime: runtime}
}

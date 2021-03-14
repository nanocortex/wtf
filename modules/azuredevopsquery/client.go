package azuredevopsquery

import (
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
	azrWorkItemTracking "github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
	"github.com/pkg/errors"
)

func (widget *Widget) getWorkItems() ([]azrWorkItemTracking.WorkItem, error) {
	var workItems []azrWorkItemTracking.WorkItem

	u := uuid.MustParse(widget.settings.queryUuid)

	queryResult, err := widget.api.QueryById(widget.ctx, workitemtracking.QueryByIdArgs{Id: &u})

	workItemsRef := *queryResult.WorkItems

	if err != nil {
		return workItems, errors.Wrap(err, "could not get query")
	}

	var ids []int

	for _, wir := range workItemsRef {
		ids = append(ids, *wir.Id)
	}

	wi, err := widget.api.GetWorkItemsBatch(widget.ctx, workitemtracking.GetWorkItemsBatchArgs{WorkItemGetRequest: &azrWorkItemTracking.WorkItemBatchGetRequest{Ids: &ids}})

	if err != nil {
		return workItems, errors.Wrap(err, "could not get work items")
	}

	return *wi, nil
}

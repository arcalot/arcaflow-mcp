package workflowtools

import (
	"fmt"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
)

func selectWorkflowWithDiscovery(
	details workflow.LoadDetails,
	selector LoadSelectorParams,
) (workflow.Workflow, *DiscoveryResult, error) {
	workflows := details.Index.Workflows
	normalized := normalizeSelector(selector)
	if normalized.ID == "" && normalized.Path == "" {
		if len(workflows) == 0 {
			return workflow.Workflow{}, nil, fmt.Errorf("no workflows found")
		}
		if len(workflows) == 1 {
			return workflows[0], nil, nil
		}
		if selected, ok := autoSelectPrimaryWorkflow(workflows); ok {
			return selected, nil, nil
		}
		result := buildDiscoveryResult(details)
		result.Selection = selectionGuidanceForWorkflows(workflows)
		return workflow.Workflow{}, &result, nil
	}

	selected, err := selectWorkflow(workflows, normalized)
	if err != nil {
		result := buildDiscoveryResult(details)
		result.Selection = selectionGuidanceForWorkflows(workflows)
		return workflow.Workflow{}, &result, err
	}
	return selected, nil, nil
}

func selectionErrorData(
	err error,
	result *DiscoveryResult,
	workflows []workflow.Workflow,
) map[string]interface{} {
	data := map[string]interface{}{
		"error": err.Error(),
	}
	paths := availableWorkflowPathsList(workflows)
	if len(paths) > 0 {
		data["available_paths"] = paths
	}
	ids := availableWorkflowIDsList(workflows)
	if len(ids) > 0 {
		data["available_ids"] = ids
	}
	if result != nil {
		data["discovery"] = result
	}
	return data
}

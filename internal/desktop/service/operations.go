package service

import (
	"fmt"
	"time"

	"fastReadFile/internal/desktop/viewmodel"
	"fastReadFile/internal/ops"
)

func RunVerify(root string) (viewmodel.OperationResult, error) {
	output, err := ops.Verify(root)
	if err != nil {
		return viewmodel.OperationResult{}, wrapServiceError("run_verify", root, err)
	}
	return viewmodel.OperationResult{
		Summary: "Verification completed",
		Changed: false,
		Details: []viewmodel.KeyValue{
			{Key: "status", Value: output},
			{Key: "finishedAt", Value: time.Now().Format(time.RFC3339)},
		},
	}, nil
}

func RunCloseCheck(root string) (viewmodel.OperationResult, error) {
	output, err := ops.CloseCheck(root)
	if err != nil {
		return viewmodel.OperationResult{}, wrapServiceError("run_close_check", root, err)
	}
	return viewmodel.OperationResult{
		Summary: "Close-check completed",
		Changed: false,
		Details: []viewmodel.KeyValue{
			{Key: "lifecycle", Value: output},
		},
	}, nil
}

func RunRepairTail(root string, segmentID uint64) (viewmodel.OperationResult, error) {
	output, err := ops.RepairTail(root, segmentID)
	if err != nil {
		return viewmodel.OperationResult{}, wrapServiceError("run_repair_tail", root, err)
	}
	return viewmodel.OperationResult{
		Summary: fmt.Sprintf("Repair-tail completed for segment %d", segmentID),
		Changed: true,
		Details: []viewmodel.KeyValue{
			{Key: "segmentID", Value: fmt.Sprintf("%d", segmentID)},
			{Key: "status", Value: output},
		},
	}, nil
}

func RunShutdown(root string) (viewmodel.OperationResult, error) {
	return viewmodel.OperationResult{
		Summary: "Shutdown handoff recorded",
		Changed: true,
		Details: []viewmodel.KeyValue{
			{Key: "root", Value: root},
			{Key: "status", Value: "requested"},
		},
	}, nil
}

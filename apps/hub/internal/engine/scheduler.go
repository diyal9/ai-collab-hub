package engine

import (
	"fmt"
	"ai-collab-hub/internal/db"
	"ai-collab-hub/internal/model"
	"ai-collab-hub/internal/ws"
	"time"
)

// StartTask creates a task and dispatches steps
func StartTask(taskID, title, desc, taskType string, steps []string, creatorID uint) error {
	task := model.Task{
		ID: taskID, Title: title, Description: desc, Type: taskType, CreatorID: creatorID, Status: "running",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	db.DB.Create(&task)
	for _, cmd := range steps {
		step := model.TaskStep{TaskID: taskID, Command: cmd, Status: "pending", CreatedAt: time.Now()}
		db.DB.Create(&step)
	}
	Dispatch(taskID, taskType)
	return nil
}

func Dispatch(taskID, taskType string) {
	var steps []model.TaskStep
	db.DB.Where("task_id = ? AND status = ?", taskID, "pending").Order("id asc").Find(&steps)
	
	if len(steps) == 0 {
		db.DB.Model(&model.Task{}).Where("id = ?", taskID).Update("status", "completed")
		return
	}

	if taskType == "serial" {
		step := steps[0]
		step.Status = "running"
		db.DB.Save(&step)
		// Assign to first online agent or broadcast
		msg := map[string]interface{}{"type": "task", "task_id": taskID, "step_id": step.ID, "command": step.Command}
		ws.Hub.Broadcast <- []byte(fmt.Sprintf(`{"type":"broadcast","payload":%v}`, msg))
	} else { // parallel
		for i := range steps {
			step := &steps[i]
			step.Status = "running"
			db.DB.Save(step)
			msg := map[string]interface{}{"type": "task", "task_id": taskID, "step_id": step.ID, "command": step.Command}
			ws.Hub.Broadcast <- []byte(fmt.Sprintf(`{"type":"broadcast","payload":%v}`, msg))
		}
	}
}

func UpdateStep(stepID uint, status, output, inputReq, reply string) {
	db.DB.Model(&model.TaskStep{}).Where("id = ?", stepID).Updates(map[string]interface{}{
		"status": status, "output": output, "input_req": inputReq, "user_reply": reply,
	})
	var step model.TaskStep
	db.DB.First(&step, stepID)
	
	if status == "completed" || status == "success" {
		var pendingCount int64
		db.DB.Model(&model.TaskStep{}).Where("task_id = ? AND status != ?", step.TaskID, "success").Count(&pendingCount)
		if pendingCount == 0 {
			db.DB.Model(&model.Task{}).Where("id = ?", step.TaskID).Update("status", "completed")
			// Trigger next serial steps if any (simplified)
		} else if step.TaskID != "" {
			Dispatch(step.TaskID, "serial") // Continue pipeline
		}
	}
}

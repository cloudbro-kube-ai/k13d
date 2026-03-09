package ui

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/rivo/tview"
)

// triggerCronJob manually triggers a cronjob (k9s t key)
func (a *App) triggerCronJob() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "cronjobs" && resource != "cj" {
		a.flashMsg("Trigger is only available for cronjobs. Navigate to cronjobs view first using :cronjobs", true)
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.getTableCellText(row, 0)
	name := a.getTableCellText(row, 1)

	modal := tview.NewModal().
		SetText(fmt.Sprintf("Trigger CronJob?\n\n%s/%s\n\nThis will create a new job from this cronjob.", ns, name)).
		AddButtons([]string{"Cancel", "Trigger"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.closeModal("trigger-confirm")
			a.SetFocus(a.table)

			if buttonLabel == "Trigger" {
				a.safeGo("triggerCronJob", func() {
					a.flashMsg(fmt.Sprintf("Triggering cronjob %s/%s...", ns, name), false)

					// Use kubectl to create job from cronjob
					jobName := fmt.Sprintf("%s-manual-%d", name, time.Now().Unix())
					resourcePath := fmt.Sprintf("%s/cronjob/%s", ns, name)
					cmd := exec.Command("kubectl", "create", "job", jobName, "--from=cronjob/"+name, "-n", ns)
					output, err := cmd.CombinedOutput()
					if err != nil {
						a.flashMsg(fmt.Sprintf("Trigger failed: %s", string(output)), true)
						a.recordTUIAudit("trigger", resourcePath, fmt.Sprintf("Failed to trigger cronjob %s", name), false, string(output))
						return
					}

					a.flashMsg(fmt.Sprintf("Created job %s from cronjob %s", jobName, name), false)
					a.recordTUIAudit("trigger", resourcePath, fmt.Sprintf("Triggered cronjob %s, created job %s", name, jobName), true, "")
					a.refresh()
				})
			}
		})

	a.showModal("trigger-confirm", modal, true)
}

// scaleResource scales a deployment/statefulset (k9s Shift+S key)
func (a *App) scaleResource() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	// Only scalable resources
	scalable := map[string]bool{
		"deployments": true, "deploy": true,
		"statefulsets": true, "sts": true,
		"replicasets": true, "rs": true,
	}

	if !scalable[resource] {
		a.flashMsg("Scale is only available for deployments, statefulsets, and replicasets. Navigate to one of these resources first.", true)
		return
	}

	// RBAC check
	if !a.checkTUIPermission(resource, "scale") {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.getTableCellText(row, 0)
	name := a.getTableCellText(row, 1)

	// Create scale dialog
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf(" Scale: %s/%s ", ns, name))

	var replicas string
	form.AddInputField("Replicas:", "1", 10, tview.InputFieldInteger, func(text string) {
		replicas = text
	})
	form.AddButton("Scale", func() {
		// Validate replica count
		n, err := fmt.Sscanf(replicas, "%d", new(int))
		if n != 1 || err != nil {
			a.flashMsg("Invalid replica count. Please enter a valid number (0-999).", true)
			return
		}
		var replicaCount int
		_, _ = fmt.Sscanf(replicas, "%d", &replicaCount)
		if replicaCount < 0 || replicaCount > 999 {
			a.flashMsg("Replica count must be between 0 and 999. Please enter a valid number.", true)
			return
		}

		a.closeModal("scale-dialog")
		a.SetFocus(a.table)

		a.safeGo("scaleResource", func() {
			a.flashMsg(fmt.Sprintf("Scaling %s/%s to %s replicas...", ns, name, replicas), false)

			resourceType := resource
			if resourceType == "deploy" {
				resourceType = "deployment"
			} else if resourceType == "sts" {
				resourceType = "statefulset"
			} else if resourceType == "rs" {
				resourceType = "replicaset"
			}

			resourcePath := fmt.Sprintf("%s/%s/%s", ns, resourceType, name)
			cmd := exec.Command("kubectl", "scale", resourceType, name, "-n", ns, "--replicas="+replicas)
			output, err := cmd.CombinedOutput()
			if err != nil {
				a.flashMsg(fmt.Sprintf("Scale failed: %s", string(output)), true)
				a.recordTUIAudit("scale", resourcePath, fmt.Sprintf("Failed to scale to %s replicas", replicas), false, string(output))
				return
			}

			a.flashMsg(fmt.Sprintf("Scaled %s/%s to %s replicas", ns, name, replicas), false)
			a.recordTUIAudit("scale", resourcePath, fmt.Sprintf("Scaled to %s replicas", replicas), true, "")
			a.refresh()
		})
	})
	form.AddButton("Cancel", func() {
		a.closeModal("scale-dialog")
		a.SetFocus(a.table)
	})

	a.showModal("scale-dialog", centered(form, 50, 10), true)
}

// restartResource restarts a deployment/statefulset (k9s Shift+R key)
func (a *App) restartResource() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	restartable := map[string]bool{
		"deployments": true, "deploy": true,
		"statefulsets": true, "sts": true,
		"daemonsets": true, "ds": true,
	}

	if !restartable[resource] {
		a.flashMsg("Restart is only available for deployments, statefulsets, and daemonsets. Navigate to one of these resources first.", true)
		return
	}

	// RBAC check
	if !a.checkTUIPermission(resource, "restart") {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.getTableCellText(row, 0)
	name := a.getTableCellText(row, 1)

	modal := tview.NewModal().
		SetText(fmt.Sprintf("Restart %s?\n\n%s/%s\n\nThis will trigger a rolling restart.", resource, ns, name)).
		AddButtons([]string{"Cancel", "Restart"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.closeModal("restart-confirm")
			a.SetFocus(a.table)

			if buttonLabel == "Restart" {
				a.safeGo("restartResource", func() {
					a.flashMsg(fmt.Sprintf("Restarting %s/%s...", ns, name), false)

					resourceType := resource
					if resourceType == "deploy" {
						resourceType = "deployment"
					} else if resourceType == "sts" {
						resourceType = "statefulset"
					} else if resourceType == "ds" {
						resourceType = "daemonset"
					}

					resourcePath := fmt.Sprintf("%s/%s/%s", ns, resourceType, name)
					cmd := exec.Command("kubectl", "rollout", "restart", resourceType, name, "-n", ns)
					output, err := cmd.CombinedOutput()
					if err != nil {
						a.flashMsg(fmt.Sprintf("Restart failed: %s", string(output)), true)
						a.recordTUIAudit("restart", resourcePath, fmt.Sprintf("Failed to rollout restart %s", name), false, string(output))
						return
					}

					a.flashMsg(fmt.Sprintf("Restarted %s/%s", ns, name), false)
					a.recordTUIAudit("restart", resourcePath, fmt.Sprintf("Rollout restart %s", name), true, "")
					a.refresh()
				})
			}
		})

	a.showModal("restart-confirm", modal, true)
}

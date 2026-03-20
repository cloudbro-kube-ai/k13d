package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// portForwardInfo tracks a running port-forward process
type portForwardInfo struct {
	Cmd        *exec.Cmd
	Namespace  string
	Name       string
	LocalPort  string
	RemotePort string
}

type podContainerEntry struct {
	Name  string
	Image string
	Role  string
	State string
}

func containerStateSummary(status corev1.ContainerStatus) string {
	switch {
	case status.State.Running != nil:
		return "Running"
	case status.State.Waiting != nil && status.State.Waiting.Reason != "":
		return status.State.Waiting.Reason
	case status.State.Waiting != nil:
		return "Waiting"
	case status.State.Terminated != nil && status.State.Terminated.Reason != "":
		return status.State.Terminated.Reason
	case status.State.Terminated != nil:
		return "Terminated"
	case status.Ready:
		return "Ready"
	default:
		return "Unknown"
	}
}

func (a *App) listPodContainers(ctx context.Context, namespace, name string) ([]podContainerEntry, error) {
	pod, err := a.k8s.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	initStates := make(map[string]corev1.ContainerStatus, len(pod.Status.InitContainerStatuses))
	for _, status := range pod.Status.InitContainerStatuses {
		initStates[status.Name] = status
	}
	appStates := make(map[string]corev1.ContainerStatus, len(pod.Status.ContainerStatuses))
	for _, status := range pod.Status.ContainerStatuses {
		appStates[status.Name] = status
	}
	ephemeralStates := make(map[string]corev1.ContainerStatus, len(pod.Status.EphemeralContainerStatuses))
	for _, status := range pod.Status.EphemeralContainerStatuses {
		ephemeralStates[status.Name] = status
	}

	entries := make([]podContainerEntry, 0, len(pod.Spec.InitContainers)+len(pod.Spec.Containers)+len(pod.Spec.EphemeralContainers))
	for _, container := range pod.Spec.InitContainers {
		entries = append(entries, podContainerEntry{
			Name:  container.Name,
			Image: container.Image,
			Role:  "init",
			State: containerStateSummary(initStates[container.Name]),
		})
	}
	for _, container := range pod.Spec.Containers {
		entries = append(entries, podContainerEntry{
			Name:  container.Name,
			Image: container.Image,
			Role:  "container",
			State: containerStateSummary(appStates[container.Name]),
		})
	}
	for _, container := range pod.Spec.EphemeralContainers {
		entries = append(entries, podContainerEntry{
			Name:  container.Name,
			Image: container.Image,
			Role:  "ephemeral",
			State: containerStateSummary(ephemeralStates[container.Name]),
		})
	}

	return entries, nil
}

func (a *App) showPodContainers() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.getTableCellText(row, 0)
	name := a.getTableCellText(row, 1)

	run := func() {
		ctx, cancel := context.WithTimeout(a.getAppContext(), 5*time.Second)
		defer cancel()

		entries, err := a.listPodContainers(ctx, ns, name)
		a.QueueUpdateDraw(func() {
			if err != nil {
				a.flashMsg(fmt.Sprintf("Failed to list containers: %v", err), true)
				return
			}
			if len(entries) == 0 {
				a.flashMsg(fmt.Sprintf("No containers found for %s/%s", ns, name), true)
				return
			}

			list := tview.NewList()
			list.ShowSecondaryText(true)
			list.SetBorder(true).
				SetTitle(fmt.Sprintf(" Containers: %s/%s [gray](l logs, p previous, Esc close)[white] ", ns, name))

			for _, entry := range entries {
				containerName := entry.Name
				list.AddItem(
					fmt.Sprintf("%s  [%s]", containerName, entry.Role),
					fmt.Sprintf("State: %s  Image: %s", entry.State, entry.Image),
					0,
					nil,
				)
			}

			list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Key() {
				case tcell.KeyEsc:
					a.closeModal("pod-containers")
					a.SetFocus(a.table)
					return nil
				}

				switch event.Rune() {
				case 'q':
					a.closeModal("pod-containers")
					a.SetFocus(a.table)
					return nil
				case 'l':
					index := list.GetCurrentItem()
					if index >= 0 && index < len(entries) {
						a.closeModal("pod-containers")
						a.SetFocus(a.table)
						a.showLogsForContainer(ns, name, entries[index].Name, false)
						return nil
					}
				case 'p':
					index := list.GetCurrentItem()
					if index >= 0 && index < len(entries) {
						a.closeModal("pod-containers")
						a.SetFocus(a.table)
						a.showLogsForContainer(ns, name, entries[index].Name, true)
						return nil
					}
				}
				return event
			})

			height := len(entries)*2 + 4
			if height > 20 {
				height = 20
			}
			a.showModal("pod-containers", centered(list, 72, height), true)
			a.SetFocus(list)
		})
	}

	if atomic.LoadInt32(&a.running) == 0 {
		run()
		return
	}

	a.safeGo("showPodContainers", run)
}

// showLogs shows logs for selected pod with Vim-style navigation
func (a *App) showLogs() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.getTableCellText(row, 0)
	name := a.getTableCellText(row, 1)

	a.showLogsForContainer(ns, name, "", false)
}

func (a *App) showLogsForContainer(ns, name, container string, previous bool) {
	title := fmt.Sprintf(" Logs: %s/%s", ns, name)
	if container != "" {
		title = fmt.Sprintf("%s [%s]", title, container)
	}
	if previous {
		title = strings.Replace(title, " Logs: ", " Previous Logs: ", 1)
	}

	// Use VimViewer for Vim-style navigation and search
	logView := NewVimViewer(a, "logs",
		fmt.Sprintf("%s [gray](Esc:close /search s:autoscroll w:wrap m:mark)[white] ", title))
	logView.isLogView = true
	logView.autoScroll = true
	logView.textWrap = true

	logView.SetContent("[yellow]Loading...[white]")
	logView.updateTitle()

	a.showModal("logs", logView, true)
	a.SetFocus(logView)

	fetchName := "showLogs-fetch"
	if previous {
		fetchName = "showPreviousLogs-fetch"
	}
	a.safeGo(fetchName, func() {
		ctx, cancel := context.WithTimeout(a.getAppContext(), 30*time.Second)
		defer cancel()

		var (
			logs string
			err  error
		)
		if previous {
			logs, err = a.k8s.GetPodLogsPrevious(ctx, ns, name, container, 100)
		} else {
			logs, err = a.k8s.GetPodLogs(ctx, ns, name, container, 100)
		}
		a.QueueUpdateDraw(func() {
			if err != nil {
				logView.SetContent(fmt.Sprintf("[red]Error: %v", err))
			} else if logs == "" {
				logView.SetContent("[gray]No logs available")
			} else {
				logView.SetContent(logs)
				logView.ScrollToEnd()
			}
		})
	})
}

// showLogsPrevious shows logs for previous container (k9s p key) with Vim-style navigation
func (a *App) showLogsPrevious() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		a.flashMsg("Log viewing is only available for pods. Navigate to pods view first using :pods", true)
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.getTableCellText(row, 0)
	name := a.getTableCellText(row, 1)
	a.showLogsForContainer(ns, name, "", true)
}

// execShell opens an interactive shell in the selected pod
func (a *App) execShell() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	// RBAC check
	if !a.checkTUIPermission("pods", "exec") {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.getTableCellText(row, 0)
	name := a.getTableCellText(row, 1)

	// Direct shell for pods
	if resource == "pods" || resource == "po" {
		a.runShellForPod(ns, name)
		return
	}

	// For workloads, show pod selector
	workloadResources := map[string]bool{
		"deployments": true, "deploy": true,
		"statefulsets": true, "sts": true,
		"daemonsets": true, "ds": true,
		"replicasets": true, "rs": true,
	}
	if !workloadResources[resource] {
		a.flashMsg("Shell is available for pods and workloads (deploy/sts/ds/rs)", true)
		return
	}

	a.safeGo("selectPodAndShell", func() { a.selectPodAndShell(ns, name, resource) })
}

// runShellForPod suspends the TUI and opens a shell into the given pod
func (a *App) runShellForPod(ns, name string) {
	a.safeSuspend(func() {
		// Try bash first, fall back to sh
		cmd := exec.Command("kubectl", "exec", "-it", "-n", ns, name, "--", "/bin/bash")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			// Try sh if bash fails
			cmd2 := exec.Command("kubectl", "exec", "-it", "-n", ns, name, "--", "/bin/sh")
			cmd2.Stdin = os.Stdin
			cmd2.Stdout = os.Stdout
			cmd2.Stderr = os.Stderr
			if err2 := cmd2.Run(); err2 != nil {
				fmt.Fprintf(os.Stderr, "\nShell failed: %v\nPress Enter to return...\n", err2)
				_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
			}
		}
	})
}

// selectPodAndShell lists pods for a workload and lets the user pick one for shell access
func (a *App) selectPodAndShell(ns, name, resource string) {
	ctx, cancel := context.WithTimeout(a.getAppContext(), 10*time.Second)
	defer cancel()

	// Get the workload's label selector
	var labelSelector string
	switch resource {
	case "deployments", "deploy":
		dep, err := a.k8s.Clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			a.QueueUpdateDraw(func() {
				a.flashMsg(fmt.Sprintf("Failed to get deployment: %v", err), true)
			})
			return
		}
		labelSelector = metav1.FormatLabelSelector(dep.Spec.Selector)
	case "statefulsets", "sts":
		sts, err := a.k8s.Clientset.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			a.QueueUpdateDraw(func() {
				a.flashMsg(fmt.Sprintf("Failed to get statefulset: %v", err), true)
			})
			return
		}
		labelSelector = metav1.FormatLabelSelector(sts.Spec.Selector)
	case "daemonsets", "ds":
		ds, err := a.k8s.Clientset.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			a.QueueUpdateDraw(func() {
				a.flashMsg(fmt.Sprintf("Failed to get daemonset: %v", err), true)
			})
			return
		}
		labelSelector = metav1.FormatLabelSelector(ds.Spec.Selector)
	case "replicasets", "rs":
		rs, err := a.k8s.Clientset.AppsV1().ReplicaSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			a.QueueUpdateDraw(func() {
				a.flashMsg(fmt.Sprintf("Failed to get replicaset: %v", err), true)
			})
			return
		}
		labelSelector = metav1.FormatLabelSelector(rs.Spec.Selector)
	default:
		return
	}

	// List pods matching the selector
	podList, err := a.k8s.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		a.QueueUpdateDraw(func() {
			a.flashMsg(fmt.Sprintf("Failed to list pods: %v", err), true)
		})
		return
	}

	// Filter to running pods
	var runningPods []struct {
		Name   string
		Status string
	}
	for _, pod := range podList.Items {
		status := string(pod.Status.Phase)
		if status == "Running" {
			runningPods = append(runningPods, struct {
				Name   string
				Status string
			}{Name: pod.Name, Status: status})
		}
	}

	if len(runningPods) == 0 {
		a.QueueUpdateDraw(func() {
			a.flashMsg(fmt.Sprintf("No running pods found for %s/%s", resource, name), true)
		})
		return
	}

	// If only one pod, shell directly
	if len(runningPods) == 1 {
		a.QueueUpdateDraw(func() {
			a.runShellForPod(ns, runningPods[0].Name)
		})
		return
	}

	// Show pod selector modal
	a.QueueUpdateDraw(func() {
		list := tview.NewList()
		list.SetBorder(true).SetTitle(fmt.Sprintf(" Select Pod for Shell (%s/%s) ", resource, name))

		for _, pod := range runningPods {
			podName := pod.Name
			list.AddItem(podName, fmt.Sprintf("  Status: %s", pod.Status), 0, nil)
		}

		list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			selectedPod := runningPods[index].Name
			a.closeModal("pod-shell-selector")
			a.SetFocus(a.table)
			a.runShellForPod(ns, selectedPod)
		})

		list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEsc {
				a.closeModal("pod-shell-selector")
				a.SetFocus(a.table)
				return nil
			}
			return event
		})

		height := len(runningPods)*2 + 4
		if height > 20 {
			height = 20
		}
		a.showModal("pod-shell-selector", centered(list, 65, height), true)
		a.SetFocus(list)
	})
}

// portForward shows port forwarding dialog
func (a *App) portForward() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" && resource != "services" && resource != "svc" {
		a.flashMsg("Port forwarding is only available for pods and services. Navigate to one of these resources first using :pods or :services", true)
		return
	}

	// RBAC check
	if !a.checkTUIPermission(resource, "port-forward") {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.getTableCellText(row, 0)
	name := a.getTableCellText(row, 1)

	// Create port forward dialog
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf(" Port Forward: %s/%s ", ns, name))

	var localPort, remotePort string
	form.AddInputField("Local Port:", "8080", 10, nil, func(text string) {
		localPort = text
	})
	form.AddInputField("Remote Port:", "80", 10, nil, func(text string) {
		remotePort = text
	})
	form.AddButton("Forward", func() {
		a.closeModal("port-forward")
		a.SetFocus(a.table)

		if localPort == "" || remotePort == "" {
			a.flashMsg("Both ports are required", true)
			return
		}

		a.safeGo("startPortForward", func() { a.startPortForward(ns, name, resource, localPort, remotePort) })
	})
	form.AddButton("Cancel", func() {
		a.closeModal("port-forward")
		a.SetFocus(a.table)
	})

	a.showModal("port-forward", centered(form, 50, 12), true)
}

// startPortForward starts port forwarding in background
func (a *App) startPortForward(ns, name, resource, localPort, remotePort string) {
	resourceType := "pod"
	if resource == "services" || resource == "svc" {
		resourceType = "svc"
	}

	target := fmt.Sprintf("%s/%s", resourceType, name)
	portMap := fmt.Sprintf("%s:%s", localPort, remotePort)

	a.flashMsg(fmt.Sprintf("Starting port forward %s -> %s:%s", localPort, name, remotePort), false)

	cmd := exec.Command("kubectl", "port-forward", "-n", ns, target, portMap)
	err := cmd.Start()
	if err != nil {
		a.flashMsg(fmt.Sprintf("Port forward failed: %v", err), true)
		return
	}

	// Track the port-forward process
	pf := &portForwardInfo{
		Cmd:        cmd,
		Namespace:  ns,
		Name:       name,
		LocalPort:  localPort,
		RemotePort: remotePort,
	}
	a.pfMx.Lock()
	a.portForwards = append(a.portForwards, pf)
	a.pfMx.Unlock()

	// Wait for process to exit in background and clean up
	a.safeGo("portforward-cleanup", func() {
		_ = cmd.Wait()
		a.pfMx.Lock()
		for i, p := range a.portForwards {
			if p == pf {
				a.portForwards = append(a.portForwards[:i], a.portForwards[i+1:]...)
				break
			}
		}
		a.pfMx.Unlock()
	})

	a.flashMsg(fmt.Sprintf("Port forward active: localhost:%s -> %s:%s (PID: %d)", localPort, name, remotePort, cmd.Process.Pid), false)
}

// showPortForwards displays active port-forward processes
func (a *App) showPortForwards() {
	a.pfMx.Lock()
	forwards := make([]*portForwardInfo, len(a.portForwards))
	copy(forwards, a.portForwards)
	a.pfMx.Unlock()

	if len(forwards) == 0 {
		a.flashMsg("No active port-forwards", false)
		return
	}

	list := tview.NewList()
	list.SetBorder(true).SetTitle(fmt.Sprintf(" Active Port Forwards (%d) - Enter to stop, Esc to close ", len(forwards)))

	for i, pf := range forwards {
		pid := 0
		if pf.Cmd.Process != nil {
			pid = pf.Cmd.Process.Pid
		}
		list.AddItem(
			fmt.Sprintf("localhost:%s -> %s/%s:%s", pf.LocalPort, pf.Namespace, pf.Name, pf.RemotePort),
			fmt.Sprintf("PID: %d", pid),
			rune('1'+i),
			nil,
		)
	}

	list.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		// Re-acquire lock and look up by identity to avoid stale snapshot
		a.pfMx.Lock()
		if idx < len(a.portForwards) {
			pf := a.portForwards[idx]
			if pf.Cmd.Process != nil {
				_ = pf.Cmd.Process.Kill()
				a.flashMsg(fmt.Sprintf("Stopped port-forward localhost:%s", pf.LocalPort), false)
			}
		}
		a.pfMx.Unlock()
		a.closeModal("portforwards")
		a.SetFocus(a.table)
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.closeModal("portforwards")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})

	a.showModal("portforwards", centered(list, 65, 20), true)
	a.SetFocus(list)
}

// cleanupPortForwards kills all active port-forward processes
func (a *App) cleanupPortForwards() {
	a.pfMx.Lock()
	defer a.pfMx.Unlock()
	for _, pf := range a.portForwards {
		if pf.Cmd.Process != nil {
			_ = pf.Cmd.Process.Kill()
		}
	}
	a.portForwards = nil
}

// attachContainer attaches to a container (k9s a key)
func (a *App) attachContainer() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		a.flashMsg("Attach is only available for pods. Navigate to pods view first using :pods", true)
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.getTableCellText(row, 0)
	name := a.getTableCellText(row, 1)

	// Suspend TUI and run kubectl attach
	a.safeSuspend(func() {
		cmd := exec.Command("kubectl", "attach", "-it", "-n", ns, name)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "\nAttach failed: %v\nPress Enter to return...\n", err)
			_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		}
	})
}

// killPod force deletes a pod (k9s k or Ctrl+K key)
func (a *App) killPod() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		a.flashMsg("Kill operation is only available for pods. Navigate to pods view first using :pods", true)
		return
	}

	// RBAC check
	if !a.checkTUIPermission("pods", "delete") {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.getTableCellText(row, 0)
	name := a.getTableCellText(row, 1)

	modal := tview.NewModal().
		SetText(fmt.Sprintf("[red]Kill pod?[white]\n\n%s/%s\n\nThis will force delete the pod.", ns, name)).
		AddButtons([]string{"Cancel", "Kill"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.closeModal("kill-confirm")
			a.SetFocus(a.table)

			if buttonLabel == "Kill" {
				a.safeGo("killPod", func() {
					ctx, cancel := context.WithTimeout(a.getAppContext(), 30*time.Second)
					defer cancel()

					a.flashMsg(fmt.Sprintf("Killing pod %s/%s...", ns, name), false)

					resourcePath := fmt.Sprintf("%s/pod/%s", ns, name)
					err := a.k8s.DeletePodForce(ctx, ns, name)
					if err != nil {
						a.flashMsg(fmt.Sprintf("Kill failed: %v", err), true)
						a.recordTUIAudit("kill", resourcePath, fmt.Sprintf("Failed to force delete pod %s", name), false, err.Error())
						return
					}

					a.flashMsg(fmt.Sprintf("Killed pod %s/%s", ns, name), false)
					a.recordTUIAudit("kill", resourcePath, fmt.Sprintf("Force deleted pod %s", name), true, "")
					a.refresh()
				})
			}
		})

	modal.SetBackgroundColor(tcell.ColorDarkRed)
	a.showModal("kill-confirm", modal, true)
}

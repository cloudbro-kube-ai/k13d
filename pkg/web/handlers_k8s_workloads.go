package web

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cronJobSourceAnnotation = "k13d.io/source-cronjob"
	maxCronJobRecentRuns    = 8
	maxCronJobUpcomingRuns  = 3
)

type cronScheduleDetails struct {
	locationName string
	source       string
	note         string
	estimated    bool
	nextRun      *time.Time
	upcomingRuns []time.Time
	parseError   string
}

func buildJobItems(jobs []batchv1.Job) []map[string]interface{} {
	items := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		items[i] = buildJobItem(job)
	}
	return items
}

func buildJobItem(job batchv1.Job) map[string]interface{} {
	status := jobStatus(&job)
	ownerKind, ownerName := jobOwner(job)
	sourceCronJob := sourceCronJobName(job)
	sourceLabel := "Standalone"
	switch {
	case sourceCronJob != "":
		sourceLabel = fmt.Sprintf("CronJob/%s", sourceCronJob)
	case isManualJob(job):
		sourceLabel = "Manual"
	case ownerName != "":
		sourceLabel = fmt.Sprintf("%s/%s", ownerKind, ownerName)
	}

	return map[string]interface{}{
		"name":              job.Name,
		"namespace":         job.Namespace,
		"status":            status,
		"statusReason":      strings.Join(summarizeJobConditions(job.Status.Conditions), ", "),
		"completions":       getJobCompletions(&job),
		"duration":          getJobDuration(&job),
		"age":               formatAge(job.CreationTimestamp.Time),
		"createdAt":         timestampString(job.CreationTimestamp.Time),
		"startTime":         timestampStringPtr(job.Status.StartTime),
		"completionTime":    timestampStringPtr(job.Status.CompletionTime),
		"succeeded":         job.Status.Succeeded,
		"failed":            job.Status.Failed,
		"active":            job.Status.Active,
		"parallelism":       derefInt32(job.Spec.Parallelism),
		"completionsTarget": derefInt32(job.Spec.Completions),
		"backoffLimit":      derefInt32(job.Spec.BackoffLimit),
		"ownerKind":         ownerKind,
		"ownerName":         ownerName,
		"sourceCronJob":     sourceCronJob,
		"sourceLabel":       sourceLabel,
		"manualTrigger":     isManualJob(job),
		"image":             firstContainerImage(job.Spec.Template.Spec.Containers),
		"conditions":        summarizeJobConditions(job.Status.Conditions),
		"security":          buildPodSecurityDetails(job.Spec.Template.Spec),
	}
}

func buildCronJobItems(cjs []batchv1.CronJob, jobs []batchv1.Job) []map[string]interface{} {
	items := make([]map[string]interface{}, len(cjs))
	for i, cj := range cjs {
		items[i] = buildCronJobItem(cj, jobs)
	}
	return items
}

func buildCronJobItem(cj batchv1.CronJob, jobs []batchv1.Job) map[string]interface{} {
	suspended := cj.Spec.Suspend != nil && *cj.Spec.Suspend
	lastSchedule := "<never>"
	if cj.Status.LastScheduleTime != nil {
		lastSchedule = formatAge(cj.Status.LastScheduleTime.Time) + " ago"
	}

	relatedJobs := relatedJobsForCronJob(cj, jobs)
	recentRuns := make([]map[string]interface{}, 0, minInt(len(relatedJobs), maxCronJobRecentRuns))
	var lastSuccessfulRun *time.Time
	var lastFailedRun *time.Time
	for _, job := range relatedJobs {
		recentRuns = append(recentRuns, buildCronJobRun(job))
		status := jobStatus(&job)
		if lastSuccessfulRun == nil && status == "Complete" {
			t := jobReferenceTime(job)
			lastSuccessfulRun = &t
		}
		if lastFailedRun == nil && status == "Failed" {
			t := jobReferenceTime(job)
			lastFailedRun = &t
		}
		if len(recentRuns) >= maxCronJobRecentRuns {
			break
		}
	}

	schedule := evaluateCronSchedule(cj, time.Now())
	nextRunDisplay := "Not scheduled"
	if suspended {
		nextRunDisplay = "Paused"
	} else if schedule.parseError != "" {
		nextRunDisplay = "Invalid schedule"
	} else if schedule.nextRun != nil {
		nextRunDisplay = "Scheduled"
	}

	return map[string]interface{}{
		"name":                     cj.Name,
		"namespace":                cj.Namespace,
		"schedule":                 cj.Spec.Schedule,
		"status":                   cronJobStatusLabel(suspended),
		"suspend":                  suspended,
		"active":                   len(cj.Status.Active),
		"lastSchedule":             lastSchedule,
		"lastScheduleTime":         timestampStringPtr(cj.Status.LastScheduleTime),
		"age":                      formatAge(cj.CreationTimestamp.Time),
		"createdAt":                timestampString(cj.CreationTimestamp.Time),
		"timeZone":                 schedule.locationName,
		"timeZoneSource":           schedule.source,
		"scheduleNote":             schedule.note,
		"scheduleError":            schedule.parseError,
		"nextRun":                  nextRunDisplay,
		"nextRunTime":              timestampStringTimePtr(schedule.nextRun),
		"upcomingRuns":             timestampStringSlice(schedule.upcomingRuns),
		"nextRunEstimated":         schedule.estimated,
		"recentRuns":               recentRuns,
		"concurrencyPolicy":        string(cj.Spec.ConcurrencyPolicy),
		"startingDeadlineSeconds":  derefInt64(cj.Spec.StartingDeadlineSeconds),
		"successfulJobsHistory":    derefInt32(cj.Spec.SuccessfulJobsHistoryLimit),
		"failedJobsHistory":        derefInt32(cj.Spec.FailedJobsHistoryLimit),
		"lastSuccessfulRunTime":    timestampStringTimePtr(lastSuccessfulRun),
		"lastFailedRunTime":        timestampStringTimePtr(lastFailedRun),
		"image":                    firstContainerImage(cj.Spec.JobTemplate.Spec.Template.Spec.Containers),
		"jobTemplateParallelism":   derefInt32(cj.Spec.JobTemplate.Spec.Parallelism),
		"jobTemplateCompletions":   derefInt32(cj.Spec.JobTemplate.Spec.Completions),
		"jobTemplateBackoffLimit":  derefInt32(cj.Spec.JobTemplate.Spec.BackoffLimit),
		"historyRunsObservedCount": len(relatedJobs),
		"security":                 buildPodSecurityDetails(cj.Spec.JobTemplate.Spec.Template.Spec),
	}
}

func buildCronJobRun(job batchv1.Job) map[string]interface{} {
	return map[string]interface{}{
		"name":           job.Name,
		"status":         jobStatus(&job),
		"startTime":      timestampStringPtr(job.Status.StartTime),
		"completionTime": timestampStringPtr(job.Status.CompletionTime),
		"duration":       getJobDuration(&job),
		"age":            formatAge(job.CreationTimestamp.Time),
		"manualTrigger":  isManualJob(job),
		"sourceCronJob":  sourceCronJobName(job),
		"succeeded":      job.Status.Succeeded,
		"failed":         job.Status.Failed,
		"active":         job.Status.Active,
	}
}

func relatedJobsForCronJob(cj batchv1.CronJob, jobs []batchv1.Job) []batchv1.Job {
	filtered := make([]batchv1.Job, 0)
	for _, job := range jobs {
		if job.Namespace != cj.Namespace {
			continue
		}
		if !jobBelongsToCronJob(job, cj.Name) {
			continue
		}
		filtered = append(filtered, job)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return jobReferenceTime(filtered[i]).After(jobReferenceTime(filtered[j]))
	})
	return filtered
}

func jobBelongsToCronJob(job batchv1.Job, cronJobName string) bool {
	for _, owner := range job.OwnerReferences {
		if owner.Kind == "CronJob" && owner.Name == cronJobName {
			return true
		}
	}

	if source := strings.TrimSpace(job.Annotations[cronJobSourceAnnotation]); source == cronJobName {
		return true
	}

	// Best-effort fallback for manually-triggered jobs created from a CronJob.
	if isManualJob(job) && strings.HasPrefix(job.Name, cronJobName+"-") {
		return true
	}

	return false
}

func jobOwner(job batchv1.Job) (string, string) {
	for _, owner := range job.OwnerReferences {
		if owner.Controller != nil && *owner.Controller {
			return owner.Kind, owner.Name
		}
	}
	for _, owner := range job.OwnerReferences {
		return owner.Kind, owner.Name
	}
	return "", ""
}

func sourceCronJobName(job batchv1.Job) string {
	for _, owner := range job.OwnerReferences {
		if owner.Kind == "CronJob" {
			return owner.Name
		}
	}
	return strings.TrimSpace(job.Annotations[cronJobSourceAnnotation])
}

func isManualJob(job batchv1.Job) bool {
	return job.Annotations["cronjob.kubernetes.io/instantiate"] == "manual"
}

func jobStatus(job *batchv1.Job) string {
	for _, cond := range job.Status.Conditions {
		if cond.Status != "True" {
			continue
		}
		switch cond.Type {
		case batchv1.JobComplete:
			return "Complete"
		case batchv1.JobFailed:
			return "Failed"
		}
	}

	switch {
	case job.Status.Active > 0:
		return "Running"
	case job.Status.Succeeded > 0:
		return "Complete"
	case job.Status.Failed > 0:
		return "Failed"
	case job.Status.StartTime != nil:
		return "Starting"
	default:
		return "Pending"
	}
}

func jobReferenceTime(job batchv1.Job) time.Time {
	if job.Status.StartTime != nil {
		return job.Status.StartTime.Time
	}
	if job.Status.CompletionTime != nil {
		return job.Status.CompletionTime.Time
	}
	return job.CreationTimestamp.Time
}

func summarizeJobConditions(conditions []batchv1.JobCondition) []string {
	summaries := make([]string, 0, len(conditions))
	for _, cond := range conditions {
		if cond.Status != corev1.ConditionTrue {
			continue
		}
		summary := string(cond.Type)
		if cond.Reason != "" {
			summary += ": " + cond.Reason
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func cronJobStatusLabel(suspended bool) string {
	if suspended {
		return "Suspended"
	}
	return "Active"
}

func evaluateCronSchedule(cj batchv1.CronJob, now time.Time) cronScheduleDetails {
	details := cronScheduleDetails{}
	location, locationName, source, estimated, note := cronJobLocation(cj)
	details.locationName = locationName
	details.source = source
	details.estimated = estimated
	details.note = note

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(cj.Spec.Schedule)
	if err != nil {
		details.parseError = err.Error()
		return details
	}

	base := now.In(location)
	if cj.Status.LastScheduleTime != nil && cj.Status.LastScheduleTime.Time.After(base) {
		base = cj.Status.LastScheduleTime.Time.In(location)
	}

	nextRun := schedule.Next(base)
	details.nextRun = &nextRun
	details.upcomingRuns = make([]time.Time, 0, maxCronJobUpcomingRuns)
	current := nextRun
	for i := 0; i < maxCronJobUpcomingRuns; i++ {
		details.upcomingRuns = append(details.upcomingRuns, current)
		current = schedule.Next(current)
	}
	return details
}

func cronJobLocation(cj batchv1.CronJob) (*time.Location, string, string, bool, string) {
	if cj.Spec.TimeZone != nil {
		timeZone := strings.TrimSpace(*cj.Spec.TimeZone)
		if timeZone != "" {
			location, err := time.LoadLocation(timeZone)
			if err == nil {
				return location, timeZone, "spec.timeZone", false, "Next runs are calculated from the CronJob time zone and shown in your local time below."
			}
			return time.Local, fmt.Sprintf("%s (invalid)", timeZone), "invalid spec.timeZone", true,
				fmt.Sprintf("The CronJob timeZone %q could not be loaded. Next runs are estimated using the k13d server timezone.", timeZone)
		}
	}

	location := time.Local
	return location, fmt.Sprintf("%s (cluster default assumed)", location.String()), "cluster default", true,
		"spec.timeZone is not set. Kubernetes uses the controller-manager timezone, so next runs are estimated using the k13d server timezone."
}

func firstContainerImage(containers []corev1.Container) string {
	if len(containers) == 0 {
		return "-"
	}
	if len(containers) == 1 {
		return containers[0].Image
	}
	return fmt.Sprintf("%s +%d more", containers[0].Image, len(containers)-1)
}

func timestampStringPtr(t *metav1.Time) string {
	if t == nil {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func timestampStringTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func timestampString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func timestampStringSlice(times []time.Time) []string {
	items := make([]string, 0, len(times))
	for _, ts := range times {
		if ts.IsZero() {
			continue
		}
		items = append(items, ts.Format(time.RFC3339))
	}
	return items
}

func derefInt32(v *int32) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func derefInt64(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
